package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OMSHandler struct {
	omsService *service.OMSService
	aiParser   *service.AIOrderParserService
}

func NewOMSHandler(omsService *service.OMSService, aiParser *service.AIOrderParserService) *OMSHandler {
	return &OMSHandler{
		omsService: omsService,
		aiParser:   aiParser,
	}
}

// AIParseOrder handles POST /api/v1/orders/ai-parse
func (h *OMSHandler) AIParseOrder(w http.ResponseWriter, r *http.Request) {
	var req domain.AIParseRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload", "INVALID_JSON", err.Error())
		return
	}

	if req.MerchantID == uuid.Nil {
		// Default to primary seeded merchant when omitted
		req.MerchantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	}

	parsed, err := h.aiParser.ParseSocialCommerceChat(r.Context(), req.MerchantID, req.RawText)
	if err != nil {
		response.BadRequest(w, "failed to parse social commerce chat", "AI_PARSE_ERROR", err.Error())
		return
	}

	response.OK(w, "WhatsApp/Live chat extracted into structured order draft", parsed)
}

// CreateOrder handles POST /api/v1/orders
func (h *OMSHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateOrderRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload", "INVALID_JSON", err.Error())
		return
	}

	if req.MerchantID == uuid.Nil {
		req.MerchantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	}

	ord, err := h.omsService.CreateOrder(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, postgresrepo.ErrInsufficientStock):
			response.Conflict(w, "Insufficient Stock", "INSUFFICIENT_STOCK", err.Error())
		case errors.Is(err, postgresrepo.ErrProductNotFound):
			response.BadRequest(w, "Product SKU not found", "SKU_NOT_FOUND", err.Error())
		case errors.Is(err, service.ErrInvalidOrderInput):
			response.BadRequest(w, "Invalid order payload", "VALIDATION_ERROR", err.Error())
		default:
			response.InternalServerError(w, err.Error())
		}
		return
	}

	response.Created(w, "Order created and stock reserved atomically", ord)
}

// CancelOrder handles POST /api/v1/orders/{id}/cancel
func (h *OMSHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		response.BadRequest(w, "invalid order UUID", "INVALID_ID", err.Error())
		return
	}

	ord, err := h.omsService.CancelOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrInvalidOrderTransition) {
			response.Conflict(w, "Cannot cancel order in current status", "INVALID_STATE_TRANSITION", err.Error())
			return
		}
		response.BadRequest(w, "failed to cancel order", "CANCEL_FAILED", err.Error())
		return
	}

	response.OK(w, "Order cancelled and reserved stock released to available inventory", ord)
}

// SimulatePayOrder handles POST /api/v1/orders/{id}/simulate-pay
func (h *OMSHandler) SimulatePayOrder(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		response.BadRequest(w, "invalid order UUID", "INVALID_ID", err.Error())
		return
	}

	res, err := h.omsService.SimulatePayOrder(r.Context(), orderID)
	if err != nil {
		response.BadRequest(w, "failed to dispatch payment webhook", "PAYMENT_SIM_ERROR", err.Error())
		return
	}

	response.OK(w, "Signed HMAC-SHA256 payment webhook dispatched", res)
}

// SetProductStock handles POST /api/v1/products/{id}/stock
func (h *OMSHandler) SetProductStock(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	productID, err := uuid.Parse(idParam)
	if err != nil {
		response.BadRequest(w, "invalid product UUID", "INVALID_ID", err.Error())
		return
	}

	var req domain.UpdateStockRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload", "INVALID_JSON", err.Error())
		return
	}

	prod, err := h.omsService.SetProductStock(r.Context(), productID, req.AvailableStock)
	if err != nil {
		response.BadRequest(w, "failed to update product stock", "STOCK_UPDATE_ERROR", err.Error())
		return
	}

	response.OK(w, "Product available stock updated", prod)
}

