-- 183_imported_strategies_bytecode_cache.down.sql
ALTER TABLE imported_strategies DROP COLUMN IF EXISTS bytecode_cache;
