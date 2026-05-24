# ADR-0010 · SLO + Alert + DLQ + Trace 框架

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/15-observability.md` `docs/spec/20-slo.md`（新建）
- **关联 ADR**：ADR-0005（CircuitBreaker + Spill）

## 1. 背景

M7 已有 22 个 metric + 7 条 alert，但缺三个关键能力：

1. **SLO 缺失**：没有数值化的"健康"定义。tick 端到端延迟、drop 率、spill 长度的可接受阈值不明，导致告警阈值靠拍脑袋
2. **DLQ 缺失**：`quality.go` drop 的 tick（`bid_gt_ask` / `non_positive` / `parse_error`）没有样本留存，出 bug 时只能看 metric 总数，无法回溯哪一条数据触发
3. **Trace 缺失**：tick 端到端 P99 超 SLO 时无法定位卡在 `normalize` / `quality` / `publish` / `chWrite` 哪一段
4. **clock skew 不告警**：`md_clock_skew_seconds` 仅 gauge，>30s 表示 broker 时钟错乱、bar 时间不可信，必须升级为 alert

## 2. 决策

### 2.1 SLO 定义（`docs/spec/20-slo.md`）

数据基础四条 SLO：

| SLO | 指标 | 目标 | 测量窗 | 失败动作 |
|---|---|---|---|---|
| **可用性** | tick 持续到达 | 99.9% (>30 days) | 1min 检查无 tick = down | alert MdGatewayUnavailable |
| **延迟 P99** | tick e2e (broker→CH 落盘) | < 500ms | rolling 5min | alert TickLatencyHigh |
| **数据完整性** | drop 率 | < 0.1% | rolling 5min | alert HighDropRate |
| **降级耗时** | spill backlog | < 5min 持续 | rolling | alert SpillBacklog |

每条 SLO 有 error budget = 1 - target；月度复盘 budget 消耗。

### 2.2 DLQ（Dead Letter Queue）

新增 CH 表 `md_ticks_dlq`（schema 见 spec/13 §2.6）：
- TTL 7 天
- `quality.go` drop 时按 reason **采样写入**：
  - `parse_error`：100% 写（数据损坏，必须排查）
  - `bid_gt_ask` / `non_positive`：1% 采样写（broker bug，量大）
  - 通过 `metrics.go` 计数器 `md_dlq_sampled_total{reason}`

实现：`quality.go` 接受可选注入 `dlq DLQWriter`，drop 时根据 reason 决定调用频率。`DLQWriter` 与 `CHWriter` 复用 spill 路径（CH 不通时落 spill 同目录）。

### 2.3 OpenTelemetry Trace

数据基础引入 OTel（M8 计划提前到 M10），跨度链：

```
adapter.OnQuote
  └─ manager.HandleTick
       ├─ normalizer.Resolve
       ├─ quality.Check
       ├─ tick_dedup.Seen
       ├─ bar_aggregator.AddTick
       ├─ publisher.PublishTick (异步)
       └─ chWriter.Enqueue → batch.INSERT (异步)
```

- exemplar 关联到 `md_publish_latency_seconds` `md_ch_write_latency_seconds` histogram
- exporter：OTLP gRPC 到本地 Tempo / Jaeger（`OTEL_EXPORTER_OTLP_ENDPOINT` env，缺省关闭）
- 采样：实时路径 1%；spill replay / backfiller 100%（量小且需要排错）

### 2.4 Alert 补全

`deploy/prometheus/alerts.yml` 新增：

```yaml
- alert: BrokerClockSkewHigh
  expr: abs(md_clock_skew_seconds) > 30
  for: 2m
- alert: TickLatencyP99High
  expr: histogram_quantile(0.99, rate(md_e2e_latency_seconds_bucket[5m])) > 0.5
  for: 5m
- alert: SpillBacklog
  expr: md_spill_pending_files > 0
  for: 5m
- alert: SpillUnwritable
  expr: rate(md_tick_dropped_total{reason="spill_failed"}[5m]) > 0
  for: 1m
