-- 053_strategy_templates_is_system.up.sql
-- 给 strategy_templates 增加 is_system 标记：
--   * true  = 系统内置模板（由 seeder 维护，用户不能删除）
--   * false = 用户自建模板
-- 并回填已有"preset"标签的行。

ALTER TABLE strategy_templates
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

-- 回填：已有 tags 中带 'preset' 的视为系统模板。
UPDATE strategy_templates
SET is_system = TRUE
WHERE is_system = FALSE AND 'preset' = ANY(tags);

-- 索引用于 seeder 的查询 (user_id, name, is_system)。
CREATE INDEX IF NOT EXISTS idx_strategy_templates_user_system
    ON strategy_templates (user_id, is_system);

COMMENT ON COLUMN strategy_templates.is_system IS
    '系统内置模板标记。true 表示由 seeder 维护，不允许用户删除。';
