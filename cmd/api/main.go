package main

import (
	"context"
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
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	redisrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	redisClient, _ := redisrepo.NewRedisClient(cfg)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Repositories
	accountRepo := postgresrepo.NewAccountRepository(pgPool)
	transferRepo := postgresrepo.NewTransferRepository(pgPool)
	reconRepo := postgresrepo.NewReconciliationRepository(pgPool)
	omsRepo := postgresrepo.NewOMSRepository(pgPool)
	idempotencyRepo := redisrepo.NewIdempotencyRepository(redisClient)
	queueRepo := redisrepo.NewQueueRepository(redisClient)

	// Idempotent schema bootstrap for local & containerized environments
	for _, mig := range []string{
		"migrations/000001_init_schema.up.sql",
		"migrations/000002_reconciliation_schema.up.sql",
		"migrations/000003_oms_schema.up.sql",
	} {
		if _, statErr := os.Stat(mig); statErr == nil {
			if migErr := omsRepo.EnsureSchema(ctx, mig); migErr != nil {
				slog.Warn("Migration bootstrap warning", "file", mig, "error", migErr)
			}
		}
	}

	// Services
	accountService := service.NewAccountService(accountRepo, cfg)
	transferService := service.NewTransferService(transferRepo, idempotencyRepo, cfg)
	reconService := service.NewReconciliationService(reconRepo, transferRepo)
	aiParserService := service.NewAIOrderParserService(omsRepo, cfg)
	omsService := service.NewOMSService(omsRepo, queueRepo, aiParserService, cfg)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	if cfg.EnableEmbeddedWorker {
		go omsService.StartBackgroundWorker(workerCtx)
	}

	// HTTP Handlers
	accountHandler := httphandler.NewAccountHandler(accountService)
	transferHandler := httphandler.NewTransferHandler(transferService)
	reconHandler := httphandler.NewReconciliationHandler(reconService)
	omsHandler := httphandler.NewOMSHandler(omsService, aiParserService)

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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "Omni-Channel OMS & Secure Payment Gateway is active and healthy", map[string]string{
			"status":        "UP",
			"queue_backend": queueRepo.BackendName(),
			"time":          time.Now().UTC().Format(time.RFC3339),
		})
	})

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
