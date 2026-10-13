package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Transactional Outbox Worker...")

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("Failed to connect to database for outbox worker", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	outboxRepo := postgresrepo.NewOutboxRepository(pgPool)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	slog.Info("Outbox worker listening for pending events...")

	for {
		select {
		case <-stopChan:
			slog.Info("Worker stop signal received, shutting down gracefully...")
			return
		case <-ticker.C:
			processOutboxBatch(context.Background(), outboxRepo)
		}
	}
}

func processOutboxBatch(ctx context.Context, repo *postgresrepo.OutboxRepository) {
	events, err := repo.GetPendingEvents(ctx, 10)
	if err != nil {
		slog.Error("Failed to fetch pending outbox events", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	slog.Info("Processing outbox events batch", "count", len(events))

	for _, ev := range events {
		slog.Info("Dispatching outbox event",
			"event_id", ev.ID,
			"event_type", ev.EventType,
			"payload", string(ev.Payload),
		)

		if err := repo.MarkProcessed(ctx, ev.ID); err != nil {
			slog.Error("Failed to mark outbox event as processed", "event_id", ev.ID, "error", err)
		}
	}
}

