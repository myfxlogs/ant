-- 260_marketplace_quality_gates_activate.down.sql
-- Revert quality gate thresholds to migration 213 defaults.
-- Note: deleted waivers cannot be restored.

UPDATE system_config SET value = '0.5',  updated_at = now()
  WHERE key = 'marketplace.quality.min_sharpe_ratio';
UPDATE system_config SET value = '0.30', updated_at = now()
  WHERE key = 'marketplace.quality.max_drawdown_pct';
UPDATE system_config SET value = '20',   updated_at = now()
  WHERE key = 'marketplace.quality.min_total_trades';
UPDATE system_config SET value = '0.35', updated_at = now()
  WHERE key = 'marketplace.quality.min_win_rate';
