-- ADR-0012: Remove tick persistence. PG md_ticks table is no longer needed.
-- Ticks are transient via NATS, latest quote cached in Redis.
-- This is irreversible — md_ticks data is not recoverable after this.
DROP TABLE IF EXISTS md_ticks CASCADE;
