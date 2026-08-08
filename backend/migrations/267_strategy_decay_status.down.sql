-- 267_strategy_decay_status.down.sql
-- Remove decay status tracking columns.

ALTER TABLE marketplace_strategies
    DROP COLUMN IF EXISTS decay_status,
    DROP COLUMN IF EXISTS last_decay_at;
