package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
)

func TestReconciliationService_EmptyRecords(t *testing.T) {
	svc := service.NewReconciliationService(nil, nil)
	_, err := svc.ReconcileBankSettlement(context.Background(), "BCA", []*domain.BankSettlementRecord{})
	if err == nil {
		t.Fatal("expected error for empty records, got nil")
	}
}

func TestReconciliationService_LiveMatching(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skip("skipping db test, config failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		t.Skip("skipping db test, postgres not reachable")
	}
	defer pgPool.Close()

	reconRepo := postgresrepo.NewReconciliationRepository(pgPool)
	transferRepo := postgresrepo.NewTransferRepository(pgPool)
	svc := service.NewReconciliationService(reconRepo, transferRepo)

	records := []*domain.BankSettlementRecord{
		{
			ReferenceID: "NON-EXISTENT-REF-999",
			Amount:      100000,
			BankStatus:  "SUCCESS",
			Timestamp:   time.Now().UTC(),
		},
	}

	batch, err := svc.ReconcileBankSettlement(ctx, "MANDIRI", records)
	if err != nil {
		t.Fatalf("failed to reconcile: %v", err)
	}

	if batch.TotalRecords != 1 {
		t.Errorf("expected 1 record, got %d", batch.TotalRecords)
	}

	if batch.MissingInGatewayCount != 1 {
		t.Errorf("expected 1 missing in gateway, got %d", batch.MissingInGatewayCount)
	}

	// Fetch detail
	detail, err := svc.GetBatchDetail(ctx, batch.ID)
	if err != nil {
		t.Fatalf("failed to fetch detail: %v", err)
	}

	if len(detail.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(detail.Items))
	}

	if detail.Items[0].MatchStatus != domain.MatchStatusMissingInGateway {
		t.Errorf("expected MISSING_IN_GATEWAY, got %s", detail.Items[0].MatchStatus)
	}
}

