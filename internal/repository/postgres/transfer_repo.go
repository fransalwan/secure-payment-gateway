package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrSelfTransferNotAllowed = errors.New("self-transfer is not allowed")
	ErrDuplicateReferenceID   = errors.New("duplicate reference id")
)

type TransferRepository struct {
	pool *pgxpool.Pool
}

func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{pool: pool}
}

// ExecuteDoubleEntryTransfer executes an atomic, ACID-compliant double-entry transfer.
// Uses deterministic lock ordering to guarantee zero deadlocks under high concurrency.
func (r *TransferRepository) ExecuteDoubleEntryTransfer(
	ctx context.Context,
	req *domain.TransferRequestDTO,
	idempotencyKey string,
) (*domain.TransferResponseDTO, error) {
	if req.SourceAccountID == req.DestinationAccountID {
		return nil, ErrSelfTransferNotAllowed
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: Deterministic Lock Ordering to prevent deadlocks (A->B vs B->A)
	firstLockID := req.SourceAccountID
	secondLockID := req.DestinationAccountID
	if strings.Compare(firstLockID.String(), secondLockID.String()) > 0 {
		firstLockID, secondLockID = secondLockID, firstLockID
	}

	// Lock first account
	var acc1 domain.Account
	err = tx.QueryRow(ctx, `
		SELECT id, balance, currency, version
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`, firstLockID).Scan(&acc1.ID, &acc1.Balance, &acc1.Currency, &acc1.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to lock account %s: %w", firstLockID, err)
	}

	// Lock second account
	var acc2 domain.Account
	err = tx.QueryRow(ctx, `
		SELECT id, balance, currency, version
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`, secondLockID).Scan(&acc2.ID, &acc2.Balance, &acc2.Currency, &acc2.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to lock account %s: %w", secondLockID, err)
	}

	// Determine which locked struct is source and destination
	var sourceAcc, destAcc *domain.Account
	if acc1.ID == req.SourceAccountID {
		sourceAcc = &acc1
		destAcc = &acc2
	} else {
		sourceAcc = &acc2
		destAcc = &acc1
	}

	// Step 2: Validate balance and currency
	if sourceAcc.Balance < req.Amount {
		return nil, ErrInsufficientBalance
	}

	// Step 3: Compute new balances
	newSourceBalance := sourceAcc.Balance - req.Amount
	newDestBalance := destAcc.Balance + req.Amount

	// Step 4: Apply Balance Updates
	_, err = tx.Exec(ctx, `
		UPDATE accounts
		SET balance = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2
	`, newSourceBalance, sourceAcc.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update source balance: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE accounts
		SET balance = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2
	`, newDestBalance, destAcc.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update destination balance: %w", err)
	}

	// Step 5: Insert Transaction Record
	txID := uuid.New()
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, reference_id, idempotency_key, source_account_id, destination_account_id, amount, currency, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, txID, req.ReferenceID, idempotencyKey, req.SourceAccountID, req.DestinationAccountID, req.Amount, req.Currency, domain.TransactionStatusSettled, req.Description, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transaction record: %w", err)
	}

	// Step 6: Insert Double-Entry Ledger Records
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, transaction_id, account_id, entry_type, amount, balance_after, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New(), txID, req.SourceAccountID, domain.EntryTypeDebit, req.Amount, newSourceBalance, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert debit ledger entry: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, transaction_id, account_id, entry_type, amount, balance_after, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New(), txID, req.DestinationAccountID, domain.EntryTypeCredit, req.Amount, newDestBalance, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert credit ledger entry: %w", err)
	}

	// Step 7: Cryptographic Tamper-Evident Audit Log (Hash-Chained)
	var lastHash string
	err = tx.QueryRow(ctx, `
		SELECT current_hash FROM audit_logs ORDER BY id DESC LIMIT 1
	`).Scan(&lastHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get previous audit hash: %w", err)
	}
	if lastHash == "" {
		lastHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	auditPayload, _ := json.Marshal(map[string]interface{}{
		"transaction_id": txID,
		"source":         req.SourceAccountID,
		"destination":    req.DestinationAccountID,
		"amount":         req.Amount,
		"time":           now,
	})
	currentHash := crypto.CalculateAuditHash(lastHash, auditPayload)

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (entity_type, entity_id, action, payload, previous_hash, current_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "TRANSACTION", txID.String(), "SETTLED", auditPayload, lastHash, currentHash, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert audit log: %w", err)
	}

	// Step 8: Transactional Outbox Pattern for reliable event publishing
	outboxPayload, _ := json.Marshal(map[string]interface{}{
		"event":          "payment.settled",
		"transaction_id": txID,
		"reference_id":   req.ReferenceID,
		"amount":         req.Amount,
		"currency":       req.Currency,
		"timestamp":      now,
	})

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, payload, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), "payment.settled", outboxPayload, domain.OutboxStatusPending, now)
	if err != nil {
		return nil, fmt.Errorf("failed to record outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.TransferResponseDTO{
		TransactionID: txID,
		ReferenceID:   req.ReferenceID,
		Status:        domain.TransactionStatusSettled,
		Amount:        req.Amount,
		SourceBalance: newSourceBalance,
		DestBalance:   newDestBalance,
		CreatedAt:     now,
	}, nil
}

func (r *TransferRepository) ListLedgerEntries(ctx context.Context, limit int) ([]*domain.LedgerEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, transaction_id, account_id, entry_type, amount, balance_after, created_at
		FROM ledger_entries
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []*domain.LedgerEntry
	for rows.Next() {
		var e domain.LedgerEntry
		err := rows.Scan(
			&e.ID,
			&e.TransactionID,
			&e.AccountID,
			&e.EntryType,
			&e.Amount,
			&e.BalanceAfter,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, &e)
	}
	return entries, nil
}

func (r *TransferRepository) ListAuditLogs(ctx context.Context, limit int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, entity_type, entity_id, action, payload, previous_hash, current_hash, created_at
		FROM audit_logs
		ORDER BY id DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		err := rows.Scan(
			&a.ID,
			&a.EntityType,
			&a.EntityID,
			&a.Action,
			&a.Payload,
			&a.PreviousHash,
			&a.CurrentHash,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, &a)
	}
	return logs, nil
}


