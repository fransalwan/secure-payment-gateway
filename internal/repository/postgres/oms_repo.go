package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProductNotFound        = errors.New("product SKU not found")
	ErrInsufficientStock      = errors.New("insufficient available stock for requested SKU")
	ErrOrderNotFound          = errors.New("order not found")
	ErrInvalidOrderTransition = errors.New("invalid one-way order state transition")
)

type OMSRepository struct {
	pool *pgxpool.Pool
}

func NewOMSRepository(pool *pgxpool.Pool) *OMSRepository {
	return &OMSRepository{pool: pool}
}

// EnsureSchema idempotently applies the OMS SQL migration file if tables do not exist yet.
func (r *OMSRepository) EnsureSchema(ctx context.Context, sqlFilePath string) error {
	sqlBytes, err := os.ReadFile(sqlFilePath)
	if err != nil {
		return fmt.Errorf("failed to read OMS migration file %s: %w", sqlFilePath, err)
	}
	if _, err := r.pool.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute OMS schema migration: %w", err)
	}
	return nil
}

func (r *OMSRepository) ListMerchants(ctx context.Context) ([]domain.Merchant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, city, phone, created_at
		FROM merchants
		ORDER BY code ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants: %w", err)
	}
	defer rows.Close()

	var merchants []domain.Merchant
	for rows.Next() {
		var m domain.Merchant
		if err := rows.Scan(&m.ID, &m.Code, &m.Name, &m.City, &m.Phone, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		merchants = append(merchants, m)
	}
	return merchants, nil
}

func (r *OMSRepository) GetMerchantByID(ctx context.Context, id uuid.UUID) (*domain.Merchant, error) {
	var m domain.Merchant
	err := r.pool.QueryRow(ctx, `
		SELECT id, code, name, city, phone, created_at
		FROM merchants
		WHERE id = $1
	`, id).Scan(&m.ID, &m.Code, &m.Name, &m.City, &m.Phone, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}
	return &m, nil
}

