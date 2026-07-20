-- 213_marketplace_quality_gates.up.sql
-- Backtest quality gate configuration + admin waiver table.

INSERT INTO system_config (key, value, value_type, description, enabled, admin_visible) VALUES
    ('marketplace.quality.min_sharpe_ratio',        '0.5',   'decimal', 'Minimum Sharpe ratio for publishing', true, true),
    ('marketplace.quality.max_drawdown_pct',        '0.30',  'decimal', 'Maximum drawdown percentage (0-1)', true, true),
    ('marketplace.quality.min_total_trades',        '20',    'int',     'Minimum total trades in backtest', true, true),
    ('marketplace.quality.min_win_rate',            '0.35',  'decimal', 'Minimum win rate (0-1)', true, true),
    ('marketplace.quality.max_is_oos_degradation',  '0.5',   'decimal', 'Max IS vs OOS metric degradation ratio', true, true),
    ('marketplace.quality.enforce_backtest_snapshot', 'true', 'bool',   'Require backtest snapshot for publishing', true, true)
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS marketplace_quality_waivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
    waived_by UUID NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
