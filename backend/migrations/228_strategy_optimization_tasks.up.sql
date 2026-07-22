-- Phase 5.1: AI Strategy Iteration — optimization tasks table
CREATE TABLE IF NOT EXISTS marketplace_strategy_optimization_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     UUID NOT NULL,
    publisher_id    UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending | generating | completed | rejected | published
    trigger_reason  TEXT NOT NULL DEFAULT '',          -- decay_detected | manual | scheduled
    decay_metrics   JSONB,                             -- snapshot of decay detection results
    suggested_code  TEXT,                              -- AI-generated optimized source code
    suggested_params TEXT,                             -- suggested parameter changes (JSON)
    backtest_snapshot BYTEA,                           -- proto-serialized BacktestSnapshot of optimized version
    change_summary  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_opt_tasks_strategy ON marketplace_strategy_optimization_tasks(strategy_id);
CREATE INDEX idx_opt_tasks_publisher ON marketplace_strategy_optimization_tasks(publisher_id);
CREATE INDEX idx_opt_tasks_status ON marketplace_strategy_optimization_tasks(status);

-- Track which version an optimization task produced (if published)
ALTER TABLE marketplace_strategy_optimization_tasks
    ADD COLUMN IF NOT EXISTS published_version_id UUID;
