DROP TABLE IF EXISTS ai_telemetry_events;
ALTER TABLE payment_proofs DROP COLUMN IF EXISTS is_duplicate_image;
ALTER TABLE payment_proofs DROP COLUMN IF EXISTS image_phash;
DROP TABLE IF EXISTS outbox_events;
