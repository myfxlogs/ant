-- Phase 2: AI Token credit-based billing.
-- credit_accounts: per-user credit balance (1 credit = $0.01).
-- credit_transactions: immutable audit trail of all credit changes.
-- Reuses wallet table structure pattern (NUMERIC(20,8)).

CREATE TABLE IF NOT EXISTS credit_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    balance         NUMERIC(20,8) NOT NULL DEFAULT 0,
    frozen_balance  NUMERIC(20,8) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_accounts_user_id ON credit_accounts(user_id);

CREATE TABLE IF NOT EXISTS credit_transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES credit_accounts(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    tx_type         VARCHAR(30) NOT NULL,
    -- tx_type values:
    --   'deposit'             — admin manual top-up
    --   'subscription_grant'  — monthly subscription credits
    --   'free_grant'          — daily free credits
    --   'ai_usage'            — actual AI call settlement
    --   'ai_hold'             — pre-deduction (frozen)
    --   'ai_release'          — release unused hold back
    --   'refund'              — admin refund
    --   'adjustment'          — manual correction
    amount          NUMERIC(20,8) NOT NULL,
    balance_before  NUMERIC(20,8) NOT NULL,
    balance_after   NUMERIC(20,8) NOT NULL,
    source          VARCHAR(50),
    description     TEXT,
    operator_id     UUID,
    related_tx_id   UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_transactions_user_id ON credit_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_credit_transactions_created_at ON credit_transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_credit_transactions_tx_type ON credit_transactions(tx_type);

-- Add markup_rate and model_tier to ai_models for credit pricing.
-- markup_rate: multiplier on vendor cost (1.5x flagship, 2.5x lightweight).
-- model_tier: 'flagship' or 'lightweight' for pricing policy.
ALTER TABLE ai_models
    ADD COLUMN IF NOT EXISTS markup_rate NUMERIC(4,2) NOT NULL DEFAULT 1.5,
    ADD COLUMN IF NOT EXISTS model_tier VARCHAR(20) NOT NULL DEFAULT 'flagship';
