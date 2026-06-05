-- Revert: BYTEA → JSONB
ALTER TABLE backtest_runs ADD COLUMN parameter_overrides_jsonb JSONB DEFAULT '{}';
UPDATE backtest_runs SET parameter_overrides_jsonb = parameter_overrides::text::jsonb WHERE parameter_overrides IS NOT NULL;
ALTER TABLE backtest_runs DROP COLUMN parameter_overrides;
ALTER TABLE backtest_runs RENAME COLUMN parameter_overrides_jsonb TO parameter_overrides;

ALTER TABLE strategy_experiments ADD COLUMN parameter_space_jsonb JSONB DEFAULT '{}';
ALTER TABLE strategy_experiments DROP COLUMN parameter_space;
ALTER TABLE strategy_experiments RENAME COLUMN parameter_space_jsonb TO parameter_space;

ALTER TABLE strategy_experiment_candidates ADD COLUMN parameters_jsonb JSONB DEFAULT '{}';
ALTER TABLE strategy_experiment_candidates DROP COLUMN parameters;
ALTER TABLE strategy_experiment_candidates RENAME COLUMN parameters_jsonb TO parameters;

ALTER TABLE strategy_experiment_candidates ADD COLUMN score_components_jsonb JSONB DEFAULT '{}';
ALTER TABLE strategy_experiment_candidates DROP COLUMN score_components;
ALTER TABLE strategy_experiment_candidates RENAME COLUMN score_components_jsonb TO score_components;
