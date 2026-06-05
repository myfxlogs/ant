-- 136_backtest_drop_legacy_jsonb: remove metrics/equity_curve JSONB columns
-- These were superseded by proto_response BYTEA (migration 133).
-- The API reads metrics, equity_curve, and risk from proto_response via
-- parseMetrics/parseEquityCurve/parseRisk which unmarshal ExecuteBacktestResponse.

ALTER TABLE backtest_runs DROP COLUMN IF EXISTS metrics;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS equity_curve;