func (r *OMSRepository) ListProductsByMerchant(ctx context.Context, merchantID uuid.UUID) ([]domain.Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, merchant_id, sku, name, price::FLOAT8, available_stock, reserved_stock, sold_stock, version, created_at, updated_at
		FROM products
		WHERE merchant_id = $1
		ORDER BY sku ASC
	`, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ID, &p.MerchantID, &p.SKU, &p.Name, &p.Price,
			&p.AvailableStock, &p.ReservedStock, &p.SoldStock,
			&p.Version, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *OMSRepository) GetProductByID(ctx context.Context, productID uuid.UUID) (*domain.Product, error) {
	var p domain.Product
	err := r.pool.QueryRow(ctx, `
		SELECT id, merchant_id, sku, name, price::FLOAT8, available_stock, reserved_stock, sold_stock, version, created_at, updated_at
		FROM products
		WHERE id = $1
	`, productID).Scan(
		&p.ID, &p.MerchantID, &p.SKU, &p.Name, &p.Price,
		&p.AvailableStock, &p.ReservedStock, &p.SoldStock,
		&p.Version, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	return &p, nil
}

func (r *OMSRepository) SetAvailableStock(ctx context.Context, productID uuid.UUID, availableStock int) (*domain.Product, error) {
	if availableStock < 0 {
		return nil, fmt.Errorf("available stock cannot be negative")
	}
	var p domain.Product
	err := r.pool.QueryRow(ctx, `
		UPDATE products
		SET available_stock = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, merchant_id, sku, name, price::FLOAT8, available_stock, reserved_stock, sold_stock, version, created_at, updated_at
	`, availableStock, productID).Scan(
		&p.ID, &p.MerchantID, &p.SKU, &p.Name, &p.Price,
		&p.AvailableStock, &p.ReservedStock, &p.SoldStock,
		&p.Version, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to set available stock: %w", err)
	}
	return &p, nil
}

// CreateOrderAtomic executes Anti-Overselling Order Creation inside an ACID transaction:
// 1. Deterministically sorts ProductIDs to prevent multi-SKU deadlocks.
// 2. Acquires Pessimistic Row-Level Lock (SELECT ... FOR UPDATE) on each SKU.
// 3. Atomically moves qty from available_stock -> reserved_stock with WHERE available_stock >= qty guard.
// 4. Inserts Order (PENDING_PAYMENT) and OrderItems.
func (r *OMSRepository) CreateOrderAtomic(
	ctx context.Context,
	req *domain.CreateOrderRequestDTO,
	orderNumber string,
	paymentURL string,
	expiresAt time.Time,
) (*domain.Order, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Consolidate duplicate SKUs in request & sort by ProductID to eliminate lock-order deadlocks
	qtyByProduct := make(map[uuid.UUID]int, len(req.Items))
	for _, item := range req.Items {
		qtyByProduct[item.ProductID] += item.Qty
	}

	sortedIDs := make([]uuid.UUID, 0, len(qtyByProduct))
	for pid := range qtyByProduct {
		sortedIDs = append(sortedIDs, pid)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return sortedIDs[i].String() < sortedIDs[j].String()
	})

	now := time.Now().UTC()
	orderID := uuid.New()
	var totalAmount float64
	orderItems := make([]domain.OrderItem, 0, len(sortedIDs))

	for _, pid := range sortedIDs {
		reqQty := qtyByProduct[pid]
		var sku, prodName string
		var price float64
		var availStock int

		// Pessimistic Row-Level Lock on products row
		err := tx.QueryRow(ctx, `
			SELECT sku, name, price::FLOAT8, available_stock
			FROM products
			WHERE id = $1 AND merchant_id = $2
			FOR UPDATE
		`, pid, req.MerchantID).Scan(&sku, &prodName, &price, &availStock)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: %s", ErrProductNotFound, pid.String())
			}
			return nil, fmt.Errorf("failed to lock product row: %w", err)
		}

		if availStock < reqQty {
			return nil, fmt.Errorf("%w: SKU %s has %d available, requested %d", ErrInsufficientStock, sku, availStock, reqQty)
		}

		// Atomic conditional stock reservation
		cmdTag, err := tx.Exec(ctx, `
			UPDATE products
			SET available_stock = available_stock - $1,
			    reserved_stock = reserved_stock + $1,
			    version = version + 1,
			    updated_at = $2
			WHERE id = $3 AND available_stock >= $1
		`, reqQty, now, pid)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve stock: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return nil, fmt.Errorf("%w: SKU %s concurrent reservation conflict", ErrInsufficientStock, sku)
		}

		subtotal := price * float64(reqQty)
		totalAmount += subtotal
		orderItems = append(orderItems, domain.OrderItem{
			ID:          uuid.New(),
			OrderID:     orderID,
			ProductID:   pid,
			SKU:         sku,
			ProductName: prodName,
			Qty:         reqQty,
			UnitPrice:   price,
			Subtotal:    subtotal,
		})
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (
			id, merchant_id, order_number, channel, external_order_id,
			customer_name, customer_phone, shipping_address, courier_code,
			status, total_amount, payment_url, payment_expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		orderID, req.MerchantID, orderNumber, req.Channel, req.ExternalOrderID,
		req.CustomerName, req.CustomerPhone, req.ShippingAddress, req.CourierCode,
		domain.OrderStatusPendingPayment, totalAmount, paymentURL, expiresAt, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	for _, item := range orderItems {
		_, err := tx.Exec(ctx, `
			INSERT INTO order_items (
				id, order_id, product_id, sku, product_name, qty, unit_price, subtotal
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, item.ID, item.OrderID, item.ProductID, item.SKU, item.ProductName, item.Qty, item.UnitPrice, item.Subtotal)
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit order creation transaction: %w", err)
	}

	return &domain.Order{
		ID:               orderID,
		MerchantID:       req.MerchantID,
		OrderNumber:      orderNumber,
		Channel:          req.Channel,
		ExternalOrderID:  req.ExternalOrderID,
		CustomerName:     req.CustomerName,
		CustomerPhone:    req.CustomerPhone,
		ShippingAddress:  req.ShippingAddress,
		CourierCode:      req.CourierCode,
		Status:           domain.OrderStatusPendingPayment,
		TotalAmount:      totalAmount,
		PaymentURL:       paymentURL,
		PaymentExpiresAt: expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
		Items:            orderItems,
	}, nil
}

