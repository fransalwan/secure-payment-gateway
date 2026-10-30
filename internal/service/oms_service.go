package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	redisrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
	"github.com/google/uuid"
)

var (
	ErrWebhookUnauthorized = errors.New("webhook signature or timestamp verification failed")
	ErrInvalidOrderInput   = errors.New("invalid order creation payload")
)

type OMSService struct {
	omsRepo      *postgresrepo.OMSRepository
	queueRepo    *redisrepo.QueueRepository
	aiParser     *AIOrderParserService
	cfg          *config.Config
	mu           sync.RWMutex
	chaosMode    domain.ChaosFailMode
	retryBackoff time.Duration
}

func NewOMSService(
	omsRepo *postgresrepo.OMSRepository,
	queueRepo *redisrepo.QueueRepository,
	aiParser *AIOrderParserService,
	cfg *config.Config,
) *OMSService {
	return &OMSService{
		omsRepo:      omsRepo,
		queueRepo:    queueRepo,
		aiParser:     aiParser,
		cfg:          cfg,
		chaosMode:    domain.ChaosModeOff,
		retryBackoff: 2 * time.Second, // Matches AC: retry at 2s, 4s, 8s, 16s
	}
}

// SetRetryBackoff allows fast backoff intervals inside automated integration tests while keeping 2s in production.
func (s *OMSService) SetRetryBackoff(d time.Duration) {
	s.mu.Lock()
	s.retryBackoff = d
	s.mu.Unlock()
}

func (s *OMSService) SetChaosMode(mode domain.ChaosFailMode) {
	s.mu.Lock()
	s.chaosMode = mode
	s.mu.Unlock()
}

func (s *OMSService) GetChaosMode() domain.ChaosFailMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chaosMode
}

func (s *OMSService) CreateOrder(ctx context.Context, req *domain.CreateOrderRequestDTO) (*domain.Order, error) {
	if req == nil || req.MerchantID == uuid.Nil {
		return nil, fmt.Errorf("%w: merchant_id is required", ErrInvalidOrderInput)
	}
	if strings.TrimSpace(req.CustomerName) == "" || strings.TrimSpace(req.CustomerPhone) == "" {
		return nil, fmt.Errorf("%w: customer_name and customer_phone are required", ErrInvalidOrderInput)
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: at least one order item is required", ErrInvalidOrderInput)
	}
	for _, item := range req.Items {
		if item.ProductID == uuid.Nil || item.Qty <= 0 {
			return nil, fmt.Errorf("%w: each item must have a valid product_id and qty > 0", ErrInvalidOrderInput)
		}
	}

	if req.Channel == "" {
		req.Channel = domain.ChannelWeb
	}
	if req.CourierCode == "" {
		req.CourierCode = "JNE"
	}

	shortHash := strings.ToUpper(uuid.New().String()[:6])
	orderNumber := fmt.Sprintf("ORD-%s-%s", time.Now().UTC().Format("20060102"), shortHash)
	paymentURL := fmt.Sprintf("https://pay.spg-gateway.local/checkout/%s", orderNumber)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	return s.omsRepo.CreateOrderAtomic(ctx, req, orderNumber, paymentURL, expiresAt)
}

func (s *OMSService) CancelOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	return s.omsRepo.CancelOrderAtomic(ctx, orderID)
}

func (s *OMSService) SetProductStock(ctx context.Context, productID uuid.UUID, availableStock int) (*domain.Product, error) {
	return s.omsRepo.SetAvailableStock(ctx, productID, availableStock)
}

type WebhookResponseDTO struct {
	Status       string `json:"status"`
	WebhookID    string `json:"webhook_id"`
	Idempotent   bool   `json:"idempotent"`
	QueuedJobID  string `json:"queued_job_id,omitempty"`
	ProcessingMs int64  `json:"processing_ms"`
}

