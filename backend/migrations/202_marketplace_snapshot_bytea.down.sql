-- Revert: convert marketplace_strategies.backtest_snapshot BYTEA back to JSONB
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'marketplace_strategies'
          AND column_name = 'backtest_snapshot'
          AND data_type = 'bytea'
    ) THEN
        ALTER TABLE marketplace_strategies ADD COLUMN backtest_snapshot_jsonb JSONB;
        ALTER TABLE marketplace_strategies DROP COLUMN backtest_snapshot;
        ALTER TABLE marketplace_strategies RENAME COLUMN backtest_snapshot_jsonb TO backtest_snapshot;
    END IF;
END $$;
