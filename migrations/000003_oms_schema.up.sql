-- ============================================================================
-- Migration 000003: Omni-Channel Mini E-Commerce OMS & Idempotent Webhooks
-- Multi-Tenant Isolation, 3-State Inventory, Composite Indexes, DLQ Error Logs
-- ============================================================================

-- 0. Merchants Table (Multi-Tenant Isolation)
CREATE TABLE IF NOT EXISTS merchants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(32) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    city VARCHAR(100) NOT NULL DEFAULT 'Jakarta Selatan',
    phone VARCHAR(30) NOT NULL DEFAULT '021-5550199',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 1. Tabel Products & Inventory (3-State Stock: Available, Reserved, Sold)
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    sku VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    price NUMERIC(14, 2) NOT NULL CHECK (price >= 0),
    available_stock INT NOT NULL DEFAULT 0 CHECK (available_stock >= 0),
    reserved_stock INT NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
    sold_stock INT NOT NULL DEFAULT 0 CHECK (sold_stock >= 0),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (merchant_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_products_merchant_sku ON products (merchant_id, sku);

-- 2. Tabel Orders (Multi-Channel Unification)
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    order_number VARCHAR(50) NOT NULL UNIQUE,
    channel VARCHAR(30) NOT NULL CHECK (channel IN ('WEB', 'WHATSAPP_LIVE', 'SHOPEE', 'TIKTOK_SHOP')),
    external_order_id VARCHAR(100),
    customer_name VARCHAR(150) NOT NULL,
    customer_phone VARCHAR(30) NOT NULL,
    shipping_address TEXT NOT NULL,
    courier_code VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_PAYMENT'
        CHECK (status IN ('PENDING_PAYMENT', 'PAID', 'READY_TO_SHIP', 'SHIPPED', 'COMPLETED', 'CANCELLED')),
    total_amount NUMERIC(14, 2) NOT NULL CHECK (total_amount >= 0),
    waybill_number VARCHAR(64),
    payment_url TEXT,
    payment_expires_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Composite Index untuk Query Dashboard Multi-Merchant
CREATE INDEX IF NOT EXISTS idx_orders_merchant_status_created
ON orders (merchant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_orders_merchant_channel_created
ON orders (merchant_id, channel, created_at DESC);

-- 3. Tabel Order Items
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    sku VARCHAR(64) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    qty INT NOT NULL CHECK (qty > 0),
    unit_price NUMERIC(14, 2) NOT NULL CHECK (unit_price >= 0),
    subtotal NUMERIC(14, 2) NOT NULL CHECK (subtotal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);

-- 4. Tabel Webhook Events (Jantung Idempotency & Audit Trail)
CREATE TABLE IF NOT EXISTS webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    external_event_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'RECEIVED'
        CHECK (status IN ('RECEIVED', 'PROCESSED', 'FAILED', 'IGNORED_DUPLICATE')),
    retry_count INT NOT NULL DEFAULT 0,
    processing_ms INT NOT NULL DEFAULT 0,
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, external_event_id)
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_created ON webhook_events (created_at DESC);

-- 5. Tabel Shipments (Logistics Waybill Registry)
CREATE TABLE IF NOT EXISTS shipments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    courier_code VARCHAR(30) NOT NULL,
    waybill_number VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(30) NOT NULL DEFAULT 'READY_TO_SHIP',
    shipping_label_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Tabel System Error Logs (Dead-Letter Queue & Worker Failure Audit)
CREATE TABLE IF NOT EXISTS system_error_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(64) NOT NULL,
    queue_name VARCHAR(64) NOT NULL,
    job_id VARCHAR(128) NOT NULL,
    reference_id VARCHAR(128),
    error_code VARCHAR(64) NOT NULL,
    error_message TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    payload JSONB,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_system_error_logs_created ON system_error_logs (created_at DESC);

-- ============================================================================
-- Seed Data Idempotent (Merchants, 3-State Products, Multi-Channel Sample Orders)
-- ============================================================================

INSERT INTO merchants (id, code, name, city, phone)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'MID-8801', 'PT Toko Maju Omni-Retail', 'Yogyakarta', '0274-889911'),
    ('22222222-2222-2222-2222-222222222222', 'MID-8802', 'PT Nusantara Apparel Hub', 'Bandung', '022-7712345'),
    ('33333333-3333-3333-3333-333333333333', 'MID-8803', 'PT Finansial Retail Indonesia', 'Jakarta Selatan', '021-50998877')
