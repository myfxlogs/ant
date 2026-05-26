# 27 · 信号→执行延迟 SLO 规范

> **关联 ADR**：ADR-0018
> **关联 spec**：`docs/spec/20-slo.md`、`docs/spec/15-observability.md`

## 1. 为什么这个 SLO 重要

量化交易中，信号→执行的延迟直接决定策略的滑点和实际收益。如果因子触发了正确的信号，但订单到达 broker 时价格已经移动了，策略的 alpha 就消失了。

这是整个量化系统最关键的性能指标。

## 2. 延迟定义

```
起点 T0: 因子值跨越阈值触发信号的时刻 (factor_ts_unix_ms)
终点 T_end: mtapi OrderSend 返回 broker 确认的时刻 (broker_ack_ts)
总延迟 = T_end - T0
```

## 3. 分阶段预算

```
T0 (factor trigger)
 │
 ├─ factorsvc → NATS publish ─────────────────── < 5ms
 │
 ├─ quantengine.OnFactor → signal ────────────── < 20ms P99
 │     测量: quant_inference_latency_seconds
 │
 ├─ oms.SignalRouter → risk.PreCheck ─────────── < 5ms P99
 │     测量: oms_risk_check_latency_seconds
 │
 ├─ oms → mthub.Place ────────────────────────── < 10ms P99
 │     测量: mthub_place_latency_seconds
 │
 ├─ mthub → adapter.OrderSend → mtapi gRPC ──── < 200ms P99
 │     测量: mthub_broker_ack_latency_seconds
 │     (这是最大的单项；broker RTT 决定)
 │
 T_end (broker ack)
```

## 4. SLO 定义

| SLO ID | 指标 | 目标 | 窗口 | Error Budget |
|--------|------|------|------|--------------|
| SLO-SIG-1 | `histogram_quantile(0.99, signal_to_execution_latency_seconds_bucket)` | < 235ms | 5min 滚动 | P99 > 235ms 持续 > 1h/月 |
| SLO-SIG-2 | `histogram_quantile(0.50, signal_to_execution_latency_seconds_bucket)` | < 62ms | 5min 滚动 | 仅参考，不告警 |
| SLO-SIG-3 | `rate(oms_signal_dropped_total{reason!="risk_deny"}[5m]) / rate(oms_signal_total[5m])` | < 0.01% | 5min 滚动 | 丢弃 > 10min/月 |

## 5. 实现

### 5.1 OTel Trace 传播

```go
// factorsvc: 附加 factor_ts_unix_ms 到 NATS 消息 header
msg.Header.Set("factor-ts-unix-ms", strconv.FormatInt(fv.TsUnixMs, 10))

// quantengine: 创建根 span
ctx, span := tracer.Start(ctx, "signal-to-execution",
    trace.WithTimestamp(time.UnixMilli(factorTs)))
defer span.End()

// oms: 子 span
ctx, riskSpan := tracer.Start(ctx, "risk-check")
result := risk.PreCheck(ctx, signal)
riskSpan.End()

// mthub: 子 span → broker_ack
ctx, orderSpan := tracer.Start(ctx, "mthub-place-order")
resp := adapter.OrderSend(ctx, req)
orderSpan.End()

// 记录 broker_ack_ts
signal_to_execution_latency.Observe(float64(resp.AckTs - signal.FactorTsMs) / 1000)
```

### 5.2 Prometheus Histogram

```go
var signalToExecutionLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
    Name:    "signal_to_execution_latency_seconds",
    Help:    "End-to-end latency from factor trigger to broker acknowledgment.",
    Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.15, 0.235, 0.3, 0.5, 1, 2},
})
```

### 5.3 分阶段 Histogram

```go
var quantInferenceLatency = prometheus.NewHistogram(...)  // bucket: [0.001, 0.005, 0.01, 0.02, 0.05]
var omsRiskCheckLatency = prometheus.NewHistogram(...)     // bucket: [0.001, 0.002, 0.005, 0.01]
var mthubPlaceLatency = prometheus.NewHistogram(...)       // bucket: [0.001, 0.005, 0.01, 0.02]
var mthubBrokerAckLatency = prometheus.NewHistogram(...)   // bucket: [0.01, 0.05, 0.1, 0.2, 0.5, 1]
```

## 6. 告警

```yaml
# Burn rate alert (Google SRE 模型)
- alert: SignalExecutionLatencyHigh
  expr: |
    histogram_quantile(0.99,
      rate(signal_to_execution_latency_seconds_bucket[5m])
    ) > 0.500
  for: 5m
  labels: { severity: page }
  annotations: {
    summary: "信号→执行 P99 延迟超过 500ms",
    runbook: "docs/runbook/mt-incidents.md#信号执行延迟"
  }

- alert: SignalExecutionLatencyWarning
  expr: |
    histogram_quantile(0.99,
      rate(signal_to_execution_latency_seconds_bucket[5m])
    ) > 0.300
  for: 15m
  labels: { severity: warning }

# Error budget burn
- alert: ErrorBudgetBurnSignal
  expr: |
    (
      1 - (
        sum(rate(signal_to_execution_latency_seconds_bucket{le="0.235"}[5m]))
        / sum(rate(signal_to_execution_latency_seconds_count[5m]))
      )
    ) > 14.4 * (1 - 0.99)
  for: 1h
  labels: { severity: page }
  annotations: {
    summary: "信号 SLO error budget 快速燃烧",
    description: "1h 内消耗了 30d budget 的 14.4%"
  }
```

## 7. 瓶颈分析

broker ack 阶段（200ms P99）占总延迟的 85%。ant 侧优化（factor + risk + mthub）最多压缩到 ~35ms。

**优化优先级**：
1. 选择更快的 broker / 更近的 mtapi 网关（收益最大）
2. mthub + adapter 内部延迟优化（减少内存分配、使用连接池）
3. OMS → mthub 使用进程内调用（已是最优）
4. factorsvc batch 优化（多因子并行计算）

## 8. 与行情 SLO 的关系

| SLO | 覆盖路径 | 目标 |
|-----|----------|------|
| SLO-MD-2 | broker → CH（tick 延迟）| P99 < 500ms |
| SLO-SIG-1 | factor → broker ack（交易延迟）| P99 < 235ms |

两者相加：tick 到达 ant (500ms) + 信号到执行 (235ms) = 总共 < 735ms P99 从市场数据刷新到订单成交。这在中低频策略（持仓 ≥ 5min）中是可接受的水平。

## 9. 验收命令

```bash
# 1. Histogram 定义
curl -s localhost:8080/metrics | grep signal_to_execution_latency_seconds_bucket

# 2. 分阶段 histogram
curl -s localhost:8080/metrics | grep -E "(quant_inference|oms_risk_check|mthub_place|mthub_broker_ack)_latency"

# 3. 告警规则存在
grep -q "SignalExecutionLatencyHigh" deploy/prometheus/alerts.yml

# 4. 负载测试：25k signals/h，P99 < 235ms
go test -tags=loadtest ./tests/loadtest/ -run TestSignalExecutionLatency -v
```
