-- 032_strategy_schedules_add_enable_count.up.sql

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'strategy_schedules'
  ) THEN
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'strategy_schedules' AND column_name = 'enable_count'
    ) THEN
      ALTER TABLE strategy_schedules ADD COLUMN enable_count INT NOT NULL DEFAULT 0;
      CREATE INDEX IF NOT EXISTS idx_strategy_schedules_enable_count ON strategy_schedules(enable_count);

      -- Backfill: schedules that are already active should have at least 1 enable.
      UPDATE strategy_schedules SET enable_count = 1 WHERE is_active = true AND enable_count = 0;
    END IF;
  END IF;
END $$;
