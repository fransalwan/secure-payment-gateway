package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type OrderParseHandler func(ctx context.Context, payload *domain.OrderParseTaskPayload) error
type PaymentProofVerifyHandler func(ctx context.Context, payload *domain.PaymentProofVerifyTaskPayload) error

type EventConsumer struct {
	client        *Client
	prefetchCount int
}

func NewConsumer(client *Client, prefetchCount int) *EventConsumer {
	if prefetchCount <= 0 {
		prefetchCount = 10
	}
	return &EventConsumer{
		client:        client,
		prefetchCount: prefetchCount,
	}
}

// StartOrderParseWorker listens to order.parse.queue with manual ACK and backoff
func (c *EventConsumer) StartOrderParseWorker(ctx context.Context, handler OrderParseHandler) {
	if !c.client.IsConnected() {
		log.Info().Msg("Starting in-memory fallback worker for OrderParse")
		go c.runFallbackOrderParse(ctx, handler)
		return
	}

	c.client.mu.Lock()
	if err := c.client.ch.Qos(c.prefetchCount, 0, false); err != nil {
		log.Warn().Err(err).Msg("Failed to set QoS on RabbitMQ channel")
	}
	msgs, err := c.client.ch.Consume(
		domain.RabbitMQQueueOrderParse,
		"oms-order-parse-worker",
		false, // manual ACK
		false,
		false,
		false,
		nil,
	)
	c.client.mu.Unlock()

	if err != nil {
		log.Error().Err(err).Msg("Failed to start RabbitMQ consumer for OrderParse, switching to memory fallback")
		go c.runFallbackOrderParse(ctx, handler)
		return
	}

	log.Info().
		Str("queue", domain.RabbitMQQueueOrderParse).
		Int("prefetch", c.prefetchCount).
		Msg("RabbitMQ OrderParse Worker started listening")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Warn().Msg("RabbitMQ OrderParse delivery channel closed")
					return
				}
				c.processOrderParseDelivery(ctx, msg, handler)
			}
		}
	}()
}

func (c *EventConsumer) processOrderParseDelivery(ctx context.Context, msg amqp.Delivery, handler OrderParseHandler) {
	var payload domain.OrderParseTaskPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Error().Err(err).Msg("Malformed OrderParse message body, rejecting to DLQ")
		_ = msg.Reject(false)
		return
	}

	taskLog := log.With().
		Str("trace_id", payload.TraceID).
		Str("queue", domain.RabbitMQQueueOrderParse).
		Int("attempt", payload.Attempt).
		Logger()

	// Strict SLA timeout (3500ms) for AI text extraction
	procCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
	defer cancel()

	err := handler(procCtx, &payload)
	if err == nil {
		taskLog.Info().Msg("OrderParse task processed successfully, sending ACK")
		_ = msg.Ack(false)
		return
	}

	taskLog.Warn().Err(err).Msg("OrderParse handler failed, evaluating retry backoff")

	payload.Attempt++
	if payload.Attempt < 3 {
		backoff := time.Duration(1<<payload.Attempt) * time.Second
		time.Sleep(backoff)
		_ = msg.Ack(false)

		// Re-enqueue updated attempt
		producer := NewProducer(c.client)
		_ = producer.PublishOrderParse(ctx, &payload)
		return
	}

	taskLog.Error().
		Int("max_attempts", 3).
		Msg("Max retry attempts reached for OrderParse. Rejecting message to DLX (oms.dlq)")
	_ = msg.Reject(false) // Sent to Dead-Letter Exchange
}

// StartPaymentProofWorker listens to payment.verify.proof with manual ACK and backoff
func (c *EventConsumer) StartPaymentProofWorker(ctx context.Context, handler PaymentProofVerifyHandler) {
	if !c.client.IsConnected() {
		log.Info().Msg("Starting in-memory fallback worker for PaymentProofVerify")
		go c.runFallbackPaymentProof(ctx, handler)
		return
	}

	c.client.mu.Lock()
	if err := c.client.ch.Qos(c.prefetchCount, 0, false); err != nil {
		log.Warn().Err(err).Msg("Failed to set QoS on RabbitMQ channel")
	}
	msgs, err := c.client.ch.Consume(
		domain.RabbitMQQueuePaymentVerify,
		"oms-payment-verify-worker",
		false, // manual ACK
		false,
		false,
		false,
		nil,
	)
	c.client.mu.Unlock()

	if err != nil {
		log.Error().Err(err).Msg("Failed to start RabbitMQ consumer for PaymentProof, switching to memory fallback")
		go c.runFallbackPaymentProof(ctx, handler)
		return
	}

	log.Info().
		Str("queue", domain.RabbitMQQueuePaymentVerify).
		Int("prefetch", c.prefetchCount).
		Msg("RabbitMQ PaymentProof Worker started listening")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Warn().Msg("RabbitMQ PaymentProof delivery channel closed")
					return
				}
				c.processPaymentProofDelivery(ctx, msg, handler)
			}
		}
	}()
}

func (c *EventConsumer) processPaymentProofDelivery(ctx context.Context, msg amqp.Delivery, handler PaymentProofVerifyHandler) {
	var payload domain.PaymentProofVerifyTaskPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Error().Err(err).Msg("Malformed PaymentProof message body, rejecting to DLQ")
		_ = msg.Reject(false)
		return
	}

	taskLog := log.With().
		Str("trace_id", payload.TraceID).
		Str("proof_id", payload.ProofID.String()).
		Str("order_id", payload.OrderID.String()).
		Int("attempt", payload.Attempt).
		Logger()

	// Strict SLA timeout (5000ms) for Vision AI slip analysis
	procCtx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()

	err := handler(procCtx, &payload)
	if err == nil {
		taskLog.Info().Msg("PaymentProof task processed successfully, sending ACK")
		_ = msg.Ack(false)
		return
	}

	taskLog.Warn().Err(err).Msg("PaymentProof handler failed, evaluating retry backoff")

	payload.Attempt++
	if payload.Attempt < 3 {
		backoff := time.Duration(1<<payload.Attempt) * time.Second
		time.Sleep(backoff)
		_ = msg.Ack(false)

		producer := NewProducer(c.client)
		_ = producer.PublishPaymentProofVerify(ctx, &payload)
		return
	}

	taskLog.Error().
		Int("max_attempts", 3).
		Msg("Max retry attempts reached for PaymentProof. Rejecting message to DLX (oms.dlq)")
	_ = msg.Reject(false)
}

// Fallback handlers for development / testing when RabbitMQ broker is offline
func (c *EventConsumer) runFallbackOrderParse(ctx context.Context, handler OrderParseHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case body := <-c.client.fallbackOrderParseChan:
			var payload domain.OrderParseTaskPayload
			if err := json.Unmarshal(body, &payload); err == nil {
				procCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
				_ = handler(procCtx, &payload)
				cancel()
			}
		}
	}
}

func (c *EventConsumer) runFallbackPaymentProof(ctx context.Context, handler PaymentProofVerifyHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case body := <-c.client.fallbackPaymentChan:
			var payload domain.PaymentProofVerifyTaskPayload
			if err := json.Unmarshal(body, &payload); err == nil {
				procCtx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
				_ = handler(procCtx, &payload)
				cancel()
			}
		}
	}
}
