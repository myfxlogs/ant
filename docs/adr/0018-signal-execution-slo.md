# ADR-0018 · 信号→执行延迟 SLO

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/27-signal-execution-slo.md`、`docs/spec/20-slo.md`
- **关联 ADR**：ADR-0010

## 1. 背景

量化交易系统的核心性能指标不是行情延迟（tick→CH），而是信号→执行延迟（factor trigger→broker ack）。当前 SLO-MD-2 只覆盖前者（< 500ms P99），未定义后者。

如果因子触发了交易信号，但 2 秒后才到达 broker，策略的滑点会让任何 alpha 消失。必须定义 signal→execution 端到端延迟的 SLO 和分阶段预算。

## 2. 决策

### 2.1 延迟定义

- **起点**：因子值跨越阈值触发信号的时刻（`factor_ts_unix_ms`）
- **终点**：mtapi OrderSend 返回 broker 确认的时刻（`broker_ack_ts`）
- **总延迟** = `broker_ack_ts - factor_ts_unix_ms`

### 2.2 分阶段预算

| 阶段 | P50 预算 | P99 预算 | 测量方式 |
|------|----------|----------|----------|
| 因子触发 → 量化引擎信号输出 | < 5ms | < 20ms | `quant_inference_latency_seconds` |
| 信号 → 风控检查完成 | < 2ms | < 5ms | `oms_risk_check_latency_seconds` |
| 风控通过 → mthub.Place 调用 | < 5ms | < 10ms | `mthub_place_latency_seconds` |
| mthub.Place → broker 确认 | < 50ms | < 200ms | `mthub_broker_ack_latency_seconds` |
| **总计** | **< 62ms** | **< 235ms** | `signal_to_execution_latency_seconds` |

### 2.3 SLO 定义

| SLO ID | 指标 | 目标 | 窗口 | Error Budget |
|--------|------|------|------|--------------|
| SLO-SIG-1 | P99 signal→execution latency | < 235ms | 5min 滚动 | P99 超标时长 ≤ 1h/月 |
| SLO-SIG-2 | P50 signal→execution latency | < 62ms | 5min 滚动 | （仅参考，不告警）|
| SLO-SIG-3 | 信号丢弃率（非风控原因）| < 0.01% | 5min 滚动 | 丢弃超过 10min/月 |

### 2.4 实现方式

- OTel trace span 贯穿全链路：`factor.value → signal → risk → order → broker_ack`
- 每个阶段记录 span start/end 时间戳
- Prometheus histogram 按阶段分别聚合
- `signal_to_execution_latency_seconds` 为根 span 的 duration

### 2.5 Broker 延迟的务实认知

broker → broker ack 阶段（200ms P99）占总预算的 85%。ant 侧优化（factor + risk + mthub）最大可压缩到 ~35ms。优化 ant 侧代码的边际收益低；如果要降低总延迟，需要选择更快的 broker 或更近的 mtapi 网关。

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| SLO = "越快越好" | 不可测量 |
| P99 < 50ms | 不现实；MT5 broker RTT 通常 100-200ms |
| 不设信号 SLO，只设行情 SLO | 不知道端到端交易性能，无法优化 |

## 4. 后果

- **正面**：端到端交易延迟可测量、可优化、可告警
- **负面**：每个 pipeline 阶段需要暴露 histogram + OTel span；增加约 50 行观测代码
- **中性**：broker 延迟是最大瓶颈，ant 侧优化空间有限

## 5. 实施约束

1. `internal/observability/trace.go`：信号→执行 span 传播
2. factorsvc、quantengine、oms、mthub 各阶段添加 span + histogram
3. Prometheus recording rules：`signal_execution:p99:5m`
4. Alert rules：`SignalExecutionLatencyHigh` (P99 > 500ms 的 page 级别告警)
5. spec/27 详细规范

## 6. 验证方式

```bash
# 负载测试：25k signals/hour，P99 < 235ms，OTel trace 验证各阶段
go test -tags=loadtest ./tests/loadtest/ -run TestSignalExecutionLatency -v
```
