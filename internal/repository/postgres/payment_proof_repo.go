package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentProofRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentProofRepository(pool *pgxpool.Pool) *PaymentProofRepository {
	return &PaymentProofRepository{pool: pool}
}

func (r *PaymentProofRepository) CreatePaymentProof(ctx context.Context, proof *domain.PaymentProof) error {
	if proof.ID == uuid.Nil {
		proof.ID = uuid.New()
	}
	if proof.CreatedAt.IsZero() {
		proof.CreatedAt = time.Now().UTC()
	}
	if proof.Status == "" {
		proof.Status = domain.ProofStatusPending
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO payment_proofs (
			id, order_id, merchant_id, file_name, image_data,
			bank_name, sender_name, amount, reference_number,
			tamper_detected, confidence_score, status, image_phash, is_duplicate_image, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		proof.ID, proof.OrderID, proof.MerchantID, proof.FileName, "",
		proof.BankName, proof.SenderName, proof.Amount, proof.ReferenceNumber,
		proof.TamperDetected, proof.ConfidenceScore, proof.Status,
		proof.ImagePHash, proof.IsDuplicateImage, proof.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment proof: %w", err)
	}
	return nil
}

// CreatePaymentProofWithOutbox inserts a payment proof AND its outbox event inside a single ACID transaction.
// This completely resolves the Dual-Write Problem between PostgreSQL and the RabbitMQ broker.
func (r *PaymentProofRepository) CreatePaymentProofWithOutbox(
	ctx context.Context,
	proof *domain.PaymentProof,
	outboxRepo *OutboxRepository,
	payload interface{},
) (*domain.OutboxEvent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if proof.ID == uuid.Nil {
		proof.ID = uuid.New()
	}
	if proof.CreatedAt.IsZero() {
		proof.CreatedAt = time.Now().UTC()
	}
	if proof.Status == "" {
		proof.Status = domain.ProofStatusPending
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO payment_proofs (
			id, order_id, merchant_id, file_name, image_data,
			bank_name, sender_name, amount, reference_number,
			tamper_detected, confidence_score, status, image_phash, is_duplicate_image, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		proof.ID, proof.OrderID, proof.MerchantID, proof.FileName, "",
		proof.BankName, proof.SenderName, proof.Amount, proof.ReferenceNumber,
		proof.TamperDetected, proof.ConfidenceScore, proof.Status,
		proof.ImagePHash, proof.IsDuplicateImage, proof.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert payment proof tx: %w", err)
	}

	var outboxEvent *domain.OutboxEvent
	if outboxRepo != nil {
		outboxEvent, err = outboxRepo.InsertOutboxEventTx(
			ctx,
			tx,
			"PAYMENT_PROOF",
			proof.ID.String(),
			domain.RabbitMQRoutingPaymentVerify,
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("insert outbox event tx: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit outbox tx: %w", err)
	}

	return outboxEvent, nil
}

// FindNearDuplicatePHash checks recent payment proofs to see if another slip has Hamming distance <= maxDistance.
func (r *PaymentProofRepository) FindNearDuplicatePHash(
	ctx context.Context,
	merchantID uuid.UUID,
	targetPHash string,
	maxDistance int,
) (*domain.PaymentProof, int, error) {
	if targetPHash == "" {
		return nil, -1, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, merchant_id, file_name, bank_name, sender_name,
		       amount::FLOAT8, reference_number, status, image_phash, created_at
		FROM payment_proofs
		WHERE merchant_id = $1 AND image_phash IS NOT NULL AND status IN ('VERIFIED', 'SUSPICIOUS_HOLD', 'PENDING')
		ORDER BY created_at DESC
		LIMIT 100
	`, merchantID)
	if err != nil {
		return nil, -1, fmt.Errorf("query phashes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p domain.PaymentProof
		if err := rows.Scan(
			&p.ID, &p.OrderID, &p.MerchantID, &p.FileName, &p.BankName, &p.SenderName,
			&p.Amount, &p.ReferenceNumber, &p.Status, &p.ImagePHash, &p.CreatedAt,
		); err != nil {
			return nil, -1, err
		}

		dist := crypto.HammingDistanceHex(targetPHash, p.ImagePHash)
		if dist >= 0 && dist <= maxDistance {
			return &p, dist, nil
		}
	}

	return nil, -1, nil
}

func (r *PaymentProofRepository) GetPaymentProof(ctx context.Context, id uuid.UUID) (*domain.PaymentProof, error) {
	var p domain.PaymentProof
	var transferTime *time.Time
	var processedAt *time.Time
	var rawVision []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, merchant_id, file_name, bank_name, sender_name,
		       amount::FLOAT8, reference_number, transfer_time, tamper_detected,
		       confidence_score::FLOAT8, status, rejection_reason, raw_vision_response,
		       COALESCE(image_phash, ''), COALESCE(is_duplicate_image, FALSE),
		       created_at, processed_at
		FROM payment_proofs
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.OrderID, &p.MerchantID, &p.FileName, &p.BankName, &p.SenderName,
		&p.Amount, &p.ReferenceNumber, &transferTime, &p.TamperDetected,
		&p.ConfidenceScore, &p.Status, &p.RejectionReason, &rawVision,
		&p.ImagePHash, &p.IsDuplicateImage,
		&p.CreatedAt, &processedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("payment proof not found")
		}
		return nil, fmt.Errorf("query payment proof: %w", err)
	}

	p.TransferTime = transferTime
	p.ProcessedAt = processedAt
	return &p, nil
}

func (r *PaymentProofRepository) ListPaymentProofs(ctx context.Context, merchantID uuid.UUID, limit int) ([]domain.PaymentProof, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, merchant_id, file_name, bank_name, sender_name,
		       amount::FLOAT8, reference_number, transfer_time, tamper_detected,
		       confidence_score::FLOAT8, status, rejection_reason,
		       COALESCE(image_phash, ''), COALESCE(is_duplicate_image, FALSE),
		       created_at, processed_at
		FROM payment_proofs
		WHERE merchant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, merchantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list payment proofs: %w", err)
	}
	defer rows.Close()

	var proofs []domain.PaymentProof
	for rows.Next() {
		var p domain.PaymentProof
		var transferTime *time.Time
		var processedAt *time.Time
		if err := rows.Scan(
			&p.ID, &p.OrderID, &p.MerchantID, &p.FileName, &p.BankName, &p.SenderName,
			&p.Amount, &p.ReferenceNumber, &transferTime, &p.TamperDetected,
			&p.ConfidenceScore, &p.Status, &p.RejectionReason,
			&p.ImagePHash, &p.IsDuplicateImage,
			&p.CreatedAt, &processedAt,
		); err != nil {
			return nil, err
		}
		p.TransferTime = transferTime
		p.ProcessedAt = processedAt
		proofs = append(proofs, p)
	}
	return proofs, nil
}

// VerifyPaymentSlipAndTransitionOrderAtomic applies ACID verification and stock mutation:
// If tamper detected or amount mismatch -> transitions to SUSPICIOUS_HOLD
// If verified -> transitions order to PAID and reserved_stock -> sold_stock
func (r *PaymentProofRepository) VerifyPaymentSlipAndTransitionOrderAtomic(
	ctx context.Context,
	proofID uuid.UUID,
	orderID uuid.UUID,
	res *domain.VisionSlipExtractionResult,
) (domain.OrderStatus, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()

	// 1. Pessimistic row-level lock on order
	var currentStatus domain.OrderStatus
	var totalAmount float64
	err = tx.QueryRow(ctx, `
		SELECT status, total_amount::FLOAT8
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`, orderID).Scan(&currentStatus, &totalAmount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrOrderNotFound
		}
		return "", fmt.Errorf("lock order: %w", err)
	}

	rawJSON, _ := json.Marshal(res)

	// Check validity conditions
	amountMismatch := math.Abs(res.Amount-totalAmount) > 0.01
	isSuspicious := res.TamperDetected || res.ConfidenceScore < 0.70 || amountMismatch

	var rejectionReason string
	var newOrderStatus domain.OrderStatus
	var proofStatus domain.PaymentProofStatus

	if isSuspicious {
		newOrderStatus = domain.OrderStatusSuspiciousHold
		proofStatus = domain.ProofStatusSuspiciousHold
		if res.TamperDetected {
			rejectionReason = "Tampering/photoshop manipulation detected in slip image"
		} else if amountMismatch {
			rejectionReason = fmt.Sprintf("Extracted amount (%.2f) does not match order total (%.2f)", res.Amount, totalAmount)
		} else {
			rejectionReason = fmt.Sprintf("Low extraction confidence (%.2f < 0.70)", res.ConfidenceScore)
		}

		// Update order to SUSPICIOUS_HOLD
		_, err = tx.Exec(ctx, `
			UPDATE orders
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, newOrderStatus, now, orderID)
		if err != nil {
			return "", fmt.Errorf("update order suspicious hold: %w", err)
		}
	} else {
		newOrderStatus = domain.OrderStatusPaid
		proofStatus = domain.ProofStatusVerified

		// Transition order to PAID
		_, err = tx.Exec(ctx, `
			UPDATE orders
			SET status = $1, paid_at = $2, updated_at = $2
			WHERE id = $3
		`, newOrderStatus, now, orderID)
		if err != nil {
			return "", fmt.Errorf("update order paid: %w", err)
		}

		// Atomically move stock: reserved_stock -> sold_stock for all order items
		rows, err := tx.Query(ctx, `
			SELECT product_id, qty
			FROM order_items
			WHERE order_id = $1
		`, orderID)
		if err != nil {
			return "", fmt.Errorf("query order items: %w", err)
		}
		defer rows.Close()

		type itemStock struct {
			productID uuid.UUID
			qty       int
		}
		var items []itemStock
		for rows.Next() {
			var it itemStock
			if err := rows.Scan(&it.productID, &it.qty); err != nil {
				return "", err
			}
			items = append(items, it)
		}
		rows.Close()

		for _, it := range items {
			_, err = tx.Exec(ctx, `
				UPDATE products
				SET reserved_stock = reserved_stock - $1,
				    sold_stock = sold_stock + $1,
				    version = version + 1,
				    updated_at = $2
				WHERE id = $3 AND reserved_stock >= $1
			`, it.qty, now, it.productID)
			if err != nil {
				return "", fmt.Errorf("finalize product stock: %w", err)
			}
		}
	}

	// 2. Update payment_proofs record
	_, err = tx.Exec(ctx, `
		UPDATE payment_proofs
		SET bank_name = $1, sender_name = $2, amount = $3,
		    reference_number = $4, transfer_time = $5,
		    tamper_detected = $6, confidence_score = $7,
		    status = $8, rejection_reason = $9,
		    raw_vision_response = $10, processed_at = $11
		WHERE id = $12
	`,
		res.BankName, res.SenderName, res.Amount,
		res.ReferenceNumber, res.TransferTime,
		res.TamperDetected, res.ConfidenceScore,
		proofStatus, rejectionReason,
		rawJSON, now, proofID,
	)
	if err != nil {
		return "", fmt.Errorf("update payment proof record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit verification transaction: %w", err)
	}

	return newOrderStatus, nil
}

