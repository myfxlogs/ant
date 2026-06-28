-- 177_global_settings_numeric_precision: Convert DOUBLE PRECISION financial columns to NUMERIC.
ALTER TABLE global_settings
  ALTER COLUMN max_risk_percent TYPE NUMERIC(5,2) USING max_risk_percent::NUMERIC(5,2),
  ALTER COLUMN max_lot_size TYPE NUMERIC(18,6) USING max_lot_size::NUMERIC(18,6),
  ALTER COLUMN max_drawdown_percent TYPE NUMERIC(5,2) USING max_drawdown_percent::NUMERIC(5,2);
