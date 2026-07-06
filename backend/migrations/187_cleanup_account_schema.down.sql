-- 187_cleanup_account_schema.down.sql: reverse of up migration

-- Drop added FKs
ALTER TABLE backtest_runs DROP CONSTRAINT IF EXISTS fk_backtest_runs_account;
ALTER TABLE account_balance_history DROP CONSTRAINT IF EXISTS fk_balance_history_account;
ALTER TABLE order_history DROP CONSTRAINT IF EXISTS fk_order_history_account;
ALTER TABLE strategy_execution_logs DROP CONSTRAINT IF EXISTS fk_exec_logs_account;
ALTER TABLE account_connection_logs DROP CONSTRAINT IF EXISTS fk_conn_logs_account;

-- Drop CHECK constraint
ALTER TABLE mt_accounts DROP CONSTRAINT IF EXISTS chk_account_status;

-- Restore dropped columns
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS is_disabled BOOLEAN DEFAULT FALSE;
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS stream_status VARCHAR(20) NOT NULL DEFAULT 'inactive';

-- Restore mt_accounts_v2 view with is_disabled
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
    NOT a.is_disabled AS is_active,
    a.canonical_subscribed_symbols,
    a.created_at,
    a.updated_at
FROM mt_accounts a
LEFT JOIN brokers b ON a.broker_id = b.id
WHERE NOT a.is_disabled;

-- Restore partial indexes
CREATE INDEX IF NOT EXISTS idx_mt_accounts_disabled_status
    ON mt_accounts(user_id, account_status)
    WHERE is_disabled = false;

CREATE INDEX IF NOT EXISTS idx_mt_accounts_disabled
    ON mt_accounts(user_id, updated_at)
    WHERE is_disabled = true;
