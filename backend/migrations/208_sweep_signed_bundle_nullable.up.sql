-- 208_sweep_signed_bundle_nullable.up.sql
-- Fix: signed_bundle was NOT NULL but PENDING_SIGN bundles have no signed data yet.
-- Make it nullable so SaveUnsignedBundle can insert without a signed bundle.
ALTER TABLE sweep_bundles ALTER COLUMN signed_bundle DROP NOT NULL;
