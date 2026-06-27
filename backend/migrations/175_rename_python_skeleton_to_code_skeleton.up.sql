-- 175_rename_python_skeleton_to_code_skeleton.up.sql
-- Rename legacy column name from Python era to reflect Go-native strategy code.

ALTER TABLE ai_strategy_templates
  RENAME COLUMN python_skeleton TO code_skeleton;
