package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusProcessed OutboxStatus = "PROCESSED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

type OutboxEvent struct {
	ID          uuid.UUID    `json:"id"`
	EventType   string       `json:"event_type"`
	Payload     []byte       `json:"payload"`
	Status      OutboxStatus `json:"status"`
	RetryCount  int          `json:"retry_count"`
	LastError   *string      `json:"last_error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	ProcessedAt *time.Time   `json:"processed_at,omitempty"`
}

type OutboxRepository interface {
	Create(ctx context.Context, event *OutboxEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}
