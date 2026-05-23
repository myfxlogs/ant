-- 031_strategy_schedules_add_manual_run_fields.up.sql

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'strategy_schedules'
  ) THEN
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'strategy_schedules' AND column_name = 'manual_run_count'
    ) THEN
      ALTER TABLE strategy_schedules ADD COLUMN manual_run_count INT NOT NULL DEFAULT 0;
    END IF;

    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'strategy_schedules' AND column_name = 'last_manual_run_at'
    ) THEN
      ALTER TABLE strategy_schedules ADD COLUMN last_manual_run_at TIMESTAMP NULL;
    END IF;

    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'strategy_schedules' AND column_name = 'last_manual_error'
    ) THEN
      ALTER TABLE strategy_schedules ADD COLUMN last_manual_error TEXT NOT NULL DEFAULT '';
    END IF;

    CREATE INDEX IF NOT EXISTS idx_strategy_schedules_manual_run_count ON strategy_schedules(manual_run_count);
  END IF;
END $$;
