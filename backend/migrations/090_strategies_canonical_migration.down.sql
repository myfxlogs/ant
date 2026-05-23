-- 090_strategies_canonical_migration.down.sql
-- M1-5: 回滚数据迁移 — 删除通过 090 up 插入的 strategy_symbols 记录。
-- 安全起见，仅删除 created_at 与 migration 时间窗口匹配的记录（非精确回滚，记录级）。
-- 如需完全回滚，DROP strategy_symbols 即可（已在 089 down 中处理）。
DELETE FROM strategy_symbols
WHERE created_at >= now() - interval '1 hour';
