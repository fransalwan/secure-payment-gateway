package main

import (
	"context"
	"encoding/base64"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	pkglogger "github.com/fransalwan/secure-payment-gateway/internal/pkg/logger"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	rabbitmqrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/rabbitmq"
	redisrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/redis"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		os.Exit(1)
	}

	pkglogger.InitLogger(cfg.AppEnv)
	log.Info().Msg("Starting Transactional Outbox, RabbitMQ & Redis Queue Worker...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgPool, err := postgresrepo.NewConnectionPool(ctx, cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database for worker")
		os.Exit(1)
	}
	defer pgPool.Close()

	redisClient, _ := redisrepo.NewRedisClient(cfg)
	if redisClient != nil {
		defer redisClient.Close()
	}

	rabbitClient, _ := rabbitmqrepo.NewRabbitMQClient(cfg)
	if rabbitClient != nil {
		defer rabbitClient.Close()
	}
	rabbitConsumer := rabbitmqrepo.NewConsumer(rabbitClient, 10)

	outboxRepo := postgresrepo.NewOutboxRepository(pgPool)
	omsRepo := postgresrepo.NewOMSRepository(pgPool)
	proofRepo := postgresrepo.NewPaymentProofRepository(pgPool)
	queueRepo := redisrepo.NewQueueRepository(redisClient)
	aiParser := service.NewAIOrderParserService(omsRepo, cfg)
	visionService := service.NewAIVisionService(cfg)
	omsService := service.NewOMSService(omsRepo, queueRepo, aiParser, cfg)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go omsService.StartBackgroundWorker(workerCtx)

	// Consume RabbitMQ Vision AI bank slip verification tasks
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

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Info().Msg("Worker listening for outbox, RabbitMQ & Redis queue jobs...")

	for {
		select {
		case <-stopChan:
			log.Info().Msg("Worker stop signal received, shutting down gracefully...")
			workerCancel()
			return
		case <-ticker.C:
			processOutboxBatch(context.Background(), outboxRepo)
		}
	}
}

func processOutboxBatch(ctx context.Context, repo *postgresrepo.OutboxRepository) {
	events, err := repo.GetPendingEvents(ctx, 10)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch pending outbox events")
		return
	}

	if len(events) == 0 {
		return
	}

	log.Info().Int("count", len(events)).Msg("Processing outbox events batch")

	for _, ev := range events {
		log.Info().
			Str("event_id", ev.ID.String()).
			Str("event_type", ev.EventType).
			Msg("Dispatching outbox event")

		if err := repo.MarkProcessed(ctx, ev.ID); err != nil {
			log.Error().Err(err).Str("event_id", ev.ID.String()).Msg("Failed to mark outbox event as processed")
		}
	}
}
