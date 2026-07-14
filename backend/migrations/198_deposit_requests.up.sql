-- 198_deposit_requests.up.sql
-- USDT deposit requests: user submits deposit → admin reviews → approve credits wallet.

CREATE TABLE IF NOT EXISTS deposit_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount          NUMERIC(20,8) NOT NULL,          -- USDT amount user claims to have sent
    amount_usd      NUMERIC(20,2) NOT NULL,          -- USD equivalent to credit on approval
    tx_hash         TEXT,                             -- optional: user-provided on-chain tx hash
    status          TEXT NOT NULL DEFAULT 'PENDING', -- PENDING / APPROVED / REJECTED
    reviewer_id     UUID REFERENCES users(id),       -- admin who approved/rejected
    review_note     TEXT,                             -- admin note on approval/rejection
    reviewed_at     TIMESTAMPTZ,
    wallet_tx_id    UUID,                             -- wallet_transactions.id created on approval
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_deposit_requests_status ON deposit_requests(status);
CREATE INDEX idx_deposit_requests_user   ON deposit_requests(user_id);
CREATE INDEX idx_deposit_requests_created ON deposit_requests(created_at DESC);

-- Seed USDT receiving address config (admin can update via AdminConfigService).
INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_receiving_address', '', 'USDT (TRC20) receiving address for user deposits', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_network', 'TRC20', 'USDT network for deposits (TRC20/ERC20)', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_exchange_rate', '1', 'USD amount credited per 1 USDT (default 1:1)', true)
ON CONFLICT (key) DO NOTHING;
