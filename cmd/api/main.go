package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	httphandler "github.com/fransalwan/secure-payment-gateway/internal/handler/http"
	custommiddleware "github.com/fransalwan/secure-payment-gateway/internal/handler/http/middleware"
	pkglogger "github.com/fransalwan/secure-payment-gateway/internal/pkg/logger"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	rabbitmqrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/rabbitmq"
	redisrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Omni-Channel Mini E-Commerce OMS & Secure Payment Gateway...")

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize Enterprise Structured Logging (Zerolog)
	pkglogger.InitLogger(cfg.AppEnv)
	log.Info().
		Str("app_env", cfg.AppEnv).
		Str("port", cfg.AppPort).
		Msg("Enterprise AI-Powered FinTech OMS & Payment Gateway starting...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to PostgreSQL")
		os.Exit(1)
	}
	defer pgPool.Close()

	redisClient, _ := redisrepo.NewRedisClient(cfg)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// RabbitMQ Message Broker (Asynchronous AI Worker Pipeline)
	rabbitClient, _ := rabbitmqrepo.NewRabbitMQClient(cfg)
	if rabbitClient != nil {
		defer rabbitClient.Close()
	}
	rabbitProducer := rabbitmqrepo.NewProducer(rabbitClient)
	rabbitConsumer := rabbitmqrepo.NewConsumer(rabbitClient, 10)

	// Repositories
	accountRepo := postgresrepo.NewAccountRepository(pgPool)
	transferRepo := postgresrepo.NewTransferRepository(pgPool)
	reconRepo := postgresrepo.NewReconciliationRepository(pgPool)
	omsRepo := postgresrepo.NewOMSRepository(pgPool)
	proofRepo := postgresrepo.NewPaymentProofRepository(pgPool)
	reportingRepo := postgresrepo.NewReportingRepository(pgPool)
	outboxRepo := postgresrepo.NewOutboxRepository(pgPool)
	idempotencyRepo := redisrepo.NewIdempotencyRepository(redisClient)
	queueRepo := redisrepo.NewQueueRepository(redisClient)

	// Idempotent schema bootstrap for local & containerized environments
	for _, mig := range []string{
		"migrations/000001_init_schema.up.sql",
		"migrations/000002_reconciliation_schema.up.sql",
		"migrations/000003_oms_schema.up.sql",
		"migrations/000004_payment_proofs_and_ai_audit.up.sql",
		"migrations/000005_outbox_and_ai_telemetry.up.sql",
	} {
		if _, statErr := os.Stat(mig); statErr == nil {
			if migErr := omsRepo.EnsureSchema(ctx, mig); migErr != nil {
				log.Warn().Str("file", mig).Err(migErr).Msg("Migration bootstrap warning")
			}
		}
	}

	// Services
	accountService := service.NewAccountService(accountRepo, cfg)
	transferService := service.NewTransferService(transferRepo, idempotencyRepo, cfg)
	reconService := service.NewReconciliationService(reconRepo, transferRepo)
	aiParserService := service.NewAIOrderParserService(omsRepo, cfg)
	omsService := service.NewOMSService(omsRepo, queueRepo, aiParserService, cfg)
	visionService := service.NewAIVisionService(cfg)
	visionService.SetOutboxRepo(outboxRepo)
	reportService := service.NewReportAIService(cfg, reportingRepo, outboxRepo)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	if cfg.EnableEmbeddedWorker {
		go omsService.StartBackgroundWorker(workerCtx)

		// Start Transactional Outbox Publisher Worker to eliminate Dual-Write Problem
		rabbitmqrepo.StartOutboxPublisherWorker(workerCtx, outboxRepo, rabbitProducer, 1*time.Second)

		// Start RabbitMQ Asynchronous Vision AI Payment Proof Worker
		rabbitConsumer.StartPaymentProofWorker(workerCtx, func(pCtx context.Context, payload *domain.PaymentProofVerifyTaskPayload) error {
			imgBytes, _ := base64.StdEncoding.DecodeString(payload.ImageBase64)
			res, err := visionService.AnalyzeBankSlip(pCtx, imgBytes, payload.ImageMimeType, payload.ExpectedAmount)
			if err != nil {
				return err
			}
			newStatus, err := proofRepo.VerifyPaymentSlipAndTransitionOrderAtomic(pCtx, payload.ProofID, payload.OrderID, res)
			if err != nil {
				return err
			}
			if newStatus == domain.OrderStatusPaid {
				ord, _ := omsRepo.GetOrderByID(pCtx, payload.OrderID)
				omsService.FanOutPostPaymentJobs(pCtx, ord)
			}
			return nil
		})
	}

	// HTTP Handlers
	accountHandler := httphandler.NewAccountHandler(accountService)
	transferHandler := httphandler.NewTransferHandler(transferService)
	reconHandler := httphandler.NewReconciliationHandler(reconService)
	omsHandler := httphandler.NewOMSHandler(omsService, aiParserService)
	proofHandler := httphandler.NewPaymentProofHandler(proofRepo, rabbitProducer, omsRepo, outboxRepo)
	reportHandler := httphandler.NewReportHandler(reportService)
	telemetryHandler := httphandler.NewAITelemetryHandler(outboxRepo)

	r := chi.NewRouter()

	// CORS Configuration for Vue Dev Server, Tunnel, & Cross-Origin Webhook Demos
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept", "Authorization", "Content-Type", "X-CSRF-Token",
			"X-Idempotency-Key", "X-Timestamp", "X-Signature",
			"X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature",
			"ngrok-skip-browser-warning",
		},
		ExposedHeaders:   []string{"Link", "X-Idempotency-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(custommiddleware.RecovererMiddleware)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "Omni-Channel OMS & Secure Payment Gateway is active and healthy", map[string]string{
			"status":        "UP",
			"queue_backend": queueRepo.BackendName(),
			"time":          time.Now().UTC().Format(time.RFC3339),
		})
	}
	r.Get("/health", healthHandler)
	r.Head("/health", healthHandler)

	// Serve Vue 3 + TypeScript + Pinia Operational Dashboard (web/dist)
	distDir := filepath.Join("web", "dist")
	assetsDir := filepath.Join(distDir, "assets")
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))

	serveSPA := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	}
	r.Get("/", serveSPA)
	r.Get("/dashboard", serveSPA)

	// API Routes V1
	r.Route("/api/v1", func(r chi.Router) {
		// --- Omni-Channel E-Commerce OMS Endpoints ---
		r.Get("/dashboard/overview", omsHandler.GetDashboardOverview)

		r.Post("/orders/ai-parse", omsHandler.AIParseOrder)
		r.Post("/orders", omsHandler.CreateOrder)
		r.Post("/orders/{id}/cancel", omsHandler.CancelOrder)
		r.Post("/orders/{id}/simulate-pay", omsHandler.SimulatePayOrder)

		r.Post("/products/{id}/stock", omsHandler.SetProductStock)

		// Idempotent & HMAC-SHA256 Verified Payment Webhook
		r.Post("/webhooks/payment", omsHandler.ReceivePaymentWebhook)

		// Multimodal Vision AI Bank Slip Verification (Async RabbitMQ Pipeline)
		r.Post("/payments/proof", proofHandler.UploadProof)
		r.Get("/payments/proof", proofHandler.ListProofs)
		r.Get("/payments/proof/{id}", proofHandler.GetProof)
		r.Get("/payments/proofs", proofHandler.ListProofs)
		r.Get("/payments/proofs/{id}", proofHandler.GetProof)

		// Enterprise Text-to-SQL Reporting Engine (Grounded Schema & AST Guardrails)
		r.Post("/reports/ask-ai", reportHandler.AskAIReport)

		// AI Observability, FinOps Telemetry & Outbox Health
		r.Get("/ai/telemetry", telemetryHandler.GetTelemetrySummary)
		r.Get("/telemetry", telemetryHandler.GetTelemetrySummary)

		// Stress-Test & Queue Chaos Simulators
		r.Post("/simulator/race-condition", omsHandler.SimulateRaceCondition)
		r.Post("/simulator/duplicate-webhook", omsHandler.SimulateDuplicateWebhook)
		r.Post("/simulator/chaos-mode", omsHandler.SetChaosMode)
		r.Post("/queue/jobs/{id}/retry", omsHandler.ManualRetryJob)

		// --- Core Payment Gateway & Double-Entry Ledger Endpoints ---
		r.Get("/accounts", accountHandler.ListAccounts)
		r.Post("/accounts", accountHandler.CreateAccount)
		r.Get("/accounts/{id}", accountHandler.GetAccount)

		r.Get("/ledger", transferHandler.ListLedger)
		r.Get("/audit-logs", transferHandler.ListAuditLogs)

		r.Route("/reconciliation", func(recon chi.Router) {
			recon.Post("/upload", reconHandler.UploadAndReconcile)
			recon.Get("/batches", reconHandler.ListBatches)
			recon.Get("/batches/{id}", reconHandler.GetBatchDetail)
		})

		r.Group(func(protected chi.Router) {
			protected.Use(custommiddleware.HMACAuthMiddleware(cfg))
			protected.Post("/transfers", transferHandler.Transfer)
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Server listening on port", "port", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to listen", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownChan
	slog.Info("Shutdown signal received, shutting down gracefully...")
	workerCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful shutdown failed, forcing exit", "error", err)
		os.Exit(1)
	}

	slog.Info("Omni-Channel OMS & Secure Payment Gateway terminated cleanly")
}
