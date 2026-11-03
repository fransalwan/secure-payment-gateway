DROP TABLE IF EXISTS ai_reporting_audit_logs;
DROP TABLE IF EXISTS payment_proofs;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING_PAYMENT', 'PAID', 'READY_TO_SHIP', 'SHIPPED', 'COMPLETED', 'CANCELLED'));
