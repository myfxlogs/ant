-- 207_sweep_unsigned_bundle.down.sql
ALTER TABLE sweep_bundles DROP COLUMN IF EXISTS unsigned_bundle;
ALTER TABLE sweep_bundles DROP COLUMN IF EXISTS deposit_address_id;
