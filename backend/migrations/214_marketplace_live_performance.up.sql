-- 214_marketplace_live_performance.up.sql
-- Live performance tracking for published strategies.

CREATE TABLE IF NOT EXISTS marketplace_live_performance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
    account_id UUID NOT NULL,
    date DATE NOT NULL,
    daily_pnl NUMERIC(20,8) NOT NULL DEFAULT 0,
    daily_return NUMERIC(10,6) NOT NULL DEFAULT 0,
    equity NUMERIC(20,8) NOT NULL,
    drawdown NUMERIC(10,6) NOT NULL DEFAULT 0,
    total_trades INT NOT NULL DEFAULT 0,
    winning_trades INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_id, account_id, date)
);

CREATE TABLE IF NOT EXISTS marketplace_live_performance_summary (
    strategy_id UUID PRIMARY KEY REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
    account_id UUID NOT NULL,
    total_return NUMERIC(10,6) NOT NULL DEFAULT 0,
    annual_return NUMERIC(10,6),
    max_drawdown NUMERIC(10,6) NOT NULL DEFAULT 0,
    sharpe_ratio NUMERIC(10,6),
    win_rate NUMERIC(10,6),
    total_trades INT NOT NULL DEFAULT 0,
    avg_monthly_return NUMERIC(10,6),
    tracking_since DATE NOT NULL,
    last_updated DATE NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS linked_account_id UUID;
