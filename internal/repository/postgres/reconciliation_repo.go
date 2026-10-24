package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrReconBatchNotFound = errors.New("reconciliation batch not found")
)

type ReconciliationRepository struct {
	pool *pgxpool.Pool
}

func NewReconciliationRepository(pool *pgxpool.Pool) *ReconciliationRepository {
	return &ReconciliationRepository{pool: pool}
}

func (r *ReconciliationRepository) CreateBatch(ctx context.Context, batch *domain.ReconciliationBatch) error {
	query := `
		INSERT INTO reconciliation_batches (
			id, batch_number, bank_name, total_records, matched_count, discrepancy_count,
			missing_in_gateway_count, missing_in_bank_count, total_bank_amount, total_gateway_amount,
			status, created_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		batch.ID,
		batch.BatchNumber,
		batch.BankName,
		batch.TotalRecords,
		batch.MatchedCount,
		batch.DiscrepancyCount,
		batch.MissingInGatewayCount,
		batch.MissingInBankCount,
		batch.TotalBankAmount,
		batch.TotalGatewayAmount,
		batch.Status,
		batch.CreatedAt,
		batch.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create reconciliation batch: %w", err)
	}
	return nil
}

func (r *ReconciliationRepository) CreateItems(ctx context.Context, items []*domain.ReconciliationItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for batch items: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO reconciliation_items (
			id, batch_id, reference_id, bank_amount, gateway_amount, difference_amount, match_status, remarks, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, item := range items {
		_, err := tx.Exec(ctx, query,
			item.ID,
			item.BatchID,
			item.ReferenceID,
			item.BankAmount,
			item.GatewayAmount,
			item.DifferenceAmount,
			item.MatchStatus,
			item.Remarks,
			item.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert reconciliation item (%s): %w", item.ReferenceID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ReconciliationRepository) UpdateBatchSummary(ctx context.Context, batch *domain.ReconciliationBatch) error {
	now := time.Now().UTC()
	query := `
		UPDATE reconciliation_batches
		SET total_records = $1, matched_count = $2, discrepancy_count = $3,
		    missing_in_gateway_count = $4, missing_in_bank_count = $5,
		    total_bank_amount = $6, total_gateway_amount = $7,
		    status = $8, completed_at = $9
		WHERE id = $10
	`
	_, err := r.pool.Exec(ctx, query,
		batch.TotalRecords,
		batch.MatchedCount,
		batch.DiscrepancyCount,
		batch.MissingInGatewayCount,
		batch.MissingInBankCount,
		batch.TotalBankAmount,
		batch.TotalGatewayAmount,
		batch.Status,
		now,
		batch.ID,
	)
	return err
}

func (r *ReconciliationRepository) GetBatchByID(ctx context.Context, id uuid.UUID) (*domain.ReconciliationBatch, error) {
	query := `
		SELECT id, batch_number, bank_name, total_records, matched_count, discrepancy_count,
		       missing_in_gateway_count, missing_in_bank_count, total_bank_amount, total_gateway_amount,
		       status, created_at, completed_at
		FROM reconciliation_batches
		WHERE id = $1
	`
	var b domain.ReconciliationBatch
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&b.ID,
		&b.BatchNumber,
		&b.BankName,
		&b.TotalRecords,
		&b.MatchedCount,
		&b.DiscrepancyCount,
		&b.MissingInGatewayCount,
		&b.MissingInBankCount,
		&b.TotalBankAmount,
		&b.TotalGatewayAmount,
		&b.Status,
		&b.CreatedAt,
		&b.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReconBatchNotFound
		}
		return nil, fmt.Errorf("failed to get reconciliation batch: %w", err)
	}
	return &b, nil
}

func (r *ReconciliationRepository) ListBatches(ctx context.Context, limit int) ([]*domain.ReconciliationBatch, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, batch_number, bank_name, total_records, matched_count, discrepancy_count,
		       missing_in_gateway_count, missing_in_bank_count, total_bank_amount, total_gateway_amount,
		       status, created_at, completed_at
		FROM reconciliation_batches
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query batches: %w", err)
	}
	defer rows.Close()

	var batches []*domain.ReconciliationBatch
	for rows.Next() {
		var b domain.ReconciliationBatch
		err := rows.Scan(
			&b.ID,
			&b.BatchNumber,
			&b.BankName,
			&b.TotalRecords,
			&b.MatchedCount,
			&b.DiscrepancyCount,
			&b.MissingInGatewayCount,
			&b.MissingInBankCount,
			&b.TotalBankAmount,
			&b.TotalGatewayAmount,
			&b.Status,
			&b.CreatedAt,
			&b.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch: %w", err)
		}
		batches = append(batches, &b)
	}
	return batches, nil
}

func (r *ReconciliationRepository) GetItemsByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.ReconciliationItem, error) {
	query := `
		SELECT id, batch_id, reference_id, bank_amount, gateway_amount, difference_amount, match_status, remarks, created_at
		FROM reconciliation_items
		WHERE batch_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reconciliation items: %w", err)
	}
	defer rows.Close()

	var items []*domain.ReconciliationItem
	for rows.Next() {
		var item domain.ReconciliationItem
		err := rows.Scan(
			&item.ID,
			&item.BatchID,
			&item.ReferenceID,
			&item.BankAmount,
			&item.GatewayAmount,
			&item.DifferenceAmount,
			&item.MatchStatus,
			&item.Remarks,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation item: %w", err)
		}
		items = append(items, &item)
	}
	return items, nil
}
