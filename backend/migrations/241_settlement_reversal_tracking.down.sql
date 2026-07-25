ALTER TABLE marketplace_settlements
    DROP COLUMN IF EXISTS reversal_failure_note,
    DROP COLUMN IF EXISTS reversal_failed;
