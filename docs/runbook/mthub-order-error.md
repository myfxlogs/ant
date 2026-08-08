# Runbook · MtHubOrderErrorRateHigh

> 占位（post-launch 补全）。关联告警：`MtHubOrderErrorRateHigh`（`mthub_orders_placed_total{status="err"}` >0.1/s，2m）。

## 含义
下单路径出错（非拒绝，是系统错误——mtapi RPC 失败、连接断、超时）。

## 影响
订单可能未达 broker 或状态不明 → 需对账（mthub 幂等门/对账门介入）。

## 初步处置
1. `docker logs alphaforge-backend | grep -iE "order.*err|mtapi"` 看错误类型。
2. 查 MT 连接健康（`OnQuote` 流是否在、session 是否断）。
3. 触发熔断（circuit breaker）则查 broker 侧。

## TODO（post-launch）
错误分类（mtapi/网络/OMS）、对账流程、补偿动作。
