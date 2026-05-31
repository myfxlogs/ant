-- 128_account_indexes: performance indexes + updated_at trigger
-- Date: 2026-05-31

-- Composite index for ListAccounts filtered by user + status
CREATE INDEX IF NOT EXISTS idx_mt_accounts_user_status
    ON mt_accounts(user_id, account_status);

-- Partial index for Dashboard queries (exclude disabled accounts)
CREATE INDEX IF NOT EXISTS idx_mt_accounts_disabled_status
    ON mt_accounts(user_id, account_status)
    WHERE is_disabled = false;

-- Partial index for disabled accounts section
CREATE INDEX IF NOT EXISTS idx_mt_accounts_disabled
    ON mt_accounts(user_id, updated_at)
    WHERE is_disabled = true;

-- Trade records index for GetRecentTrades paginated by account
CREATE INDEX IF NOT EXISTS idx_trade_records_account_close
    ON trade_records(account_id, close_time DESC);

-- Balance history index for GetEquityCurve date-range lookups
CREATE INDEX IF NOT EXISTS idx_account_balance_history_lookup
    ON account_balance_history(account_id, recorded_at DESC);

-- trigger: auto-update updated_at on every row change
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mt_accounts_updated_at ON mt_accounts;
CREATE TRIGGER trg_mt_accounts_updated_at
    BEFORE UPDATE ON mt_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
