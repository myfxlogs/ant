-- Reverse of 266: remove magic_number column and index.
DROP INDEX IF EXISTS idx_strategy_schedules_magic;
ALTER TABLE strategy_schedules DROP COLUMN IF EXISTS magic_number;
