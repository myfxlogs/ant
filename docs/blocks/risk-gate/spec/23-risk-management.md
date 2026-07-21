# 23 · 持仓级风控规范

> **关联 ADR**：ADR-0014
> **关联 spec**：`docs/spec/12-mthub.md`

## 1. 风控架构

```
Signal → [risk.PreCheck — 同步阻断] → pass → OMS → mthub.Place
                │
                └─ deny → audit log + metric + signal 丢弃

[risk.Monitor — 异步 30s ticker]
  → margin check → margin call / stop-out
  → concentration check → warning
```

## 2. PreCheck 接口

```go
// internal/risk/precheck.go

type PreCheckResult struct {
    Allowed bool
    Reason  string   // 拒绝原因（中文，用于 audit log 和用户提示）
    Rule    string   // 触发的规则名
}

type RiskEngine struct {
    limits    LimitStore
    positions PositionStore
}

func (e *RiskEngine) PreCheck(ctx context.Context, accountID string, signal *Signal) PreCheckResult {
    limits := e.limits.Get(ctx, accountID)
    positions := e.positions.GetOpenPositions(ctx, accountID)

    // 1. 品种持仓上限
    if check := e.checkSymbolLimit(signal, positions, limits); !check.Allowed {
        return check
    }
    // 2. 总敞口上限
    if check := e.checkTotalExposure(signal, positions, limits); !check.Allowed {
        return check
    }
    // 3. 跨账户净敞口
    if check := e.checkNetExposure(signal, positions, limits); !check.Allowed {
        return check
    }
    // 4. 保证金利用率
    if check := e.checkMarginRatio(signal, positions, limits); !check.Allowed {
        return check
    }

    return PreCheckResult{Allowed: true}
}
```

## 3. 四项检查规则

### 3.1 品种持仓上限

```
abs(position(symbol)) + signal.volume <= max_position(symbol)
```

`max_position(symbol)` 来自 `risk_limits` 表。默认值：单品种 ≤ 10 手（可配置）。

### 3.2 总敞口上限

```
sum(abs(position_value_i)) + signal_value <= max_total_exposure
```

`position_value = position_volume * current_price * contract_size`。
`max_total_exposure` 默认 = 账户余额 * 杠杆 / 2（即最多用一半购买力）。

### 3.3 跨账户净敞口

```
abs(sum(position_value across all accounts)) <= max_net_exposure
```

考虑到用户可能 MT4 做多 BTC + MT5 做空 BTC 的套利场景，净敞口为对冲后的净值。

### 3.4 保证金利用率

```
(used_margin + signal.margin_required) / equity <= max_margin_ratio
```

`max_margin_ratio` 默认 0.80。即保证金占用不超过净值的 80%。

## 4. 可选检查

| 检查项 | 默认关闭 | 说明 |
|--------|----------|------|
| 每日亏损上限 | ✓ | `realized_pnl_today + unrealized_pnl >= max_daily_loss` |
| 最大回撤 | ✓ | `equity / peak_equity >= 1 - max_drawdown_pct` |

## 5. Post-trade 风控

```go
// internal/risk/monitor.go

func (m *Monitor) Run(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            for _, accountID := range m.GetActiveAccounts() {
                m.checkMarginLevels(ctx, accountID)
                m.checkConcentration(ctx, accountID)
            }
        }
    }
}

func (m *Monitor) checkMarginLevels(ctx context.Context, accountID string) {
    ratio := m.getMarginUtilization(ctx, accountID)
    switch {
    case ratio > 1.0:
        m.closeAllPositions(ctx, accountID)   // stop-out
        m.alert("stop_out", accountID, ratio)
    case ratio > 0.90:
        m.closeSmallestPosition(ctx, accountID)  // margin call
        m.alert("margin_call", accountID, ratio)
    case ratio > 0.80:
        m.alert("margin_warning", accountID, ratio)
    }
}
```

## 6. 限额配置

```sql
CREATE TABLE risk_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    account_id UUID REFERENCES mt_accounts(id),  -- NULL = 用户级默认
    max_position_per_symbol NUMERIC(20,8) DEFAULT 10.0,
    max_total_exposure NUMERIC(20,8) DEFAULT 0,  -- 0 = 自动计算（余额*杠杆/2）
    max_net_exposure NUMERIC(20,8) DEFAULT 0,
    max_margin_ratio NUMERIC(5,4) DEFAULT 0.8000,
    max_daily_loss NUMERIC(20,8) DEFAULT 0,       -- 0 = 不限制
    max_drawdown_pct NUMERIC(5,4) DEFAULT 0,      -- 0 = 不限制
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 7. 指标

| 指标 | 说明 |
|------|------|
| `oms_risk_block_total{rule}` | 风控拒绝计数（按规则） |
| `oms_margin_utilization{account_id}` | 保证金利用率 Gauge |
| `oms_exposure_ratio{account_id}` | 总敞口/净值 比率 |
| `oms_risk_margin_call_total` | margin call 次数 |
| `oms_risk_stop_out_total` | stop-out 次数 |

## 8. 错误码

| Code | 消息 |
|------|------|
| `RISK_POSITION_LIMIT` | 品种 {symbol} 持仓已达上限 {limit} 手 |
| `RISK_EXPOSURE_LIMIT` | 总敞口已达上限 {limit} |
| `RISK_MARGIN_LIMIT` | 保证金不足，当前利用率 {ratio}% |
| `RISK_DAILY_LOSS` | 今日亏损已达上限 {limit} |
| `RISK_DRAWDOWN` | 回撤已达上限 {limit}% |

## 9. 验收命令

```bash
# 1. PreCheck 四项检查全部实现
grep -E "checkSymbolLimit|checkTotalExposure|checkNetExposure|checkMarginRatio" \
  backend/internal/risk/precheck.go

# 2. 集成测试
go test -tags=integration ./internal/risk/ -run TestPreCheck -v
```
