-- 171_copytrade_signal_dedup.up.sql
-- Idempotency table for copy-trade signal processing.
-- Rows auto-expire after 24h to keep the table small.

CREATE TABLE IF NOT EXISTS copytrade_signals (
    signal_id   VARCHAR(128) PRIMARY KEY,
    strategy_id UUID NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_copytrade_signals_expires ON copytrade_signals(expires_at);

COMMENT ON TABLE copytrade_signals IS 'Processed copy-trade signal IDs for idempotency (24h TTL).';
