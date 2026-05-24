-- 111_broker_symbols_notify.down.sql
DROP TRIGGER IF EXISTS trg_broker_symbols_changed ON broker_symbols;
DROP FUNCTION IF EXISTS notify_broker_symbols_changed();
