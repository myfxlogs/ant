# Runbook · MtHubOrderRejectRateHigh

> 占位（post-launch 补全）。关联告警：`deploy/prometheus/alerts.yml` `MtHubOrderRejectRateHigh`（`mthub_orders_placed_total{status="rejected"}` 占比 >10%，2m）。

## 含义
订单被风控 Gate / broker 拒绝的比例飙升。常因：保证金不足、账户余额低、策略手数超限、broker 端拒绝（品种/交易时段）。

## 影响
策略信号下不出单 → 实盘与回测发散、用户策略"看似运行实则空转"。

## 初步处置
1. Grafana 钱路径 dashboard 看是哪个 broker / 哪些策略在集中被拒。
2. `docker logs alphaforge-backend | grep -i reject` 看拒绝原因（Gate rule 命中 / broker error）。
3. 若 Gate 规则（MaxPositionCount/MaxLotSize/MarginPreCheck）集中命中 → 看是否阈值需调；若 broker 端 → 查 MT 连接/账户状态。

## TODO（post-launch）
完整诊断决策树、缓解（阈值调整/账户充值）、升级路径、复盘模板。
