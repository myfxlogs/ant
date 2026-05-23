-- 090_strategies_canonical_migration.up.sql
-- M1-5: 历史 strategies.symbol → strategy_symbols 数据回填
-- 根据现有 strategies.symbol 裸字符串反向填充 strategy_symbols 表。
-- 注意：此脚本在 broker_symbols 已填充后才能产生有意义的结果；
-- 在 broker_symbols 为空时，canonical 列将为 NULL（需要后续手动回填）。

INSERT INTO strategy_symbols (strategy_id, canonical, enabled, created_at)
SELECT
    s.id AS strategy_id,
    s.symbol AS canonical,
    true AS enabled,
    s.created_at
FROM strategies s
WHERE s.symbol IS NOT NULL
  AND s.symbol != ''
  AND NOT EXISTS (
      SELECT 1 FROM strategy_symbols ss WHERE ss.strategy_id = s.id AND ss.canonical = s.symbol
  );
