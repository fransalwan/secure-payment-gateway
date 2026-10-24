package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/google/uuid"
)

type ReconciliationService struct {
	reconRepo    *postgres.ReconciliationRepository
	transferRepo *postgres.TransferRepository
}

func NewReconciliationService(
	reconRepo *postgres.ReconciliationRepository,
	transferRepo *postgres.TransferRepository,
) *ReconciliationService {
	return &ReconciliationService{
		reconRepo:    reconRepo,
		transferRepo: transferRepo,
	}
}

// ReconcileBankSettlement performs 3-way matching between bank settlement records and internal ledger transactions.
func (s *ReconciliationService) ReconcileBankSettlement(
	ctx context.Context,
	bankName string,
	records []*domain.BankSettlementRecord,
) (*domain.ReconciliationBatch, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no bank settlement records provided")
	}

	batchID := uuid.New()
	batchNumber := fmt.Sprintf("RECON-%s-%s-%s", bankName, time.Now().Format("20060102150405"), batchID.String()[:6])
	now := time.Now().UTC()

	batch := &domain.ReconciliationBatch{
		ID:           batchID,
		BatchNumber:  batchNumber,
		BankName:     bankName,
		TotalRecords: len(records),
		Status:       domain.ReconStatusProcessing,
		CreatedAt:    now,
	}

	if err := s.reconRepo.CreateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to initialize reconciliation batch: %w", err)
	}

	// Extract reference IDs
	refIDs := make([]string, 0, len(records))
	for _, r := range records {
		refIDs = append(refIDs, r.ReferenceID)
	}

	// Fetch matching gateway transactions
	gatewayTxs, err := s.transferRepo.GetTransactionsByReferenceIDs(ctx, refIDs)
	if err != nil {
		_ = s.reconRepo.UpdateBatchSummary(ctx, &domain.ReconciliationBatch{
			ID:     batchID,
			Status: domain.ReconStatusFailed,
		})
		return nil, fmt.Errorf("failed to fetch gateway transactions: %w", err)
	}

	var items []*domain.ReconciliationItem
	var totalBankAmt, totalGatewayAmt int64
	var matchedCount, discrepancyCount, missingInGatewayCount int

	for _, bankRec := range records {
		totalBankAmt += bankRec.Amount
		itemID := uuid.New()

		gtwTx, exists := gatewayTxs[bankRec.ReferenceID]
		if !exists {
			// Record exists in Bank report, but missing in our gateway!
			missingInGatewayCount++
			items = append(items, &domain.ReconciliationItem{
				ID:               itemID,
				BatchID:          batchID,
				ReferenceID:      bankRec.ReferenceID,
				BankAmount:       bankRec.Amount,
				GatewayAmount:    0,
				DifferenceAmount: bankRec.Amount,
				MatchStatus:      domain.MatchStatusMissingInGateway,
				Remarks:          "Transaction recorded by bank but missing in internal gateway ledger",
				CreatedAt:        now,
			})
			continue
		}

		totalGatewayAmt += gtwTx.Amount
		diff := bankRec.Amount - gtwTx.Amount

		if diff == 0 && gtwTx.Status == domain.TransactionStatusSettled {
			// Perfect Match!
			matchedCount++
			items = append(items, &domain.ReconciliationItem{
				ID:               itemID,
				BatchID:          batchID,
				ReferenceID:      bankRec.ReferenceID,
				BankAmount:       bankRec.Amount,
				GatewayAmount:    gtwTx.Amount,
				DifferenceAmount: 0,
				MatchStatus:      domain.MatchStatusMatched,
				Remarks:          "100% Matched & Settled",
				CreatedAt:        now,
			})
		} else {
			// Discrepancy in amount or status
			discrepancyCount++
			remarks := fmt.Sprintf("Amount discrepancy of %d (Bank: %d, Gateway: %d)", diff, bankRec.Amount, gtwTx.Amount)
			if gtwTx.Status != domain.TransactionStatusSettled {
				remarks += fmt.Sprintf(" with internal status: %s", gtwTx.Status)
			}

			items = append(items, &domain.ReconciliationItem{
				ID:               itemID,
				BatchID:          batchID,
				ReferenceID:      bankRec.ReferenceID,
				BankAmount:       bankRec.Amount,
				GatewayAmount:    gtwTx.Amount,
				DifferenceAmount: diff,
				MatchStatus:      domain.MatchStatusDiscrepancyAmount,
				Remarks:          remarks,
				CreatedAt:        now,
			})
		}
	}

	// Persist items
	if err := s.reconRepo.CreateItems(ctx, items); err != nil {
		return nil, fmt.Errorf("failed to save reconciliation items: %w", err)
	}

	// Update Batch summary
	batch.MatchedCount = matchedCount
	batch.DiscrepancyCount = discrepancyCount
	batch.MissingInGatewayCount = missingInGatewayCount
	batch.TotalBankAmount = totalBankAmt
	batch.TotalGatewayAmount = totalGatewayAmt
	batch.Status = domain.ReconStatusCompleted

	if err := s.reconRepo.UpdateBatchSummary(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to update batch summary: %w", err)
	}

	return batch, nil
}

func (s *ReconciliationService) GetBatchDetail(ctx context.Context, batchID uuid.UUID) (*domain.ReconciliationBatchDetailDTO, error) {
	batch, err := s.reconRepo.GetBatchByID(ctx, batchID)
	if err != nil {
		return nil, err
	}

	items, err := s.reconRepo.GetItemsByBatchID(ctx, batchID)
	if err != nil {
		return nil, err
	}

	return &domain.ReconciliationBatchDetailDTO{
		Batch: *batch,
		Items: itemsToValueSlice(items),
	}, nil
}

func (s *ReconciliationService) ListBatches(ctx context.Context) ([]*domain.ReconciliationBatch, error) {
	return s.reconRepo.ListBatches(ctx, 50)
}

func itemsToValueSlice(items []*domain.ReconciliationItem) []domain.ReconciliationItem {
	res := make([]domain.ReconciliationItem, len(items))
	for i, item := range items {
		res[i] = *item
	}
	return res
}

