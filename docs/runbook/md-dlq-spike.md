# Runbook · MdGatewayDLQSpike

> 占位。告警：MdGatewayDLQSpike（DLQ drop 量飙升）。

**含义**：行情 tick/bar 被丢弃的量异常高（PgWriter channel 满 / PG 故障 / quality drop）。
**影响**：K线数据缺口 → 回测/实盘数据不全 → Backfiller 会补但延迟。
**初步**：查 `md_dlq_sampled_total{reason}` → reason 分类（channel_full/pg_error/quality_drop）→ 对应处置（PG 扩容 / 降吞吐 / 修 quality 规则）。
**TODO**：drop 阈值 + Backfiller 恢复验证 + 数据完整性检查。