- alert: DLQSpike
  expr: rate(md_dlq_sampled_total{reason="parse_error"}[5m]) > 1
  for: 2m
- alert: NormalizerFallbackHigh
  expr: rate(md_canonical_fallback_total[10m]) > 0.05 * rate(md_tick_total[10m])
  for: 10m
```

新增 metric：
```
md_e2e_latency_seconds  Histogram  buckets [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5]
md_spill_pending_files  Gauge      未 replay 的 spill 文件数
md_dlq_sampled_total{reason}  Counter
```

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决 |
|---|---|---|---|
| 仅加 alert 不加 SLO | 工作量小 | 阈值无依据；月度复盘无锚点 | 失去工程纪律 |
| DLQ 全量写 | 数据完整 | broker bug 时打爆 CH（每秒 10k 行 dlq） | 反面教材 |
| Trace 延后到 M11 | 减少 M10 范围 | M10 引入双写 + finality 增加复杂度，定位 bug 时缺 trace 极痛 | 必须前置 |
| 用 zap log 模拟 trace | 零依赖 | 跨服务/异步无法关联；exemplar 不能挂 metric | 工程级不够 |
| **本决策：SLO + DLQ + OTel 三件套同 ADR 引入** | 工程基础设施一次到位；之后所有性能问题都有抓手 | 工作量大（~5 卡片） | ✓ |

## 4. 后果

- **正面**：所有 incident 可定位（trace）、可重现（DLQ）、可量化（SLO）；alert 阈值有据可依
- **负面**：
  - OTel SDK 增加 ~3MB 二进制；运行时 CPU 1% 开销（1% 采样下）
  - DLQ CH 表占用磁盘（按当前流量估 < 100MB / 7d）
  - 多一份 spec（20-slo.md）需要维护
- **中性**：`quality.go` API 变化（注入 DLQWriter），所有调用方需更新

## 5. 实施约束

1. `metrics.go` 新增 3 个 metric（含 buckets 定义）
2. `quality.go` 接收可选 `dlq DLQWriter` 注入；nil 时跳过 DLQ 写
3. `deploy/prometheus/alerts.yml` 同步更新
4. `cmd/ant-server/main.go` 启动 OTel SDK（按 env 开关）；`internal/trace/otel.go` 封装初始化
5. spec/15 §6 alert 列表追加；spec/20-slo.md 新建
6. 所有 incident runbook（`docs/runbook/`）参考 SLO 表
7. DLQ 表写入路径必须经过 spill fallback（与 main path 一致）

## 6. 验证方式

```bash
# 1. SLO 文档存在且四条完整
test -f docs/spec/20-slo.md
grep -E '可用性|延迟 P99|数据完整性|降级耗时' docs/spec/20-slo.md | wc -l | grep -q '^4$'

# 2. DLQ 表存在
docker exec ant-clickhouse clickhouse-client --query "DESCRIBE ant.md_ticks_dlq" | grep -q reason

# 3. parse_error 100% 入 DLQ
go test -tags=integration ./internal/mdgateway/ -run TestDLQParseError -v
# 注入 100 条 parse_error → DLQ 100 行

# 4. bid_gt_ask 1% 采样
go test -tags=integration ./internal/mdgateway/ -run TestDLQSampling -v
# 注入 10000 条 bid_gt_ask → DLQ 80~120 行（1% ± 抖动）

# 5. OTel trace 输出
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 go test -tags=integration ./internal/mdgateway/ -run TestTraceExport -v

# 6. Alert 全语法正确
promtool check rules deploy/prometheus/alerts.yml

# 7. 6 条新 alert 都在
for a in BrokerClockSkewHigh TickLatencyP99High SpillBacklog SpillUnwritable DLQSpike NormalizerFallbackHigh; do
  grep -q "alert: $a" deploy/prometheus/alerts.yml || { echo "MISSING $a"; exit 1; }
done
```
