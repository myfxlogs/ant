# ADR-0003 PostgreSQL + Redis 主存方案

## Status
Accepted (2026-05-23)

## Context
ant 需要：① 持久化（用户、账户、策略、订单、回测）；② 短时缓存（行情、会话）；③ 分布式锁（策略调度去重）；④ 时序数据（K 线、tick）。是否引入 ClickHouse / TimescaleDB / InfluxDB？

## Options
- **A. PG 18 + Redis 8**（已选）：主数据 PG，缓存/锁 Redis；时序数据用 PG 表 + 索引（暂不引入专用时序库）
- B. + ClickHouse：时序查询强，但单机部署形态下复杂度高
- C. + TimescaleDB：PG 扩展，最小化新栈，但增加维护成本

## Decision
**A**。M0-M4 阶段用 PG 单库 + Redis；时序需求暂以 PG 索引 + 分区表满足。**触发器**：当行情/tick 表 ≥ 1 亿行或 P95 查询 > 500ms 时，立 ADR 评估时序方案。

## Consequences
- **+** 单机 compose 部署形态保持；运维简单
- **+** 数据一致性约束在单库内可强制
- **−** 时序数据扩展性受限（触发器再决策）
- **−** 不利于跨地域部署（短期不需要）

## Related
- AGENT.md 数据规则
- AlfQ 迁移计划 §M1（broker_symbols / canonical_symbols 全在 PG）

## History
- 2026-05-23 Accepted
