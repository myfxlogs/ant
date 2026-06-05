-- 142_jsonb_to_bytea_cleanup: complete the proto binary migration
-- Converts remaining JSONB columns to BYTEA that store proto binary.
-- Migrations 133/134/136/137 handled backtest_runs + experiments.
-- This migration covers strategy_templates, strategy_schedules, and ai_strategy_templates.

-- 1. strategy_templates.parameters JSONB → BYTEA (StrategyTemplate proto, repeated TemplateParameter)
ALTER TABLE strategy_templates ADD COLUMN parameters_proto BYTEA;
UPDATE strategy_templates SET parameters_proto = parameters::text::bytea WHERE parameters IS NOT NULL;
ALTER TABLE strategy_templates DROP COLUMN parameters;
ALTER TABLE strategy_templates RENAME COLUMN parameters_proto TO parameters;

-- 2. strategy_schedules.parameters JSONB → BYTEA (structpb.Struct proto)
ALTER TABLE strategy_schedules ADD COLUMN parameters_proto BYTEA;
UPDATE strategy_schedules SET parameters_proto = parameters::text::bytea WHERE parameters IS NOT NULL;
ALTER TABLE strategy_schedules DROP COLUMN parameters;
ALTER TABLE strategy_schedules RENAME COLUMN parameters_proto TO parameters;

-- 3. strategy_schedules.schedule_config JSONB → BYTEA (ScheduleConfig proto)
ALTER TABLE strategy_schedules ADD COLUMN schedule_config_proto BYTEA;
UPDATE strategy_schedules SET schedule_config_proto = schedule_config::text::bytea WHERE schedule_config IS NOT NULL;
ALTER TABLE strategy_schedules DROP COLUMN schedule_config;
ALTER TABLE strategy_schedules RENAME COLUMN schedule_config_proto TO schedule_config;

-- 4. strategy_schedules.backtest_metrics JSONB → BYTEA (BacktestMetrics proto)
ALTER TABLE strategy_schedules ADD COLUMN backtest_metrics_proto BYTEA;
UPDATE strategy_schedules SET backtest_metrics_proto = backtest_metrics::text::bytea WHERE backtest_metrics IS NOT NULL;
ALTER TABLE strategy_schedules DROP COLUMN backtest_metrics;
ALTER TABLE strategy_schedules RENAME COLUMN backtest_metrics_proto TO backtest_metrics;

-- 5. strategy_schedules.risk_reasons JSONB → BYTEA (structpb.ListValue proto)
ALTER TABLE strategy_schedules ADD COLUMN risk_reasons_proto BYTEA;
UPDATE strategy_schedules SET risk_reasons_proto = risk_reasons::text::bytea WHERE risk_reasons IS NOT NULL;
ALTER TABLE strategy_schedules DROP COLUMN risk_reasons;
ALTER TABLE strategy_schedules RENAME COLUMN risk_reasons_proto TO risk_reasons;

-- 6. strategy_schedules.risk_warnings JSONB → BYTEA (structpb.ListValue proto)
ALTER TABLE strategy_schedules ADD COLUMN risk_warnings_proto BYTEA;
UPDATE strategy_schedules SET risk_warnings_proto = risk_warnings::text::bytea WHERE risk_warnings IS NOT NULL;
ALTER TABLE strategy_schedules DROP COLUMN risk_warnings;
ALTER TABLE strategy_schedules RENAME COLUMN risk_warnings_proto TO risk_warnings;

-- 7. ai_strategy_templates.parameter_slots JSONB → BYTEA (ParameterSlots proto)
ALTER TABLE ai_strategy_templates ADD COLUMN parameter_slots_proto BYTEA;
UPDATE ai_strategy_templates SET parameter_slots_proto = parameter_slots::text::bytea WHERE parameter_slots IS NOT NULL;
ALTER TABLE ai_strategy_templates DROP COLUMN parameter_slots;
ALTER TABLE ai_strategy_templates RENAME COLUMN parameter_slots_proto TO parameter_slots;
