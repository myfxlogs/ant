-- 144_pg_notify_delimited_payloads: replace JSON payloads with simple delimited strings.
-- PG NOTIFY is a signal mechanism, not a data channel. Payloads should be simple strings.

-- broker_symbols_changed: "broker|symbol_raw" instead of JSON
CREATE OR REPLACE FUNCTION notify_broker_symbols_changed() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('broker_symbols_changed',
        COALESCE(NEW.broker, '') || '|' || COALESCE(NEW.symbol_raw, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