// HandlePaymentWebhook verifies HMAC-SHA256 + 5-min anti-replay timestamp, enforces atomic DB idempotency
// via webhook_events UNIQUE(provider, external_event_id), pushes async settlement to Redis Queue, and ACKs < 50ms.
func (s *OMSService) HandlePaymentWebhook(
	ctx context.Context,
	webhookIDHeader string,
	timestampHeader string,
	signatureHeader string,
	rawBody []byte,
) (*WebhookResponseDTO, error) {
	start := time.Now()

	if strings.TrimSpace(webhookIDHeader) == "" || strings.TrimSpace(timestampHeader) == "" || strings.TrimSpace(signatureHeader) == "" {
		return nil, fmt.Errorf("%w: X-Webhook-Id, X-Webhook-Timestamp, and X-Webhook-Signature headers are mandatory", ErrWebhookUnauthorized)
	}

	tsUnix, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid X-Webhook-Timestamp format", ErrWebhookUnauthorized)
	}

	tolerance := s.cfg.WebhookTimestampTolerance
	if tolerance <= 0 {
		tolerance = 300
	}
	if !crypto.VerifyTimestamp(tsUnix, tolerance) {
		return nil, fmt.Errorf("%w: webhook timestamp outside 5-minute anti-replay window", ErrWebhookUnauthorized)
	}

	signedMessage := []byte(fmt.Sprintf("%d.%s", tsUnix, string(rawBody)))
	if !crypto.VerifyHMACSignature(signedMessage, signatureHeader, s.cfg.WebhookSecretKey) {
		return nil, fmt.Errorf("%w: HMAC-SHA256 signature mismatch", ErrWebhookUnauthorized)
	}

	var payload domain.PaymentWebhookPayloadDTO
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, fmt.Errorf("invalid webhook JSON body: %w", err)
	}
	if payload.Provider == "" {
		payload.Provider = "PAYMENT_GW"
	}
	if payload.EventType == "" {
		payload.EventType = "payment.succeeded"
	}

	// Database-level Idempotency Guard
	eventDBID, isDuplicate, err := s.omsRepo.TryInsertWebhookEvent(
		ctx,
		payload.Provider,
		webhookIDHeader,
		payload.EventType,
		rawBody,
	)
	if err != nil {
		return nil, err
	}

	if isDuplicate {
		elapsed := time.Since(start).Milliseconds()
		return &WebhookResponseDTO{
			Status:       "duplicate_ignored",
			WebhookID:    webhookIDHeader,
			Idempotent:   true,
			ProcessingMs: elapsed,
		}, nil
	}

	// Enqueue asynchronous payment settlement job to Redis Queue for < 50ms fast ACK
	jobPayload, _ := json.Marshal(map[string]interface{}{
		"webhook_db_id": eventDBID.String(),
		"event_type":    payload.EventType,
		"order_id":      payload.OrderID,
		"order_number":  payload.OrderNumber,
		"amount":        payload.Amount,
	})

	jobID := fmt.Sprintf("job-pay-%s", uuid.New().String()[:8])
	_ = s.queueRepo.Enqueue(ctx, &domain.QueueJob{
		ID:          jobID,
		QueueName:   domain.QueuePaymentWebhook,
		OrderID:     payload.OrderID,
		OrderNumber: payload.OrderNumber,
		Payload:     jobPayload,
		MaxAttempts: 4,
	})

	elapsed := time.Since(start).Milliseconds()
	return &WebhookResponseDTO{
		Status:       "accepted",
		WebhookID:    webhookIDHeader,
		Idempotent:   false,
		QueuedJobID:  jobID,
		ProcessingMs: elapsed,
	}, nil
}

// SignWebhookPayload helper builds valid HMAC-SHA256 headers for merchant payment simulation & automated tests.
func (s *OMSService) SignWebhookPayload(rawBody []byte, tsUnix int64) string {
	msg := []byte(fmt.Sprintf("%d.%s", tsUnix, string(rawBody)))
	return crypto.GenerateHMACSignature(msg, s.cfg.WebhookSecretKey)
}

