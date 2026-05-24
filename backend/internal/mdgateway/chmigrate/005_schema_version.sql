-- 005_schema_version.sql
-- Migration version tracking for chmigrate idempotent execution.

CREATE TABLE IF NOT EXISTS _schema_migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3, 'UTC') DEFAULT now64(3),
    checksum   String
) ENGINE = ReplacingMergeTree(applied_at)
ORDER BY version;
