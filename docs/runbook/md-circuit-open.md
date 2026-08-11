# Runbook · MdGatewayAccountDisconnected

> 关联告警：`MdGatewayAccountDisconnected`（`md_circuit_breaker_state == 1`，持续 30s，severity=warning）

## 症状

`md_circuit_breaker_state == 1` 持续 30 秒。某个 broker 的 circuit breaker 处于 open 状态。受影响账户的新订单被抑制（`ErrCircuitOpen`）。Grafana 显示该 broker 的行情中断。

## 影响

该 broker 下所有账户的行情数据和下单路径被熔断。策略收不到新 bar → 停止交易。`live_dispatch.go:364` 检测 `ErrCircuitOpen` 后跳过下单并标记 `circuit_open` 状态。

## 诊断步骤

```bash
# 1. 找出哪个 broker 的熔断器打开了
curl -s http://localhost:8080/metrics | grep 'md_circuit_breaker_state' | grep -v '0$'

# 2. 查后端日志中的熔断器状态变更
docker logs alphaforge-backend --since 10m | grep -iE "circuit.*open|breaker.*trip|circuit.*half"

# 3. 查触发熔断的原始错误
docker logs alphaforge-backend --since 15m | grep -iE "OnFailure|breaker.*failure|broker.*unreachable"

# 4. 检查受影响账户的 session 状态
curl -s http://localhost:8080/metrics | grep 'mthub_session_active'
```

## 应急处置

1. **等待自动恢复** → circuit breaker cooldown=30s，之后自动进入 half-open 探测。若探测成功（`successThreshold=2`），自动恢复 closed。通常无需手动干预。
2. **half-open 反复失败** → broker 持续不可达，查根因：
   ```bash
   docker logs alphaforge-backend --since 30m | grep -iE "dial.*refused|connection.*reset|broker.*timeout"
   ```
3. **需手动重连** → 重启受影响账户：
   ```bash
   curl -X POST http://localhost:8080/admin/v1/restart-account \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -d '{"account_id":"<ACCOUNT_ID>"}'
   ```

## 常见根因

- **broker 网络不可达**：`circuit_breaker.go` 中 `failureThreshold=5`，连续 5 次 RPC 失败后熔断。通常由网络抖动或 broker 服务器宕机引起
- **mtapi.io 代理故障**：gRPC 连接到 mtapi.io 超时/断开，触发 `OnFailure()`
- **broker 服务器维护**：MT broker 维护期间连接被关闭，连续失败触发熔断
- **DNS 解析失败**：mtapi.io 的 DNS 记录过期或 DNS 服务器故障
