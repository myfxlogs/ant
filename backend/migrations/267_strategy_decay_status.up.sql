-- 267_strategy_decay_status.up.sql
-- FEAT-5: Decay status tracking on marketplace strategies.
-- decay_status: none | decaying | decayed
-- last_decay_at: when decay was last detected (throttle: at most once per day per strategy)

ALTER TABLE marketplace_strategies
    ADD COLUMN IF NOT EXISTS decay_status TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS last_decay_at TIMESTAMPTZ;
