-- 200_system_config_value_type.up.sql
-- Add value_type column so the frontend can render the correct editor
-- (text, json, number) without hardcoding key strings.
-- Also seed missing config rows that code expects to exist.

ALTER TABLE system_config
    ADD COLUMN IF NOT EXISTS value_type VARCHAR(20) NOT NULL DEFAULT 'text';

-- Mark known JSON configs.
UPDATE system_config SET value_type = 'json'
WHERE key IN (
    'ai.provider_catalog',
    'econ.translation.ai_config',
    'strategy.schedule.health_grading_config'
);

-- Mark numeric configs.
UPDATE system_config SET value_type = 'number'
WHERE key IN (
    'usdt_exchange_rate',
    'max_accounts_per_user',
    'max_positions_per_account',
    'default_leverage',
    'min_lot_size'
);

-- Seed missing rows that code queries but were never inserted by migrations.
INSERT INTO system_config (key, value, description, enabled, admin_visible, value_type)
VALUES
    ('marketplace.platform_fee_rate', '0', 'Platform fee rate for marketplace sales (0-100)', true, true, 'number')
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_config (key, value, description, enabled, admin_visible, value_type)
VALUES
    ('strategy.schedule.health_grading_config',
     '{"green_success_rate":90,"green_max_failed_runs":1,"yellow_success_rate":60,"min_sample_size":1}',
     'Strategy health grading thresholds',
     true, true, 'json')
ON CONFLICT (key) DO NOTHING;
