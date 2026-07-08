-- 193 down: Remove turn_data column.
ALTER TABLE ai_messages DROP COLUMN IF EXISTS turn_data;
