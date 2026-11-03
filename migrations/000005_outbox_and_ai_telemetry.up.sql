-- ============================================================================
-- Migration 000005: Transactional Outbox Pattern & AI Telemetry Metrics
-- ============================================================================

-- 1. Extend outbox_events (Created in 000001) with aggregate metadata if not exists
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED')),
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS aggregate_type VARCHAR(50) NOT NULL DEFAULT 'PAYMENT_PROOF';
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS aggregate_id VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_outbox_status_created ON outbox_events (status, created_at ASC);

-- 2. Add Perceptual Hash columns to payment_proofs for duplicate image fraud detection
ALTER TABLE payment_proofs ADD COLUMN IF NOT EXISTS image_phash VARCHAR(64);
ALTER TABLE payment_proofs ADD COLUMN IF NOT EXISTS is_duplicate_image BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_payment_proofs_phash ON payment_proofs (image_phash) WHERE image_phash IS NOT NULL;

-- 3. Table ai_telemetry_events for FinOps, Token Cost Tracking & AI Observability
CREATE TABLE IF NOT EXISTS ai_telemetry_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_type VARCHAR(50) NOT NULL,
    model_name VARCHAR(50) NOT NULL,
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    latency_ms INT NOT NULL DEFAULT 0,
    cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    is_success BOOLEAN NOT NULL DEFAULT TRUE,
    fraud_flag BOOLEAN NOT NULL DEFAULT FALSE,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_telemetry_created ON ai_telemetry_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_telemetry_service ON ai_telemetry_events (service_type, created_at DESC);
