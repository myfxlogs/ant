-- 205_hd_wallet_phase_a_cleanup.up.sql
-- ADR-0026 Phase A: Remove v1 hot-wallet model, switch to watch-only xpub derivation.

-- 1. Drop wallet_secrets table (online server no longer stores wallet private keys — R1).
DROP TABLE IF EXISTS wallet_secrets;

-- 2. Drop encrypted_privkey column from user_deposit_addresses.
ALTER TABLE user_deposit_addresses DROP COLUMN IF EXISTS encrypted_privkey;

-- 3. Remove v1 config items.
DELETE FROM system_config WHERE key = 'hot_wallet_address';
DELETE FROM system_config WHERE key = 'address_pool_min_threshold';

-- 4. Add SEQUENCE for on-demand derivation index allocation (Q1: no MAX(index)+1).
CREATE SEQUENCE IF NOT EXISTS deposit_addr_index_seq START 1 INCREMENT 1;

-- 5. Unique partial index: one ASSIGNED address per user (idempotent on-demand derivation).
CREATE UNIQUE INDEX IF NOT EXISTS idx_deposit_addr_one_per_user
    ON user_deposit_addresses(user_id) WHERE status = 'ASSIGNED';

-- 6. New config items for watch-only + cold signing model (A9).
INSERT INTO system_config (key, value, description, enabled)
VALUES ('deposit_xpub', '', 'Account-level extended public key (m/44\'/195\'/0\'/0) for watch-only address derivation', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('deposit_xpub_fingerprint', '', 'SHA-256 fingerprint of deposit_xpub for startup integrity verification (R5)', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('cold_wallet_address', '', 'Cold wallet TRC20 address for fund consolidation (sweep destination)', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('energy_account_address', '', 'Energy provider account TRC20 address for staking/delegation', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_alert_threshold', '1000', 'Single address balance highlight alert threshold USDT', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('dem_factor', '1.3', 'Dynamic Energy Multiplier factor for energy cost estimation', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('energy_buffer_percent', '10', 'Extra energy buffer percentage above estimated requirement', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('stake_trx_amount', '5000', 'TRX amount staked for energy delegation', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_threshold', '0.01', 'Minimum USDT balance to trigger sweep', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_batch_size', '10', 'Number of addresses per sweep batch', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('sweep_min_confirmations', '20', 'Minimum confirmations required for sweep transactions', true)
ON CONFLICT (key) DO NOTHING;
