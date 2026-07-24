-- 237_marketplace_trial_days.up.sql
-- I2: Publisher-configurable trial period (replaces hardcoded 7-day).
ALTER TABLE marketplace_strategies
    ADD COLUMN IF NOT EXISTS trial_days INT NOT NULL DEFAULT 7;
