ALTER TABLE strategy_experiments
    DROP COLUMN IF EXISTS to_ts_unix_ms,
    DROP COLUMN IF EXISTS from_ts_unix_ms,
    DROP COLUMN IF EXISTS timeframe,
    DROP COLUMN IF EXISTS symbol;
