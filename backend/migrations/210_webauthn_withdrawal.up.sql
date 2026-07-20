-- Phase E: WebAuthn withdrawal authorization (R11 R12)
-- Tables: webauthn_credentials, withdrawal_requests, withdrawal_whitelist, credential_change_log

-- 1. WebAuthn credentials — user passkey public keys.
-- The online server stores these for registration ceremony.
-- coldsign maintains its own copy (synced via USB, Q2/R12) — it does NOT trust this table.
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   TEXT NOT NULL UNIQUE,       -- WebAuthn credential ID (base64url)
    public_key      BYTEA NOT NULL,              -- COSE-encoded public key
    attestation_type TEXT NOT NULL DEFAULT 'none',
    aaguid          TEXT NOT NULL DEFAULT '',
    sign_count      BIGINT NOT NULL DEFAULT 0,   -- replay protection (clone detection)
    transports      TEXT[] NOT NULL DEFAULT '{}',
    name            TEXT NOT NULL DEFAULT '',    -- user-assigned label
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, credential_id)
);

CREATE INDEX idx_webauthn_credentials_user ON webauthn_credentials(user_id);

-- 2. Withdrawal requests — tracks each withdrawal through its lifecycle.
-- PENDING → SIGNED → BROADCASTING → DONE / FAILED / CANCELLED
CREATE TABLE IF NOT EXISTS withdrawal_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount          NUMERIC(20,8) NOT NULL,      -- USDT amount
    dest_address    TEXT NOT NULL,                -- TRC20 destination
    nonce           BIGINT NOT NULL,              -- unique per withdrawal (replay protection)
    credential_id   TEXT,                         -- which passkey signed (set at FinishWithdrawal)
    assertion       BYTEA,                        -- WebAuthn assertion blob (set at FinishWithdrawal)
    status          TEXT NOT NULL DEFAULT 'PENDING',
                    -- PENDING / SIGNED / BROADCASTING / DONE / FAILED / CANCELLED
    bundle_id       UUID,                         -- links to sweep_bundles when built as TransferTx
    tx_hash         TEXT,                         -- on-chain tx hash after broadcast
    idem_key        TEXT NOT NULL UNIQUE,          -- idempotency (R7): "withdrawal-{id}"
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_withdrawal_requests_user ON withdrawal_requests(user_id);
CREATE INDEX idx_withdrawal_requests_status ON withdrawal_requests(status);
CREATE UNIQUE INDEX idx_withdrawal_requests_nonce_user ON withdrawal_requests(user_id, nonce);

-- 3. Withdrawal whitelist — user-approved destination addresses (R12).
-- Changes require 2FA + cooldown + hash chain ledger entry.
CREATE TABLE IF NOT EXISTS withdrawal_whitelist (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address         TEXT NOT NULL,                -- TRC20 Base58
    label           TEXT NOT NULL DEFAULT '',     -- user-assigned name
    status          TEXT NOT NULL DEFAULT 'PENDING_CONFIRMATION',
                    -- PENDING_CONFIRMATION / ACTIVE / REMOVED
    confirmed_at    TIMESTAMPTZ,                  -- set after 2FA + cooldown
    cooldown_until  TIMESTAMPTZ,                  -- 24h cooldown before activation
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, address)
);

CREATE INDEX idx_withdrawal_whitelist_user ON withdrawal_whitelist(user_id);
CREATE INDEX idx_withdrawal_whitelist_active ON withdrawal_whitelist(user_id, status) WHERE status = 'ACTIVE';

-- 4. Credential change log — tracks all credential/whitelist mutations (R12).
-- Each change is written to the hash chain via wallet_transactions (idem_key pattern).
-- This enables coldsign to detect unauthorized changes by comparing with its mirror.
CREATE TABLE IF NOT EXISTS credential_change_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    change_type     TEXT NOT NULL,                -- CREDENTIAL_ADD / CREDENTIAL_REMOVE / WHITELIST_ADD / WHITELIST_REMOVE
    target_id       TEXT NOT NULL,                -- credential_id or whitelist address
    status          TEXT NOT NULL DEFAULT 'PENDING',  -- PENDING / CONFIRMED / EXPIRED
    idem_key        TEXT NOT NULL UNIQUE,          -- links to wallet_transactions hash chain
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at    TIMESTAMPTZ
);

CREATE INDEX idx_credential_change_log_user ON credential_change_log(user_id);
CREATE INDEX idx_credential_change_log_pending ON credential_change_log(status) WHERE status = 'PENDING';
