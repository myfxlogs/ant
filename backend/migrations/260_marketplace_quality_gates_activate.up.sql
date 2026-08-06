-- 260_marketplace_quality_gates_activate.up.sql
-- Activate marketplace quality gates: update thresholds to spec defaults + clear test waivers.
-- Layer 1: gates table had overly strict test values (sharpe=0.5, drawdown=0.30, trades=20).
--          Replace with宽松 defaults — quality gate blocks假数据/无数据, not "bad" strategies.
-- Layer 3: marketplace_quality_waivers had 6 test rows (= all published strategies exempt).
--          Waivers should be admin-approved per-strategy, not batch-preloaded.

UPDATE system_config SET value = '-1.0',  updated_at = now()
  WHERE key = 'marketplace.quality.min_sharpe_ratio';
UPDATE system_config SET value = '0.80',  updated_at = now()
  WHERE key = 'marketplace.quality.max_drawdown_pct';
UPDATE system_config SET value = '10',    updated_at = now()
  WHERE key = 'marketplace.quality.min_total_trades';
UPDATE system_config SET value = '0.0',   updated_at = now()
  WHERE key = 'marketplace.quality.min_win_rate';
UPDATE system_config SET value = 'true',  updated_at = now()
  WHERE key = 'marketplace.quality.enforce_backtest_snapshot';

-- Ensure rows exist in case migration 213 was partially applied.
INSERT INTO system_config (key, value, value_type, description, enabled, admin_visible) VALUES
    ('marketplace.quality.min_sharpe_ratio',          '-1.0', 'decimal', 'Minimum Sharpe ratio for publishing',          true, true),
    ('marketplace.quality.max_drawdown_pct',          '0.80', 'decimal', 'Maximum drawdown percentage (0-1)',             true, true),
    ('marketplace.quality.min_total_trades',          '10',   'int',     'Minimum total trades in backtest',              true, true),
    ('marketplace.quality.min_win_rate',              '0.0',  'decimal', 'Minimum win rate (0-1)',                        true, true),
    ('marketplace.quality.max_is_oos_degradation',    '0.5',  'decimal', 'Max IS vs OOS metric degradation ratio',        true, true),
    ('marketplace.quality.enforce_backtest_snapshot', 'true', 'bool',    'Require backtest snapshot for publishing',      true, true)
ON CONFLICT (key) DO UPDATE SET
    value       = EXCLUDED.value,
    updated_at  = now();

-- Layer 3: clear all test waivers. Admin must grant waivers individually going forward.
DELETE FROM marketplace_quality_waivers;
