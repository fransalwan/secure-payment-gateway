package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusReadyToShip    OrderStatus = "READY_TO_SHIP"
	OrderStatusShipped        OrderStatus = "SHIPPED"
	OrderStatusCompleted      OrderStatus = "COMPLETED"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
)

type OrderChannel string

const (
	ChannelWeb          OrderChannel = "WEB"
	ChannelWhatsAppLive OrderChannel = "WHATSAPP_LIVE"
	ChannelShopee       OrderChannel = "SHOPEE"
	ChannelTikTokShop   OrderChannel = "TIKTOK_SHOP"
)

type WebhookEventStatus string

const (
	WebhookStatusReceived         WebhookEventStatus = "RECEIVED"
	WebhookStatusProcessed        WebhookEventStatus = "PROCESSED"
	WebhookStatusFailed           WebhookEventStatus = "FAILED"
	WebhookStatusIgnoredDuplicate WebhookEventStatus = "IGNORED_DUPLICATE"
)

type QueueJobStatus string

const (
	JobStatusPending   QueueJobStatus = "PENDING"
	JobStatusActive    QueueJobStatus = "ACTIVE"
	JobStatusRetrying  QueueJobStatus = "RETRYING"
	JobStatusCompleted QueueJobStatus = "COMPLETED"
	JobStatusFailedDLQ QueueJobStatus = "FAILED"
)

const (
	QueuePaymentWebhook   = "payment-webhook-queue"
	QueueLogisticsWaybill = "logistics-waybill-queue"
	QueueChannelStockSync = "channel-stock-sync-queue"
	QueueNotification     = "notification-queue"
)

type ChaosFailMode string

const (
	ChaosModeOff            ChaosFailMode = "OFF"
	ChaosModeTransientTwice ChaosFailMode = "TRANSIENT_TWICE"
	ChaosModeAlways429      ChaosFailMode = "ALWAYS_429"
)

type Merchant struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID             uuid.UUID `json:"id"`
	MerchantID     uuid.UUID `json:"merchant_id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	Price          float64   `json:"price"`
	AvailableStock int       `json:"available_stock"`
	ReservedStock  int       `json:"reserved_stock"`
	SoldStock      int       `json:"sold_stock"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type OrderItem struct {
	ID          uuid.UUID `json:"id"`
	OrderID     uuid.UUID `json:"order_id"`
	ProductID   uuid.UUID `json:"product_id"`
	SKU         string    `json:"sku"`
	ProductName string    `json:"product_name"`
	Qty         int       `json:"qty"`
	UnitPrice   float64   `json:"unit_price"`
	Subtotal    float64   `json:"subtotal"`
}

