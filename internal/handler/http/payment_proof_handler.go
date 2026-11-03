package http

import (
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/pkg/logger"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/rabbitmq"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type PaymentProofHandler struct {
	proofRepo *postgresrepo.PaymentProofRepository
	producer  rabbitmq.Producer
	omsRepo   *postgresrepo.OMSRepository
}

func NewPaymentProofHandler(
	proofRepo *postgresrepo.PaymentProofRepository,
	producer rabbitmq.Producer,
	omsRepo *postgresrepo.OMSRepository,
) *PaymentProofHandler {
	return &PaymentProofHandler{
		proofRepo: proofRepo,
		producer:  producer,
		omsRepo:   omsRepo,
	}
}

// UploadProof handles multipart image upload and returns Fast ACK 202 Accepted in < 15ms
func (h *PaymentProofHandler) UploadProof(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := logger.GetTraceID(r.Context())
	if traceID == "" {
		traceID = uuid.New().String()
	}

	// Limit upload size to 6 MB
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		response.BadRequest(w, "Upload exceeds 6MB size limit", "INVALID_FILE_SIZE", err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Field 'file' is required (multipart/form-data)", "MISSING_FILE", err.Error())
		return
	}
	defer file.Close()

	orderIDStr := r.FormValue("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		response.BadRequest(w, "Valid 'order_id' (UUID) is required", "INVALID_ORDER_ID", err.Error())
		return
	}

	// Read image bytes
	imgBytes, err := io.ReadAll(file)
	if err != nil {
		response.InternalServerError(w, "Failed to read uploaded file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(imgBytes)
	}

	// Validate allowed image types
	if !strings.HasPrefix(mimeType, "image/") {
		response.BadRequest(w, "Only image files (JPEG, PNG, WebP) are accepted", "INVALID_FILE_TYPE", mimeType)
		return
	}

	// Look up order to verify merchant and total amount
	order, err := h.omsRepo.GetOrderByID(r.Context(), orderID)
	if err != nil {
		response.NotFound(w, "Order not found", err.Error())
		return
	}

	proofID := uuid.New()
	proof := &domain.PaymentProof{
		ID:         proofID,
		OrderID:    orderID,
		MerchantID: order.MerchantID,
		FileName:   header.Filename,
		Status:     domain.ProofStatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if err := h.proofRepo.CreatePaymentProof(r.Context(), proof); err != nil {
		response.InternalServerError(w, "Failed to register payment proof")
		return
	}

	// Enqueue asynchronous Vision AI validation task to RabbitMQ
	task := &domain.PaymentProofVerifyTaskPayload{
		TraceID:        traceID,
		ProofID:        proofID,
		OrderID:        orderID,
		MerchantID:     order.MerchantID,
		ImageBase64:    base64.StdEncoding.EncodeToString(imgBytes),
		ImageMimeType:  mimeType,
		FileName:       header.Filename,
		ExpectedAmount: order.TotalAmount,
		Attempt:        0,
		CreatedAt:      time.Now().UTC(),
	}

	if err := h.producer.PublishPaymentProofVerify(r.Context(), task); err != nil {
		log.Error().Err(err).Msg("Failed to enqueue payment proof to RabbitMQ")
	}

	elapsed := time.Since(start).Milliseconds()

	response.Accepted(w, "Payment proof receipt uploaded and queued for async Vision AI verification", map[string]interface{}{
		"proof_id":       proofID,
		"order_id":       orderID,
		"order_number":   order.OrderNumber,
		"expected_total": order.TotalAmount,
		"status":         domain.ProofStatusPending,
		"queue_status":   "QUEUED_RABBITMQ",
		"latency_ms":     elapsed,
	})
}

func (h *PaymentProofHandler) ListProofs(w http.ResponseWriter, r *http.Request) {
	merchantIDStr := r.URL.Query().Get("merchant_id")
	var merchantID uuid.UUID
	if merchantIDStr != "" {
		merchantID, _ = uuid.Parse(merchantIDStr)
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	proofs, err := h.proofRepo.ListPaymentProofs(r.Context(), merchantID, limit)
	if err != nil {
		response.InternalServerError(w, "Failed to list payment proofs")
		return
	}

	response.OK(w, "Payment proofs retrieved", proofs)
}

func (h *PaymentProofHandler) GetProof(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid proof ID", "INVALID_PROOF_ID", err.Error())
		return
	}

	proof, err := h.proofRepo.GetPaymentProof(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Payment proof not found", err.Error())
		return
	}

	response.OK(w, "Payment proof details retrieved", proof)
}
