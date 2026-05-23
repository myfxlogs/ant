-- 041_backtest_runs_template_ref.up.sql
-- Add optional linkage from backtest runs to strategy template (published) or template draft.

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS template_id UUID;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS template_draft_id UUID;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_template_id ON backtest_runs(template_id);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_template_draft_id ON backtest_runs(template_draft_id);
