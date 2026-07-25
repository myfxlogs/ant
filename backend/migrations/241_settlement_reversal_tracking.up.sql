-- 241: Add reversal tracking to marketplace_settlements for failed refund reversals.
-- When a refund reversal (debiting publisher/platform) fails due to insufficient
-- balance, the settlement row is marked so it can be reconciled later.
ALTER TABLE marketplace_settlements
    ADD COLUMN IF NOT EXISTS reversal_failed BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS reversal_failure_note TEXT;
