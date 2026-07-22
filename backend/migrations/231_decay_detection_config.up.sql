-- 231_decay_detection_config.up.sql
-- Phase 5.1a: Seed decay detection thresholds into system_config.

INSERT INTO system_config (key, value, value_type, description, enabled, admin_visible) VALUES
    ('marketplace.decay.sharpe_decline_threshold', '0.3',  'decimal', '夏普下滑超过此值触发衰减告警 (0-1)', true, true),
    ('marketplace.decay.winrate_decline_threshold', '0.15', 'decimal', '胜率下滑超过此值触发衰减告警 (0-1)', true, true),
    ('marketplace.decay.lookback_days',             '30',   'int',     '衰减检测回溯天数 (recent window)', true, true),
    ('marketplace.decay.min_live_days',             '30',   'int',     '最少实盘天数才启动检测', true, true)
ON CONFLICT (key) DO NOTHING;
