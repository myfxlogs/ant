# Runbook · MdGatewayTickDropRateHigh

> 关联告警：`MdGatewayTickDropRateHigh`（`md_dlq_sampled_total{reason="parse_error"}` rate >1/s，持续 2m，severity=warning）

## 症状

`rate(md_dlq_sampled_total{reason="parse_error"}[5m]) > 1` 持续触发。行情 tick 被丢弃到 DLQ（Dead Letter Queue）的速率超过每秒 1 条。

## 影响

K线数据出现缺口。回测/实盘数据不完整。Backfiller 会延迟补全，但期间策略可能收到不完整的 bar。

## 诊断步骤

```bash
# 1. 按 reason 分类查看 DLQ 丢弃量
curl -s http://localhost:8080/metrics | grep 'md_dlq_sampled_total'

# 2. 查后端日志中的 DLQ 丢弃详情
docker logs alphaforge-backend --since 10m | grep -iE "dlq|drop.*tick|parse.*error|tick.*dropped"

# 3. 确认是哪个 broker/symbol 的数据有问题
docker logs alphaforge-backend --since 10m | grep -iE "parse.*error|invalid.*tick|malformed" | head -20

# 4. 检查 PG 写入是否正常（channel_full reason 会指向 PG 瓶颈）
curl -s http://localhost:8080/metrics | grep -E 'md_pg_write_errors_total|md_pg_writer_queue_depth'
```

## 应急处置

1. **`reason="parse_error"`** → broker 数据格式异常：
   - 单 symbol 持续 → broker 该品种数据脏，考虑临时禁用该 symbol
   - 全 broker 多 symbol → mtapi.io 代理可能损坏数据包，检查 mtapi.io 状态
2. **`reason="channel_full"`** → PgWriter channel 积压：
   - 检查 PG 写入性能：`docker exec alphaforge-postgres psql -U ant -d ant -c "SELECT pg_size_pretty(pg_database_size('ant'))"`
   - 临时：重启 backend 释放积压
3. **`reason="quality_drop"`** → bid > ask 等质量检查失败：
   - 参考 `mt-incidents.md` §6 处理脏数据策略

## 常见根因

- **broker 数据脏**：bid > ask / 价格为 0 / 精度异常，被 quality gate 丢弃
- **PG 写入慢**：PgWriter channel 满（`channel_full`），通常是 PG 磁盘满或查询慢导致写入跟不上
- **mtapi.io 数据包损坏**：代理层传输错误导致 tick 格式异常
- **broker 维护期间异常数据**：维护开始/结束时 broker 可能发送格式异常的 tick
