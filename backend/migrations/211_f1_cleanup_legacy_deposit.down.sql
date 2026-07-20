-- 211_f1_cleanup_legacy_deposit.down.sql

-- Restore currency default and values.
ALTER TABLE user_wallets ALTER COLUMN currency SET DEFAULT 'USD';
UPDATE user_wallets SET currency = 'USD' WHERE currency = 'USDT';

-- Restore obsolete config items.
INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_receiving_address', '', 'USDT (TRC20) receiving address for user deposits', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_network', 'TRC20', 'USDT network for deposits (TRC20/ERC20)', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled)
VALUES ('usdt_exchange_rate', '1', 'USD amount credited per 1 USDT (default 1:1)', true)
ON CONFLICT (key) DO NOTHING;

-- Note: deposit_requests table not restored (data was already empty in production).
