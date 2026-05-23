-- 091_orders_state_machine.down.sql
-- Rollback M2 OMS state machine additions.
ALTER TABLE orders DROP COLUMN IF EXISTS broker_symbol_raw;
ALTER TABLE orders DROP COLUMN IF EXISTS state;
DROP INDEX IF EXISTS idx_orders_state;
