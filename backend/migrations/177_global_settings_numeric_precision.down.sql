-- 177_global_settings_numeric_precision: Revert NUMERIC back to DOUBLE PRECISION.
ALTER TABLE global_settings
  ALTER COLUMN max_risk_percent TYPE DOUBLE PRECISION USING max_risk_percent::DOUBLE PRECISION,
  ALTER COLUMN max_lot_size TYPE DOUBLE PRECISION USING max_lot_size::DOUBLE PRECISION,
  ALTER COLUMN max_drawdown_percent TYPE DOUBLE PRECISION USING max_drawdown_percent::DOUBLE PRECISION;
