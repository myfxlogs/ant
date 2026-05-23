-- 091_orders_state_machine.up.sql
-- M2 OMS state machine — add state and broker_symbol_raw to orders table.
-- Backfills existing orders to SUBMITTED (conservative default).

-- 1. Add state column with check constraint
ALTER TABLE orders ADD COLUMN IF NOT EXISTS state VARCHAR(20) NOT NULL DEFAULT 'SUBMITTED';

-- 2. Add broker_symbol_raw for canonical → broker-native resolution trace
ALTER TABLE orders ADD COLUMN IF NOT EXISTS broker_symbol_raw VARCHAR(32);

-- 3. Add state index for querying by status
CREATE INDEX IF NOT EXISTS idx_orders_state ON orders(state);

-- 4. Backfill: all existing orders default to SUBMITTED (already handled by DEFAULT)
--    Future states: NEW, VALIDATED, RISK_APPROVED, SUBMITTED, WORKING,
--    PARTIALLY_FILLED, FILLED, CANCELLED, REJECTED, FAILED, EXPIRED
