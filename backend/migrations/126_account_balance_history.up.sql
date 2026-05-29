-- Tracks periodic equity/balance snapshots so the equity curve is available
-- even for periods with no closed trades (open positions still show net value).
CREATE TABLE IF NOT EXISTS account_balance_history (
    id BIGSERIAL PRIMARY KEY,
    account_id UUID NOT NULL,
    balance DOUBLE PRECISION NOT NULL DEFAULT 0,
    equity DOUBLE PRECISION NOT NULL DEFAULT 0,
    margin DOUBLE PRECISION NOT NULL DEFAULT 0,
    free_margin DOUBLE PRECISION NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ab_hist_account_time
    ON account_balance_history (account_id, recorded_at DESC);
