package domain

import (
	"time"

	"github.com/google/uuid"
)

// RabbitMQ Exchange & Queue Names
const (
	RabbitMQExchangeEvents = "oms.events"
	RabbitMQExchangeDLX    = "oms.dlx"

	RabbitMQRoutingOrderParse     = "order.parse"
	RabbitMQRoutingPaymentVerify  = "payment.verify"

	RabbitMQQueueOrderParse     = "order.parse.queue"
	RabbitMQQueuePaymentVerify  = "payment.verify.proof"
	RabbitMQQueueDLQ            = "oms.dlq"
)

// RabbitMQ Message Payloads
type OrderParseTaskPayload struct {
	TraceID    string    `json:"trace_id"`
	MerchantID uuid.UUID `json:"merchant_id"`
	OrderID    uuid.UUID `json:"order_id,omitempty"`
	RawChat    string    `json:"raw_chat"`
	Attempt    int       `json:"attempt"`
	CreatedAt  time.Time `json:"created_at"`
}

type PaymentProofVerifyTaskPayload struct {
	TraceID         string    `json:"trace_id"`
	ProofID         uuid.UUID `json:"proof_id"`
	OrderID         uuid.UUID `json:"order_id"`
	MerchantID      uuid.UUID `json:"merchant_id"`
	ImageBase64     string    `json:"image_base64,omitempty"`
	ImageMimeType   string    `json:"image_mime_type,omitempty"`
	FileName        string    `json:"file_name"`
	ExpectedAmount  float64   `json:"expected_amount"`
	Attempt         int       `json:"attempt"`
	CreatedAt       time.Time `json:"created_at"`
}

// PaymentProofStatus represents verification lifecycle of bank slips
type PaymentProofStatus string

const (
	ProofStatusPending        PaymentProofStatus = "PENDING"
	ProofStatusVerified       PaymentProofStatus = "VERIFIED"
	ProofStatusSuspiciousHold PaymentProofStatus = "SUSPICIOUS_HOLD"
	ProofStatusRejected       PaymentProofStatus = "REJECTED"
)

// PaymentProof Entity
type PaymentProof struct {
	ID              uuid.UUID          `json:"id"`
	OrderID         uuid.UUID          `json:"order_id"`
	MerchantID      uuid.UUID          `json:"merchant_id"`
	FileName        string             `json:"file_name"`
	BankName        string             `json:"bank_name"`
	SenderName      string             `json:"sender_name"`
	Amount          float64            `json:"amount"`
	ReferenceNumber string             `json:"reference_number"`
	TransferTime    *time.Time         `json:"transfer_time,omitempty"`
	TamperDetected  bool               `json:"tamper_detected"`
	ConfidenceScore float64            `json:"confidence_score"`
	Status          PaymentProofStatus `json:"status"`
	RejectionReason string             `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	ProcessedAt     *time.Time         `json:"processed_at,omitempty"`
}

// VisionSlipExtractionResult from multimodal LLM
type VisionSlipExtractionResult struct {
	BankName        string     `json:"bank_name"`
	SenderName      string     `json:"sender_name"`
	Amount          float64    `json:"amount"`
	ReferenceNumber string     `json:"reference_number"`
	TransferTime    *time.Time `json:"transfer_time,omitempty"`
	TamperDetected  bool       `json:"tamper_detected"`
	ConfidenceScore float64    `json:"confidence_score"`
	Notes           string     `json:"notes,omitempty"`
}

// Natural Language Text-to-SQL Reporting DTOs
type AskAIReportRequestDTO struct {
	Question   string    `json:"question"`
	MerchantID uuid.UUID `json:"merchant_id,omitempty"`
}

type AskAIReportResponseDTO struct {
	Question      string                   `json:"question"`
	GeneratedSQL  string                   `json:"generated_sql"`
	Explanation   string                   `json:"explanation"`
	ChartType     string                   `json:"chart_type"` // e.g. "bar", "line", "pie", "stat", "table"
	Columns       []string                 `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	TotalRows     int                      `json:"total_rows"`
	ExecutionMs   int64                    `json:"execution_ms"`
	SanitizedSafe bool                     `json:"sanitized_safe"`
	TokensUsed    int                      `json:"tokens_used,omitempty"`
}

