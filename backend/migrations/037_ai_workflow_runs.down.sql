-- 037_ai_workflow_runs.down.sql
-- Auto-generated rollback for 037_ai_workflow_runs

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_workflow_runs_updated_at;
DROP INDEX IF EXISTS idx_ai_workflow_runs_user_id;
DROP INDEX IF EXISTS idx_ai_workflow_steps_created_at;
DROP INDEX IF EXISTS idx_ai_workflow_steps_run_id;

-- Drop tables
DROP TABLE IF EXISTS ai_workflow_steps CASCADE;
DROP TABLE IF EXISTS ai_workflow_runs CASCADE;