// MarkOrderPaidAtomic transitions an order strictly one-way from PENDING_PAYMENT -> PAID
// and moves reserved_stock -> sold_stock atomically. If already PAID or further along,
// returns (order, true, nil) safely without double-deducting stock.
func (r *OMSRepository) MarkOrderPaidAtomic(ctx context.Context, orderIDStr string, orderNumber string) (*domain.Order, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin payment settlement tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ord domain.Order
	var extID, waybill, payURL *string
	var query string
	var arg interface{}

	if parsedUUID, err := uuid.Parse(orderIDStr); err == nil {
		query = `
			SELECT id, merchant_id, order_number, channel, external_order_id,
			       customer_name, customer_phone, shipping_address, courier_code,
			       status, total_amount::FLOAT8, waybill_number, payment_url,
			       payment_expires_at, paid_at, created_at, updated_at
			FROM orders WHERE id = $1 FOR UPDATE
		`
		arg = parsedUUID
	} else {
		query = `
			SELECT id, merchant_id, order_number, channel, external_order_id,
			       customer_name, customer_phone, shipping_address, courier_code,
			       status, total_amount::FLOAT8, waybill_number, payment_url,
			       payment_expires_at, paid_at, created_at, updated_at
			FROM orders WHERE order_number = $1 FOR UPDATE
		`
		arg = orderNumber
	}

	err = tx.QueryRow(ctx, query, arg).Scan(
		&ord.ID, &ord.MerchantID, &ord.OrderNumber, &ord.Channel, &extID,
		&ord.CustomerName, &ord.CustomerPhone, &ord.ShippingAddress, &ord.CourierCode,
		&ord.Status, &ord.TotalAmount, &waybill, &payURL,
		&ord.PaymentExpiresAt, &ord.PaidAt, &ord.CreatedAt, &ord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrOrderNotFound
		}
		return nil, false, fmt.Errorf("failed to lock order for payment: %w", err)
	}
	if extID != nil {
		ord.ExternalOrderID = *extID
	}
	if waybill != nil {
		ord.WaybillNumber = *waybill
	}
	if payURL != nil {
		ord.PaymentURL = *payURL
	}

	// Enforce one-way state machine: never revert or double-deduct stock if already settled
	if ord.Status == domain.OrderStatusPaid ||
		ord.Status == domain.OrderStatusReadyToShip ||
		ord.Status == domain.OrderStatusShipped ||
		ord.Status == domain.OrderStatusCompleted {
		return &ord, true, nil
	}
	if ord.Status == domain.OrderStatusCancelled {
		return nil, false, fmt.Errorf("%w: cannot pay a CANCELLED order", ErrInvalidOrderTransition)
	}

	rows, err := tx.Query(ctx, `
		SELECT product_id, qty
		FROM order_items
		WHERE order_id = $1
		ORDER BY product_id ASC
	`, ord.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load order items: %w", err)
	}

	type itemQty struct {
		productID uuid.UUID
		qty       int
	}
	var items []itemQty
	for rows.Next() {
		var iq itemQty
		if err := rows.Scan(&iq.productID, &iq.qty); err != nil {
			rows.Close()
			return nil, false, err
		}
		items = append(items, iq)
	}
	rows.Close()

	now := time.Now().UTC()
	for _, iq := range items {
		_, err := tx.Exec(ctx, `
			UPDATE products
			SET reserved_stock = GREATEST(reserved_stock - $1, 0),
			    sold_stock = sold_stock + $1,
			    version = version + 1,
			    updated_at = $2
			WHERE id = $3
		`, iq.qty, now, iq.productID)
		if err != nil {
			return nil, false, fmt.Errorf("failed to finalize sold_stock: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status = $1, paid_at = $2, updated_at = $2
		WHERE id = $3
	`, domain.OrderStatusPaid, now, ord.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to mark order PAID: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("failed to commit payment settlement: %w", err)
	}

	ord.Status = domain.OrderStatusPaid
	ord.PaidAt = &now
	ord.UpdatedAt = now
	return &ord, false, nil
}

// CancelOrderAtomic cancels a PENDING_PAYMENT order and releases reserved_stock back to available_stock.
func (r *OMSRepository) CancelOrderAtomic(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin cancel transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ord domain.Order
	var extID, waybill, payURL *string
	err = tx.QueryRow(ctx, `
		SELECT id, merchant_id, order_number, channel, external_order_id,
		       customer_name, customer_phone, shipping_address, courier_code,
		       status, total_amount::FLOAT8, waybill_number, payment_url,
		       payment_expires_at, paid_at, created_at, updated_at
		FROM orders WHERE id = $1 FOR UPDATE
	`, orderID).Scan(
		&ord.ID, &ord.MerchantID, &ord.OrderNumber, &ord.Channel, &extID,
		&ord.CustomerName, &ord.CustomerPhone, &ord.ShippingAddress, &ord.CourierCode,
		&ord.Status, &ord.TotalAmount, &waybill, &payURL,
		&ord.PaymentExpiresAt, &ord.PaidAt, &ord.CreatedAt, &ord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if ord.Status != domain.OrderStatusPendingPayment {
		return nil, fmt.Errorf("%w: only PENDING_PAYMENT orders can be cancelled (current: %s)", ErrInvalidOrderTransition, ord.Status)
	}

	rows, err := tx.Query(ctx, `SELECT product_id, qty FROM order_items WHERE order_id = $1 ORDER BY product_id ASC`, ord.ID)
	if err != nil {
		return nil, err
	}
	type itemQty struct {
		productID uuid.UUID
		qty       int
	}
	var items []itemQty
	for rows.Next() {
		var iq itemQty
		if err := rows.Scan(&iq.productID, &iq.qty); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, iq)
	}
	rows.Close()

	now := time.Now().UTC()
	for _, iq := range items {
		_, err := tx.Exec(ctx, `
			UPDATE products
			SET reserved_stock = GREATEST(reserved_stock - $1, 0),
			    available_stock = available_stock + $1,
			    version = version + 1,
			    updated_at = $2
			WHERE id = $3
		`, iq.qty, now, iq.productID)
		if err != nil {
			return nil, fmt.Errorf("failed to release reserved stock: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3
	`, domain.OrderStatusCancelled, now, ord.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	ord.Status = domain.OrderStatusCancelled
	ord.UpdatedAt = now
	return &ord, nil
}

// AttachWaybillAndReadyToShip stores generated logistics AWB and transitions PAID -> READY_TO_SHIP.
func (r *OMSRepository) AttachWaybillAndReadyToShip(
	ctx context.Context,
	orderID uuid.UUID,
	courierCode string,
	waybillNumber string,
) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	labelMeta, _ := json.Marshal(map[string]interface{}{
		"courier":      courierCode,
		"service":      "REG",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})

	_, err = tx.Exec(ctx, `
		INSERT INTO shipments (order_id, courier_code, waybill_number, status, shipping_label_payload)
		VALUES ($1, $2, $3, 'READY_TO_SHIP', $4)
		ON CONFLICT (order_id) DO UPDATE
		SET waybill_number = EXCLUDED.waybill_number,
		    status = 'READY_TO_SHIP'
	`, orderID, courierCode, waybillNumber, labelMeta)
	if err != nil {
		return fmt.Errorf("failed to upsert shipment waybill: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET waybill_number = $1,
		    status = CASE WHEN status = 'PAID' THEN 'READY_TO_SHIP' ELSE status END,
		    updated_at = NOW()
		WHERE id = $2
	`, waybillNumber, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order waybill: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *OMSRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	var ord domain.Order
	var extID, waybill, payURL *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, merchant_id, order_number, channel, external_order_id,
		       customer_name, customer_phone, shipping_address, courier_code,
		       status, total_amount::FLOAT8, waybill_number, payment_url,
		       payment_expires_at, paid_at, created_at, updated_at
		FROM orders WHERE id = $1
	`, orderID).Scan(
		&ord.ID, &ord.MerchantID, &ord.OrderNumber, &ord.Channel, &extID,
		&ord.CustomerName, &ord.CustomerPhone, &ord.ShippingAddress, &ord.CourierCode,
		&ord.Status, &ord.TotalAmount, &waybill, &payURL,
		&ord.PaymentExpiresAt, &ord.PaidAt, &ord.CreatedAt, &ord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if extID != nil {
		ord.ExternalOrderID = *extID
	}
	if waybill != nil {
		ord.WaybillNumber = *waybill
	}
	if payURL != nil {
		ord.PaymentURL = *payURL
	}

	itemsMap, err := r.getItemsForOrders(ctx, []uuid.UUID{ord.ID})
	if err == nil {
		ord.Items = itemsMap[ord.ID]
	}
	return &ord, nil
}

func (r *OMSRepository) GetOrderByNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	var ord domain.Order
	var extID, waybill, payURL *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, merchant_id, order_number, channel, external_order_id,
		       customer_name, customer_phone, shipping_address, courier_code,
		       status, total_amount::FLOAT8, waybill_number, payment_url,
		       payment_expires_at, paid_at, created_at, updated_at
		FROM orders WHERE order_number = $1
	`, orderNumber).Scan(
		&ord.ID, &ord.MerchantID, &ord.OrderNumber, &ord.Channel, &extID,
		&ord.CustomerName, &ord.CustomerPhone, &ord.ShippingAddress, &ord.CourierCode,
		&ord.Status, &ord.TotalAmount, &waybill, &payURL,
		&ord.PaymentExpiresAt, &ord.PaidAt, &ord.CreatedAt, &ord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if extID != nil {
		ord.ExternalOrderID = *extID
	}
	if waybill != nil {
		ord.WaybillNumber = *waybill
	}
	if payURL != nil {
		ord.PaymentURL = *payURL
	}

	itemsMap, err := r.getItemsForOrders(ctx, []uuid.UUID{ord.ID})
	if err == nil {
		ord.Items = itemsMap[ord.ID]
	}
	return &ord, nil
}

func (r *OMSRepository) ListOrders(
	ctx context.Context,
	merchantID uuid.UUID,
	channelFilter string,
	statusFilter string,
	limit int,
) ([]domain.Order, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, merchant_id, order_number, channel, external_order_id,
		       customer_name, customer_phone, shipping_address, courier_code,
		       status, total_amount::FLOAT8, waybill_number, payment_url,
		       payment_expires_at, paid_at, created_at, updated_at
		FROM orders
		WHERE merchant_id = $1
		  AND ($2 = '' OR channel = $2)
		  AND ($3 = '' OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, merchantID, channelFilter, statusFilter, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	orderIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var ord domain.Order
		var extID, waybill, payURL *string
		if err := rows.Scan(
			&ord.ID, &ord.MerchantID, &ord.OrderNumber, &ord.Channel, &extID,
			&ord.CustomerName, &ord.CustomerPhone, &ord.ShippingAddress, &ord.CourierCode,
			&ord.Status, &ord.TotalAmount, &waybill, &payURL,
			&ord.PaymentExpiresAt, &ord.PaidAt, &ord.CreatedAt, &ord.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		if extID != nil {
			ord.ExternalOrderID = *extID
		}
		if waybill != nil {
			ord.WaybillNumber = *waybill
		}
		if payURL != nil {
			ord.PaymentURL = *payURL
		}
		orders = append(orders, ord)
		orderIDs = append(orderIDs, ord.ID)
	}

	if len(orderIDs) > 0 {
		itemsMap, err := r.getItemsForOrders(ctx, orderIDs)
		if err == nil {
			for i := range orders {
				orders[i].Items = itemsMap[orders[i].ID]
			}
		}
	}

	return orders, nil
}

func (r *OMSRepository) getItemsForOrders(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]domain.OrderItem, error) {
	result := make(map[uuid.UUID][]domain.OrderItem, len(orderIDs))
	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, product_id, sku, product_name, qty, unit_price::FLOAT8, subtotal::FLOAT8
		FROM order_items
		WHERE order_id = ANY($1)
	`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.SKU,
			&item.ProductName, &item.Qty, &item.UnitPrice, &item.Subtotal,
		); err != nil {
			return nil, err
		}
		result[item.OrderID] = append(result[item.OrderID], item)
	}
	return result, nil
}

