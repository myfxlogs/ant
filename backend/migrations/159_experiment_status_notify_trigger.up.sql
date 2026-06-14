-- 159_experiment_status_notify_trigger.up.sql
-- Atomic PG NOTIFY on experiment status changes.
-- Replaces application-level fire-and-forget pg_notify calls,
-- guaranteeing notification delivery whenever status column is updated.

CREATE OR REPLACE FUNCTION notify_experiment_status() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('experiment_status', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER experiment_status_notify
    AFTER UPDATE OF status ON strategy_experiments
    FOR EACH ROW
    EXECUTE FUNCTION notify_experiment_status();