// SimulatePayOrder dispatches a genuinely signed payment.succeeded webhook for an existing PENDING_PAYMENT order.
func (s *OMSService) SimulatePayOrder(ctx context.Context, orderID uuid.UUID) (*WebhookResponseDTO, error) {
	ord, err := s.omsRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	payload := domain.PaymentWebhookPayloadDTO{
		Provider:    "PAYMENT_GW",
		EventType:   "payment.succeeded",
		OrderID:     ord.ID.String(),
		OrderNumber: ord.OrderNumber,
		Amount:      ord.TotalAmount,
		PaidAt:      time.Now().UTC().Format(time.RFC3339),
	}
	rawBody, _ := json.Marshal(payload)
	ts := time.Now().Unix()
	sig := s.SignWebhookPayload(rawBody, ts)
	webhookID := fmt.Sprintf("evt-pay-%s", ord.OrderNumber)

	return s.HandlePaymentWebhook(ctx, webhookID, strconv.FormatInt(ts, 10), sig, rawBody)
}

// StartBackgroundWorker runs the asynchronous Redis Queue consumer loop.
func (s *OMSService) StartBackgroundWorker(ctx context.Context) {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	slog.Info("OMS Redis Background Queue Worker started", "backend", s.queueRepo.BackendName())
	for {
		select {
		case <-ctx.Done():
			slog.Info("OMS Redis Background Queue Worker stopped")
			return
		case <-ticker.C:
			for i := 0; i < 8; i++ {
				processed := s.ProcessNextQueueJob(ctx)
				if !processed {
					break
				}
			}
		}
	}
}

// ProcessNextQueueJob executes one due job from the Redis Queue with Exponential Backoff & Dead-Letter Queue logging.
func (s *OMSService) ProcessNextQueueJob(ctx context.Context) bool {
	job, err := s.queueRepo.PopNextDueJob(ctx)
	if err != nil || job == nil {
		return false
	}

	execErr := s.executeQueueJob(ctx, job)
	if execErr == nil {
		s.queueRepo.MarkCompleted(ctx, job.ID)
		_ = s.omsRepo.ResolveSystemErrorByJobID(ctx, job.ID)
		return true
	}

	s.mu.RLock()
	baseBackoff := s.retryBackoff
	s.mu.RUnlock()

	movedToDLQ, updatedJob := s.queueRepo.FailOrScheduleRetry(ctx, job.ID, execErr.Error(), baseBackoff)
	if movedToDLQ && updatedJob != nil {
		slog.Error("Job exhausted retries and moved to Dead-Letter Queue (DLQ)",
			"job_id", updatedJob.ID,
			"queue", updatedJob.QueueName,
			"attempts", updatedJob.Attempts,
			"error", execErr,
		)
		_ = s.omsRepo.LogSystemError(ctx, &domain.SystemErrorLog{
			ServiceName:  "oms-background-worker",
			QueueName:    updatedJob.QueueName,
			JobID:        updatedJob.ID,
			ReferenceID:  updatedJob.OrderNumber,
			ErrorCode:    "DLQ_MAX_RETRIES_EXCEEDED",
			ErrorMessage: execErr.Error(),
			Attempts:     updatedJob.Attempts,
			Payload:      updatedJob.Payload,
		})
	}

	return true
}

