-- Rollback migration for ingestion_idempotency table

DROP INDEX IF EXISTS idx_ingestion_idempotency_did_collection;
DROP INDEX IF EXISTS idx_ingestion_idempotency_created_at;
DROP TABLE IF EXISTS ingestion_idempotency;
