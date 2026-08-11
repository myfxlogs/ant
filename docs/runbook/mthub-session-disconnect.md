# Runbook · MtHubNoActiveSessions

> 关联告警：`MtHubNoActiveSessions`（`mthub_session_active == 0`，持续 60s，severity=warning）

## 症状

`mthub_session_active == 0` 持续 60 秒。所有 MT 会话断开，无活跃 session。Grafana session_active 指标归零。

## 影响

**严重**——所有实盘策略停止收行情和下单。行情链路中断导致策略无 bar 输入，订单路径中断导致信号无法执行。

## 诊断步骤

```bash
# 1. 确认是全断还是单账户
curl -s http://localhost:8080/metrics | grep 'mthub_session_active'

# 2. 查后端日志中的 session 断开原因
docker logs alphaforge-backend --since 5m | grep -iE "session.*disconnect|session.*closed|login.*fail|auth.*fail"

# 3. 检查 mtapi.io 代理可达性
docker exec alphaforge-backend ping -c 5 mt4grpc.mtapi.io

# 4. 检查依赖健康
curl -s http://localhost:8080/healthz | jq '.'
curl -s http://localhost:8080/readyz | jq '.'
```

## 应急处置

1. **mtapi.io 不可达** → 网络故障或 mtapi.io 服务宕机。等待网络恢复后自动重连。
2. **密码/认证失败** → broker 密码可能已更改：
   ```bash
   docker logs alphaforge-backend --since 10m | grep -iE "login.*fail|auth.*fail|invalid.*password"
   ```
   需用户更新账户密码。
3. **自动重连未恢复** → 手动重启受影响账户连接：
   ```bash
   curl -X POST http://localhost:8080/admin/v1/restart-account \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -d '{"account_id":"<ACCOUNT_ID>"}'
   ```
4. **全 broker 断** → 可能是 broker 服务器维护/宕机，检查 broker 官方状态页。

## 常见根因

- **mtapi.io 代理故障**：mtapi.io 服务端宕机或网络不可达，所有 gRPC 连接断开
- **broker 服务器维护**：MT broker 定期维护，期间所有连接被关闭
- **账户密码过期/更改**：broker 端密码变更后旧密码登录失败
- **网络分区**：平台与 mtapi.io 之间的网络中断（DNS 解析失败/防火墙规则变更）
