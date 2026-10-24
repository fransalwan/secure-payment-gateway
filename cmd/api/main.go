package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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

	slog.Info("Starting Secure Payment Gateway...")

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	accountRepo := postgresrepo.NewAccountRepository(pgPool)
	transferRepo := postgresrepo.NewTransferRepository(pgPool)
	reconRepo := postgresrepo.NewReconciliationRepository(pgPool)
	idempotencyRepo := redisrepo.NewIdempotencyRepository(redisClient)

	accountService := service.NewAccountService(accountRepo, cfg)
	transferService := service.NewTransferService(transferRepo, idempotencyRepo, cfg)
	reconService := service.NewReconciliationService(reconRepo, transferRepo)

	accountHandler := httphandler.NewAccountHandler(accountService)
	transferHandler := httphandler.NewTransferHandler(transferService)
	reconHandler := httphandler.NewReconciliationHandler(reconService)

	r := chi.NewRouter()

	// CORS Configuration for Tunnel & Cross-Origin Public Demos
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Idempotency-Key", "X-Timestamp", "X-Signature", "ngrok-skip-browser-warning"},
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
		response.OK(w, "Secure Payment Gateway is active and healthy", map[string]string{
			"status": "UP",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Serve Frontend Dashboard
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})
	r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	// API Routes V1
	r.Route("/api/v1", func(r chi.Router) {
		// Accounts Endpoints
		r.Get("/accounts", accountHandler.ListAccounts)
		r.Post("/accounts", accountHandler.CreateAccount)
		r.Get("/accounts/{id}", accountHandler.GetAccount)

		// Ledger & Audit Chain Endpoints
		r.Get("/ledger", transferHandler.ListLedger)
		r.Get("/audit-logs", transferHandler.ListAuditLogs)

		// Bank Reconciliation Endpoints
		r.Route("/reconciliation", func(recon chi.Router) {
			recon.Post("/upload", reconHandler.UploadAndReconcile)
			recon.Get("/batches", reconHandler.ListBatches)
			recon.Get("/batches/{id}", reconHandler.GetBatchDetail)
		})

		// Transfers Endpoint Protected by HMAC-SHA256 & Anti-Replay
		r.Group(func(protected chi.Router) {
			protected.Use(custommiddleware.HMACAuthMiddleware(cfg))
			protected.Post("/transfers", transferHandler.Transfer)
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful shutdown failed, forcing exit", "error", err)
		os.Exit(1)
	}

	slog.Info("Secure Payment Gateway terminated cleanly")
}
