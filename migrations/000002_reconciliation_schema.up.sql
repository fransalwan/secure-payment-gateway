-- 1. Reconciliation Batches
CREATE TABLE IF NOT EXISTS reconciliation_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_number VARCHAR(64) UNIQUE NOT NULL,
    bank_name VARCHAR(64) NOT NULL,
    total_records INT NOT NULL DEFAULT 0,
    matched_count INT NOT NULL DEFAULT 0,
    discrepancy_count INT NOT NULL DEFAULT 0,
    missing_in_gateway_count INT NOT NULL DEFAULT 0,
    missing_in_bank_count INT NOT NULL DEFAULT 0,
    total_bank_amount BIGINT NOT NULL DEFAULT 0,
    total_gateway_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'PROCESSING' CHECK (status IN ('PROCESSING', 'COMPLETED', 'FAILED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_recon_batches_number ON reconciliation_batches(batch_number);

-- 2. Reconciliation Items
CREATE TABLE IF NOT EXISTS reconciliation_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES reconciliation_batches(id) ON DELETE CASCADE,
    reference_id VARCHAR(64) NOT NULL,
    bank_amount BIGINT NOT NULL DEFAULT 0,
    gateway_amount BIGINT NOT NULL DEFAULT 0,
    difference_amount BIGINT NOT NULL DEFAULT 0,
    match_status VARCHAR(30) NOT NULL CHECK (match_status IN ('MATCHED', 'DISCREPANCY_AMOUNT', 'MISSING_IN_GATEWAY', 'MISSING_IN_BANK')),
    remarks TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recon_items_batch_id ON reconciliation_items(batch_id);
CREATE INDEX IF NOT EXISTS idx_recon_items_status ON reconciliation_items(match_status);
CREATE INDEX IF NOT EXISTS idx_recon_items_ref_id ON reconciliation_items(reference_id);
