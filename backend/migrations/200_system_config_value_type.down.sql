ALTER TABLE system_config DROP COLUMN IF EXISTS value_type;
DELETE FROM system_config WHERE key IN ('marketplace.platform_fee_rate', 'strategy.schedule.health_grading_config');
