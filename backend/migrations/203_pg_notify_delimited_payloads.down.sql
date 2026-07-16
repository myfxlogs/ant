-- Revert: restore JSON payloads for broker_symbols_changed NOTIFY
CREATE OR REPLACE FUNCTION notify_broker_symbols_changed() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('broker_symbols_changed',
        json_build_object('broker', NEW.broker, 'symbol_raw', NEW.symbol_raw)::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
