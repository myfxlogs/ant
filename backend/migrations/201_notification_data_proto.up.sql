-- 201_notification_data_proto: convert notifications.data_json TEXT to data_proto BYTEA
-- Stores google.protobuf.Struct serialized as proto binary instead of JSON text.

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS data_proto BYTEA;

-- Migrate existing rows: store raw JSON bytes temporarily.
-- The Go startup migration (MigrateNotificationDataProto) will
-- convert any remaining JSON to proto binary on first boot.
-- For new rows, the application writes proto binary directly.
UPDATE notifications SET data_proto = convert_to(data_json, 'UTF8') WHERE data_json IS NOT NULL AND data_json != '{}' AND data_proto IS NULL;

ALTER TABLE notifications DROP COLUMN IF EXISTS data_json;
