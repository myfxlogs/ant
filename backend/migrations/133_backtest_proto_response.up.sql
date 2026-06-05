-- 133_backtest_proto_response: store full ExecuteBacktestResponse proto binary
-- Replaces metrics/equity_curve JSONB columns with a single proto_response BYTEA.
-- Proto is the canonical wire format; JSONB was a temporary transport hack.

ALTER TABLE backtest_runs ADD COLUMN proto_response BYTEA;

COMMENT ON COLUMN backtest_runs.proto_response IS 'Serialized ant.v1.ExecuteBacktestResponse proto binary';
