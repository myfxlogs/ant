-- 202_marketplace_snapshot_bytea: convert backtest_snapshot JSONB → BYTEA for proto binary.
-- Existing JSON data is dropped (marketplace has no production data per migration 162).
-- New writes use proto-serialized antv1.BacktestSnapshot bytes.

-- Only run if backtest_snapshot is still JSONB (not already BYTEA).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'marketplace_strategies'
          AND column_name = 'backtest_snapshot'
          AND data_type = 'jsonb'
    ) THEN
        ALTER TABLE marketplace_strategies ADD COLUMN backtest_snapshot_proto BYTEA;
        ALTER TABLE marketplace_strategies DROP COLUMN backtest_snapshot;
        ALTER TABLE marketplace_strategies RENAME COLUMN backtest_snapshot_proto TO backtest_snapshot;
    END IF;
END $$;