type Order struct {
	ID               uuid.UUID    `json:"id"`
	MerchantID       uuid.UUID    `json:"merchant_id"`
	OrderNumber      string       `json:"order_number"`
	Channel          OrderChannel `json:"channel"`
	ExternalOrderID  string       `json:"external_order_id,omitempty"`
	CustomerName     string       `json:"customer_name"`
	CustomerPhone    string       `json:"customer_phone"`
	ShippingAddress  string       `json:"shipping_address"`
	CourierCode      string       `json:"courier_code"`
	Status           OrderStatus  `json:"status"`
	TotalAmount      float64      `json:"total_amount"`
	WaybillNumber    string       `json:"waybill_number,omitempty"`
	PaymentURL       string       `json:"payment_url,omitempty"`
	PaymentExpiresAt time.Time    `json:"payment_expires_at"`
	PaidAt           *time.Time   `json:"paid_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	Items            []OrderItem  `json:"items,omitempty"`
}

type CreateOrderItemDTO struct {
	ProductID uuid.UUID `json:"product_id"`
	Qty       int       `json:"qty"`
}

type CreateOrderRequestDTO struct {
	MerchantID      uuid.UUID            `json:"merchant_id"`
	Channel         OrderChannel         `json:"channel"`
	ExternalOrderID string               `json:"external_order_id,omitempty"`
	CustomerName    string               `json:"customer_name"`
	CustomerPhone   string               `json:"customer_phone"`
	ShippingAddress string               `json:"shipping_address"`
	CourierCode     string               `json:"courier_code"`
	Items           []CreateOrderItemDTO `json:"items"`
}

type UpdateStockRequestDTO struct {
	AvailableStock int `json:"available_stock"`
}

// AI Social Commerce Order Parser DTOs
type AIParseRequestDTO struct {
	MerchantID uuid.UUID `json:"merchant_id"`
	RawText    string    `json:"raw_text"`
}

type AIParsedItemDTO struct {
	ProductID          uuid.UUID `json:"sku_id"`
	SKU                string    `json:"sku"`
	MatchedProductName string    `json:"matched_product_name"`
	Qty                int       `json:"qty"`
	UnitPrice          float64   `json:"unit_price"`
	Subtotal           float64   `json:"subtotal"`
	AvailableStock     int       `json:"available_stock"`
	StockSufficient    bool      `json:"stock_sufficient"`
	StockWarning       string    `json:"stock_warning,omitempty"`
}

type AIParseResponseDTO struct {
	CustomerName         string            `json:"customer_name"`
	CustomerPhone        string            `json:"customer_phone"`
	ShippingAddress      string            `json:"shipping_address"`
	CourierCode          string            `json:"courier_code"`
	Items                []AIParsedItemDTO `json:"items"`
	EstimatedTotal       float64           `json:"estimated_total"`
	HasInsufficientStock bool              `json:"has_insufficient_stock"`
	ParserEngine         string            `json:"parser_engine"`
	LatencyMs            int64             `json:"latency_ms"`
}

// Webhook & Idempotency DTOs
type WebhookEvent struct {
	ID              uuid.UUID          `json:"id"`
	Provider        string             `json:"provider"`
	ExternalEventID string             `json:"external_event_id"`
	EventType       string             `json:"event_type"`
	Payload         json.RawMessage    `json:"payload"`
	Status          WebhookEventStatus `json:"status"`
	RetryCount      int                `json:"retry_count"`
	ProcessingMs    int                `json:"processing_ms"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	ProcessedAt     *time.Time         `json:"processed_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type PaymentWebhookPayloadDTO struct {
	Provider    string  `json:"provider"`
	EventType   string  `json:"event_type"`
	OrderID     string  `json:"order_id,omitempty"`
	OrderNumber string  `json:"order_number"`
	Amount      float64 `json:"amount"`
	PaidAt      string  `json:"paid_at,omitempty"`
}

type Shipment struct {
	ID                   uuid.UUID       `json:"id"`
	OrderID              uuid.UUID       `json:"order_id"`
	CourierCode          string          `json:"courier_code"`
	WaybillNumber        string          `json:"waybill_number"`
	Status               string          `json:"status"`
	ShippingLabelPayload json.RawMessage `json:"shipping_label_payload,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}

// Redis + Background Queue Job & DLQ Models
type QueueJob struct {
	ID          string          `json:"id"`
	QueueName   string          `json:"queue_name"`
	OrderID     string          `json:"order_id"`
	OrderNumber string          `json:"order_number"`
	Payload     json.RawMessage `json:"payload"`
	Status      QueueJobStatus  `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	LastError   string          `json:"last_error,omitempty"`
	NextRetryAt *time.Time      `json:"next_retry_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type QueueMetricsDTO struct {
	ActiveCount    int           `json:"active_count"`
	RetryingCount  int           `json:"retrying_count"`
	CompletedCount int           `json:"completed_count"`
	FailedDLQCount int           `json:"failed_dlq_count"`
	ChaosMode      ChaosFailMode `json:"chaos_mode"`
	QueueBackend   string        `json:"queue_backend"`
	RecentJobs     []*QueueJob   `json:"recent_jobs"`
}

type SystemErrorLog struct {
	ID           uuid.UUID       `json:"id"`
	ServiceName  string          `json:"service_name"`
	QueueName    string          `json:"queue_name"`
	JobID        string          `json:"job_id"`
	ReferenceID  string          `json:"reference_id,omitempty"`
	ErrorCode    string          `json:"error_code"`
	ErrorMessage string          `json:"error_message"`
	Attempts     int             `json:"attempts"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Resolved     bool            `json:"resolved"`
	CreatedAt    time.Time       `json:"created_at"`
	ResolvedAt   *time.Time      `json:"resolved_at,omitempty"`
}

// Stress-Test Simulator DTOs
type RaceSimulationRequestDTO struct {
	MerchantID       uuid.UUID `json:"merchant_id"`
	ProductID        uuid.UUID `json:"product_id"`
	ConcurrentBuyers int       `json:"concurrent_buyers"`
	ResetStockTo     int       `json:"reset_stock_to"`
}

type RaceBuyerAttemptDTO struct {
	BuyerIndex  int     `json:"buyer_index"`
	BuyerName   string  `json:"buyer_name"`
	Channel     string  `json:"channel"`
	HTTPStatus  int     `json:"http_status"`
	Outcome     string  `json:"outcome"`
	OrderNumber string  `json:"order_number,omitempty"`
	Reason      string  `json:"reason"`
	LatencyMs   float64 `json:"latency_ms"`
}

type RaceSimulationResultDTO struct {
	ProductSKU         string                `json:"product_sku"`
	ProductName        string                `json:"product_name"`
	InitialStock       int                   `json:"initial_stock"`
	ConcurrentBuyers   int                   `json:"concurrent_buyers"`
	SuccessCount       int                   `json:"success_count"`
	ConflictCount      int                   `json:"conflict_count"`
	FinalAvailable     int                   `json:"final_available_stock"`
	FinalReserved      int                   `json:"final_reserved_stock"`
	ZeroOversellProven bool                  `json:"zero_oversell_proven"`
	TotalDurationMs    float64               `json:"total_duration_ms"`
	Attempts           []RaceBuyerAttemptDTO `json:"attempts"`
}

type DuplicateWebhookSimRequestDTO struct {
	MerchantID     uuid.UUID `json:"merchant_id"`
	OrderID        string    `json:"order_id,omitempty"`
	DuplicateCount int       `json:"duplicate_count"`
}

type WebhookAttemptDTO struct {
	AttemptIndex  int     `json:"attempt_index"`
	WebhookID     string  `json:"webhook_id"`
	HTTPStatus    int     `json:"http_status"`
	StatusResult  string  `json:"status_result"`
	LatencyMs     float64 `json:"latency_ms"`
	SignatureUsed string  `json:"signature_used"`
}

type DuplicateWebhookSimResultDTO struct {
	OrderID            string              `json:"order_id"`
	OrderNumber        string              `json:"order_number"`
	ExternalEventID    string              `json:"external_event_id"`
	TotalSent          int                 `json:"total_sent"`
	ProcessedCount     int                 `json:"processed_count"`
	DuplicateIgnored   int                 `json:"duplicate_ignored"`
	IdempotencyProven  bool                `json:"idempotency_proven"`
	OrderFinalStatus   OrderStatus         `json:"order_final_status"`
	Attempts           []WebhookAttemptDTO `json:"attempts"`
}

type DashboardOverviewDTO struct {
	Merchants     []Merchant       `json:"merchants"`
	ActiveMerchant *Merchant       `json:"active_merchant,omitempty"`
	Products      []Product        `json:"products"`
	Orders        []Order          `json:"orders"`
	WebhookEvents []WebhookEvent   `json:"webhook_events"`
	QueueMetrics  QueueMetricsDTO  `json:"queue_metrics"`
	ErrorLogs     []SystemErrorLog `json:"error_logs"`
}

