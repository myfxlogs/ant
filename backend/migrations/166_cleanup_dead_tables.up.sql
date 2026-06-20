-- 166_cleanup_dead_tables.up.sql
-- Drop tables that are no longer referenced by any code and contain 0 rows.
-- These are legacy tables from superseded architectures.
-- Old migrations still reference them with IF NOT EXISTS, but this ensures
-- both existing and new deployments end up clean.

DROP TABLE IF EXISTS kline_data CASCADE;
DROP TABLE IF EXISTS kline_cache_metadata CASCADE;
DROP TABLE IF EXISTS strategy_schedules_legacy CASCADE;
DROP TABLE IF EXISTS strategy_schedule_runtime_states CASCADE;
DROP TABLE IF EXISTS strategy_models CASCADE;
DROP TABLE IF EXISTS strategy_symbols CASCADE;
DROP TABLE IF EXISTS market_quotes CASCADE;
DROP TABLE IF EXISTS platform_ai_agents CASCADE;
DROP TABLE IF EXISTS user_ai_agents CASCADE;
DROP TABLE IF EXISTS tenant_config CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
