# Runbook · MtHubOrderRejectRateHigh

> 关联告警：`MtHubOrderRejectRateHigh`（rejected 占比 >10%，持续 2m，severity=warning）

## 症状

`sum(rate(mthub_orders_placed_total{status="rejected"}[5m])) / sum(rate(mthub_orders_placed_total[5m])) > 0.10` 持续触发。策略产生信号但订单被 Gate 风控或 broker 端拒绝。用户策略"看似运行实则空转"。

## 影响

实盘与回测发散——策略逻辑产生了交易信号，但订单未成交。若持续，用户策略 P&L 与回测预期严重偏离。

## 诊断步骤

```bash
# 1. 确认是 Gate 拒绝还是 broker 拒绝
docker logs alphaforge-backend --since 10m | grep -iE "reject|gate.*deny|pre.*check.*fail"

# 2. 查 Gate 规则命中详情
docker logs alphaforge-backend --since 10m | grep -iE "MaxPosition|MaxLotSize|MarginPre|gate.*rule"

# 3. 查 broker 端拒绝原因
docker logs alphaforge-backend --since 10m | grep -iE "broker.*reject|trade.*disabled|invalid.*volume|invalid.*price"

# 4. 看是哪个账户/策略集中被拒
curl -s http://localhost:8080/metrics | grep 'mthub_orders_placed_total{status="rejected"}'
```

## 应急处置

1. **Gate 风控规则集中命中**：
   - `MaxPositionCount` → 账户持仓数超限，检查是否有僵尸仓位未关闭
   - `MaxLotSize` → 策略手数超限，调整策略参数或 Gate 阈值
   - `MarginPreCheck` → 保证金不足，用户需充值或降低杠杆
2. **broker 端拒绝**：
   - `trade disabled` → 品种交易时段关闭（周末/假期/休市），正常行为
   - `invalid volume` → 手数不符合 broker 的 `min_lot`/`max_lot`/`lot_step` 约束
   - `invalid price` → 下单价格偏离市场价过多（滑点超限）
3. **单策略持续被拒** → 暂停该策略 schedule，排查策略参数配置

## 常见根因

- **保证金不足**：账户余额低 + 高杠杆 → `MarginPreCheck` 拒绝。最常见
- **品种交易时段关闭**：周末/假期 FX 不交易，broker 返回 `trade disabled`
- **手数不合规**：策略计算的手数不符合 `broker_symbols` 表中 `min_lot`/`max_lot`/`lot_step` 约束
- **Gate 阈值过严**：`MaxPositionCount` 或 `MaxLotSize` 配置与策略实际需求不匹配
