package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	redisrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupOMSTestSuite(t *testing.T) (*service.OMSService, *service.AIOrderParserService, *postgresrepo.OMSRepository, *pgxpool.Pool) {
	t.Helper()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skipf("skipping OMS integration test, config error: %v", err)
	}
	// Force deterministic NLP fallback during unit/integration tests so tests never depend on external internet
	cfg.GeminiAPIKey = ""

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		t.Skipf("skipping OMS integration test, postgres unavailable: %v", err)
	}

	omsRepo := postgresrepo.NewOMSRepository(pgPool)
	migPath := filepath.Join("..", "..", "migrations", "000003_oms_schema.up.sql")
	if err := omsRepo.EnsureSchema(ctx, migPath); err != nil {
		pgPool.Close()
		t.Fatalf("failed to apply OMS migration in test: %v", err)
	}

	redisClient, _ := redisrepo.NewRedisClient(cfg)
	t.Cleanup(func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
		pgPool.Close()
	})

	queueRepo := redisrepo.NewQueueRepository(redisClient)
	aiParser := service.NewAIOrderParserService(omsRepo, cfg)
	omsSvc := service.NewOMSService(omsRepo, queueRepo, aiParser, cfg)
	omsSvc.SetRetryBackoff(15 * time.Millisecond)

	return omsSvc, aiParser, omsRepo, pgPool
}

// 1. Test Race Condition: 10 concurrent buyers competing for 1 last stock -> exact 1 succeeds (201), 9 rejected (409), stock = 0
func TestCriticalWorkflows_RaceCondition_10ConcurrentBuyers_1LastStock(t *testing.T) {
	omsSvc, _, _, _ := setupOMSTestSuite(t)
	ctx := context.Background()

	merchantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	productID := uuid.MustParse("a1111111-1111-1111-1111-111111111101")

	res, err := omsSvc.SimulateRaceCondition(ctx, &domain.RaceSimulationRequestDTO{
		MerchantID:       merchantID,
		ProductID:        productID,
		ConcurrentBuyers: 10,
		ResetStockTo:     1,
	})
	if err != nil {
		t.Fatalf("race condition simulation failed: %v", err)
	}

	if res.SuccessCount != 1 {
		t.Fatalf("expected exact 1 winning buyer (201 Created), got %d", res.SuccessCount)
	}
	if res.ConflictCount != 9 {
		t.Fatalf("expected exact 9 rejected buyers (409 Conflict), got %d", res.ConflictCount)
	}
	if res.FinalAvailable != 0 {
		t.Fatalf("expected final available_stock == 0 (never negative), got %d", res.FinalAvailable)
	}
	if !res.ZeroOversellProven {
		t.Fatalf("expected ZeroOversellProven == true")
	}
}

