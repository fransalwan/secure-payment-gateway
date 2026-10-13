package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID                uuid.UUID `json:"id"`
	AccountNumber     string    `json:"account_number"`
	HolderName        string    `json:"holder_name"`
	EncryptedIdentity string    `json:"-"`
	Balance           int64     `json:"balance"`
	Currency          string    `json:"currency"`
	Version           int       `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateAccountDTO struct {
	AccountNumber  string `json:"account_number"`
	HolderName     string `json:"holder_name"`
	IdentityData   string `json:"identity_data"`
	InitialBalance int64  `json:"initial_balance"`
	Currency       string `json:"currency"`
}

type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetByAccountNumber(ctx context.Context, accNum string) (*Account, error)
	GetByIDWithLock(ctx context.Context, id uuid.UUID) (*Account, error)
}

