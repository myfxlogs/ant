# ADR-0016 · Bar 修订级联处理

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/25-bar-revision-cascade.md`、`docs/spec/11-mdgateway.md`
- **关联 ADR**：ADR-0009

## 1. 背景

Broker 可能修订已推送的历史 bar 数据。常见场景：
- 外汇 broker 收盘后修正日线 OHLC（流动性提供商结算差异）
- 数据清洗修正
- Broker 重启后重新推送时间窗口

ADR-0009 定义了"bar finality"概念，但只规定了 bar_aggregator 侧不覆盖已闭合 bar。没有定义当 broker 真正修订 bar 后，下游因子、信号、订单如何处理。

## 2. 决策

### 2.1 检测机制

`md_bars` CH 表新增 `version UInt32 DEFAULT 1` 列。当 backfiller 拉取到与已有 bar `(broker, canonical, period, close_ts_unix_ms)` 相同但 OHLCV 不同的数据时，INSERT 新版本行（ReplacingMergeTree 自动保留最新 version）。

`bar_aggregator` 在 finality 检查时检测修订：若 broker 推送的 bar `close_ts_unix_ms` 已存在于 CH，且 OHLCV 不同 → 发布 `bar.revision.{broker}.{canonical}.{period}` NATS 消息。

### 2.2 级联策略

```
Bar 修订检测
  ├─→ factorsvc 重算受影响窗口的因子
  │     └─→ factor_values INSERT 新版本（ReplacingMergeTree 覆盖）
  ├─→ quantengine 重算受影响窗口的信号
  │     ├─→ 信号尚未发送至 OMS → 就地更新（新版本）
  │     └─→ 信号已执行（订单已发）→ 记录偏差到 CH bar_revision_log
  │           → 不修改/撤销已执行的订单（订单不可变）
  │           → 若偏差 > 阈值（方向改变 或 数量偏离 > 20%）→ BarRevisionPostExecution 告警
  └─→ 审计：bar_revision_total 指标
```

### 2.3 订单不可变性原则

一旦 `broker_ticket` 已分配，订单记录不可变。bar 修订不会：
- 自动平仓
- 自动修改订单
- 自动撤销订单

理由是 broker 侧没有"因行情修订而修改已成交订单"的语义。修正 bar 后如果发现不应该成交，这是 broker 的问题，不是 ant 能自动修复的。

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| 忽略 bar 修订 | 静默错误数据；因子/信号不可靠 |
| 自动修改/撤销已执行订单 | 违反 broker 端订单不可变性；可能造成实际资金损失 |
| 禁止 broker 修订（拒绝接收）| ant 无法控制 broker 行为 |

## 4. 后果

- **正面**：bar 修订不再静默污染因子和信号；已执行订单有完整审计追踪
- **负面**：CH `md_bars` 存储量因版本行增加（估计 +5%）
- **中性**：`BackfillPostExecution` 告警需要人工介入判断

## 5. 实施约束

1. `chmigrate/015_bar_version.sql`：`md_bars` 新增 `version UInt32 DEFAULT 1`
2. `bar_aggregator.go`：finality 检查时检测修订，发布 NATS 消息
3. `factorsvc`：新增 `bar.revision.>` NATS 订阅
4. `quantengine`：信号重算逻辑 + 偏差记录
5. CH `bar_revision_log` 表
6. spec/25 详细规范

## 6. 验证方式

```bash
# 注入修订 bar → 验证因子重算触发 → 验证信号更新（未执行）/ 偏差记录（已执行）
go test -tags=integration ./internal/mdgateway/ -run TestBarRevision -v
```
