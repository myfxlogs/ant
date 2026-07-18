-- 204_hd_wallet_deposit.up.sql
-- ADR-0026: HD wallet deposit system — per-user addresses + auto-confirmation.

-- Address pool: offline-generated TRC20 addresses, atomically claimed by users.
CREATE TABLE IF NOT EXISTS user_deposit_addresses (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID REFERENCES users(id) ON DELETE SET NULL,
    address           VARCHAR(64) NOT NULL UNIQUE,
    derivation_index  INT NOT NULL UNIQUE,
    encrypted_privkey BYTEA NOT NULL,
    network           VARCHAR(16) NOT NULL DEFAULT 'TRC20',
    status            VARCHAR(16) NOT NULL DEFAULT 'AVAILABLE',
    has_received_usdt BOOLEAN NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at       TIMESTAMPTZ
);

CREATE INDEX idx_deposit_addresses_user_id ON user_deposit_addresses(user_id);
CREATE INDEX idx_deposit_addresses_address ON user_deposit_addresses(address);
CREATE INDEX idx_deposit_addresses_status ON user_deposit_addresses(status);

-- Deposits: on-chain facts, auto-confirmed by chain monitor.
CREATE TABLE IF NOT EXISTS deposits (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deposit_address_id  UUID NOT NULL REFERENCES user_deposit_addresses(id),
    tx_hash             VARCHAR(64) NOT NULL UNIQUE,
    amount              NUMERIC(20,8) NOT NULL,
    block_number        BIGINT NOT NULL,
    confirmations       INT NOT NULL DEFAULT 0,
    status              VARCHAR(16) NOT NULL DEFAULT 'CONFIRMED',
    confirmed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deposits_user_id ON deposits(user_id);
CREATE INDEX idx_deposits_address_id ON deposits(deposit_address_id);
CREATE INDEX idx_deposits_status ON deposits(status);

-- Sweep logs: single source of truth for sweep status.
CREATE TABLE IF NOT EXISTS sweep_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_address_id  UUID NOT NULL REFERENCES user_deposit_addresses(id),
    tx_hash             VARCHAR(64) UNIQUE,
    amount              NUMERIC(20,8) NOT NULL,
    energy_used         BIGINT NOT NULL DEFAULT 0,
    status              VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ
);

CREATE INDEX idx_sweep_logs_address_id ON sweep_logs(deposit_address_id);
CREATE INDEX idx_sweep_logs_status ON sweep_logs(status);

-- Hot wallet private key encrypted storage.
CREATE TABLE IF NOT EXISTS wallet_secrets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose         VARCHAR(32) NOT NULL,
    encrypted_data  BYTEA NOT NULL,
    key_version     INT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Chain monitor state: last scanned block for restart recovery.
INSERT INTO system_config (key, value, description, enabled)
VALUES ('last_scanned_block', '0', 'Last block scanned by chain monitor', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('min_confirmations', '20', 'Minimum block confirmations for deposit', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('min_deposit_amount', '1', 'Minimum deposit amount in USDT', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_contract_address', 'TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t', 'USDT TRC20 contract address', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('address_pool_min_threshold', '100', 'Alert when available addresses below this', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('dem_factor', '1.3', 'USDT Dynamic Energy Model factor', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('energy_buffer_percent', '10', 'Extra energy buffer percentage', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('hot_wallet_address', '', 'Platform hot wallet TRC20 address for fund consolidation', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_threshold', '1', 'Minimum USDT balance to trigger sweep', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_batch_size', '10', 'Max addresses to sweep per cycle', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_min_confirmations', '20', 'Min confirmations before sweep eligible', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('reconcile_alert_threshold', '10', 'USD diff threshold for reconciliation alert', true)
ON CONFLICT (key) DO NOTHING;
