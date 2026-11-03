package domain

import (
	"encoding/json"
	"time"
)

// OutboxStatus defines state of transactional outbox record
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusProcessed OutboxStatus = "PROCESSED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

// OutboxEvent represents an atomic event record persisted in the same DB transaction
// to eliminate the Dual-Write Problem between PostgreSQL and the RabbitMQ Message Broker.
type OutboxEvent struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Status        OutboxStatus    `json:"status"`
	RetryCount    int             `json:"retry_count"`
	ErrorMessage  *string         `json:"error_message,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
}

// AITelemetryEvent records token consumption, model latencies, and FinOps metrics
type AITelemetryEvent struct {
	ID               string                 `json:"id"`
	ServiceType      string                 `json:"service_type"` // 'VISION_SLIP', 'TEXT_TO_SQL', 'SOCIAL_CHAT'
	ModelName        string                 `json:"model_name"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	LatencyMs        int                    `json:"latency_ms"`
	CostUSD          float64                `json:"cost_usd"`
	IsSuccess        bool                   `json:"is_success"`
	FraudFlag        bool                   `json:"fraud_flag"`
	Details          map[string]interface{} `json:"details,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// AITelemetrySummary provides high-level observability for the FinOps & AI Telemetry Cockpit
type AITelemetrySummary struct {
	TotalCalls           int64   `json:"total_calls"`
	VisionCalls          int64   `json:"vision_calls"`
	TextToSqlCalls       int64   `json:"text_to_sql_calls"`
	TotalTokens          int64   `json:"total_tokens"`
	TotalCostUSD         float64 `json:"total_cost_usd"`
	AvgLatencyMs         float64 `json:"avg_latency_ms"`
	FraudQuarantineCount int64   `json:"fraud_quarantine_count"`
	FraudRatePct         float64 `json:"fraud_rate_pct"`
	ASTBlocksCount       int64   `json:"ast_blocks_count"`
	DuplicatePhashCount  int64   `json:"duplicate_phash_count"`
	OutboxPendingCount   int64   `json:"outbox_pending_count"`
	OutboxProcessedCount int64   `json:"outbox_processed_count"`
}
