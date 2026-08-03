-- Phase 0.2: Last-known position snapshot persistence for mtapi disconnection display.
-- Stores serialized MtPositionSnapshotRecord proto as BYTEA, throttled 30s per account.
CREATE TABLE IF NOT EXISTS mt_position_snapshots (
    account_id   TEXT        NOT NULL,
    payload_proto BYTEA       NOT NULL,
    captured_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id)
);

CREATE INDEX IF NOT EXISTS idx_mt_position_snapshots_captured_at
    ON mt_position_snapshots (captured_at DESC);
