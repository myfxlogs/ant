-- 262_deposit_address_schema_fix.up.sql
-- ADR-0026 §2.3 schema correction: remove AVAILABLE status (addresses are always ASSIGNED on creation).
-- The code (deposit_address_repo.go) always INSERTs with status='ASSIGNED'; AVAILABLE was never used.

-- 1. Change DEFAULT from 'AVAILABLE' to 'ASSIGNED' (matches actual code behavior).
ALTER TABLE user_deposit_addresses
    ALTER COLUMN status SET DEFAULT 'ASSIGNED';

-- 2. Update any existing rows with status='AVAILABLE' to 'ASSIGNED' (should be zero or negligible).
UPDATE user_deposit_addresses SET status = 'ASSIGNED' WHERE status = 'AVAILABLE';

-- 3. Add CHECK constraint to enforce valid status values only.
ALTER TABLE user_deposit_addresses
    ADD CONSTRAINT chk_deposit_address_status
    CHECK (status IN ('ASSIGNED', 'RETIRED'));
