# Runbook · MtHubOrderErrorRateHigh

> 关联告警：`MtHubOrderErrorRateHigh`（`mthub_orders_placed_total{status="err"}` rate >0.1/s，持续 2m，severity=critical）

## 症状

Prometheus 查询 `sum(rate(mthub_orders_placed_total{status="err"}[5m])) > 0.1` 持续触发。后端日志出现 `order submission failed` 或 `mtapi` RPC 错误。策略信号产生但订单未落 broker。

## 影响

订单未达 broker 或状态不明。实盘策略空转（有信号无成交），实盘与回测发散。若伴随 circuit breaker 打开，该 broker 下所有账户下单被抑制。

## 诊断步骤

```bash
# 1. 确认 error 来源（哪个账户/broker）
curl -s http://localhost:8080/metrics | grep 'mthub_orders_placed_total{status="err"}'

# 2. 看后端日志中的订单错误详情
docker logs alphaforge-backend --since 10m | grep -iE "order.*submission.*failed|mtapi.*err|OrderSend.*err"

# 3. 检查 circuit breaker 状态（error 可能已触发熔断）
curl -s http://localhost:8080/metrics | grep 'md_circuit_breaker_state'

# 4. 检查 MT session 是否存活
curl -s http://localhost:8080/metrics | grep 'mthub_session_active'
```

## 应急处置

1. **circuit breaker 已打开** → 该 broker 被熔断，等 30s cooldown 后 half-open 自动探测恢复。无需手动干预。
2. **mtapi RPC 超时/连接断** → 检查 mtapi.io 代理可达性：
   ```bash
   docker logs alphaforge-backend --since 5m | grep -iE "dial.*refused|connection.*closed|rpc.*error"
   ```
   若 mtapi 代理不可达，等待自动重连；若持续不恢复，重启受影响账户连接。
3. **单账户持续 error** → 重启该账户 gateway：
   ```bash
   # 通过 admin API 重启单账户（不影响其他账户）
   curl -X POST http://localhost:8080/admin/v1/restart-account \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -d '{"account_id":"<ACCOUNT_ID>"}'
   ```
4. **全 broker error** → 可能 mtapi.io 代理故障，检查 mtapi.io 服务状态页或网络。

## 常见根因

- **mtapi.io 代理不可达**：网络抖动或 mtapi.io 服务端故障，gRPC 连接断开
- **broker 服务器维护**：MT broker 定期维护，期间所有 RPC 返回错误
- **circuit breaker 连锁**：`circuit_breaker.go` 中 failureThreshold=5 连续失败后熔断，`live_dispatch.go:364` 检测 `ErrCircuitOpen` 并跳过下单
- **账户 session 过期**：MT session 断开后 `mthub_session_active=0`，后续订单全部 error
