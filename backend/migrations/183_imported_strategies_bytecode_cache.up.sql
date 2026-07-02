-- 183_imported_strategies_bytecode_cache.up.sql
-- Cache compiled Bytecode alongside MQL source to avoid recompilation on every
-- backtest/live execution. The cache is invalidated when source_code changes
-- (re-import clears bytecode_cache). ADR-0023 §8 P3.

ALTER TABLE imported_strategies
  ADD COLUMN IF NOT EXISTS bytecode_cache BYTEA;

COMMENT ON COLUMN imported_strategies.bytecode_cache IS 'Compiled MQL Bytecode (binary format). NULL = not yet compiled or cache invalidated.';
