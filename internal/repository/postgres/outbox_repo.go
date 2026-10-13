package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	query := `
		SELECT id, event_type, payload, status, retry_count, last_error, created_at, processed_at
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		var ev domain.OutboxEvent
		err := rows.Scan(
			&ev.ID,
			&ev.EventType,
			&ev.Payload,
			&ev.Status,
			&ev.RetryCount,
			&ev.LastError,
			&ev.CreatedAt,
			&ev.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}
		events = append(events, &ev)
	}

	return events, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE outbox_events
		SET status = 'PROCESSED', processed_at = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, now, id)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `
		UPDATE outbox_events
		SET status = 'FAILED', retry_count = retry_count + 1, last_error = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, errMsg, id)
	return err
}

func (r *OutboxRepository) ListAll(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, event_type, payload, status, retry_count, last_error, created_at, processed_at
		FROM outbox_events
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list outbox events: %w", err)
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		var ev domain.OutboxEvent
		err := rows.Scan(
			&ev.ID,
			&ev.EventType,
			&ev.Payload,
			&ev.Status,
			&ev.RetryCount,
			&ev.LastError,
			&ev.CreatedAt,
			&ev.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}
		events = append(events, &ev)
	}
	return events, nil
}


