-- 205_hd_wallet_phase_a_cleanup.down.sql

-- Remove new config items.
DELETE FROM system_config WHERE key = 'deposit_xpub';
DELETE FROM system_config WHERE key = 'deposit_xpub_fingerprint';
DELETE FROM system_config WHERE key = 'cold_wallet_address';
DELETE FROM system_config WHERE key = 'energy_account_address';
DELETE FROM system_config WHERE key = 'sweep_alert_threshold';
DELETE FROM system_config WHERE key = 'dem_factor';
DELETE FROM system_config WHERE key = 'energy_buffer_percent';
DELETE FROM system_config WHERE key = 'stake_trx_amount';
DELETE FROM system_config WHERE key = 'sweep_threshold';
DELETE FROM system_config WHERE key = 'sweep_batch_size';
DELETE FROM system_config WHERE key = 'sweep_min_confirmations';

-- Drop unique partial index.
DROP INDEX IF EXISTS idx_deposit_addr_one_per_user;

-- Drop SEQUENCE.
DROP SEQUENCE IF EXISTS deposit_addr_index_seq;

-- Restore v1 config items.
INSERT INTO system_config (key, value, description, enabled)
VALUES ('hot_wallet_address', '', 'Platform hot wallet TRC20 address for fund consolidation', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('address_pool_min_threshold', '100', 'Alert when available addresses below this', true)
ON CONFLICT (key) DO NOTHING;

-- Restore encrypted_privkey column.
ALTER TABLE user_deposit_addresses ADD COLUMN IF NOT EXISTS encrypted_privkey BYTEA;

-- Restore wallet_secrets table.
CREATE TABLE IF NOT EXISTS wallet_secrets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose         VARCHAR(32) NOT NULL,
    encrypted_data  BYTEA NOT NULL,
    key_version     INT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
