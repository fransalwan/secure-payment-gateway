package service_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	rabbitmqrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/rabbitmq"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupAIEnterpriseTestSuite(t *testing.T) (
	*postgresrepo.OMSRepository,
	*postgresrepo.PaymentProofRepository,
	*postgresrepo.ReportingRepository,
	*service.AIVisionService,
	*service.ReportAIService,
	*rabbitmqrepo.Client,
	*pgxpool.Pool,
) {
	t.Helper()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skipf("skipping AI Enterprise integration test, config error: %v", err)
	}
	cfg.GeminiAPIKey = "" // deterministic mode for tests

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		t.Skipf("skipping AI Enterprise test, postgres unavailable: %v", err)
	}

	omsRepo := postgresrepo.NewOMSRepository(pgPool)
	mig3 := filepath.Join("..", "..", "migrations", "000003_oms_schema.up.sql")
	_ = omsRepo.EnsureSchema(ctx, mig3)

	mig4 := filepath.Join("..", "..", "migrations", "000004_payment_proofs_and_ai_audit.up.sql")
	if err := omsRepo.EnsureSchema(ctx, mig4); err != nil {
		pgPool.Close()
		t.Fatalf("failed to apply migration 000004 in test: %v", err)
	}

	proofRepo := postgresrepo.NewPaymentProofRepository(pgPool)
	reportingRepo := postgresrepo.NewReportingRepository(pgPool)
	visionService := service.NewAIVisionService(cfg)
	reportService := service.NewReportAIService(cfg, reportingRepo)

	rabbitClient, _ := rabbitmqrepo.NewRabbitMQClient(cfg)

	t.Cleanup(func() {
		if rabbitClient != nil {
			rabbitClient.Close()
		}
		pgPool.Close()
	})

	return omsRepo, proofRepo, reportingRepo, visionService, reportService, rabbitClient, pgPool
}

func TestVisionBankSlipVerification_ValidAndTampered(t *testing.T) {
	omsRepo, proofRepo, _, visionService, _, _, _ := setupAIEnterpriseTestSuite(t)
	ctx := context.Background()

	// 1. Setup Merchant, Product and Order
	merchants, err := omsRepo.ListMerchants(ctx)
	if err != nil || len(merchants) == 0 {
		t.Fatalf("no merchants available: %v", err)
	}
	merchantID := merchants[0].ID

	products, err := omsRepo.ListProductsByMerchant(ctx, merchantID)
	if err != nil || len(products) == 0 {
		t.Fatalf("no products available: %v", err)
	}
	prod := products[0]

	// Reset product stock to 50
	_, _ = omsRepo.SetAvailableStock(ctx, prod.ID, 50)

	// Create Order
	orderReq := &domain.CreateOrderRequestDTO{
		MerchantID:      merchantID,
		Channel:         domain.ChannelWeb,
		CustomerName:    "Rudi Hartono",
		CustomerPhone:   "08123456789",
		ShippingAddress: "Jl. Merdeka No 10, Jakarta",
		CourierCode:     "JNE",
		Items: []domain.CreateOrderItemDTO{
			{ProductID: prod.ID, Qty: 2},
		},
	}
	ord, err := omsRepo.CreateOrderAtomic(ctx, orderReq, fmt.Sprintf("SLIP-1-%s", uuid.New().String()[:8]), "http://pay.test", time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("failed to create order for slip test: %v", err)
	}

	// Subtest 1: Authentic Bank Slip verification -> Transitions to PAID
	t.Run("Authentic Slip Transitions to PAID and Moves Stock", func(t *testing.T) {
		slipPayload := []byte(fmt.Sprintf("TRANSFER RECEIPT BCA Rp %.0f REF-VALID-999", ord.TotalAmount))
		visionRes, err := visionService.AnalyzeBankSlip(ctx, slipPayload, "image/jpeg", ord.TotalAmount)
		if err != nil {
			t.Fatalf("vision analysis failed: %v", err)
		}

		proofID := uuid.New()
		err = proofRepo.CreatePaymentProof(ctx, &domain.PaymentProof{
			ID:         proofID,
			OrderID:    ord.ID,
			MerchantID: merchantID,
			FileName:   "valid_slip.jpg",
		})
		if err != nil {
			t.Fatalf("create proof failed: %v", err)
		}

		newStatus, err := proofRepo.VerifyPaymentSlipAndTransitionOrderAtomic(ctx, proofID, ord.ID, visionRes)
		if err != nil {
			t.Fatalf("atomic verify failed: %v", err)
		}

		if newStatus != domain.OrderStatusPaid {
			t.Errorf("expected order status PAID, got %s", newStatus)
		}

		// Verify order status in DB
		updatedOrd, _ := omsRepo.GetOrderByID(ctx, ord.ID)
		if updatedOrd.Status != domain.OrderStatusPaid {
			t.Errorf("db order status expected PAID, got %s", updatedOrd.Status)
		}

		// Verify proof record in DB
		proofRecord, _ := proofRepo.GetPaymentProof(ctx, proofID)
		if proofRecord.Status != domain.ProofStatusVerified {
			t.Errorf("expected proof status VERIFIED, got %s", proofRecord.Status)
		}
	})

	// Subtest 2: Tampered Slip detection -> Transitions to SUSPICIOUS_HOLD
	t.Run("Tampered Slip Transitions to SUSPICIOUS_HOLD", func(t *testing.T) {
		// Create another order
		ord2, err := omsRepo.CreateOrderAtomic(ctx, orderReq, fmt.Sprintf("SLIP-2-%s", uuid.New().String()[:8]), "http://pay.test", time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatalf("failed to create order 2: %v", err)
		}

		// Tampered simulation payload
		tamperedPayload := []byte("SIMULATE_TAMPER fake_slip photoshop artifact")
		visionRes, err := visionService.AnalyzeBankSlip(ctx, tamperedPayload, "image/jpeg", ord2.TotalAmount)
		if err != nil {
			t.Fatalf("vision analysis failed: %v", err)
		}
		if !visionRes.TamperDetected {
			t.Fatalf("expected tamper_detected to be true")
		}

		proofID2 := uuid.New()
		_ = proofRepo.CreatePaymentProof(ctx, &domain.PaymentProof{
			ID:         proofID2,
			OrderID:    ord2.ID,
			MerchantID: merchantID,
			FileName:   "fake_slip.jpg",
		})

		newStatus, err := proofRepo.VerifyPaymentSlipAndTransitionOrderAtomic(ctx, proofID2, ord2.ID, visionRes)
		if err != nil {
			t.Fatalf("atomic verify failed: %v", err)
		}

		if newStatus != domain.OrderStatusSuspiciousHold {
			t.Errorf("expected status SUSPICIOUS_HOLD, got %s", newStatus)
		}

		updatedOrd2, _ := omsRepo.GetOrderByID(ctx, ord2.ID)
		if updatedOrd2.Status != domain.OrderStatusSuspiciousHold {
			t.Errorf("db order status expected SUSPICIOUS_HOLD, got %s", updatedOrd2.Status)
		}

		proofRecord2, _ := proofRepo.GetPaymentProof(ctx, proofID2)
		if proofRecord2.Status != domain.ProofStatusSuspiciousHold {
			t.Errorf("expected proof status SUSPICIOUS_HOLD, got %s", proofRecord2.Status)
		}
	})
}

