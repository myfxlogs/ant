-- 162_marketplace_strategy_fix.up.sql
-- Fix marketplace FK to point to strategy_templates (where Python code lives).
-- WARNING: drops marketplace rows that reference non-existent templates.
-- Add backtest snapshot + code snippet columns.

-- 0. Clean orphaned rows before FK change (safety: marketplace has no production data).
DELETE FROM marketplace_strategies
 WHERE strategy_id::text NOT IN (SELECT id::text FROM strategy_templates);

-- 1. marketplace_strategies FK: platform_strategies → strategy_templates
ALTER TABLE marketplace_strategies DROP CONSTRAINT IF EXISTS marketplace_strategies_strategy_id_fkey;
ALTER TABLE marketplace_strategies
  ADD CONSTRAINT marketplace_strategies_strategy_id_fkey
  FOREIGN KEY (strategy_id) REFERENCES strategy_templates(id) ON DELETE CASCADE;

-- 2. Add code_snippet + backtest_snapshot (JSONB)
ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS code_snippet TEXT;
ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS backtest_snapshot JSONB;

-- 3. user_strategy_publishes FK: platform_strategies → strategy_templates
DELETE FROM user_strategy_publishes
 WHERE platform_strategy_id::text NOT IN (SELECT id::text FROM strategy_templates);

ALTER TABLE user_strategy_publishes DROP CONSTRAINT IF EXISTS user_strategy_publishes_platform_strategy_id_fkey;
ALTER TABLE user_strategy_publishes
  ADD CONSTRAINT user_strategy_publishes_strategy_id_fkey
  FOREIGN KEY (platform_strategy_id) REFERENCES strategy_templates(id) ON DELETE CASCADE;
