-- 213_marketplace_quality_gates.down.sql
DROP TABLE IF EXISTS marketplace_quality_waivers CASCADE;

DELETE FROM system_config WHERE key LIKE 'marketplace.quality.%';
