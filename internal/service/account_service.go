package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
	"github.com/google/uuid"
)

var (
	ErrInvalidAccountInput = errors.New("invalid account input data")
)

type AccountService struct {
	repo *postgres.AccountRepository
	cfg  *config.Config
}

func NewAccountService(repo *postgres.AccountRepository, cfg *config.Config) *AccountService {
	return &AccountService{
		repo: repo,
		cfg:  cfg,
	}
}

func formatCiphertextPreview(enc string) string {
	if len(enc) > 28 {
		return enc[:28] + "..."
	}
	return enc
}

func (s *AccountService) CreateAccount(ctx context.Context, dto *domain.CreateAccountDTO) (*domain.Account, error) {
	if dto.IdentityData == "" && dto.IdentityNumber != "" {
		dto.IdentityData = dto.IdentityNumber
	}
	if dto.AccountNumber == "" || dto.HolderName == "" || dto.IdentityData == "" {
		return nil, ErrInvalidAccountInput
	}
	if dto.InitialBalance < 0 {
		return nil, errors.New("initial balance cannot be negative")
	}
	if dto.Currency == "" {
		dto.Currency = "IDR"
	}

	encryptedIdentity, err := crypto.AESGCMEncrypt([]byte(dto.IdentityData), s.cfg.AESSecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt identity data: %w", err)
	}

	now := time.Now().UTC()
	account := &domain.Account{
		ID:                uuid.New(),
		AccountNumber:     dto.AccountNumber,
		HolderName:        dto.HolderName,
		EncryptedIdentity: encryptedIdentity,
		CiphertextPreview: formatCiphertextPreview(encryptedIdentity),
		Balance:           dto.InitialBalance,
		Currency:          dto.Currency,
		Version:           0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

func (s *AccountService) GetAccount(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	acc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	acc.CiphertextPreview = formatCiphertextPreview(acc.EncryptedIdentity)
	if plain, decErr := crypto.AESGCMDecrypt(acc.EncryptedIdentity, s.cfg.AESSecretKey); decErr == nil {
		acc.DecryptedIdentity = string(plain)
	}
	return acc, nil
}

func (s *AccountService) GetAccountByNumber(ctx context.Context, accNum string) (*domain.Account, error) {
	acc, err := s.repo.GetByAccountNumber(ctx, accNum)
	if err != nil {
		return nil, err
	}
	acc.CiphertextPreview = formatCiphertextPreview(acc.EncryptedIdentity)
	return acc, nil
}

func (s *AccountService) ListAccounts(ctx context.Context) ([]*domain.Account, error) {
	accounts, err := s.repo.List(ctx, 50)
	if err != nil {
		return nil, err
	}
	for _, acc := range accounts {
		acc.CiphertextPreview = formatCiphertextPreview(acc.EncryptedIdentity)
	}
	return accounts, nil
}