ON CONFLICT (code) DO NOTHING;

-- Seed 6 E-Commerce SKUs for Merchant 1 (MID-8801)
-- Note: SKU-KAOS-BLK-XL starts with available_stock = 1 for immediate Flash-Sale Race Condition Demo!
INSERT INTO products (id, merchant_id, sku, name, price, available_stock, reserved_stock, sold_stock, version)
VALUES
    ('a1111111-1111-1111-1111-111111111101', '11111111-1111-1111-1111-111111111111', 'SKU-KAOS-BLK-XL', 'Kaos Oversize Hitam XL', 129000.00, 1, 2, 48, 1),
    ('a1111111-1111-1111-1111-111111111102', '11111111-1111-1111-1111-111111111111', 'SKU-TOPI-NVY-ALL', 'Topi Baseball Navy', 75000.00, 24, 1, 35, 1),
    ('a1111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111111111', 'SKU-HOODIE-GRY-L', 'Hoodie Fleece Misty Grey L', 249000.00, 14, 1, 62, 1),
    ('a1111111-1111-1111-1111-111111111104', '11111111-1111-1111-1111-111111111111', 'SKU-CHINO-KHK-32', 'Celana Chino Slimfit Khaki 32', 189000.00, 18, 0, 29, 1),
    ('a1111111-1111-1111-1111-111111111105', '11111111-1111-1111-1111-111111111111', 'SKU-TOTEBAG-CNV', 'Totebag Canvas Waterproof Hitam', 65000.00, 40, 3, 91, 1),
    ('a1111111-1111-1111-1111-111111111106', '11111111-1111-1111-1111-111111111111', 'SKU-SNEAKER-WHT-42', 'Sneakers Vulcanized Classic White 42', 329000.00, 5, 1, 19, 1),
    -- Merchant 2 SKUs
    ('a2222222-2222-2222-2222-222222222201', '22222222-2222-2222-2222-222222222222', 'SKU-BATIK-SLM-L', 'Kemeja Batik Modern Slimfit L', 215000.00, 12, 1, 44, 1),
    ('a2222222-2222-2222-2222-222222222202', '22222222-2222-2222-2222-222222222222', 'SKU-DENIM-JKT-XL', 'Jaket Denim Raw Indigo XL', 349000.00, 3, 0, 27, 1)
ON CONFLICT (merchant_id, sku) DO NOTHING;

-- Seed Multi-Channel Orders for Merchant 1
INSERT INTO orders (
    id, merchant_id, order_number, channel, external_order_id,
    customer_name, customer_phone, shipping_address, courier_code,
    status, total_amount, waybill_number, payment_url, payment_expires_at, paid_at, created_at
)
VALUES
    (
        'b1111111-1111-1111-1111-111111111101',
        '11111111-1111-1111-1111-111111111111',
        'ORD-20261031-WA01',
        'WHATSAPP_LIVE',
        'WA-LIVE-9901',
        'Rina Kusuma',
        '081234567890',
        'Jl. Malioboro No. 45, Suryatmajan, Danurejan, Kota Yogyakarta 55213',
        'JNE',
        'READY_TO_SHIP',
        324000.00,
        'JNE-AWB-20261031-8821A',
        'https://pay.spg-gateway.local/checkout/ORD-20261031-WA01',
        NOW() + INTERVAL '24 hours',
        NOW() - INTERVAL '40 minutes',
        NOW() - INTERVAL '45 minutes'
    ),
    (
        'b1111111-1111-1111-1111-111111111102',
        '11111111-1111-1111-1111-111111111111',
        'ORD-20261031-SHP2',
        'SHOPEE',
        'SHP-2610318829QX',
        'Dimas Prakoso',
        '085711223344',
        'Jl. Dipatiukur No. 112, Coblong, Kota Bandung, Jawa Barat 40132',
        'SICEPAT',
        'PENDING_PAYMENT',
        258000.00,
        NULL,
        'https://pay.spg-gateway.local/checkout/ORD-20261031-SHP2',
        NOW() + INTERVAL '23 hours',
        NULL,
        NOW() - INTERVAL '20 minutes'
    ),
    (
        'b1111111-1111-1111-1111-111111111103',
        '11111111-1111-1111-1111-111111111111',
        'ORD-20261031-TTS3',
        'TIKTOK_SHOP',
        'TTS-57729100381',
        'Nadia Putri Maharani',
        '081399887711',
        'Jl. Kemang Raya No. 18B, Mampang Prapatan, Jakarta Selatan 12730',
        'JNT',
        'READY_TO_SHIP',
        394000.00,
        'JNT-AWB-20261031-5519C',
        'https://pay.spg-gateway.local/checkout/ORD-20261031-TTS3',
        NOW() + INTERVAL '24 hours',
        NOW() - INTERVAL '90 minutes',
        NOW() - INTERVAL '95 minutes'
    ),
    (
        'b1111111-1111-1111-1111-111111111104',
        '11111111-1111-1111-1111-111111111111',
        'ORD-20261031-WEB4',
        'WEB',
        'WEB-CHK-4410',
        'Hendra Wijaya',
        '082188776655',
        'Jl. Pemuda No. 88, Sekayu, Semarang Tengah, Kota Semarang 50132',
        'JNE',
        'PENDING_PAYMENT',
        249000.00,
        NULL,
        'https://pay.spg-gateway.local/checkout/ORD-20261031-WEB4',
        NOW() + INTERVAL '22 hours',
        NULL,
        NOW() - INTERVAL '10 minutes'
    )
