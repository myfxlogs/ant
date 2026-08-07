-- 262_deposit_address_schema_fix.down.sql
-- Revert: restore AVAILABLE default, remove CHECK constraint.

ALTER TABLE user_deposit_addresses
    DROP CONSTRAINT IF EXISTS chk_deposit_address_status;

ALTER TABLE user_deposit_addresses
    ALTER COLUMN status SET DEFAULT 'AVAILABLE';
