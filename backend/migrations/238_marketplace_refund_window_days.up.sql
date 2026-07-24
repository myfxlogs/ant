-- 238_marketplace_refund_window_days.up.sql
-- Publisher-configurable refund window (replaces hardcoded 7-day).
-- When a strategy is purchased, this value flows into marketplace_settlements.refund_window_days
-- and is used by refund_request.go to determine eligibility.
ALTER TABLE marketplace_strategies
    ADD COLUMN IF NOT EXISTS refund_window_days INT NOT NULL DEFAULT 7;

COMMENT ON COLUMN marketplace_strategies.refund_window_days IS 'Publisher-configurable refund window in days (default 7). Determines how long after purchase a buyer can request a refund before the settlement is credited to the publisher.';
