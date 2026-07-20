CREATE TABLE auto_generation_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(30) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    strategy_type VARCHAR(30) NOT NULL,   -- trend_following / mean_reversion / breakout / arbitrage
    risk_level VARCHAR(15) NOT NULL DEFAULT 'moderate',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- pending → generating → compiling → backtesting → awaiting_review → published / rejected
    strategy_id UUID,                     -- assigned after generation
    result_backtest_snapshot BYTEA,        -- proto BacktestSnapshot
    quality_passed BOOLEAN,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_autogen_tasks_status ON auto_generation_tasks(status);

-- NOTIFY channel for push-first consumer wakeup.
-- Producer sends: SELECT pg_notify('auto_generation_task_ready', '');
-- Consumer LISTENs on this channel.
