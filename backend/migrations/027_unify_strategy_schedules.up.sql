DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules_v2') THEN
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules') THEN
            IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules_legacy') THEN
                ALTER TABLE strategy_schedules RENAME TO strategy_schedules_legacy;
            END IF;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules') THEN
            ALTER TABLE strategy_schedules_v2 RENAME TO strategy_schedules;
        END IF;

        IF EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_schema = 'public' AND table_name = 'strategy_executions'
        ) THEN
            EXECUTE 'ALTER TABLE strategy_executions DROP CONSTRAINT IF EXISTS strategy_executions_schedule_id_fkey';
            EXECUTE 'ALTER TABLE strategy_executions ADD CONSTRAINT strategy_executions_schedule_id_fkey FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id) ON DELETE SET NULL';
        END IF;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules') THEN
        EXECUTE 'DROP TRIGGER IF EXISTS update_strategy_schedules_v2_updated_at ON strategy_schedules';
        EXECUTE 'DROP TRIGGER IF EXISTS update_strategy_schedules_updated_at ON strategy_schedules';
        EXECUTE 'CREATE TRIGGER update_strategy_schedules_updated_at BEFORE UPDATE ON strategy_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules_legacy') THEN
        EXECUTE 'DROP TRIGGER IF EXISTS update_strategy_schedules_updated_at ON strategy_schedules_legacy';
    END IF;
END $$;

DO $$
BEGIN
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_user_id';
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_template_id';
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_account_id';
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_symbol';
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_is_active';
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_risk_level';
    EXECUTE 'DROP INDEX IF EXISTS idx_strategy_schedules_v2_next_run_at';
END $$;

CREATE INDEX IF NOT EXISTS idx_strategy_schedules_user_id ON strategy_schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_template_id ON strategy_schedules(template_id);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_account_id ON strategy_schedules(account_id);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_symbol ON strategy_schedules(symbol);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_is_active ON strategy_schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_risk_level ON strategy_schedules(risk_level);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_next_run_at ON strategy_schedules(next_run_at);
