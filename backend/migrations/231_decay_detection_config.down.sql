-- 231_decay_detection_config.down.sql
DELETE FROM system_config WHERE key IN (
    'marketplace.decay.sharpe_decline_threshold',
    'marketplace.decay.winrate_decline_threshold',
    'marketplace.decay.lookback_days',
    'marketplace.decay.min_live_days'
);
