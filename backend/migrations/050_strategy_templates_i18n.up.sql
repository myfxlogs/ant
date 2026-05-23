-- 050_strategy_templates_i18n.up.sql
-- 为策略模板添加 i18n JSONB 字段，用于存储多语言名称/描述/参数文案

ALTER TABLE strategy_templates
    ADD COLUMN IF NOT EXISTS i18n JSONB DEFAULT '{}'::JSONB;

COMMENT ON COLUMN strategy_templates.i18n IS '多语言内容（name/description/params 的 i18n 定义）';
