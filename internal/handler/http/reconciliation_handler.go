package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	pkgrecon "github.com/fransalwan/secure-payment-gateway/pkg/reconciliation"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ReconciliationHandler struct {
	svc *service.ReconciliationService
}

func NewReconciliationHandler(svc *service.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{svc: svc}
}

// UploadAndReconcile handles CSV file upload or raw CSV text body
func (h *ReconciliationHandler) UploadAndReconcile(w http.ResponseWriter, r *http.Request) {
	bankName := r.URL.Query().Get("bank")
	if bankName == "" {
		bankName = "BCA"
	}

	var records []*domain.BankSettlementRecord
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Multipart file upload
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
			response.BadRequest(w, "failed to parse multipart form", "INVALID_FORM", err.Error())
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			response.BadRequest(w, "missing 'file' in multipart form", "MISSING_FILE", err.Error())
			return
		}
		defer file.Close()
		records, err = pkgrecon.ParseBankSettlementCSV(file)
	} else {
		// Raw CSV body
		records, err = pkgrecon.ParseBankSettlementCSV(r.Body)
	}

	if err != nil {
		response.BadRequest(w, "failed to parse settlement CSV", "CSV_PARSE_ERROR", err.Error())
		return
	}

	batch, err := h.svc.ReconcileBankSettlement(r.Context(), bankName, records)
	if err != nil {
		response.InternalServerError(w, fmt.Sprintf("reconciliation process failed: %v", err))
		return
	}

	response.OK(w, "bank reconciliation completed successfully", batch)
}

func (h *ReconciliationHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.svc.ListBatches(r.Context())
	if err != nil {
		response.InternalServerError(w, "failed to list reconciliation batches")
		return
	}
	response.OK(w, "reconciliation batches retrieved", batches)
}

func (h *ReconciliationHandler) GetBatchDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid batch id format", "INVALID_UUID", "id must be UUID")
		return
	}

	detail, err := h.svc.GetBatchDetail(r.Context(), id)
	if err != nil {
		response.JSON(w, http.StatusNotFound, false, "batch not found", nil, &response.APIError{
			Code:    "BATCH_NOT_FOUND",
			Details: err.Error(),
		})
		return
	}

	response.OK(w, "reconciliation batch detail retrieved", detail)
}


