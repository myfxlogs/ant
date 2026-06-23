# EA 完全替代 · 灰度上线流程 (T4.2)

## 1. 上线前检查清单

- [ ] 所有 Phase 0–3 Task 测试通过 (358 tests)
- [ ] 风控门 11 条规则全部单测通过
- [ ] 沙箱逃逸测试 14 项全部隔离确认
- [ ] LiveBroker 三向一致性测试通过 (paper/live/backtest)
- [ ] 回测保真度基线差异 < 5%
- [ ] Kill-switch 演练完成
- [ ] 一键回滚验证通过

## 2. 灰度阶段

### Stage 0: OFF (不交易)
```go
ctrl := NewCanaryController(DefaultCanaryConfig())
// AllowedLotSize() = 0.01 (configured but inactive)
// IsCanaryAccount(any) = false
```

### Stage 1: CANARY (金丝雀账户 + 最小手数)
```go
ctrl.AddAccount("acct-canary-1")
ctrl.AddAccount("acct-canary-2")
ctrl.ActivateCanary()
// AllowedLotSize() = 0.01
// IsCanaryAccount("acct-canary-1") = true
// IsCanaryAccount("acct-other") = false
```
- 仅 2–3 个金丝雀账户可交易
- 起始手数 = 0.01
- 持续 ≥ 24 小时
- 至少 10 笔成功交易

### Stage 2: EXPANDED (扩展账户 + 递进手数)
```
每 24h + 10 笔成功交易 → 手数 +0.01
最大 2.00 手（可配置）
```
- 新增 5–10 个账户
- 监控：滑点、成交率、盈亏偏离

### Stage 3: FULL (全量上线)
```go
ctrl.PromoteToFull()
// AllowedLotSize() = max (10.0)
// IsCanaryAccount(any) = true
```
- 所有账户可交易
- 手数上限 = 策略配置上限
- 持续监控漂移看板

## 3. Kill-Switch 应急流程

### 触发条件（任一即触发）
- 单笔亏损 > 账户余额 5%
- 连续 3 笔拒绝单
- 回测/实盘订单方向偏差
- 人工判断异常

### 执行
```go
ctrl.EngageKillSwitch("具体原因")
// 立即生效：AllowedLotSize() = 0
// 所有现有订单保留（不强制平仓 — 由 OMS 单独处理）
```

### 恢复
```go
ctrl.Rollback()                // 回滚到上一阶段配置
ctrl.DisengageKillSwitch()     // 解除 kill-switch
// 确认手数、账户白名单已恢复
```

## 4. 一键回滚

```go
// 记录当前状态
lotsBefore := ctrl.AllowedLotSize()
stageBefore := ctrl.CurrentStage()

// 回滚
err := ctrl.Rollback()

// 验证
assert(ctrl.AllowedLotSize() == lotsBefore_previous)
assert(ctrl.CurrentStage() == stageBefore_previous)
```

## 5. 监控指标

| 指标 | 告警阈值 | 数据源 |
|------|----------|--------|
| 订单方向偏差 | > 0% | DriftReport |
| 成交滑点 | > 2 pips | fill.price vs intent.price |
| 保证金利用率 | > 80% | account.margin_level |
| Kill-switch 触发 | 任何 | canary audit log |
| 阶段停滞 | > 48h | canary stage entered time |
| 金丝雀账户盈亏 | < -5% | account.equity |

## 6. 审计日志

每次阶段变更自动记录：
```
2026-06-23T10:00:00Z  off → canary       lots=0.01  reason="canary activation"
2026-06-24T10:00:00Z  canary → expanded  lots=0.02  reason="step-up to 0.02 lots"
2026-06-24T14:30:00Z  expanded → off     lots=0      reason="KILL-SWITCH: drawdown"
2026-06-24T14:35:00Z  off → expanded     lots=0.02   reason="kill-switch disengaged"
2026-06-24T14:40:00Z  expanded → canary  lots=0.01   reason="rollback to canary"
```

## 7. 回滚演练 (每阶段执行一次)

```bash
# 1. 确认当前状态
curl /api/admin/canary/status

# 2. 模拟故障
curl -X POST /api/admin/canary/kill-switch -d '{"reason": "drill"}'

# 3. 验证阻断
curl /api/admin/canary/status  # → stage=off, lots=0

# 4. 回滚
curl -X POST /api/admin/canary/rollback

# 5. 解除 kill-switch
curl -X POST /api/admin/canary/disengage

# 6. 恢复验证
curl /api/admin/canary/status  # → stage=canary, lots=0.01
```