// ReceivePaymentWebhook handles POST /api/v1/webhooks/payment
func (h *OMSHandler) ReceivePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID := r.Header.Get("X-Webhook-Id")
	timestamp := r.Header.Get("X-Webhook-Timestamp")
	signature := r.Header.Get("X-Webhook-Signature")

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w, "failed to read webhook body", "BODY_READ_ERROR", err.Error())
		return
	}

	res, err := h.omsService.HandlePaymentWebhook(r.Context(), webhookID, timestamp, signature, rawBody)
	if err != nil {
		if errors.Is(err, service.ErrWebhookUnauthorized) {
			response.Unauthorized(w, "Invalid webhook signature or expired timestamp", err.Error())
			return
		}
		response.BadRequest(w, "Webhook validation failed", "WEBHOOK_ERROR", err.Error())
		return
	}

	// Return exact 200 OK fast ACK (< 50ms) including {"status": "duplicate_ignored"} when idempotent
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"status":        res.Status,
		"webhook_id":    res.WebhookID,
		"idempotent":    res.Idempotent,
		"queued_job_id": res.QueuedJobID,
		"processing_ms": res.ProcessingMs,
		"data":          res,
	})
}

// SimulateRaceCondition handles POST /api/v1/simulator/race-condition
func (h *OMSHandler) SimulateRaceCondition(w http.ResponseWriter, r *http.Request) {
	var req domain.RaceSimulationRequestDTO
	_ = json.NewDecoder(r.Body).Decode(&req)

	res, err := h.omsService.SimulateRaceCondition(r.Context(), &req)
	if err != nil {
		response.InternalServerError(w, err.Error())
		return
	}

	response.OK(w, "Concurrency anti-overselling stress test completed", res)
}

// SimulateDuplicateWebhook handles POST /api/v1/simulator/duplicate-webhook
func (h *OMSHandler) SimulateDuplicateWebhook(w http.ResponseWriter, r *http.Request) {
	var req domain.DuplicateWebhookSimRequestDTO
	_ = json.NewDecoder(r.Body).Decode(&req)

	res, err := h.omsService.SimulateDuplicateWebhook(r.Context(), &req)
	if err != nil {
		response.InternalServerError(w, err.Error())
		return
	}

	response.OK(w, "Duplicate payment webhook idempotency simulation completed", res)
}

// SetChaosMode handles POST /api/v1/simulator/chaos-mode
func (h *OMSHandler) SetChaosMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid JSON payload", "INVALID_JSON", err.Error())
		return
	}

	mode := domain.ChaosFailMode(strings.ToUpper(strings.TrimSpace(body.Mode)))
	switch mode {
	case domain.ChaosModeOff, domain.ChaosModeTransientTwice, domain.ChaosModeAlways429:
		h.omsService.SetChaosMode(mode)
	default:
		response.BadRequest(w, "mode must be OFF, TRANSIENT_TWICE, or ALWAYS_429", "INVALID_CHAOS_MODE", "")
		return
	}

	response.OK(w, "3rd-party failure chaos simulator updated", map[string]interface{}{
		"chaos_mode": mode,
	})
}

// ManualRetryJob handles POST /api/v1/queue/jobs/{id}/retry
func (h *OMSHandler) ManualRetryJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	job, err := h.omsService.ManualRetryJob(r.Context(), jobID)
	if err != nil {
		response.BadRequest(w, "failed to retry job", "JOB_RETRY_ERROR", err.Error())
		return
	}

	response.OK(w, "Failed DLQ job re-queued for background execution", job)
}

// GetDashboardOverview handles GET /api/v1/dashboard/overview
func (h *OMSHandler) GetDashboardOverview(w http.ResponseWriter, r *http.Request) {
	merchantID := r.URL.Query().Get("merchant_id")
	channel := r.URL.Query().Get("channel")
	status := r.URL.Query().Get("status")

	overview, err := h.omsService.GetDashboardOverview(r.Context(), merchantID, channel, status)
	if err != nil {
		response.InternalServerError(w, err.Error())
		return
	}

	response.OK(w, "Operational OMS dashboard overview fetched", overview)
}

