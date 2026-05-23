-- 025_backtest_runs_worker_cancel.up.sql
-- Support persistent cancel + worker leasing for async backtest runs.

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS cancel_requested_at TIMESTAMP;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS lease_until TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_cancel_requested_at ON backtest_runs(cancel_requested_at);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_lease_until ON backtest_runs(lease_until);
