-- 011_global_settings_add_risk_columns.down.sql
ALTER TABLE global_settings
  DROP COLUMN IF EXISTS max_risk_percent,
  DROP COLUMN IF EXISTS max_positions,
  DROP COLUMN IF EXISTS max_lot_size,
  DROP COLUMN IF EXISTS max_daily_loss,
  DROP COLUMN IF EXISTS max_drawdown_percent;
