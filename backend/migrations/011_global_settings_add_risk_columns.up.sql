-- 011_global_settings_add_risk_columns: Fix missing risk columns on global_settings.
-- Model & repository already reference these; migration 010 omitted them.
ALTER TABLE global_settings
  ADD COLUMN IF NOT EXISTS max_risk_percent DOUBLE PRECISION DEFAULT 2.0,
  ADD COLUMN IF NOT EXISTS max_positions INTEGER DEFAULT 10,
  ADD COLUMN IF NOT EXISTS max_lot_size DOUBLE PRECISION DEFAULT 100.0,
  ADD COLUMN IF NOT EXISTS max_daily_loss DECIMAL(18,2) DEFAULT 5000.00,
  ADD COLUMN IF NOT EXISTS max_drawdown_percent DOUBLE PRECISION DEFAULT 10.0;
