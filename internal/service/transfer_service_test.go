package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	redisrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/google/uuid"
)

func TestTransferService_Validation(t *testing.T) {
	cfg := &config.Config{
		IdempotencyTTLSeconds: 60,
	}
	idempotencyRepo := redisrepo.NewIdempotencyRepository(nil)
	svc := service.NewTransferService(nil, idempotencyRepo, cfg)

	ctx := context.Background()

	// Test case: amount <= 0
	_, err := svc.TransferFunds(ctx, &domain.TransferRequestDTO{
		ReferenceID:          "REF-101",
		SourceAccountID:      uuid.New(),
		DestinationAccountID: uuid.New(),
		Amount:               0,
	}, "idemp-001")

	if err == nil {
		t.Fatal("expected error for amount <= 0, got nil")
	}

	// Test case: missing reference ID
	_, err = svc.TransferFunds(ctx, &domain.TransferRequestDTO{
		ReferenceID:          "",
		SourceAccountID:      uuid.New(),
		DestinationAccountID: uuid.New(),
		Amount:               1000,
	}, "idemp-002")

	if err == nil {
		t.Fatal("expected error for missing reference ID, got nil")
	}
}

func TestTransferService_IdempotencyLifecycle(t *testing.T) {
	idempotencyRepo := redisrepo.NewIdempotencyRepository(nil)
	ctx := context.Background()
	key := "test-idempotency-key-1"

	status, _, err := idempotencyRepo.LockOrGet(ctx, key, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != redisrepo.LockAcquired {
		t.Fatalf("expected LockAcquired, got %s", status)
	}

	status2, _, err := idempotencyRepo.LockOrGet(ctx, key, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status2 != redisrepo.LockInProgress {
		t.Fatalf("expected LockInProgress, got %s", status2)
	}

	savedResponse := []byte(`{"transaction_id":"test-tx","status":"SETTLED"}`)
	err = idempotencyRepo.SaveCompleted(ctx, key, savedResponse, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	status3, cachedBytes, err := idempotencyRepo.LockOrGet(ctx, key, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status3 != redisrepo.LockCompleted {
		t.Fatalf("expected LockCompleted, got %s", status3)
	}
	if string(cachedBytes) != string(savedResponse) {
		t.Fatalf("expected '%s', got '%s'", string(savedResponse), string(cachedBytes))
	}
}

