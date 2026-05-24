-- 099_broker_symbols.up.sql
-- Canonical symbol resolution table for mdgateway/normalizer.go.
-- Maps (broker, symbol_raw) → canonical symbol name.

CREATE TABLE IF NOT EXISTS broker_symbols (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broker          TEXT NOT NULL,
    symbol_raw      TEXT NOT NULL,
    canonical       TEXT NOT NULL,
    digits          INT NOT NULL DEFAULT 5,
    point_value     NUMERIC(20,8) NOT NULL DEFAULT 0.00001,
    lot_size        NUMERIC(20,8) NOT NULL DEFAULT 100000,
    lot_step        NUMERIC(20,8) NOT NULL DEFAULT 0.01,
    lot_min         NUMERIC(20,8) NOT NULL DEFAULT 0.01,
    lot_max         NUMERIC(20,8) NOT NULL DEFAULT 1000,
    trade_mode      INT NOT NULL DEFAULT 4,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (broker, symbol_raw)
);

CREATE INDEX IF NOT EXISTS idx_broker_symbols_canonical ON broker_symbols(canonical);
