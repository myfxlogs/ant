-- 208_sweep_signed_bundle_nullable.down.sql
ALTER TABLE sweep_bundles ALTER COLUMN signed_bundle SET NOT NULL;