ON CONFLICT (order_number) DO NOTHING;

-- Seed Order Items
INSERT INTO order_items (id, order_id, product_id, sku, product_name, qty, unit_price, subtotal)
VALUES
    ('c1111111-1111-1111-1111-111111111101', 'b1111111-1111-1111-1111-111111111101', 'a1111111-1111-1111-1111-111111111103', 'SKU-HOODIE-GRY-L', 'Hoodie Fleece Misty Grey L', 1, 249000.00, 249000.00),
    ('c1111111-1111-1111-1111-111111111102', 'b1111111-1111-1111-1111-111111111101', 'a1111111-1111-1111-1111-111111111102', 'SKU-TOPI-NVY-ALL', 'Topi Baseball Navy', 1, 75000.00, 75000.00),
    ('c1111111-1111-1111-1111-111111111103', 'b1111111-1111-1111-1111-111111111102', 'a1111111-1111-1111-1111-111111111101', 'SKU-KAOS-BLK-XL', 'Kaos Oversize Hitam XL', 2, 129000.00, 258000.00),
    ('c1111111-1111-1111-1111-111111111104', 'b1111111-1111-1111-1111-111111111103', 'a1111111-1111-1111-1111-111111111106', 'SKU-SNEAKER-WHT-42', 'Sneakers Vulcanized Classic White 42', 1, 329000.00, 329000.00),
    ('c1111111-1111-1111-1111-111111111105', 'b1111111-1111-1111-1111-111111111103', 'a1111111-1111-1111-1111-111111111105', 'SKU-TOTEBAG-CNV', 'Totebag Canvas Waterproof Hitam', 1, 65000.00, 65000.00),
    ('c1111111-1111-1111-1111-111111111106', 'b1111111-1111-1111-1111-111111111104', 'a1111111-1111-1111-1111-111111111103', 'SKU-HOODIE-GRY-L', 'Hoodie Fleece Misty Grey L', 1, 249000.00, 249000.00)
ON CONFLICT (id) DO NOTHING;

-- Seed Shipments for READY_TO_SHIP Orders
INSERT INTO shipments (id, order_id, courier_code, waybill_number, status, shipping_label_payload)
VALUES
    ('d1111111-1111-1111-1111-111111111101', 'b1111111-1111-1111-1111-111111111101', 'JNE', 'JNE-AWB-20261031-8821A', 'READY_TO_SHIP', '{"service":"REG","weight_grams":650}'),
    ('d1111111-1111-1111-1111-111111111102', 'b1111111-1111-1111-1111-111111111103', 'JNT', 'JNT-AWB-20261031-5519C', 'READY_TO_SHIP', '{"service":"EZ","weight_grams":950}')
ON CONFLICT (order_id) DO NOTHING;

