-- ARCH-4 step⑥: persist magic_number on strategy_schedules for O(1) reverse lookup.
-- Used by ResolveScheduleIDByMagic to attribute live trades back to their schedule.
ALTER TABLE strategy_schedules ADD COLUMN IF NOT EXISTS magic_number INTEGER;

-- Unique constraint: one magic per account (FNV-1a 32-bit, ≤20 schedules/account → no collision).
-- Partial index: only enforce uniqueness for non-NULL magic_number (NULL = not yet computed).
CREATE UNIQUE INDEX IF NOT EXISTS idx_strategy_schedules_magic
  ON strategy_schedules(account_id, magic_number)
  WHERE magic_number IS NOT NULL;
