package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/google/uuid"
)

type TransferHandler struct {
	svc *service.TransferService
}

func NewTransferHandler(svc *service.TransferService) *TransferHandler {
	return &TransferHandler{svc: svc}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		response.BadRequest(w, "missing required header", "MISSING_IDEMPOTENCY_KEY", "X-Idempotency-Key header is mandatory for transfers")
		return
	}

	var req domain.TransferRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body", "INVALID_JSON", err.Error())
		return
	}

	result, err := h.svc.TransferFunds(r.Context(), &req, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConcurrentTransaction):
			response.Conflict(w, "transaction is already in progress", "CONCURRENT_TRANSACTION", err.Error())
		case errors.Is(err, postgres.ErrInsufficientBalance):
			response.JSON(w, http.StatusUnprocessableEntity, false, "insufficient account balance", nil, &response.APIError{
				Code:    "INSUFFICIENT_BALANCE",
				Details: "source account has inadequate funds for this transfer",
			})
		case errors.Is(err, postgres.ErrSelfTransferNotAllowed):
			response.BadRequest(w, "source and destination accounts must be different", "SELF_TRANSFER_DISALLOWED", err.Error())
		default:
			response.BadRequest(w, "transfer execution failed", "TRANSFER_FAILED", err.Error())
		}
		return
	}

	response.OK(w, "transfer processed and settled successfully", result)
}

func (h *TransferHandler) ListLedger(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.ListLedgerEntries(r.Context())
	if err != nil {
		response.InternalServerError(w, "failed to list ledger entries")
		return
	}
	response.OK(w, "ledger entries retrieved successfully", entries)
}

func (h *TransferHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.svc.ListAuditLogs(r.Context())
	if err != nil {
		response.InternalServerError(w, "failed to list audit logs")
		return
	}
	response.OK(w, "audit logs retrieved successfully", logs)
}

// SimulateSettlementTransfer handles POST /api/v1/simulator/ledger-transfer
// Executes a real ACID Double-Entry transfer with deterministic lock ordering & SHA-256 hash-chained audit log.
func (h *TransferHandler) SimulateSettlementTransfer(w http.ResponseWriter, r *http.Request) {
	var req domain.TransferRequestDTO
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.SourceAccountID == uuid.Nil {
		req.SourceAccountID = uuid.MustParse("e0000000-0000-0000-0000-000000000001") // ACC-ESCROW-GATEWAY
	}
	if req.DestinationAccountID == uuid.Nil {
		req.DestinationAccountID = uuid.MustParse("e1111111-1111-1111-1111-111111111111") // ACC-MERCH-8801
	}
	if req.Amount <= 0 {
		req.Amount = 275000
	}
	if req.Currency == "" {
		req.Currency = "IDR"
	}
	if req.ReferenceID == "" {
		req.ReferenceID = fmt.Sprintf("REF-BCA-SIM-%d", time.Now().Unix()%100000)
	}
	if req.Description == "" {
		req.Description = "Simulated Instant Escrow -> Merchant Settlement Transfer"
	}

	idemKey := fmt.Sprintf("idem-sim-%s", req.ReferenceID)
	result, err := h.svc.TransferFunds(r.Context(), &req, idemKey)
	if err != nil {
		response.BadRequest(w, "simulated double-entry transfer failed", "TRANSFER_SIM_FAILED", err.Error())
		return
	}

	response.OK(w, "Atomic Double-Entry transfer & SHA-256 hash-chained audit log recorded", result)
}