func (s *OMSService) executeQueueJob(ctx context.Context, job *domain.QueueJob) error {
	chaos := s.GetChaosMode()

	switch job.QueueName {
	case domain.QueuePaymentWebhook:
		var data struct {
			WebhookDBID string `json:"webhook_db_id"`
			EventType   string `json:"event_type"`
			OrderID     string `json:"order_id"`
			OrderNumber string `json:"order_number"`
		}
		if err := json.Unmarshal(job.Payload, &data); err != nil {
			return err
		}

		start := time.Now()
		ord, alreadyPaid, err := s.omsRepo.MarkOrderPaidAtomic(ctx, data.OrderID, data.OrderNumber)
		if err != nil {
			if wid, e := uuid.Parse(data.WebhookDBID); e == nil {
				_ = s.omsRepo.UpdateWebhookEventStatus(ctx, wid, domain.WebhookStatusFailed, int(time.Since(start).Milliseconds()), err.Error())
			}
			return err
		}

		if wid, e := uuid.Parse(data.WebhookDBID); e == nil {
			status := domain.WebhookStatusProcessed
			if alreadyPaid {
				status = domain.WebhookStatusIgnoredDuplicate
			}
			_ = s.omsRepo.UpdateWebhookEventStatus(ctx, wid, status, int(time.Since(start).Milliseconds()), "")
		}

		if !alreadyPaid && ord != nil {
			// Fan-out to the 3 post-payment 3rd-party fulfillment queues (FR-04)
			downstreamPayload, _ := json.Marshal(map[string]interface{}{
				"order_id":       ord.ID.String(),
				"order_number":   ord.OrderNumber,
				"courier_code":   ord.CourierCode,
				"customer_phone": ord.CustomerPhone,
			})

			_ = s.queueRepo.Enqueue(ctx, &domain.QueueJob{
				ID:          fmt.Sprintf("job-awb-%s", ord.ID.String()[:8]),
				QueueName:   domain.QueueLogisticsWaybill,
				OrderID:     ord.ID.String(),
				OrderNumber: ord.OrderNumber,
				Payload:     downstreamPayload,
				MaxAttempts: 4,
			})
			_ = s.queueRepo.Enqueue(ctx, &domain.QueueJob{
				ID:          fmt.Sprintf("job-sync-%s", ord.ID.String()[:8]),
				QueueName:   domain.QueueChannelStockSync,
				OrderID:     ord.ID.String(),
				OrderNumber: ord.OrderNumber,
				Payload:     downstreamPayload,
				MaxAttempts: 4,
			})
			_ = s.queueRepo.Enqueue(ctx, &domain.QueueJob{
				ID:          fmt.Sprintf("job-wa-%s", ord.ID.String()[:8]),
				QueueName:   domain.QueueNotification,
				OrderID:     ord.ID.String(),
				OrderNumber: ord.OrderNumber,
				Payload:     downstreamPayload,
				MaxAttempts: 4,
			})
		}
		return nil

	case domain.QueueLogisticsWaybill:
		if chaos == domain.ChaosModeAlways429 {
			return fmt.Errorf("HTTP 429 Too Many Requests: 3rd-Party Logistics Waybill API rate limit exceeded (attempt %d/%d)", job.Attempts, job.MaxAttempts)
		}
		if chaos == domain.ChaosModeTransientTwice && job.Attempts <= 2 {
			return fmt.Errorf("HTTP 429 Too Many Requests: Transient Logistics Gateway timeout (attempt %d/%d - auto-retrying)", job.Attempts, job.MaxAttempts)
		}

		var data struct {
			OrderID     string `json:"order_id"`
			OrderNumber string `json:"order_number"`
			CourierCode string `json:"courier_code"`
		}
		if err := json.Unmarshal(job.Payload, &data); err != nil {
			return err
		}
		ordUUID, err := uuid.Parse(data.OrderID)
		if err != nil {
			return err
		}
		courier := strings.ToUpper(data.CourierCode)
		if courier == "" {
			courier = "JNE"
		}
		awb := fmt.Sprintf("%s-AWB-%s-%s", courier, time.Now().UTC().Format("20060102"), strings.ToUpper(uuid.New().String()[:6]))
		return s.omsRepo.AttachWaybillAndReadyToShip(ctx, ordUUID, courier, awb)

	case domain.QueueChannelStockSync:
		if chaos == domain.ChaosModeAlways429 {
			return fmt.Errorf("HTTP 429 Too Many Requests: Marketplace Shopee/TikTok Stock Sync webhook throttled (attempt %d/%d)", job.Attempts, job.MaxAttempts)
		}
		if chaos == domain.ChaosModeTransientTwice && job.Attempts == 1 {
			return fmt.Errorf("HTTP 503 Service Unavailable: Marketplace Stock Sync temporary gateway error (attempt 1)")
		}
		return nil

	case domain.QueueNotification:
		return nil

	default:
		return nil
	}
}

