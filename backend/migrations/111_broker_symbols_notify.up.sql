-- 111_broker_symbols_notify.up.sql
-- M10 ADR-0011 §2.3: PG NOTIFY on broker_symbols changes for normalizer cache invalidation.
CREATE OR REPLACE FUNCTION notify_broker_symbols_changed() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('broker_symbols_changed',
        json_build_object('broker', NEW.broker, 'symbol_raw', NEW.symbol_raw)::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_broker_symbols_changed ON broker_symbols;
CREATE TRIGGER trg_broker_symbols_changed
    AFTER INSERT OR UPDATE OR DELETE ON broker_symbols
    FOR EACH ROW EXECUTE FUNCTION notify_broker_symbols_changed();
