package rabbitmq

import (
	"fmt"
	"sync"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type Client struct {
	cfg         *config.Config
	conn        *amqp.Connection
	ch          *amqp.Channel
	isConnected bool
	mu          sync.RWMutex

	// In-memory fallback buffers if broker is unreachable (graceful degradation)
	fallbackOrderParseChan chan []byte
	fallbackPaymentChan    chan []byte
}

func NewRabbitMQClient(cfg *config.Config) (*Client, error) {
	client := &Client{
		cfg:                    cfg,
		fallbackOrderParseChan: make(chan []byte, 1000),
		fallbackPaymentChan:    make(chan []byte, 1000),
	}

	url := cfg.GetRabbitMQURL()
	log.Info().Str("host", cfg.RabbitMQHost).Str("port", cfg.RabbitMQPort).Msg("Attempting connection to RabbitMQ broker...")

	// Attempt dial with 3.5s timeout
	conn, err := amqp.DialConfig(url, amqp.Config{
		Dial: amqp.DefaultDial(3500 * time.Millisecond),
	})
	if err != nil {
		log.Warn().Err(err).Msg("RabbitMQ not reachable, operating in resilient in-memory fallback mode")
		return client, nil
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		log.Warn().Err(err).Msg("Failed to open RabbitMQ channel, operating in fallback mode")
		return client, nil
	}

	client.conn = conn
	client.ch = ch
	client.isConnected = true

	// Initialize Topology (Topic Exchange, Dead-Letter Exchange & Queues)
	if err := client.setupTopology(); err != nil {
		log.Error().Err(err).Msg("Failed to declare RabbitMQ topology, falling back to local memory")
		client.Close()
		client.isConnected = false
		return client, nil
	}

	log.Info().
		Str("exchange", domain.RabbitMQExchangeEvents).
		Str("dlx", domain.RabbitMQExchangeDLX).
		Msg("RabbitMQ connection & enterprise queue topology established successfully")

	return client, nil
}

func (c *Client) setupTopology() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Declare Topic Exchange for primary OMS events
	if err := c.ch.ExchangeDeclare(
		domain.RabbitMQExchangeEvents,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange %s: %w", domain.RabbitMQExchangeEvents, err)
	}

	// 2. Declare Dead-Letter Exchange (DLX)
	if err := c.ch.ExchangeDeclare(
		domain.RabbitMQExchangeDLX,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dlx %s: %w", domain.RabbitMQExchangeDLX, err)
	}

	// 3. Declare Dead-Letter Queue (DLQ)
	if _, err := c.ch.QueueDeclare(
		domain.RabbitMQQueueDLQ,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare dlq %s: %w", domain.RabbitMQQueueDLQ, err)
	}
	if err := c.ch.QueueBind(domain.RabbitMQQueueDLQ, "", domain.RabbitMQExchangeDLX, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	// 4. Declare Order Parse Queue with DLX configuration
	orderQueueArgs := amqp.Table{
		"x-dead-letter-exchange": domain.RabbitMQExchangeDLX,
	}
	if _, err := c.ch.QueueDeclare(
		domain.RabbitMQQueueOrderParse,
		true,
		false,
		false,
		false,
		orderQueueArgs,
	); err != nil {
		return fmt.Errorf("declare queue %s: %w", domain.RabbitMQQueueOrderParse, err)
	}
	if err := c.ch.QueueBind(
		domain.RabbitMQQueueOrderParse,
		domain.RabbitMQRoutingOrderParse,
		domain.RabbitMQExchangeEvents,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue %s: %w", domain.RabbitMQQueueOrderParse, err)
	}

	// 5. Declare Payment Proof Verify Queue with DLX configuration
	paymentQueueArgs := amqp.Table{
		"x-dead-letter-exchange": domain.RabbitMQExchangeDLX,
	}
	if _, err := c.ch.QueueDeclare(
		domain.RabbitMQQueuePaymentVerify,
		true,
		false,
		false,
		false,
		paymentQueueArgs,
	); err != nil {
		return fmt.Errorf("declare queue %s: %w", domain.RabbitMQQueuePaymentVerify, err)
	}
	if err := c.ch.QueueBind(
		domain.RabbitMQQueuePaymentVerify,
		domain.RabbitMQRoutingPaymentVerify,
		domain.RabbitMQExchangeEvents,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue %s: %w", domain.RabbitMQQueuePaymentVerify, err)
	}

	return nil
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected && c.conn != nil && !c.conn.IsClosed()
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.isConnected = false
}
