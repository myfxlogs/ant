# ADR-0014 · 持仓级风控

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/23-risk-management.md`、`docs/spec/12-mthub.md`
- **关联 ADR**：ADR-0006

## 1. 背景

当前 v2 架构中 `risk.PreCheck` 在 L6 被提及，但仅有一句话描述，无任何具体设计。量化交易风控至少需要：

- 单品种持仓上限（防止单一品种过度集中）
- 总敞口上限（防止整体杠杆过高）
- 跨账户净敞口（MT4 做多 + MT5 做空 = 组合风险？）
- 保证金占用率（Margin / Equity 超过阈值禁止新单）

零售用户可能不理解这些概念，系统必须提供平台级安全网。

## 2. 决策

### 2.1 Pre-trade 风控（同步阻断）

在 `signal → order` 路径上，`risk.PreCheck` 是同步调用。返回 `deny` 时信号被丢弃且必须写 audit log。

4 项强制检查：

| 检查项 | 规则 | 违规动作 |
|--------|------|----------|
| 品种持仓上限 | `abs(position(symbol)) + new_volume <= max_position(symbol)` | deny + audit log |
| 总敞口上限 | `sum(abs(position_value)) + new_value <= max_total_exposure` | deny + audit log |
| 跨账户净敞口 | `abs(sum(position_value across all accounts)) <= max_net_exposure` | deny + audit log |
| 保证金利用率 | `used_margin / equity <= max_margin_ratio`（默认 0.80） | deny + audit log |

每日亏损上限和最大回撤作为可选检查（可配置开关）。

### 2.2 Post-trade 风控（异步监控）

| 条件 | 动作 |
|------|------|
| 保证金利用率 > 90% | Alert + 自动平仓最小持仓 |
| 保证金利用率 > 100%（stop-out）| 立即平仓全部持仓 |
| 单品种集中度 > 50% 总敞口 | Warning 级别告警 |

### 2.3 配置存储

风险限额存储在 PG `risk_limits` 表（用户私有，`user_id` 外键），支持：
- 平台默认值（新用户自动创建）
- 每账户覆盖
- 每策略覆盖（策略级限额 ≤ 账户级限额）

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| 风控作为独立服务（异步 RPC）| 增加信号→执行关键路径延迟 |
| 风控嵌入每个策略 | 无平台级统一执行；策略可绕过 |
| 不建风控，依赖 broker 端限制 | broker 端风控粒度粗；平台无法跨账户计算 |

## 4. 后果

- **正面**：零售用户有安全网；平台可防止单策略/单账户灾难性亏损
- **负面**：Pre-trade 风控在关键路径上增加 5-10ms（可接受，在 SLO 预算内）
- **中性**：`risk_limits` 表需初始迁移 + 默认值 seed

## 5. 实施约束

1. `internal/risk/precheck.go`：同步风控检查（实现 `PreCheck(ctx, accountID, signal, positions) -> (allow/deny, reason)`）
2. `internal/risk/monitor.go`：异步监控循环（30s ticker）
3. `internal/risk/limits.go`：限额加载（PG → 内存缓存，PG NOTIFY 失效）
4. spec/23 详细规范

## 6. 验证方式

```bash
# 集成测试：策略尝试超过持仓上限 → PreCheck deny → signal 丢弃 + audit log
go test -tags=integration ./internal/risk/ -run TestPreCheckPositionLimit -v
```
