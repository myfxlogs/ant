DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules_legacy') THEN
        IF EXISTS (
            SELECT 1
            FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            WHERE c.conname = 'strategy_schedules_pkey' AND t.relname = 'strategy_schedules_legacy'
        ) THEN
            EXECUTE 'ALTER TABLE strategy_schedules_legacy RENAME CONSTRAINT strategy_schedules_pkey TO strategy_schedules_legacy_pkey';
        END IF;
        IF EXISTS (
            SELECT 1
            FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            WHERE c.conname = 'strategy_schedules_account_id_fkey' AND t.relname = 'strategy_schedules_legacy'
        ) THEN
            EXECUTE 'ALTER TABLE strategy_schedules_legacy RENAME CONSTRAINT strategy_schedules_account_id_fkey TO strategy_schedules_legacy_account_id_fkey';
        END IF;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules') THEN
        EXECUTE 'DROP TRIGGER IF EXISTS update_strategy_schedules_v2_updated_at ON strategy_schedules';

        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_user_id';
        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_template_id';
        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_account_id';
        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_symbol';
        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_is_active';
        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_risk_level';
        EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_next_run_at';

        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_v2_pkey') AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_pkey') THEN
            EXECUTE 'ALTER TABLE strategy_schedules RENAME CONSTRAINT strategy_schedules_v2_pkey TO strategy_schedules_pkey';
        END IF;

        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_v2_user_id_fkey') AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_user_id_fkey') THEN
            EXECUTE 'ALTER TABLE strategy_schedules RENAME CONSTRAINT strategy_schedules_v2_user_id_fkey TO strategy_schedules_user_id_fkey';
        END IF;
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_v2_template_id_fkey') AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_template_id_fkey') THEN
            EXECUTE 'ALTER TABLE strategy_schedules RENAME CONSTRAINT strategy_schedules_v2_template_id_fkey TO strategy_schedules_template_id_fkey';
        END IF;
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_v2_account_id_fkey') AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_schedules_account_id_fkey') THEN
            EXECUTE 'ALTER TABLE strategy_schedules RENAME CONSTRAINT strategy_schedules_v2_account_id_fkey TO strategy_schedules_account_id_fkey';
        END IF;

        EXECUTE 'DROP TRIGGER IF EXISTS update_strategy_schedules_updated_at ON strategy_schedules';
        EXECUTE 'CREATE TRIGGER update_strategy_schedules_updated_at BEFORE UPDATE ON strategy_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()';
    END IF;
END $$;
