package domain

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "PENDING"
	TransactionStatusSettled TransactionStatus = "SETTLED"
	TransactionStatusFailed  TransactionStatus = "FAILED"
)

type Transaction struct {
	ID                   uuid.UUID         `json:"id"`
	ReferenceID          string            `json:"reference_id"`
	IdempotencyKey       string            `json:"idempotency_key"`
	SourceAccountID      uuid.UUID         `json:"source_account_id"`
	DestinationAccountID uuid.UUID         `json:"destination_account_id"`
	Amount               int64             `json:"amount"`
	Currency             string            `json:"currency"`
	Status               TransactionStatus `json:"status"`
	Description          string            `json:"description"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

type TransferRequestDTO struct {
	ReferenceID          string    `json:"reference_id"`
	SourceAccountID      uuid.UUID `json:"source_account_id"`
	DestinationAccountID uuid.UUID `json:"destination_account_id"`
	Amount               int64     `json:"amount"`
	Currency             string    `json:"currency"`
	Description          string    `json:"description"`
}

type TransferResponseDTO struct {
	TransactionID uuid.UUID         `json:"transaction_id"`
	ReferenceID   string            `json:"reference_id"`
	Status        TransactionStatus `json:"status"`
	Amount        int64             `json:"amount"`
	SourceBalance int64             `json:"source_balance_after"`
	DestBalance   int64             `json:"dest_balance_after"`
	CreatedAt     time.Time         `json:"created_at"`
}
