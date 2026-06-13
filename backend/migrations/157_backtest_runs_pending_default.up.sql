-- Set default status to PENDING for newly created backtest runs.
-- Migration 023 originally set the default to 'SUCCEEDED', which is semantically
-- backward — a newly created run should start in PENDING state.
ALTER TABLE backtest_runs ALTER COLUMN status SET DEFAULT 'PENDING';