func (s *OMSService) ManualRetryJob(ctx context.Context, jobID string) (*domain.QueueJob, error) {
	return s.queueRepo.ManualRetry(ctx, jobID)
}

// SimulateRaceCondition executes N concurrent buyers competing at the exact same microsecond for 1 last stock!
func (s *OMSService) SimulateRaceCondition(
	ctx context.Context,
	req *domain.RaceSimulationRequestDTO,
) (*domain.RaceSimulationResultDTO, error) {
	buyers := req.ConcurrentBuyers
	if buyers <= 0 {
		buyers = 10
	}
	if buyers > 50 {
		buyers = 50
	}
	initialStock := req.ResetStockTo
	if initialStock <= 0 {
		initialStock = 1
	}

	merchantID := req.MerchantID
	if merchantID == uuid.Nil {
		merchants, err := s.omsRepo.ListMerchants(ctx)
		if err != nil || len(merchants) == 0 {
			return nil, fmt.Errorf("no merchants seeded")
		}
		merchantID = merchants[0].ID
	}

	productID := req.ProductID
	if productID == uuid.Nil {
		prods, err := s.omsRepo.ListProductsByMerchant(ctx, merchantID)
		if err != nil || len(prods) == 0 {
			return nil, fmt.Errorf("no products available for merchant")
		}
		productID = prods[0].ID
	}

	prodBefore, err := s.omsRepo.SetAvailableStock(ctx, productID, initialStock)
	if err != nil {
		return nil, err
	}

	channels := []domain.OrderChannel{
		domain.ChannelWeb,
		domain.ChannelShopee,
		domain.ChannelTikTokShop,
		domain.ChannelWhatsAppLive,
	}

	attempts := make([]domain.RaceBuyerAttemptDTO, buyers)
	startGate := make(chan struct{})
	var wg sync.WaitGroup
	overallStart := time.Now()

	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startGate // All goroutines fire simultaneously when startGate closes!

			reqStart := time.Now()
			ch := channels[idx%len(channels)]
			buyerName := fmt.Sprintf("FlashBuyer #%02d", idx+1)

			ord, err := s.CreateOrder(ctx, &domain.CreateOrderRequestDTO{
				MerchantID:      merchantID,
				Channel:         ch,
				ExternalOrderID: fmt.Sprintf("RACE-%d-%d", time.Now().UnixMilli(), idx+1),
				CustomerName:    buyerName,
				CustomerPhone:   fmt.Sprintf("08129900%04d", idx+1),
				ShippingAddress: "Jl. Sudirman Kav 52-53, SCBD, Jakarta Selatan",
				CourierCode:     "JNE",
				Items: []domain.CreateOrderItemDTO{
					{ProductID: productID, Qty: 1},
				},
			})
			latencyMs := float64(time.Since(reqStart).Microseconds()) / 1000.0

			if err == nil && ord != nil {
				attempts[idx] = domain.RaceBuyerAttemptDTO{
					BuyerIndex:  idx + 1,
					BuyerName:   buyerName,
					Channel:     string(ch),
					HTTPStatus:  201,
					Outcome:     "SUCCESS_RESERVED",
					OrderNumber: ord.OrderNumber,
					Reason:      "PostgreSQL FOR UPDATE lock acquired -> 1 stock reserved",
					LatencyMs:   latencyMs,
				}
			} else {
				attempts[idx] = domain.RaceBuyerAttemptDTO{
					BuyerIndex: idx + 1,
					BuyerName:  buyerName,
					Channel:    string(ch),
					HTTPStatus: 409,
					Outcome:    "REJECTED_CONFLICT",
					Reason:     "409 Conflict — Insufficient Stock (Row-Level Lock Guard)",
					LatencyMs:  latencyMs,
				}
			}
		}(i)
	}

	// Release all 10 concurrent buyers simultaneously!
	close(startGate)
	wg.Wait()
	totalMs := float64(time.Since(overallStart).Microseconds()) / 1000.0

	prodAfter, err := s.omsRepo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	var successCount, conflictCount int
	for _, a := range attempts {
		if a.HTTPStatus == 201 {
			successCount++
		} else {
			conflictCount++
		}
	}

	return &domain.RaceSimulationResultDTO{
		ProductSKU:         prodBefore.SKU,
		ProductName:        prodBefore.Name,
		InitialStock:       initialStock,
		ConcurrentBuyers:   buyers,
		SuccessCount:       successCount,
		ConflictCount:      conflictCount,
		FinalAvailable:     prodAfter.AvailableStock,
		FinalReserved:      prodAfter.ReservedStock,
		ZeroOversellProven: successCount == initialStock && prodAfter.AvailableStock == 0,
		TotalDurationMs:    totalMs,
		Attempts:           attempts,
	}, nil
}

