-- 159_experiment_status_notify_trigger.down.sql
DROP TRIGGER IF EXISTS experiment_status_notify ON strategy_experiments;
DROP FUNCTION IF EXISTS notify_experiment_status();
