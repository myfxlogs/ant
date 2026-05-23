-- 024_backtest_runs_mode_range.up.sql
-- Persist async backtest run input mode and optional time range.

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'KLINE_RANGE';

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS from_ts TIMESTAMP;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS to_ts TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_mode ON backtest_runs(mode);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_range ON backtest_runs(from_ts, to_ts);
