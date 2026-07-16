-- Revert: convert notifications.data_proto BYTEA back to data_json TEXT
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS data_json TEXT;
UPDATE notifications SET data_json = convert_from(data_proto, 'UTF8') WHERE data_proto IS NOT NULL AND data_json IS NULL;
ALTER TABLE notifications DROP COLUMN IF EXISTS data_proto;
