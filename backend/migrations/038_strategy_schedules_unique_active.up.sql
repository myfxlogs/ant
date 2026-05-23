DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'strategy_schedules'
  ) THEN
    EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS ux_strategy_schedules_active_unique ON strategy_schedules(user_id, account_id, template_id, symbol, timeframe) WHERE is_active = true';
  END IF;
END $$;
