-- 064_agent_job_experiment_assets.down.sql
-- Auto-generated rollback for 064_agent_job_experiment_assets

-- Drop indexes
DROP INDEX IF EXISTS idx_agent_audit_logs_agent_token_id;
DROP INDEX IF EXISTS idx_agent_audit_logs_user_id_created_at;
DROP INDEX IF EXISTS idx_agent_tokens_status;
DROP INDEX IF EXISTS idx_agent_tokens_user_id;
DROP INDEX IF EXISTS idx_job_events_job_id_seq;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_user_id_created_at;
DROP INDEX IF EXISTS idx_jobs_user_kind_idempotency;
DROP INDEX IF EXISTS idx_market_regimes_user_symbol_created_at;
DROP INDEX IF EXISTS idx_strategy_asset_clones_asset_id;
DROP INDEX IF EXISTS idx_strategy_asset_clones_user_id;
DROP INDEX IF EXISTS idx_strategy_assets_owner_user_id;
DROP INDEX IF EXISTS idx_strategy_assets_visibility_review_status;
DROP INDEX IF EXISTS idx_strategy_experiment_candidates_experiment_id_rank;
DROP INDEX IF EXISTS idx_strategy_experiments_user_id_created_at;

-- Drop tables
DROP TABLE IF EXISTS strategy_experiments CASCADE;
DROP TABLE IF EXISTS strategy_experiment_candidates CASCADE;
DROP TABLE IF EXISTS strategy_assets CASCADE;
DROP TABLE IF EXISTS strategy_asset_clones CASCADE;
DROP TABLE IF EXISTS market_regimes CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
DROP TABLE IF EXISTS job_events CASCADE;
DROP TABLE IF EXISTS agent_tokens CASCADE;
DROP TABLE IF EXISTS agent_audit_logs CASCADE;

