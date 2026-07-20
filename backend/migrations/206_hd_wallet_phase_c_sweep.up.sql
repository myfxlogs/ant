-- 206_hd_wallet_phase_c_sweep.up.sql
-- ADR-0026 Phase C: 3-leg sweep model + signed bundle persistence (Q3 crash recovery).

-- 1. Add 3-leg tracking columns to sweep_logs (ADR §2.3).
ALTER TABLE sweep_logs ADD COLUMN IF NOT EXISTS batch_id UUID;
ALTER TABLE sweep_logs ADD COLUMN IF NOT EXISTS leg_type VARCHAR(12) NOT NULL DEFAULT 'transfer';
ALTER TABLE sweep_logs ADD COLUMN IF NOT EXISTS leg_seq SMALLINT NOT NULL DEFAULT 0;

-- Backfill batch_id for any existing rows (should be none, but safe).
UPDATE sweep_logs SET batch_id = gen_random_uuid() WHERE batch_id IS NULL;

-- Now make batch_id NOT NULL.
ALTER TABLE sweep_logs ALTER COLUMN batch_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sweep_logs_batch_id ON sweep_logs(batch_id);

-- 2. Create sweep_bundles table for persisting signed bundles (ADR §2.3, Q3).
-- Signed txs contain no private keys — safe to persist. Online restart reads back
-- and resumes broadcasting from first unconfirmed leg (re-broadcast needs no private key).
CREATE TABLE IF NOT EXISTS sweep_bundles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL UNIQUE,              -- links to sweep_logs.batch_id
    signed_bundle   BYTEA NOT NULL,                    -- serialized SignedSweepBundle proto (no private keys)
    built_at_ms     BIGINT NOT NULL,                   -- construction time (raw_tx expiry check, ~24h)
    status          VARCHAR(16) NOT NULL DEFAULT 'BROADCASTING',  -- BROADCASTING / DONE / EXPIRED
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sweep_bundles_status ON sweep_bundles(status);
