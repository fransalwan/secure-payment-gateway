package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type Producer interface {
	PublishOrderParse(ctx context.Context, payload *domain.OrderParseTaskPayload) error
	PublishPaymentProofVerify(ctx context.Context, payload *domain.PaymentProofVerifyTaskPayload) error
}

type EventProducer struct {
	client *Client
}

func NewProducer(client *Client) *EventProducer {
	return &EventProducer{client: client}
}

func (p *EventProducer) PublishOrderParse(ctx context.Context, payload *domain.OrderParseTaskPayload) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = time.Now().UTC()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal order parse payload: %w", err)
	}

	traceLog := log.With().
		Str("trace_id", payload.TraceID).
		Str("order_id", payload.OrderID.String()).
		Logger()

	if !p.client.IsConnected() {
		traceLog.Debug().Msg("Publishing OrderParse to in-memory fallback queue")
		select {
		case p.client.fallbackOrderParseChan <- body:
			return nil
		default:
			return fmt.Errorf("in-memory order parse buffer is full")
		}
	}

	p.client.mu.RLock()
	defer p.client.mu.RUnlock()

	pubErr := p.client.ch.PublishWithContext(
		ctx,
		domain.RabbitMQExchangeEvents,
		domain.RabbitMQRoutingOrderParse,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			Headers: amqp.Table{
				"trace_id": payload.TraceID,
				"attempt":  payload.Attempt,
			},
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    payload.CreatedAt,
			Body:         body,
		},
	)
	if pubErr != nil {
		traceLog.Error().Err(pubErr).Msg("Failed to publish OrderParse event to RabbitMQ, falling back to buffer")
		select {
		case p.client.fallbackOrderParseChan <- body:
			return nil
		default:
			return pubErr
		}
	}

	traceLog.Info().
		Str("exchange", domain.RabbitMQExchangeEvents).
		Str("routing_key", domain.RabbitMQRoutingOrderParse).
		Msg("OrderParse event published to RabbitMQ successfully")

	return nil
}

func (p *EventProducer) PublishPaymentProofVerify(ctx context.Context, payload *domain.PaymentProofVerifyTaskPayload) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = time.Now().UTC()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payment proof payload: %w", err)
	}

	traceLog := log.With().
		Str("trace_id", payload.TraceID).
		Str("proof_id", payload.ProofID.String()).
		Str("order_id", payload.OrderID.String()).
		Logger()

	if !p.client.IsConnected() {
		traceLog.Debug().Msg("Publishing PaymentProofVerify to in-memory fallback queue")
		select {
		case p.client.fallbackPaymentChan <- body:
			return nil
		default:
			return fmt.Errorf("in-memory payment proof buffer is full")
		}
	}

	p.client.mu.RLock()
	defer p.client.mu.RUnlock()

	pubErr := p.client.ch.PublishWithContext(
		ctx,
		domain.RabbitMQExchangeEvents,
		domain.RabbitMQRoutingPaymentVerify,
		false,
		false,
		amqp.Publishing{
			Headers: amqp.Table{
				"trace_id": payload.TraceID,
				"proof_id": payload.ProofID.String(),
				"attempt":  payload.Attempt,
			},
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    payload.CreatedAt,
			Body:         body,
		},
	)
	if pubErr != nil {
		traceLog.Error().Err(pubErr).Msg("Failed to publish PaymentProofVerify event to RabbitMQ, falling back to buffer")
		select {
		case p.client.fallbackPaymentChan <- body:
			return nil
		default:
			return pubErr
		}
	}

	traceLog.Info().
		Str("exchange", domain.RabbitMQExchangeEvents).
		Str("routing_key", domain.RabbitMQRoutingPaymentVerify).
		Msg("PaymentProofVerify event published to RabbitMQ successfully")

	return nil
}
