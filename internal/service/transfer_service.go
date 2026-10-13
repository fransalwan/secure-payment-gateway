package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
)

var (
	ErrConcurrentTransaction = errors.New("a transaction with this idempotency key is already in progress")
	ErrInvalidTransferAmount = errors.New("transfer amount must be strictly greater than zero")
	ErrMissingReferenceID   = errors.New("reference_id is required")
)

type TransferService struct {
	transferRepo    *postgres.TransferRepository
	idempotencyRepo redis.IdempotencyRepository
	cfg             *config.Config
}

func NewTransferService(
	transferRepo *postgres.TransferRepository,
	idempotencyRepo redis.IdempotencyRepository,
	cfg *config.Config,
) *TransferService {
	return &TransferService{
		transferRepo:    transferRepo,
		idempotencyRepo: idempotencyRepo,
		cfg:             cfg,
	}
}

func (s *TransferService) TransferFunds(
	ctx context.Context,
	req *domain.TransferRequestDTO,
	idempotencyKey string,
) (*domain.TransferResponseDTO, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidTransferAmount
	}
	if req.ReferenceID == "" {
		return nil, ErrMissingReferenceID
	}
	if req.Currency == "" {
		req.Currency = "IDR"
	}

	ttl := time.Duration(s.cfg.IdempotencyTTLSeconds) * time.Second

	status, cachedBytes, err := s.idempotencyRepo.LockOrGet(ctx, idempotencyKey, ttl)
	if err != nil {
		return nil, fmt.Errorf("idempotency error: %w", err)
	}

	if status == redis.LockCompleted {
		var cachedResp domain.TransferResponseDTO
		if err := json.Unmarshal(cachedBytes, &cachedResp); err == nil {
			return &cachedResp, nil
		}
	}

	if status == redis.LockInProgress {
		return nil, ErrConcurrentTransaction
	}

	resp, err := s.transferRepo.ExecuteDoubleEntryTransfer(ctx, req, idempotencyKey)
	if err != nil {
		_ = s.idempotencyRepo.Release(ctx, idempotencyKey)
		return nil, err
	}

	respBytes, err := json.Marshal(resp)
	if err == nil {
		_ = s.idempotencyRepo.SaveCompleted(ctx, idempotencyKey, respBytes, ttl)
	}

	return resp, nil
}

func (s *TransferService) ListLedgerEntries(ctx context.Context) ([]*domain.LedgerEntry, error) {
	return s.transferRepo.ListLedgerEntries(ctx, 50)
}

func (s *TransferService) ListAuditLogs(ctx context.Context) ([]*domain.AuditLog, error) {
	return s.transferRepo.ListAuditLogs(ctx, 50)
}


