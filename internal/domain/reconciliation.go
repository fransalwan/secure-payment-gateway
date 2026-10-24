package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ReconciliationStatus string

const (
	ReconStatusProcessing ReconciliationStatus = "PROCESSING"
	ReconStatusCompleted  ReconciliationStatus = "COMPLETED"
	ReconStatusFailed     ReconciliationStatus = "FAILED"
)

type MatchStatus string

const (
	MatchStatusMatched          MatchStatus = "MATCHED"
	MatchStatusDiscrepancyAmount MatchStatus = "DISCREPANCY_AMOUNT"
	MatchStatusMissingInGateway MatchStatus = "MISSING_IN_GATEWAY"
	MatchStatusMissingInBank    MatchStatus = "MISSING_IN_BANK"
)

type ReconciliationBatch struct {
	ID                     uuid.UUID            `json:"id"`
	BatchNumber            string               `json:"batch_number"`
	BankName               string               `json:"bank_name"`
	TotalRecords           int                  `json:"total_records"`
	MatchedCount           int                  `json:"matched_count"`
	DiscrepancyCount       int                  `json:"discrepancy_count"`
	MissingInGatewayCount  int                  `json:"missing_in_gateway_count"`
	MissingInBankCount     int                  `json:"missing_in_bank_count"`
	TotalBankAmount        int64                `json:"total_bank_amount"`
	TotalGatewayAmount     int64                `json:"total_gateway_amount"`
	Status                 ReconciliationStatus `json:"status"`
	CreatedAt              time.Time            `json:"created_at"`
	CompletedAt            *time.Time           `json:"completed_at,omitempty"`
}

type ReconciliationItem struct {
	ID               uuid.UUID   `json:"id"`
	BatchID          uuid.UUID   `json:"batch_id"`
	ReferenceID      string      `json:"reference_id"`
	BankAmount       int64       `json:"bank_amount"`
	GatewayAmount    int64       `json:"gateway_amount"`
	DifferenceAmount int64       `json:"difference_amount"`
	MatchStatus      MatchStatus `json:"match_status"`
	Remarks          string      `json:"remarks"`
	CreatedAt        time.Time   `json:"created_at"`
}

type BankSettlementRecord struct {
	ReferenceID string    `json:"reference_id"`
	Amount      int64     `json:"amount"`
	BankStatus  string    `json:"bank_status"`
	Timestamp   time.Time `json:"timestamp"`
}

type ReconciliationBatchDetailDTO struct {
	Batch ReconciliationBatch   `json:"batch"`
	Items []ReconciliationItem `json:"items"`
}

type ReconciliationRepository interface {
	CreateBatch(ctx context.Context, batch *ReconciliationBatch) error
	CreateItems(ctx context.Context, items []*ReconciliationItem) error
	UpdateBatchSummary(ctx context.Context, batch *ReconciliationBatch) error
	GetBatchByID(ctx context.Context, id uuid.UUID) (*ReconciliationBatch, error)
	ListBatches(ctx context.Context, limit int) ([]*ReconciliationBatch, error)
	GetItemsByBatchID(ctx context.Context, batchID uuid.UUID) ([]*ReconciliationItem, error)
}

