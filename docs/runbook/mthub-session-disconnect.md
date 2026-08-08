# Runbook · MtHubNoActiveSessions

> 占位（post-launch 补全）。关联告警：`MtHubNoActiveSessions`（`mthub_session_active == 0`，60s）。

## 含义
所有 MT 会话断开（无活跃 session）。可能 broker 全断、或平台与 mtapi 代理失联。

## 影响
**严重**——所有实盘策略停止收行情/下单。

## 初步处置
1. 确认是全断还是单账户：`/healthz` 看 NATS/PG，Grafana 看 session_active 趋势。
2. 查 mtapi.io 代理可达性 + MT broker 侧状态。
3. 重连机制应自动恢复（`auto-reconnect`）；若不恢复，手动重启受影响账户连接。

## TODO（post-launch）
重连失败排查、批量重连、用户通知流程。
