-- 211_f1_cleanup_legacy_deposit.up.sql
-- F1: Drop legacy v1 manual deposit system (deposit_requests table + obsolete configs).
-- The HD wallet system (deposits table + on-demand derivation) replaces this entirely.

-- 1. Drop legacy deposit_requests table (v1 manual deposit approval flow).
DROP TABLE IF EXISTS deposit_requests;

-- 2. Remove obsolete v1 config items (no Go code references these).
DELETE FROM system_config WHERE key = 'usdt_receiving_address';
DELETE FROM system_config WHERE key = 'usdt_network';
DELETE FROM system_config WHERE key = 'usdt_exchange_rate';

-- 3. Update wallet currency label from 'USD' to 'USDT' (F2: display honesty).
--    The internal ledger has always tracked USDT amounts, not fiat USD.
UPDATE user_wallets SET currency = 'USDT' WHERE currency = 'USD';
ALTER TABLE user_wallets ALTER COLUMN currency SET DEFAULT 'USDT';
