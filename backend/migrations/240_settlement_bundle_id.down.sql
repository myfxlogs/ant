DROP INDEX IF EXISTS idx_settlements_bundle;
ALTER TABLE marketplace_settlements DROP COLUMN IF EXISTS bundle_id;