// 2. Test Webhook Idempotency: Send 3 duplicate payment.succeeded webhooks -> stock deducted ONCE & waybill generated
func TestCriticalWorkflows_WebhookIdempotency_And_WaybillGeneration(t *testing.T) {
	omsSvc, _, omsRepo, _ := setupOMSTestSuite(t)
	ctx := context.Background()

	merchantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	productID := uuid.MustParse("a1111111-1111-1111-1111-111111111102")

	// Ensure plenty of stock
	prodStart, err := omsRepo.SetAvailableStock(ctx, productID, 20)
	if err != nil {
		t.Fatalf("failed to set stock: %v", err)
	}

	ord, err := omsSvc.CreateOrder(ctx, &domain.CreateOrderRequestDTO{
		MerchantID:      merchantID,
		Channel:         domain.ChannelWeb,
		CustomerName:    "Test Buyer Idempotency",
		CustomerPhone:   "081277778888",
		ShippingAddress: "Jl. Asia Afrika No. 10, Bandung",
		CourierCode:     "JNE",
		Items:           []domain.CreateOrderItemDTO{{ProductID: productID, Qty: 2}},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	prodAfterReserve, _ := omsRepo.GetProductByID(ctx, productID)
	if prodAfterReserve.AvailableStock != 18 {
		t.Fatalf("expected available_stock 18 after reserving 2, got %d", prodAfterReserve.AvailableStock)
	}

	payloadBytes, _ := json.Marshal(domain.PaymentWebhookPayloadDTO{
		Provider:    "PAYMENT_GW",
		EventType:   "payment.succeeded",
		OrderID:     ord.ID.String(),
		OrderNumber: ord.OrderNumber,
		Amount:      ord.TotalAmount,
	})
	ts := time.Now().Unix()
	tsStr := strconv.FormatInt(ts, 10)
	sig := omsSvc.SignWebhookPayload(payloadBytes, ts)
	eventID := "evt-idem-test-" + uuid.New().String()[:8]

	// Send 3 identical webhooks
	for i := 0; i < 3; i++ {
		wRes, err := omsSvc.HandlePaymentWebhook(ctx, eventID, tsStr, sig, payloadBytes)
		if err != nil {
			t.Fatalf("webhook call %d failed unexpectedly: %v", i+1, err)
		}
		if i == 0 && (wRes.Idempotent || wRes.Status != "accepted") {
			t.Fatalf("expected 1st webhook to be accepted, got %+v", wRes)
		}
		if i > 0 && (!wRes.Idempotent || wRes.Status != "duplicate_ignored") {
			t.Fatalf("expected duplicate webhook #%d to return duplicate_ignored, got %+v", i+1, wRes)
		}
	}

	// Drain queue jobs (payment settlement + 3 downstream fulfillment queues)
	for i := 0; i < 10; i++ {
		if !omsSvc.ProcessNextQueueJob(ctx) {
			break
		}
	}

	prodAfterPay, _ := omsRepo.GetProductByID(ctx, productID)
	expectedSold := prodStart.SoldStock + 2
	if prodAfterPay.SoldStock != expectedSold {
		t.Fatalf("expected sold_stock to increase by exact 2 (to %d), got %d", expectedSold, prodAfterPay.SoldStock)
	}

	settledOrder, err := omsRepo.GetOrderByID(ctx, ord.ID)
	if err != nil {
		t.Fatalf("failed to fetch settled order: %v", err)
	}
	if settledOrder.Status != domain.OrderStatusReadyToShip {
		t.Fatalf("expected order status READY_TO_SHIP after waybill worker, got %s", settledOrder.Status)
	}
	if settledOrder.WaybillNumber == "" {
		t.Fatalf("expected non-empty generated waybill_number")
	}
}

// 3. Test Invalid & Expired Webhook Signatures -> Rejected 401 Unauthorized
func TestCriticalWorkflows_InvalidSignatureAndReplayAttack(t *testing.T) {
	omsSvc, _, _, _ := setupOMSTestSuite(t)
	ctx := context.Background()

	rawBody := []byte(`{"provider":"PAYMENT_GW","event_type":"payment.succeeded","order_number":"ORD-FAKE","amount":100000}`)
	nowTs := strconv.FormatInt(time.Now().Unix(), 10)

	// Tampered signature
	_, err := omsSvc.HandlePaymentWebhook(ctx, "evt-bad-sig-1", nowTs, "deadbeef0000invalidhexsignature", rawBody)
	if !errors.Is(err, service.ErrWebhookUnauthorized) {
		t.Fatalf("expected ErrWebhookUnauthorized for forged signature, got %v", err)
	}

	// Replay attack (> 5 minutes old timestamp)
	expiredUnix := time.Now().Add(-10 * time.Minute).Unix()
	expiredSig := omsSvc.SignWebhookPayload(rawBody, expiredUnix)
	_, err = omsSvc.HandlePaymentWebhook(ctx, "evt-replay-1", strconv.FormatInt(expiredUnix, 10), expiredSig, rawBody)
	if !errors.Is(err, service.ErrWebhookUnauthorized) {
		t.Fatalf("expected ErrWebhookUnauthorized for replay timestamp > 5m, got %v", err)
	}
}

// 4. Test Queue Exponential Backoff, Dead-Letter Queue (DLQ), and Manual Recovery
func TestCriticalWorkflows_QueueExponentialBackoff_And_DLQ(t *testing.T) {
	omsSvc, _, omsRepo, _ := setupOMSTestSuite(t)
	ctx := context.Background()

	merchantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	productID := uuid.MustParse("a1111111-1111-1111-1111-111111111105")
	_, _ = omsRepo.SetAvailableStock(ctx, productID, 15)

	ord, err := omsSvc.CreateOrder(ctx, &domain.CreateOrderRequestDTO{
		MerchantID:      merchantID,
		Channel:         domain.ChannelTikTokShop,
		CustomerName:    "DLQ Retry Tester",
		CustomerPhone:   "081122334455",
		ShippingAddress: "Jl. Thamrin No. 1, Jakarta Pusat",
		CourierCode:     "SICEPAT",
		Items:           []domain.CreateOrderItemDTO{{ProductID: productID, Qty: 1}},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// Enable persistent 3rd-party HTTP 429 failure so job exhausts all 4 attempts -> DLQ
	omsSvc.SetChaosMode(domain.ChaosModeAlways429)
	_, err = omsSvc.SimulatePayOrder(ctx, ord.ID)
	if err != nil {
		t.Fatalf("SimulatePayOrder failed: %v", err)
	}

	// Run worker cycles across backoff intervals until retries exhaust into DLQ
	for cycle := 0; cycle < 25; cycle++ {
		for drain := 0; drain < 5; drain++ {
			if !omsSvc.ProcessNextQueueJob(ctx) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	overview, err := omsSvc.GetDashboardOverview(ctx, merchantID.String(), "", "")
	if err != nil {
		t.Fatalf("GetDashboardOverview failed: %v", err)
	}
	if overview.QueueMetrics.FailedDLQCount == 0 {
		t.Fatalf("expected FailedDLQCount > 0 when 3rd-party API returns 429 across all 4 attempts")
	}

	// Recover via Chaos OFF + ManualRetryJob
	omsSvc.SetChaosMode(domain.ChaosModeOff)
	for _, j := range overview.QueueMetrics.RecentJobs {
		if j.Status == domain.JobStatusFailedDLQ {
			if _, retryErr := omsSvc.ManualRetryJob(ctx, j.ID); retryErr != nil {
				t.Fatalf("ManualRetryJob failed: %v", retryErr)
			}
		}
	}

	for i := 0; i < 10; i++ {
		omsSvc.ProcessNextQueueJob(ctx)
	}

	recoveredOrder, _ := omsRepo.GetOrderByID(ctx, ord.ID)
	if recoveredOrder.Status != domain.OrderStatusReadyToShip || recoveredOrder.WaybillNumber == "" {
		t.Fatalf("expected order to recover to READY_TO_SHIP with waybill after ManualRetry, got status=%s awb=%s",
			recoveredOrder.Status, recoveredOrder.WaybillNumber)
	}
}

// 5. Test AI Social Commerce Order Capture (WhatsApp Live Chat -> Structured Order + Insufficient Stock Alert)
func TestCriticalWorkflows_AISocialCommerceParser(t *testing.T) {
	_, aiParser, omsRepo, _ := setupOMSTestSuite(t)
	ctx := context.Background()

	merchantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	kaosID := uuid.MustParse("a1111111-1111-1111-1111-111111111101")
	// Set Kaos stock to 1 so requesting 2 pcs triggers red insufficient stock alert!
	_, _ = omsRepo.SetAvailableStock(ctx, kaosID, 1)

	chat := "Kak mau pesan Kaos Oversize Hitam XL 2 pcs sama Topi Baseball Navy 1 pcs, kirim ke Budi Santoso 081299887766, Jl. Kaliurang Km 5 No 12 Sleman via JNE REG ya"
	parsed, err := aiParser.ParseSocialCommerceChat(ctx, merchantID, chat)
	if err != nil {
		t.Fatalf("AI chat parse failed: %v", err)
	}

	if parsed.CustomerName != "Budi Santoso" {
		t.Errorf("expected customer_name 'Budi Santoso', got '%s'", parsed.CustomerName)
	}
	if parsed.CustomerPhone != "081299887766" {
		t.Errorf("expected customer_phone '081299887766', got '%s'", parsed.CustomerPhone)
	}
	if parsed.CourierCode != "JNE" {
		t.Errorf("expected courier_code 'JNE', got '%s'", parsed.CourierCode)
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("expected 2 matched items, got %d", len(parsed.Items))
	}
	if !parsed.HasInsufficientStock {
		t.Errorf("expected HasInsufficientStock == true when requesting 2 pcs with 1 available")
	}
	if parsed.LatencyMs >= 3000 {
		t.Errorf("expected parse latency < 3000ms, got %dms", parsed.LatencyMs)
	}
}

