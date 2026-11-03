-- ============================================================================
-- Migration 000004: Multimodal Bank Slip Validator & AI Reporting Audit
-- ============================================================================

-- 1. Extend orders status check constraint to include 'SUSPICIOUS_HOLD'
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING_PAYMENT', 'PAID', 'READY_TO_SHIP', 'SHIPPED', 'COMPLETED', 'CANCELLED', 'SUSPICIOUS_HOLD'));

-- 2. Table payment_proofs for Automated Bank Slip Verification
CREATE TABLE IF NOT EXISTS payment_proofs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    image_data TEXT,
    bank_name VARCHAR(100),
    sender_name VARCHAR(150),
    amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    reference_number VARCHAR(100),
    transfer_time TIMESTAMPTZ,
    tamper_detected BOOLEAN NOT NULL DEFAULT FALSE,
    confidence_score NUMERIC(5, 4) NOT NULL DEFAULT 0.0000,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'VERIFIED', 'SUSPICIOUS_HOLD', 'REJECTED')),
    rejection_reason TEXT,
    raw_vision_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payment_proofs_order_id ON payment_proofs (order_id);
CREATE INDEX IF NOT EXISTS idx_payment_proofs_merchant_created ON payment_proofs (merchant_id, created_at DESC);

-- 3. Table ai_reporting_audit_logs for Text-to-SQL Enterprise Analytics
CREATE TABLE IF NOT EXISTS ai_reporting_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    user_query TEXT NOT NULL,
    generated_sql TEXT NOT NULL,
    is_safe BOOLEAN NOT NULL DEFAULT TRUE,
    execution_ms INT NOT NULL DEFAULT 0,
    rows_returned INT NOT NULL DEFAULT 0,
    token_usage INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_reporting_audit_merchant_created ON ai_reporting_audit_logs (merchant_id, created_at DESC);
