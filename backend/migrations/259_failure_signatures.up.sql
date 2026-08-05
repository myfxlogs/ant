CREATE TABLE IF NOT EXISTS failure_signatures (
    id           BIGSERIAL PRIMARY KEY,
    signature_hash TEXT NOT NULL,
    source_hash  TEXT NOT NULL,
    rule_ids     TEXT[] NOT NULL DEFAULT '{}',
    blind_spots  TEXT[] NOT NULL DEFAULT '{}',
    total_trades INTEGER NOT NULL DEFAULT 0,
    symbol       TEXT NOT NULL DEFAULT '',
    timeframe    TEXT NOT NULL DEFAULT '',
    source_preview TEXT NOT NULL DEFAULT '',
    findings     BYTEA NOT NULL DEFAULT '',
    backtest_run_id UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_failure_sig_hash ON failure_signatures(signature_hash);
CREATE INDEX IF NOT EXISTS idx_failure_sig_source ON failure_signatures(source_hash);
CREATE INDEX IF NOT EXISTS idx_failure_sig_created ON failure_signatures(created_at DESC);
