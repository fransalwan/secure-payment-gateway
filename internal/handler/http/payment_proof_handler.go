package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/pkg/logger"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/rabbitmq"
	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type PaymentProofHandler struct {
	proofRepo  *postgresrepo.PaymentProofRepository
	producer   rabbitmq.Producer
	omsRepo    *postgresrepo.OMSRepository
	outboxRepo *postgresrepo.OutboxRepository
}

func NewPaymentProofHandler(
	proofRepo *postgresrepo.PaymentProofRepository,
	producer rabbitmq.Producer,
	omsRepo *postgresrepo.OMSRepository,
	outboxRepo *postgresrepo.OutboxRepository,
) *PaymentProofHandler {
	return &PaymentProofHandler{
		proofRepo:  proofRepo,
		producer:   producer,
		omsRepo:    omsRepo,
		outboxRepo: outboxRepo,
	}
}

// UploadProof handles both multipart file uploads and JSON/Base64 payloads.
// Returns Fast ACK 202 Accepted in < 20ms and persists event atomically via Transactional Outbox.
func (h *PaymentProofHandler) UploadProof(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := logger.GetTraceID(r.Context())
	if traceID == "" {
		traceID = uuid.New().String()
	}

	var orderID uuid.UUID
	var imgBytes []byte
	var fileName string
	var mimeType string

	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Limit upload size to 6 MB
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			response.BadRequest(w, "Upload exceeds 6MB size limit", "INVALID_FILE_SIZE", err.Error())
			return
		}

		orderIDStr := r.FormValue("order_id")
		parsedID, err := uuid.Parse(orderIDStr)
		if err != nil {
			response.BadRequest(w, "Valid 'order_id' (UUID) is required", "INVALID_ORDER_ID", err.Error())
			return
		}
		orderID = parsedID

		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			fileName = header.Filename
			mimeType = header.Header.Get("Content-Type")
			imgBytes, _ = io.ReadAll(file)
		} else {
			// Fallback to image_base64 form field
			b64 := r.FormValue("image_base64")
			if b64 != "" {
				fileName = "simulated_slip.jpg"
				mimeType = "image/jpeg"
				decoded, decErr := base64.StdEncoding.DecodeString(b64)
				if decErr == nil && len(decoded) > 0 {
					imgBytes = decoded
				} else {
					imgBytes = []byte(b64)
				}
			} else {
				response.BadRequest(w, "Field 'file' or 'image_base64' is required", "MISSING_FILE", err.Error())
				return
			}
		}
	} else {
		// Support JSON payload for direct API / Simulator calls
		var body struct {
			OrderID     string `json:"order_id"`
			ImageBase64 string `json:"image_base64"`
			FileName    string `json:"file_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.BadRequest(w, "Invalid JSON or multipart payload", "INVALID_PAYLOAD", err.Error())
			return
		}
		parsedID, err := uuid.Parse(body.OrderID)
		if err != nil {
			response.BadRequest(w, "Valid 'order_id' (UUID) is required", "INVALID_ORDER_ID", err.Error())
			return
		}
		orderID = parsedID
		fileName = body.FileName
		if fileName == "" {
			fileName = "receipt.jpg"
		}
		mimeType = "image/jpeg"
		if body.ImageBase64 != "" {
			decoded, decErr := base64.StdEncoding.DecodeString(body.ImageBase64)
			if decErr == nil && len(decoded) > 0 {
				imgBytes = decoded
			} else {
				imgBytes = []byte(body.ImageBase64)
			}
		}
	}

	if len(imgBytes) == 0 {
		imgBytes = []byte("simulated_bca_authentic_slip_payload")
	}

	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(imgBytes)
	}

	// Look up order to verify merchant and expected amount
	order, err := h.omsRepo.GetOrderByID(r.Context(), orderID)
	if err != nil {
		response.NotFound(w, "Order not found", err.Error())
		return
	}

	// 1. Calculate Perceptual Image Hash (dHash) to detect duplicate re-uploads
	pHash, _, _ := crypto.ComputeDHash(imgBytes)

	// Check if this merchant already received an identical or cropped duplicate slip
	var isDuplicateImage bool
	var duplicateReason string
	if pHash != "" {
		dupProof, distance, _ := h.proofRepo.FindNearDuplicatePHash(r.Context(), order.MerchantID, pHash, 5)
		if dupProof != nil {
			isDuplicateImage = true
			duplicateReason = fmt.Sprintf("Near-duplicate receipt detected via Perceptual Hash (Hamming distance %d to Proof %s). Quarantine applied.", distance, dupProof.ID.String()[:8])
			log.Warn().
				Str("order_id", orderID.String()).
				Str("phash", pHash).
				Int("distance", distance).
				Msg("Duplicate bank slip image detected via pHash")
		}
	}

	proofID := uuid.New()
	proofStatus := domain.ProofStatusPending
	if isDuplicateImage {
		proofStatus = domain.ProofStatusSuspiciousHold
	}

	proof := &domain.PaymentProof{
		ID:               proofID,
		OrderID:          orderID,
		MerchantID:       order.MerchantID,
		FileName:         fileName,
		ImagePHash:       pHash,
		IsDuplicateImage: isDuplicateImage,
		Status:           proofStatus,
		RejectionReason:  duplicateReason,
		CreatedAt:        time.Now().UTC(),
	}

	// Prepare RabbitMQ task payload
	b64Payload := base64.StdEncoding.EncodeToString(imgBytes)
	task := &domain.PaymentProofVerifyTaskPayload{
		TraceID:        traceID,
		ProofID:        proofID,
		OrderID:        orderID,
		MerchantID:     order.MerchantID,
		ImageBase64:    b64Payload,
		ImageMimeType:  mimeType,
		ImagePHash:     pHash,
		FileName:       fileName,
		ExpectedAmount: order.TotalAmount,
		Attempt:        1,
		CreatedAt:      time.Now().UTC(),
	}

	// 2. Transactional Outbox Pattern: Insert Proof & Outbox Event atomically
	outboxEvent, err := h.proofRepo.CreatePaymentProofWithOutbox(r.Context(), proof, h.outboxRepo, task)
	if err != nil {
		log.Error().Err(err).Msg("Failed to persist payment proof with outbox")
		response.InternalServerError(w, "Failed to register payment proof atomically")
		return
	}

	// 3. Publish to RabbitMQ Topic Exchange (with outbox mark on success)
	if h.producer != nil {
		if err := h.producer.PublishPaymentProofVerify(r.Context(), task); err != nil {
			log.Warn().Err(err).Msg("Direct RabbitMQ publish failed, will be polled by Outbox Worker")
		} else if outboxEvent != nil && h.outboxRepo != nil {
			_ = h.outboxRepo.MarkOutboxProcessed(r.Context(), outboxEvent.ID)
		}
	}

	elapsed := time.Since(start)

	log.Info().
		Str("trace_id", traceID).
		Str("proof_id", proofID.String()).
		Str("order_id", orderID.String()).
		Str("phash", pHash).
		Bool("duplicate_image", isDuplicateImage).
		Int64("latency_ms", elapsed.Milliseconds()).
		Msg("Payment proof accepted for async AI verification")

	response.Accepted(w, "Payment proof receipt uploaded and queued for async Vision AI verification", map[string]interface{}{
		"proof_id":           proofID.String(),
		"order_id":           orderID.String(),
		"order_number":       order.OrderNumber,
		"expected_total":     order.TotalAmount,
		"image_phash":        pHash,
		"is_duplicate_image": isDuplicateImage,
		"status":             proof.Status,
		"queue_status":       "QUEUED_RABBITMQ_OUTBOX",
		"latency_ms":         elapsed.Milliseconds(),
	})
}

// GetProof returns verification status and extracted vision metadata
func (h *PaymentProofHandler) GetProof(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	proofID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid proof ID (UUID)", "INVALID_ID", err.Error())
		return
	}

	proof, err := h.proofRepo.GetPaymentProof(r.Context(), proofID)
	if err != nil {
		response.NotFound(w, "Payment proof not found", err.Error())
		return
	}

	response.OK(w, "Payment proof details retrieved", proof)
}

// ListProofs returns recent payment proof audit logs
func (h *PaymentProofHandler) ListProofs(w http.ResponseWriter, r *http.Request) {
	merchantIDStr := r.URL.Query().Get("merchant_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var merchantID uuid.UUID
	if merchantIDStr != "" {
		merchantID, _ = uuid.Parse(merchantIDStr)
	}
	if merchantID == uuid.Nil {
		merchantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	}

	proofs, err := h.proofRepo.ListPaymentProofs(r.Context(), merchantID, limit)
	if err != nil {
		response.InternalServerError(w, "Failed to list payment proofs: "+err.Error())
		return
	}

	response.OK(w, "Payment proofs retrieved", proofs)
}
