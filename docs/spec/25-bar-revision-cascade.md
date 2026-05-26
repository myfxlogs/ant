# 25 · Bar 修订级联处理规范

> **关联 ADR**：ADR-0016
> **关联 spec**：`docs/spec/11-mdgateway.md`、`docs/spec/18-backfiller.md`

## 1. 背景

Broker 可能修订已推送的历史 bar（外汇收盘修正、流动性提供商数据清洗、broker 重启重推）。一旦 bar 被修订，依赖该 bar 的因子和信号都可能失效。

## 2. 检测机制

### 2.1 存储层

```sql
-- chmigrate/015_bar_version.sql
ALTER TABLE md_bars ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 1;
ALTER TABLE md_bars_buffer ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 1;
```

ReplacingMergeTree 自动按 `(broker, canonical, period, close_ts_unix_ms)` ORDER BY 保留最新 `version` 行。

### 2.2 bar_aggregator 检测

```go
// bar_aggregator.go: finality check 扩展

func (b *BarAggregator) finalizeBar(bar *mdtick.Bar) {
    existing, err := b.chWriter.GetLastBarVersion(ctx, bar)
    if err == nil && existing.Version > 0 {
        // 已存在相同 close_ts 的 bar
        if !bar.Equal(existing) {
            bar.Version = existing.Version + 1
            bar.IsRevision = true
            b.publisher.PublishRevision(ctx, bar)
        }
    }
    b.chWriter.Enqueue(ctx, bar)
}
```

### 2.3 NATS 消息

```
Subject: bar.revision.{broker}.{canonical}.{period}
Payload: JSON { broker, canonical, period, close_ts_unix_ms, old_version, new_version }
```

## 3. 级联流程

```
bar.revision.* NATS 消息
  │
  ├─→ factorsvc.OnBarRevision
  │     │ 重新计算受影响窗口的因子（使用修订后的 bar 数据）
  │     │ INSERT factor_values (新 version，ReplacingMergeTree 覆盖)
  │     │ Publish md.factor.*  NATS 消息 (is_revision=true)
  │     ▼
  ├─→ quantengine.OnFactorRevision
  │     │ 重新推理受影响窗口的信号
  │     ├─→ 原始信号尚未发送至 OMS
  │     │     → 在 PG signals 表中就地更新（version+1）
  │     │     → 不发布 oms.signal.* NATS 消息（等待新信号）
  │     │
  │     └─→ 原始信号已执行（订单已提交）
  │           → 在 CH bar_revision_log 中记录偏差
  │           → signals.is_revised = true, signals.revised_from_signal_id = 原始 signal
  │           → 不修改/撤销订单
  │           → 若偏差 > 阈值 → BarRevisionPostExecution 告警
  │
  └─→ 审计
        CH bar_revision_log 表 + bar_revision_total 指标
```

## 4. 信号修订策略

### 4.1 未执行信号

| 原始信号状态 | 修订动作 |
|-------------|----------|
| PENDING（尚未发送至 OMS） | 就地更新信号内容（version+1） |
| SUBMITTED（已发送至 OMS，未被 mthub 处理） | 由 OMS 取消原信号 → 生成新修订信号 |

### 4.2 已执行信号

已执行信号不可变。修订后的信号作为新记录写入：
- `is_revised = true`
- `revised_from_signal_id = 原始 signal UUID`
- 原始订单不变

告警条件（任一满足即触发）：
- 方向改变（BUY → SELL 或反之）
- 数量偏离 > 20%

## 5. CH 审计表

```sql
CREATE TABLE bar_revision_log (
    broker LowCardinality(String),
    canonical LowCardinality(String),
    period LowCardinality(String),
    close_ts_unix_ms Int64,
    old_version UInt32,
    new_version UInt32,
    old_close Decimal(18,6),
    new_close Decimal(18,6),
    divergence_pct Float64,
    affected_factor_count UInt32,
    affected_signal_count UInt32,
    executed_signal_divergence_count UInt32,  -- 已执行信号的偏差数
    detected_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (broker, canonical, period, close_ts_unix_ms);
```

## 6. 指标

| 指标 | 说明 |
|------|------|
| `bar_revision_total{broker, canonical}` | Bar 修订检测计数 |
| `bar_revision_factor_recalc_total` | 因子重算次数 |
| `bar_revision_signal_update_total` | 信号更新次数（未执行） |
| `bar_revision_post_execution_total` | 已执行信号偏差数 |
| `bar_revision_signal_divergence_ratio` | 信号偏差率（revision 前后信号不同 / 总修订数） |

## 7. 告警

```yaml
- alert: BarRevisionPostExecution
  expr: rate(bar_revision_post_execution_total[1h]) > 0
  for: 5m
  labels: { severity: warning }
  annotations: {
    summary: "检测到已执行订单受 bar 修订影响",
    description: "过去 1h 有 {{ $value }} 个已执行信号受 bar 修订影响，需人工审查"
  }
```

## 8. 限制

- 部分 bar 修订（仅修订部分周期）可能造成因子窗口不一致 → backfiller 全量重拉受影响时间范围缓解
- Bar 修订链（修订后又修订）→ 只触发一次级联（latest version），中间版本不追溯

## 9. 验收命令

```bash
# 1. version 列存在
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT name FROM system.columns WHERE database='ant' AND table='md_bars' AND name='version'"

# 2. bar_revision_log 表存在
docker exec ant-clickhouse clickhouse-client --query \
  "EXISTS TABLE ant.bar_revision_log"

# 3. 集成测试：注入修订 bar → 验证级联
go test -tags=integration ./internal/mdgateway/ -run TestBarRevision -v
```
