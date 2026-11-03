package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/rs/zerolog/log"
)

// StartOutboxPublisherWorker launches a resilient background poller that checks outbox_events
// and dispatches them to RabbitMQ, guaranteeing At-Least-Once delivery and solving the Dual-Write Problem.
func StartOutboxPublisherWorker(
	ctx context.Context,
	outboxRepo *postgresrepo.OutboxRepository,
	producer Producer,
	pollInterval time.Duration,
) {
	if outboxRepo == nil || producer == nil {
		return
	}
	if pollInterval <= 0 {
		pollInterval = 1 * time.Second
	}

	log.Info().
		Dur("poll_interval", pollInterval).
		Msg("Transactional Outbox Publisher Worker started")

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Outbox Publisher Worker stopping")
				return
			case <-ticker.C:
				processPendingOutbox(ctx, outboxRepo, producer)
			}
		}
	}()
}

func processPendingOutbox(
	ctx context.Context,
	outboxRepo *postgresrepo.OutboxRepository,
	producer Producer,
) {
	events, err := outboxRepo.FetchPendingOutboxEvents(ctx, 10)
	if err != nil || len(events) == 0 {
		return
	}

	for _, e := range events {
		var pubErr error
		switch e.EventType {
		case domain.RabbitMQRoutingPaymentVerify:
			var payload domain.PaymentProofVerifyTaskPayload
			if err := json.Unmarshal(e.Payload, &payload); err == nil {
				pubErr = producer.PublishPaymentProofVerify(ctx, &payload)
			} else {
				pubErr = err
			}
		case domain.RabbitMQRoutingOrderParse:
			var payload domain.OrderParseTaskPayload
			if err := json.Unmarshal(e.Payload, &payload); err == nil {
				pubErr = producer.PublishOrderParse(ctx, &payload)
			} else {
				pubErr = err
			}
		default:
			log.Warn().Str("event_type", e.EventType).Msg("Unknown outbox event type")
		}

		if pubErr == nil {
			_ = outboxRepo.MarkOutboxProcessed(ctx, e.ID)
			log.Debug().
				Str("event_id", e.ID).
				Str("event_type", e.EventType).
				Msg("Outbox event published and marked PROCESSED")
		} else {
			_ = outboxRepo.MarkOutboxFailed(ctx, e.ID, pubErr.Error())
		}
	}
}

