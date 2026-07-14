DROP TABLE IF EXISTS deposit_requests;
DELETE FROM system_config WHERE key IN ('usdt_receiving_address', 'usdt_network', 'usdt_exchange_rate');
