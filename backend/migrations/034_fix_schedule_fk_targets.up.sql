-- 034_fix_schedule_fk_targets.up.sql

DO $$
BEGIN
    -- strategy_execution_logs.schedule_id should reference strategy_schedules (not strategy_schedules_v2)
    IF to_regclass('public.strategy_execution_logs') IS NOT NULL THEN
        -- Drop old FK if exists
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_execution_logs_schedule_id_fkey') THEN
            ALTER TABLE strategy_execution_logs DROP CONSTRAINT strategy_execution_logs_schedule_id_fkey;
        END IF;

        IF to_regclass('public.strategy_schedules') IS NOT NULL THEN
            ALTER TABLE strategy_execution_logs
                ADD CONSTRAINT strategy_execution_logs_schedule_id_fkey
                FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id);
        END IF;
    END IF;

    -- order_history.schedule_id should reference strategy_schedules (not strategy_schedules_v2)
    IF to_regclass('public.order_history') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'order_history_schedule_id_fkey') THEN
            ALTER TABLE order_history DROP CONSTRAINT order_history_schedule_id_fkey;
        END IF;

        IF to_regclass('public.strategy_schedules') IS NOT NULL THEN
            ALTER TABLE order_history
                ADD CONSTRAINT order_history_schedule_id_fkey
                FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id);
        END IF;
    END IF;
END $$;

INSERT INTO schema_migrations (version) VALUES ('034_fix_schedule_fk_targets') ON CONFLICT DO NOTHING;