// TryInsertWebhookEvent enforces database-level Idempotency via UNIQUE(provider, external_event_id).
// Returns (eventID, isDuplicate=true, nil) if already seen!
func (r *OMSRepository) TryInsertWebhookEvent(
	ctx context.Context,
	provider string,
	externalEventID string,
	eventType string,
	payload []byte,
) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO webhook_events (provider, external_event_id, event_type, payload, status)
		VALUES ($1, $2, $3, $4, 'RECEIVED')
		ON CONFLICT (provider, external_event_id) DO NOTHING
		RETURNING id
	`, provider, externalEventID, eventType, payload).Scan(&id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, true, nil
		}
		return uuid.Nil, false, fmt.Errorf("failed to insert webhook event: %w", err)
	}
	return id, false, nil
}

func (r *OMSRepository) UpdateWebhookEventStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.WebhookEventStatus,
	processingMs int,
	errMsg string,
) error {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE webhook_events
		SET status = $1,
		    processing_ms = $2,
		    error_message = $3,
		    processed_at = NOW()
		WHERE id = $4
	`, status, processingMs, errPtr, id)
	return err
}

func (r *OMSRepository) ListWebhookEvents(ctx context.Context, limit int) ([]domain.WebhookEvent, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider, external_event_id, event_type, payload, status,
		       retry_count, processing_ms, error_message, processed_at, created_at
		FROM webhook_events
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhook events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.WebhookEvent, 0)
	for rows.Next() {
		var ev domain.WebhookEvent
		var errMsg *string
		if err := rows.Scan(
			&ev.ID, &ev.Provider, &ev.ExternalEventID, &ev.EventType,
			&ev.Payload, &ev.Status, &ev.RetryCount, &ev.ProcessingMs,
			&errMsg, &ev.ProcessedAt, &ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		if errMsg != nil {
			ev.ErrorMessage = *errMsg
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *OMSRepository) LogSystemError(ctx context.Context, entry *domain.SystemErrorLog) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO system_error_logs (
			service_name, queue_name, job_id, reference_id, error_code, error_message, attempts, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		entry.ServiceName, entry.QueueName, entry.JobID, entry.ReferenceID,
		entry.ErrorCode, entry.ErrorMessage, entry.Attempts, entry.Payload,
	)
	return err
}

func (r *OMSRepository) ResolveSystemErrorByJobID(ctx context.Context, jobID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE system_error_logs
		SET resolved = TRUE, resolved_at = NOW()
		WHERE job_id = $1 AND resolved = FALSE
	`, jobID)
	return err
}

func (r *OMSRepository) ListSystemErrors(ctx context.Context, limit int) ([]domain.SystemErrorLog, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, service_name, queue_name, job_id, reference_id, error_code,
		       error_message, attempts, payload, resolved, created_at, resolved_at
		FROM system_error_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]domain.SystemErrorLog, 0)
	for rows.Next() {
		var l domain.SystemErrorLog
		var refID *string
		if err := rows.Scan(
			&l.ID, &l.ServiceName, &l.QueueName, &l.JobID, &refID,
			&l.ErrorCode, &l.ErrorMessage, &l.Attempts, &l.Payload,
			&l.Resolved, &l.CreatedAt, &l.ResolvedAt,
		); err != nil {
			return nil, err
		}
		if refID != nil {
			l.ReferenceID = *refID
		}
		logs = append(logs, l)
	}
	return logs, nil
}

