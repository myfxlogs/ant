-- 134_strategy_params_proto: convert JSONB columns to BYTEA proto binary
-- Replaces parameter_overrides, parameter_space, parameters, score_components JSONB → BYTEA

-- 1. backtest_runs: parameter_overrides JSONB → BYTEA (StrategyParams proto)
ALTER TABLE backtest_runs ADD COLUMN parameter_overrides_proto BYTEA;
UPDATE backtest_runs SET parameter_overrides_proto = parameter_overrides::text::bytea WHERE parameter_overrides IS NOT NULL;
ALTER TABLE backtest_runs DROP COLUMN parameter_overrides;
ALTER TABLE backtest_runs RENAME COLUMN parameter_overrides_proto TO parameter_overrides;

-- 2. strategy_experiments: parameter_space JSONB → BYTEA (ParameterSpace proto)
ALTER TABLE strategy_experiments ADD COLUMN parameter_space_proto BYTEA;
ALTER TABLE strategy_experiments DROP COLUMN parameter_space;
ALTER TABLE strategy_experiments RENAME COLUMN parameter_space_proto TO parameter_space;

-- 3. strategy_experiment_candidates: parameters JSONB → BYTEA (CandidateParameters proto)
ALTER TABLE strategy_experiment_candidates ADD COLUMN parameters_proto BYTEA;
ALTER TABLE strategy_experiment_candidates DROP COLUMN parameters;
ALTER TABLE strategy_experiment_candidates RENAME COLUMN parameters_proto TO parameters;

-- 4. strategy_experiment_candidates: score_components JSONB → BYTEA (ScoreComponents proto)
ALTER TABLE strategy_experiment_candidates ADD COLUMN score_components_proto BYTEA;
ALTER TABLE strategy_experiment_candidates DROP COLUMN score_components;
ALTER TABLE strategy_experiment_candidates RENAME COLUMN score_components_proto TO score_components;
