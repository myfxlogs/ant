-- 030_trade_records_add_schedule_id.up.sql

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'trade_records'
  ) THEN
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'trade_records' AND column_name = 'schedule_id'
    ) THEN
      ALTER TABLE trade_records ADD COLUMN schedule_id UUID NULL;
      ALTER TABLE trade_records ADD CONSTRAINT fk_trade_records_schedule_id
        FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id) ON DELETE SET NULL;
      CREATE INDEX IF NOT EXISTS idx_trade_records_schedule_id ON trade_records(schedule_id);
    END IF;
  END IF;
END $$;
