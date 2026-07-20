-- 219_auto_generation_tasks_source_code.up.sql
-- Store AI-generated source code in the task row so it survives until approval.
-- Without this, the source code is lost between generation and publish.

ALTER TABLE auto_generation_tasks ADD COLUMN IF NOT EXISTS source_code TEXT;
