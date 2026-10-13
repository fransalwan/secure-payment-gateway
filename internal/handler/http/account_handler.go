package http

import (
	"encoding/json"
	"net/http"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AccountHandler struct {
	svc *service.AccountService
}

func NewAccountHandler(svc *service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var dto domain.CreateAccountDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.BadRequest(w, "invalid request body", "INVALID_JSON", err.Error())
		return
	}

	acc, err := h.svc.CreateAccount(r.Context(), &dto)
	if err != nil {
		response.BadRequest(w, err.Error(), "ACCOUNT_CREATION_FAILED", err.Error())
		return
	}

	response.Created(w, "account created successfully", acc)
}

func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid account id format", "INVALID_UUID", "id must be a valid UUID")
		return
	}

	acc, err := h.svc.GetAccount(r.Context(), id)
	if err != nil {
		response.JSON(w, http.StatusNotFound, false, "account not found", nil, &response.APIError{
			Code:    "ACCOUNT_NOT_FOUND",
			Details: err.Error(),
		})
		return
	}

	response.OK(w, "account retrieved successfully", acc)
}
