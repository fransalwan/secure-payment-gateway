package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportingRepository struct {
	pool *pgxpool.Pool
}

func NewReportingRepository(pool *pgxpool.Pool) *ReportingRepository {
	return &ReportingRepository{pool: pool}
}

// ExecuteSafeAnalyticalQuery executes a sanitized SELECT query inside a READ ONLY transaction with a hard 2.5s statement_timeout
func (r *ReportingRepository) ExecuteSafeAnalyticalQuery(
	ctx context.Context,
	sqlQuery string,
) ([]string, []map[string]interface{}, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("acquire db connection: %w", err)
	}
	defer conn.Release()

	// 1. Enforce strict Read-Only session with 2500ms timeout guard
	if _, err := conn.Exec(ctx, "SET statement_timeout = '2500ms'"); err != nil {
		return nil, nil, fmt.Errorf("set statement timeout: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin read transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return nil, nil, fmt.Errorf("set read-only transaction: %w", err)
	}

	// 2. Execute query
	rows, err := tx.Query(ctx, sqlQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("execute query: %w", err)
	}
	defer rows.Close()

	// 3. Extract column headers dynamically
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = fd.Name
	}

	// 4. Extract rows
	var resultRows []map[string]interface{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, nil, fmt.Errorf("scan row values: %w", err)
		}

		rowMap := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := vals[i]
			// Format timestamps and byte arrays cleanly
			if t, ok := val.(time.Time); ok {
				rowMap[col] = t.Format(time.RFC3339)
			} else {
				rowMap[col] = val
			}
		}
		resultRows = append(resultRows, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate rows error: %w", err)
	}

	_ = tx.Commit(ctx)
	return columns, resultRows, nil
}

func (r *ReportingRepository) AuditAIQuery(
	ctx context.Context,
	merchantID uuid.UUID,
	userQuery string,
	generatedSQL string,
	isSafe bool,
	execMs int64,
	rowsReturned int,
	tokenUsage int,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_reporting_audit_logs (
			merchant_id, user_query, generated_sql, is_safe,
			execution_ms, rows_returned, token_usage, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		merchantID, userQuery, generatedSQL, isSafe,
		execMs, rowsReturned, tokenUsage, time.Now().UTC(),
	)
	return err
}

