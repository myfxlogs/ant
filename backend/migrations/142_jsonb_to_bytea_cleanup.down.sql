-- 142_jsonb_to_bytea_cleanup down: revert BYTEA columns back to JSONB.

-- 1. strategy_templates.parameters
ALTER TABLE strategy_templates ADD COLUMN parameters_jsonb JSONB DEFAULT '[]';
UPDATE strategy_templates SET parameters_jsonb = convert_from(parameters, 'UTF8')::jsonb WHERE parameters IS NOT NULL;
ALTER TABLE strategy_templates DROP COLUMN parameters;
ALTER TABLE strategy_templates RENAME COLUMN parameters_jsonb TO parameters;

-- 2. strategy_schedules_v2.parameters
ALTER TABLE strategy_schedules_v2 ADD COLUMN parameters_jsonb JSONB DEFAULT '{}';
UPDATE strategy_schedules_v2 SET parameters_jsonb = convert_from(parameters, 'UTF8')::jsonb WHERE parameters IS NOT NULL;
ALTER TABLE strategy_schedules_v2 DROP COLUMN parameters;
ALTER TABLE strategy_schedules_v2 RENAME COLUMN parameters_jsonb TO parameters;

-- 3. strategy_schedules_v2.schedule_config
ALTER TABLE strategy_schedules_v2 ADD COLUMN schedule_config_jsonb JSONB DEFAULT '{}';
UPDATE strategy_schedules_v2 SET schedule_config_jsonb = convert_from(schedule_config, 'UTF8')::jsonb WHERE schedule_config IS NOT NULL;
ALTER TABLE strategy_schedules_v2 DROP COLUMN schedule_config;
ALTER TABLE strategy_schedules_v2 RENAME COLUMN schedule_config_jsonb TO schedule_config;

-- 4. strategy_schedules_v2.backtest_metrics
ALTER TABLE strategy_schedules_v2 ADD COLUMN backtest_metrics_jsonb JSONB;
UPDATE strategy_schedules_v2 SET backtest_metrics_jsonb = convert_from(backtest_metrics, 'UTF8')::jsonb WHERE backtest_metrics IS NOT NULL;
ALTER TABLE strategy_schedules_v2 DROP COLUMN backtest_metrics;
ALTER TABLE strategy_schedules_v2 RENAME COLUMN backtest_metrics_jsonb TO backtest_metrics;

-- 5. strategy_schedules_v2.risk_reasons
ALTER TABLE strategy_schedules_v2 ADD COLUMN risk_reasons_jsonb JSONB DEFAULT '[]';
UPDATE strategy_schedules_v2 SET risk_reasons_jsonb = convert_from(risk_reasons, 'UTF8')::jsonb WHERE risk_reasons IS NOT NULL;
ALTER TABLE strategy_schedules_v2 DROP COLUMN risk_reasons;
ALTER TABLE strategy_schedules_v2 RENAME COLUMN risk_reasons_jsonb TO risk_reasons;

-- 6. strategy_schedules_v2.risk_warnings
ALTER TABLE strategy_schedules_v2 ADD COLUMN risk_warnings_jsonb JSONB DEFAULT '[]';
UPDATE strategy_schedules_v2 SET risk_warnings_jsonb = convert_from(risk_warnings, 'UTF8')::jsonb WHERE risk_warnings IS NOT NULL;
ALTER TABLE strategy_schedules_v2 DROP COLUMN risk_warnings;
ALTER TABLE strategy_schedules_v2 RENAME COLUMN risk_warnings_jsonb TO risk_warnings;

-- 7. ai_strategy_templates.parameter_slots
ALTER TABLE ai_strategy_templates ADD COLUMN parameter_slots_jsonb JSONB DEFAULT '[]';
UPDATE ai_strategy_templates SET parameter_slots_jsonb = convert_from(parameter_slots, 'UTF8')::jsonb WHERE parameter_slots IS NOT NULL;
ALTER TABLE ai_strategy_templates DROP COLUMN parameter_slots;
ALTER TABLE ai_strategy_templates RENAME COLUMN parameter_slots_jsonb TO parameter_slots;
