-- 054_finalize_strategy_schedules_unification.up.sql
-- Goal: enforce a single source of truth table: strategy_schedules
-- This migration is idempotent and safe to re-run.

DO $$
BEGIN
    -- 1) Backfill from strategy_schedules_v2 when it still exists.
    IF to_regclass('public.strategy_schedules_v2') IS NOT NULL THEN
        INSERT INTO strategy_schedules (
            id, user_id, template_id, account_id, name, symbol, timeframe,
            parameters, schedule_type, schedule_config, backtest_metrics,
            risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
            is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
            created_at, updated_at
        )
        SELECT
            id, user_id, template_id, account_id, name, symbol, timeframe,
            parameters, schedule_type, schedule_config, backtest_metrics,
            risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
            is_active, last_run_at, next_run_at, run_count, last_error,
            0,
            created_at, updated_at
        FROM strategy_schedules_v2
        ON CONFLICT (id) DO UPDATE SET
            user_id = EXCLUDED.user_id,
            template_id = EXCLUDED.template_id,
            account_id = EXCLUDED.account_id,
            name = EXCLUDED.name,
            symbol = EXCLUDED.symbol,
            timeframe = EXCLUDED.timeframe,
            parameters = EXCLUDED.parameters,
            schedule_type = EXCLUDED.schedule_type,
            schedule_config = EXCLUDED.schedule_config,
            backtest_metrics = EXCLUDED.backtest_metrics,
            risk_score = EXCLUDED.risk_score,
            risk_level = EXCLUDED.risk_level,
            risk_reasons = EXCLUDED.risk_reasons,
            risk_warnings = EXCLUDED.risk_warnings,
            last_backtest_at = EXCLUDED.last_backtest_at,
            is_active = EXCLUDED.is_active,
            last_run_at = EXCLUDED.last_run_at,
            next_run_at = EXCLUDED.next_run_at,
            run_count = EXCLUDED.run_count,
            last_error = EXCLUDED.last_error,
            enable_count = GREATEST(strategy_schedules.enable_count, EXCLUDED.enable_count),
            updated_at = GREATEST(strategy_schedules.updated_at, EXCLUDED.updated_at);

        -- Safety valve: do not drop source tables in migration.
        -- We keep them as recovery fallback because this project replays
        -- all .up.sql files on each backend restart.
    END IF;
END $$;

DO $$
BEGIN
    -- 2) Backfill from strategy_schedules_legacy if there are still rows.
    IF to_regclass('public.strategy_schedules_legacy') IS NOT NULL THEN
        -- legacy v1 schema has strategy_id; newer archived schema may already be v2-like.
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'strategy_schedules_legacy'
              AND column_name = 'strategy_id'
        ) THEN
            INSERT INTO strategy_schedules (
                id, user_id, template_id, account_id, name, symbol, timeframe,
                parameters, schedule_type, schedule_config, backtest_metrics,
                risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
                is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
                created_at, updated_at
            )
            SELECT
                s.id,
                st.user_id,
                s.strategy_id,
                s.account_id,
                COALESCE(st.name, ''),
                COALESCE(st.symbol, ''),
                'H1',
                '{}'::jsonb,
                COALESCE(NULLIF(s.schedule_type, ''), 'interval'),
                COALESCE(s.schedule_config, '{}'::jsonb),
                NULL::jsonb,
                NULL::int,
                'unknown',
                '[]'::jsonb,
                '[]'::jsonb,
                NULL::timestamp,
                COALESCE(s.is_active, false),
                s.last_run_at,
                s.next_run_at,
                COALESCE(s.run_count, 0),
                s.last_error,
                0,
                s.created_at,
                s.updated_at
            FROM strategy_schedules_legacy s
            LEFT JOIN strategies st ON st.id = s.strategy_id
            ON CONFLICT (id) DO NOTHING;
        ELSE
            INSERT INTO strategy_schedules (
                id, user_id, template_id, account_id, name, symbol, timeframe,
                parameters, schedule_type, schedule_config, backtest_metrics,
                risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
                is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
                created_at, updated_at
            )
            SELECT
                id, user_id, template_id, account_id, name, symbol, timeframe,
                parameters, schedule_type, schedule_config, backtest_metrics,
                risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
                is_active, last_run_at, next_run_at, run_count, last_error,
                0,
                created_at, updated_at
            FROM strategy_schedules_legacy
            ON CONFLICT (id) DO NOTHING;
        END IF;

        -- Safety valve: keep legacy table as recovery fallback.
    END IF;
END $$;

DO $$
BEGIN
    -- 3) Ensure log FKs point to canonical strategy_schedules.
    IF to_regclass('public.strategy_execution_logs') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_execution_logs_schedule_id_fkey') THEN
            ALTER TABLE strategy_execution_logs DROP CONSTRAINT strategy_execution_logs_schedule_id_fkey;
        END IF;
        ALTER TABLE strategy_execution_logs
            ADD CONSTRAINT strategy_execution_logs_schedule_id_fkey
            FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id);
    END IF;

    IF to_regclass('public.order_history') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'order_history_schedule_id_fkey') THEN
            ALTER TABLE order_history DROP CONSTRAINT order_history_schedule_id_fkey;
        END IF;
        ALTER TABLE order_history
            ADD CONSTRAINT order_history_schedule_id_fkey
            FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id);
    END IF;
END $$;
