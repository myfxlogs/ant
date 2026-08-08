# Runbook · MdGatewayTickLatency

> 占位。告警：MdGatewayTickLatency（tick→bar 延迟过高）。

**含义**：从 MT tick 到达到 bar 聚合完成延迟过大。
**影响**：策略收到 bar 延迟 → 信号延迟 → 滑点。
**初步**：查 6 阶段管线各阶段耗时（Normalizer/Quality/Dedup/Aggregator/PgWriter/Publisher）→ 定位瓶颈阶段。
**TODO**：各阶段 SLO + 性能 profiling + 瓶颈优化。