// SimulateDuplicateWebhook fires 5 identical signed webhooks with the exact same X-Webhook-Id
// to prove HMAC-SHA256 verification, <50ms ACK latency, and 100% Idempotent deduplication.
func (s *OMSService) SimulateDuplicateWebhook(
	ctx context.Context,
	req *domain.DuplicateWebhookSimRequestDTO,
) (*domain.DuplicateWebhookSimResultDTO, error) {
	count := req.DuplicateCount
	if count <= 0 {
		count = 5
	}

	merchantID := req.MerchantID
	if merchantID == uuid.Nil {
		merchants, err := s.omsRepo.ListMerchants(ctx)
		if err != nil || len(merchants) == 0 {
			return nil, fmt.Errorf("no merchants seeded")
		}
		merchantID = merchants[0].ID
	}

	var targetOrder *domain.Order
	if req.OrderID != "" {
		if parsed, err := uuid.Parse(req.OrderID); err == nil {
			targetOrder, _ = s.omsRepo.GetOrderByID(ctx, parsed)
		}
	}

	// If no specific PENDING_PAYMENT order was supplied, find one or create a fresh demo order automatically
	if targetOrder == nil || targetOrder.Status != domain.OrderStatusPendingPayment {
		pendingOrders, _ := s.omsRepo.ListOrders(ctx, merchantID, "", string(domain.OrderStatusPendingPayment), 1)
		if len(pendingOrders) > 0 {
			targetOrder = &pendingOrders[0]
		} else {
			prods, err := s.omsRepo.ListProductsByMerchant(ctx, merchantID)
			if err != nil || len(prods) == 0 {
				return nil, fmt.Errorf("no products available to create demo order")
			}
			// Pick a product with stock >= 1, or restock the second SKU
			chosen := prods[0]
			for _, p := range prods {
				if p.AvailableStock >= 1 {
					chosen = p
					break
				}
			}
			if chosen.AvailableStock < 1 {
				updated, err := s.omsRepo.SetAvailableStock(ctx, chosen.ID, 5)
				if err == nil {
					chosen = *updated
				}
			}
			created, err := s.CreateOrder(ctx, &domain.CreateOrderRequestDTO{
				MerchantID:      merchantID,
				Channel:         domain.ChannelShopee,
				ExternalOrderID: fmt.Sprintf("SHP-SIM-%d", time.Now().Unix()%100000),
				CustomerName:    "Santoso Wibowo",
				CustomerPhone:   "081388990011",
				ShippingAddress: "Jl. Gatot Subroto No. 21, Jakarta Selatan",
				CourierCode:     "SICEPAT",
				Items:           []domain.CreateOrderItemDTO{{ProductID: chosen.ID, Qty: 1}},
			})
			if err != nil {
				return nil, err
			}
			targetOrder = created
		}
	}

	externalEventID := fmt.Sprintf("evt-dup-%s-%d", targetOrder.OrderNumber, time.Now().Unix()%10000)
	payload := domain.PaymentWebhookPayloadDTO{
		Provider:    "PAYMENT_GW",
		EventType:   "payment.succeeded",
		OrderID:     targetOrder.ID.String(),
		OrderNumber: targetOrder.OrderNumber,
		Amount:      targetOrder.TotalAmount,
		PaidAt:      time.Now().UTC().Format(time.RFC3339),
	}
	rawBody, _ := json.Marshal(payload)
	ts := time.Now().Unix()
	tsHeader := strconv.FormatInt(ts, 10)
	sig := s.SignWebhookPayload(rawBody, ts)

	attempts := make([]domain.WebhookAttemptDTO, count)
	var processedCount, duplicateCount int

	for i := 0; i < count; i++ {
		callStart := time.Now()
		res, err := s.HandlePaymentWebhook(ctx, externalEventID, tsHeader, sig, rawBody)
		latencyMs := float64(time.Since(callStart).Microseconds()) / 1000.0

		if err != nil {
			attempts[i] = domain.WebhookAttemptDTO{
				AttemptIndex:  i + 1,
				WebhookID:     externalEventID,
				HTTPStatus:    401,
				StatusResult:  "ERROR: " + err.Error(),
				LatencyMs:     latencyMs,
				SignatureUsed: sig[:16] + "...",
			}
			continue
		}

		if res.Idempotent {
			duplicateCount++
		} else {
			processedCount++
		}

		attempts[i] = domain.WebhookAttemptDTO{
			AttemptIndex:  i + 1,
			WebhookID:     externalEventID,
			HTTPStatus:    200,
			StatusResult:  res.Status,
			LatencyMs:     latencyMs,
			SignatureUsed: sig[:16] + "...",
		}
	}

	// Process the queued payment settlement job immediately so caller sees updated order status
	s.ProcessNextQueueJob(ctx)

	updatedOrder, _ := s.omsRepo.GetOrderByID(ctx, targetOrder.ID)
	finalStatus := domain.OrderStatusPaid
	if updatedOrder != nil {
		finalStatus = updatedOrder.Status
	}

	return &domain.DuplicateWebhookSimResultDTO{
		OrderID:           targetOrder.ID.String(),
		OrderNumber:       targetOrder.OrderNumber,
		ExternalEventID:   externalEventID,
		TotalSent:         count,
		ProcessedCount:    processedCount,
		DuplicateIgnored:  duplicateCount,
		IdempotencyProven: processedCount == 1 && duplicateCount == (count-1),
		OrderFinalStatus:  finalStatus,
		Attempts:          attempts,
	}, nil
}

