DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'strategy_templates'
  ) THEN
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'strategy_templates' AND column_name = 'status'
    ) THEN
      ALTER TABLE strategy_templates ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'published';
      CREATE INDEX IF NOT EXISTS idx_strategy_templates_status ON strategy_templates(status);
    END IF;
  END IF;
END $$;
