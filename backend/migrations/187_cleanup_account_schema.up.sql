-- 187_cleanup_account_schema: drop redundant columns, add CHECK, fix FK CASCADE
-- Date: 2026-07-06

-- 0. Data cleanup: normalize existing values before adding constraints

-- 0a. Fix orphan backtest_runs (zero UUID = no account reference)
DELETE FROM backtest_runs WHERE account_id = '00000000-0000-0000-0000-000000000000';

-- 0b. Normalize account_status values
UPDATE mt_accounts SET account_status = 'disconnected'
WHERE account_status IS NULL
   OR account_status NOT IN ('connecting', 'connected', 'disconnected', 'frozen');

-- 1. Drop partial indexes that reference is_disabled
DROP INDEX IF EXISTS idx_mt_accounts_disabled_status;
DROP INDEX IF EXISTS idx_mt_accounts_disabled;

-- 2. Recreate mt_accounts_v2 view without is_disabled reference
DROP VIEW IF EXISTS mt_accounts_v2;
CREATE VIEW mt_accounts_v2 AS
SELECT
    a.id,
    a.user_id,
    a.mt_type AS platform,
    a.broker_company AS broker,
    COALESCE(NULLIF(b.mtapi_endpoint, ''), '')::varchar(100) AS mtapi_host,
    a.mtapi_port,
    a.login,
    a.password,
    a.mt_token,
    a.broker_host,
    a.broker_server AS server,
    (a.account_status <> 'frozen') AS is_active,
    a.canonical_subscribed_symbols,
    a.created_at,
    a.updated_at
FROM mt_accounts a
LEFT JOIN brokers b ON a.broker_id = b.id;

-- 3. Drop redundant columns
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS is_disabled;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS stream_status;

-- 4. Add CHECK constraint on account_status (idempotent via DO block)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_account_status'
          AND conrelid = 'mt_accounts'::regclass
    ) THEN
        ALTER TABLE mt_accounts ADD CONSTRAINT chk_account_status
            CHECK (account_status IN ('connecting', 'connected', 'disconnected', 'frozen'));
    END IF;
END $$;

-- 5. Fix FK constraints: drop existing (without CASCADE) and re-add with CASCADE

-- account_connection_logs: existing FK without CASCADE
DO $$
DECLARE cname text;
BEGIN
    SELECT conname INTO cname FROM pg_constraint
    WHERE conrelid = 'account_connection_logs'::regclass
      AND contype = 'f'
      AND confrelid = 'mt_accounts'::regclass;
    IF cname IS NOT NULL THEN
        EXECUTE format('ALTER TABLE account_connection_logs DROP CONSTRAINT %I', cname);
    END IF;
END $$;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_conn_logs_account'
          AND conrelid = 'account_connection_logs'::regclass
    ) THEN
        ALTER TABLE account_connection_logs
            ADD CONSTRAINT fk_conn_logs_account
            FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
    END IF;
END $$;

-- strategy_execution_logs: existing FK without CASCADE
DO $$
DECLARE cname text;
BEGIN
    SELECT conname INTO cname FROM pg_constraint
    WHERE conrelid = 'strategy_execution_logs'::regclass
      AND contype = 'f'
      AND confrelid = 'mt_accounts'::regclass;
    IF cname IS NOT NULL THEN
        EXECUTE format('ALTER TABLE strategy_execution_logs DROP CONSTRAINT %I', cname);
    END IF;
END $$;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_exec_logs_account'
          AND conrelid = 'strategy_execution_logs'::regclass
    ) THEN
        ALTER TABLE strategy_execution_logs
            ADD CONSTRAINT fk_exec_logs_account
            FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
    END IF;
END $$;

-- order_history: existing FK without CASCADE
DO $$
DECLARE cname text;
BEGIN
    SELECT conname INTO cname FROM pg_constraint
    WHERE conrelid = 'order_history'::regclass
      AND contype = 'f'
      AND confrelid = 'mt_accounts'::regclass;
    IF cname IS NOT NULL THEN
        EXECUTE format('ALTER TABLE order_history DROP CONSTRAINT %I', cname);
    END IF;
END $$;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_order_history_account'
          AND conrelid = 'order_history'::regclass
    ) THEN
        ALTER TABLE order_history
            ADD CONSTRAINT fk_order_history_account
            FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 6. Add completely missing FKs

-- account_balance_history: no FK at all
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_balance_history_account'
          AND conrelid = 'account_balance_history'::regclass
    ) THEN
        ALTER TABLE account_balance_history
            ADD CONSTRAINT fk_balance_history_account
            FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
    END IF;
END $$;

-- backtest_runs: no FK at all
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_backtest_runs_account'
          AND conrelid = 'backtest_runs'::regclass
    ) THEN
        ALTER TABLE backtest_runs
            ADD CONSTRAINT fk_backtest_runs_account
            FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
    END IF;
END $$;