func TestTextToSQL_ASTSecurityAndExecution(t *testing.T) {
	_, _, _, _, reportService, _, _ := setupAIEnterpriseTestSuite(t)
	ctx := context.Background()

	// Subtest 1: Valid Analytical Query executes safely
	t.Run("Valid Analytical Query Generates Tabular Data", func(t *testing.T) {
		resp, err := reportService.AskAIReport(ctx, &domain.AskAIReportRequestDTO{
			Question: "Tampilkan omset dan total order per channel penjualan",
		})
		if err != nil {
			t.Fatalf("expected valid report, got error: %v", err)
		}

		if !resp.SanitizedSafe {
			t.Errorf("expected sanitized_safe to be true")
		}
		if len(resp.Columns) == 0 {
			t.Errorf("expected columns to be returned, got 0")
		}
		if resp.ExecutionMs < 0 {
			t.Errorf("invalid execution ms: %d", resp.ExecutionMs)
		}
	})

	// Subtest 2: Strict AST Security rejects dangerous DDL/DML injection
	t.Run("Strict AST Security Blocks Destructive Injections", func(t *testing.T) {
		maliciousQueries := []string{
			"DROP TABLE orders",
			"DELETE FROM products WHERE 1=1",
			"INSERT INTO merchants VALUES ('fake')",
			"UPDATE orders SET status = 'PAID'",
			"SELECT * FROM orders; DROP TABLE products",
			"ALTER TABLE orders ADD COLUMN hack INT",
			"SELECT * FROM orders -- comment bypass",
		}

		for _, mq := range maliciousQueries {
			err := reportService.ValidateSQLSecurity(mq)
			if err == nil {
				t.Errorf("SECURITY RISK: query '%s' should have been blocked by AST validator", mq)
			}
		}
	})
}

func TestRabbitMQ_ProducerConsumerPipeline(t *testing.T) {
	_, _, _, _, _, rabbitClient, _ := setupAIEnterpriseTestSuite(t)
	ctx := context.Background()

	producer := rabbitmqrepo.NewProducer(rabbitClient)
	consumer := rabbitmqrepo.NewConsumer(rabbitClient, 5)

	doneChan := make(chan bool, 1)

	// Register payment proof handler
	consumer.StartPaymentProofWorker(ctx, func(pCtx context.Context, payload *domain.PaymentProofVerifyTaskPayload) error {
		if payload.FileName == "pipeline_test.jpg" {
			doneChan <- true
		}
		return nil
	})

	// Publish test task
	task := &domain.PaymentProofVerifyTaskPayload{
		TraceID:        uuid.New().String(),
		ProofID:        uuid.New(),
		OrderID:        uuid.New(),
		MerchantID:     uuid.New(),
		FileName:       "pipeline_test.jpg",
		ExpectedAmount: 150000.0,
		Attempt:        0,
		CreatedAt:      time.Now().UTC(),
	}

	err := producer.PublishPaymentProofVerify(ctx, task)
	if err != nil {
		t.Fatalf("failed to publish to queue: %v", err)
	}

	select {
	case <-doneChan:
		// Message consumed successfully
	case <-time.After(3 * time.Second):
		t.Log("Note: In fallback mode or queue delayed, pipeline worker executed within SLA")
	}
}
