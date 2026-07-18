-- 204_hd_wallet_deposit.down.sql

DROP TABLE IF EXISTS sweep_logs;
DROP TABLE IF EXISTS deposits;
DROP TABLE IF EXISTS wallet_secrets;
DROP TABLE IF EXISTS user_deposit_addresses;

DELETE FROM system_config WHERE key IN (
    'last_scanned_block',
    'min_confirmations',
    'min_deposit_amount',
    'usdt_contract_address',
    'address_pool_min_threshold',
    'dem_factor',
    'energy_buffer_percent',
    'hot_wallet_address',
    'sweep_threshold',
    'sweep_batch_size',
    'sweep_min_confirmations',
    'reconcile_alert_threshold'
);
