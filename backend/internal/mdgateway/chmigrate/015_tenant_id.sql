-- 015_tenant_id: add tenant_id column to all mdgateway tables (M10-BASE-A4).
-- Enables multi-tenant partitioning without requiring separate databases.
-- Each tenant's data is partitioned independently within the same cluster.

ALTER TABLE ant.md_ticks ADD COLUMN IF NOT EXISTS tenant_id String DEFAULT 'default';
ALTER TABLE ant.md_bars ADD COLUMN IF NOT EXISTS tenant_id String DEFAULT 'default';
ALTER TABLE ant.factor_values ADD COLUMN IF NOT EXISTS tenant_id String DEFAULT 'default';
ALTER TABLE ant.signals ADD COLUMN IF NOT EXISTS tenant_id String DEFAULT 'default';
ALTER TABLE ant.md_ticks_dlq ADD COLUMN IF NOT EXISTS tenant_id String DEFAULT 'default';
