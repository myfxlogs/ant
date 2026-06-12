-- 156_seed_system_strategies.down.sql

DELETE FROM strategy_templates WHERE is_system = true AND user_id IS NULL;
