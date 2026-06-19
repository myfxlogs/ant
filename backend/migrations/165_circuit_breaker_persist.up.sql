-- 165_circuit_breaker_persist.up.sql
-- Replace in-memory sync.Map circuit breaker with a PG table so state
-- survives restarts and is shared across all backend instances.
-- Cleanup: stale rows (opened_at older than 30s and still open) are
-- skipped automatically — consecutive_fail < 3 rows act as "closed".

CREATE TABLE IF NOT EXISTS ai_circuit_breaker (
    user_id     UUID NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    consecutive_fails INT NOT NULL DEFAULT 0,
    opened_at   TIMESTAMPTZ,
    PRIMARY KEY (user_id, provider_id)
);
