-- 240: Add bundle_id to marketplace_settlements for bundle purchase tracking.
-- When a bundle is purchased, the settlement row records the bundle_id so
-- that refund logic can find the settlement even when refunding a subscription
-- that is not the first one in the bundle.
ALTER TABLE marketplace_settlements
    ADD COLUMN IF NOT EXISTS bundle_id UUID;

CREATE INDEX IF NOT EXISTS idx_settlements_bundle ON marketplace_settlements(bundle_id) WHERE bundle_id IS NOT NULL;
