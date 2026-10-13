package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAccountNotFound = errors.New("account not found")
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, account *domain.Account) error {
	query := `
		INSERT INTO accounts (id, account_number, holder_name, encrypted_identity, balance, currency, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		account.ID,
		account.AccountNumber,
		account.HolderName,
		account.EncryptedIdentity,
		account.Balance,
		account.Currency,
		account.Version,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert account: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	query := `
		SELECT id, account_number, holder_name, encrypted_identity, balance, currency, version, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	var acc domain.Account
	err := row.Scan(
		&acc.ID,
		&acc.AccountNumber,
		&acc.HolderName,
		&acc.EncryptedIdentity,
		&acc.Balance,
		&acc.Currency,
		&acc.Version,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to query account by id: %w", err)
	}
	return &acc, nil
}

func (r *AccountRepository) GetByAccountNumber(ctx context.Context, accNum string) (*domain.Account, error) {
	query := `
		SELECT id, account_number, holder_name, encrypted_identity, balance, currency, version, created_at, updated_at
		FROM accounts
		WHERE account_number = $1
	`
	row := r.pool.QueryRow(ctx, query, accNum)

	var acc domain.Account
	err := row.Scan(
		&acc.ID,
		&acc.AccountNumber,
		&acc.HolderName,
		&acc.EncryptedIdentity,
		&acc.Balance,
		&acc.Currency,
		&acc.Version,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to query account by number: %w", err)
	}
	return &acc, nil
}

func (r *AccountRepository) List(ctx context.Context, limit int) ([]*domain.Account, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, account_number, holder_name, encrypted_identity, balance, currency, version, created_at, updated_at
		FROM accounts
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*domain.Account
	for rows.Next() {
		var acc domain.Account
		err := rows.Scan(
			&acc.ID,
			&acc.AccountNumber,
			&acc.HolderName,
			&acc.EncryptedIdentity,
			&acc.Balance,
			&acc.Currency,
			&acc.Version,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, &acc)
	}
	return accounts, nil
}


