-- 207_sweep_unsigned_bundle.up.sql
-- ADR-0026 Phase C fix: persist unsigned bundles for crash recovery before cold signing.

-- Add unsigned_bundle column to sweep_bundles (nullable — only set before signing).
ALTER TABLE sweep_bundles ADD COLUMN IF NOT EXISTS unsigned_bundle BYTEA;

-- Add PENDING_SIGN status support (existing bundles default to BROADCASTING).
-- Statuses: PENDING_SIGN / BROADCASTING / DONE / EXPIRED

-- Add deposit_address_id to sweep_bundles for recovery lookup.
ALTER TABLE sweep_bundles ADD COLUMN IF NOT EXISTS deposit_address_id UUID;
