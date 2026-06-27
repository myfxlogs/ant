-- 175_rename_python_skeleton_to_code_skeleton.down.sql

ALTER TABLE ai_strategy_templates
  RENAME COLUMN code_skeleton TO python_skeleton;