func (s *OMSService) GetDashboardOverview(
	ctx context.Context,
	merchantIDStr string,
	channelFilter string,
	statusFilter string,
) (*domain.DashboardOverviewDTO, error) {
	merchants, err := s.omsRepo.ListMerchants(ctx)
	if err != nil {
		return nil, err
	}

	var activeMerchant *domain.Merchant
	if parsedID, err := uuid.Parse(merchantIDStr); err == nil {
		for i := range merchants {
			if merchants[i].ID == parsedID {
				activeMerchant = &merchants[i]
				break
			}
		}
	}
	if activeMerchant == nil && merchantIDStr != "" {
		for i := range merchants {
			if strings.EqualFold(merchants[i].Code, merchantIDStr) {
				activeMerchant = &merchants[i]
				break
			}
		}
	}
	if activeMerchant == nil && len(merchants) > 0 {
		activeMerchant = &merchants[0]
	}

	var products []domain.Product
	var orders []domain.Order
	if activeMerchant != nil {
		products, _ = s.omsRepo.ListProductsByMerchant(ctx, activeMerchant.ID)
		orders, _ = s.omsRepo.ListOrders(ctx, activeMerchant.ID, channelFilter, statusFilter, 50)
	}

	webhookEvents, _ := s.omsRepo.ListWebhookEvents(ctx, 25)
	errorLogs, _ := s.omsRepo.ListSystemErrors(ctx, 20)
	queueMetrics := s.queueRepo.GetMetrics(s.GetChaosMode())

	return &domain.DashboardOverviewDTO{
		Merchants:      merchants,
		ActiveMerchant: activeMerchant,
		Products:       products,
		Orders:         orders,
		WebhookEvents:  webhookEvents,
		QueueMetrics:   queueMetrics,
		ErrorLogs:      errorLogs,
	}, nil
}

