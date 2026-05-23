DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'strategy_schedules'
    ) AND EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'strategy_schedules' AND column_name = 'strategy_id'
    ) THEN
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_schema = 'public' AND table_name = 'strategy_schedules_legacy'
        ) THEN
            ALTER TABLE strategy_schedules RENAME TO strategy_schedules_legacy;
        END IF;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'strategy_schedules_v2'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'strategy_schedules'
    ) THEN
        ALTER TABLE strategy_schedules_v2 RENAME TO strategy_schedules;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'strategy_schedules'
    ) THEN
        CREATE TABLE strategy_schedules (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            template_id UUID NOT NULL REFERENCES strategy_templates(id) ON DELETE CASCADE,
            account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,

            name VARCHAR(255),
            symbol VARCHAR(20) NOT NULL,
            timeframe VARCHAR(10) NOT NULL DEFAULT 'H1',
            parameters JSONB DEFAULT '{}',

            schedule_type VARCHAR(20) NOT NULL DEFAULT 'interval',
            schedule_config JSONB NOT NULL DEFAULT '{}',

            backtest_metrics JSONB,
            risk_score INTEGER,
            risk_level VARCHAR(10),
            risk_reasons JSONB DEFAULT '[]',
            risk_warnings JSONB DEFAULT '[]',
            last_backtest_at TIMESTAMP,

            is_active BOOLEAN DEFAULT false,
            last_run_at TIMESTAMP,
            next_run_at TIMESTAMP,
            run_count INTEGER DEFAULT 0,
            last_error TEXT,

            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'strategy_schedules_legacy'
    ) THEN
        INSERT INTO strategy_templates (id, user_id, name, description, code, parameters, is_public, tags, use_count, created_at, updated_at)
        SELECT
            st.id,
            st.user_id,
            st.name,
            st.description,
            '',
            '{}'::jsonb,
            false,
            '{}'::varchar(100)[],
            0,
            st.created_at,
            st.updated_at
        FROM strategies st
        WHERE st.id IN (SELECT DISTINCT strategy_id FROM strategy_schedules_legacy)
        ON CONFLICT (id) DO NOTHING;

        INSERT INTO strategy_schedules (
            id, user_id, template_id, account_id, name, symbol, timeframe,
            parameters, schedule_type, schedule_config,
            backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
            is_active, last_run_at, next_run_at, run_count, last_error,
            created_at, updated_at
        )
        SELECT
            s.id,
            st.user_id,
            s.strategy_id,
            s.account_id,
            st.name,
            st.symbol,
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
            s.created_at,
            s.updated_at
        FROM strategy_schedules_legacy s
        JOIN strategies st ON st.id = s.strategy_id
        ON CONFLICT (id) DO NOTHING;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'strategy_executions'
    ) THEN
        EXECUTE 'ALTER TABLE strategy_executions DROP CONSTRAINT IF EXISTS strategy_executions_schedule_id_fkey';
        EXECUTE 'ALTER TABLE strategy_executions ADD CONSTRAINT strategy_executions_schedule_id_fkey FOREIGN KEY (schedule_id) REFERENCES strategy_schedules(id) ON DELETE SET NULL';
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'strategy_schedules') THEN
        EXECUTE 'DROP TRIGGER IF EXISTS update_strategy_schedules_updated_at ON strategy_schedules';
        EXECUTE 'CREATE TRIGGER update_strategy_schedules_updated_at BEFORE UPDATE ON strategy_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_strategy_schedules_user_id ON strategy_schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_template_id ON strategy_schedules(template_id);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_account_id ON strategy_schedules(account_id);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_symbol ON strategy_schedules(symbol);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_is_active ON strategy_schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_risk_level ON strategy_schedules(risk_level);
CREATE INDEX IF NOT EXISTS idx_strategy_schedules_next_run_at ON strategy_schedules(next_run_at);
