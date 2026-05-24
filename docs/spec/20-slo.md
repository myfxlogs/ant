# 20 · SLO 与 Error Budget 规范

> 关联 ADR：ADR-0010
> 关联 spec：`docs/spec/15-observability.md` §6（alert）

## 1. 数据基础四条 SLO

| ID | 名称 | 指标定义 | 目标 | 测量窗 | error budget |
|---|---|---|---|---|---|
| **SLO-MD-1** | 可用性 | (1 - downtime_minutes/total_minutes) | 99.9% | 30 天滚动 | 43.2 min/月 |
| **SLO-MD-2** | tick e2e 延迟 P99 | `histogram_quantile(0.99, rate(md_e2e_latency_seconds_bucket[5m]))` | < 0.5 s | 5min 滚动 | P99 超阈值时长 / 30d ≤ 0.1% |
| **SLO-MD-3** | 数据完整性 | `1 - rate(md_tick_dropped_total[5m]) / rate(md_tick_total[5m])` | ≥ 99.9% | 5min 滚动 | drop 率 > 0.1% 时长 ≤ 1h/月 |
| **SLO-MD-4** | 降级窗口 | `md_spill_pending_files` | < 1 | 持续时间 | spill 持续 > 5min 次数 ≤ 3/月 |

## 2. SLO 实现细节

### 2.1 SLO-MD-1（可用性）

"down" 定义：连续 60s 无任何 tick 写入 CH（across all accounts）。

实现：Prometheus recording rule（存入 `deploy/prometheus/rules.yml`，与 alert rules 文件 `deploy/prometheus/alerts.yml` 分文件管理；alerts.yml 详见 spec/15 §6 / §6.x）：
```yaml
- record: md:up:1m
  expr: (rate(md_tick_total[1m]) > 0)
- record: md:availability:30d
  expr: avg_over_time(md:up:1m[30d])
```

### 2.2 SLO-MD-2（延迟）

`md_e2e_latency_seconds`：在 `clickhouse_writer.go` flush 成功时记录 `now - tick.ArrivedUnixMs`：

```go
hist.Observe(float64(time.Now().UnixMilli() - tick.ArrivedUnixMs) / 1000)
```

buckets：`[0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5]`（覆盖 SLO 阈值 0.5）

### 2.3 SLO-MD-3（完整性）

drop 率 = drop tick / total tick。**`reason="spill_failed"` 计入 drop**（这是真正的丢数据）；`outlier` `gap` 等不算 drop（不丢，只 metric）。

### 2.4 SLO-MD-4（降级）

`md_spill_pending_files` 由 spill_replay goroutine 每 30s 扫目录更新：
```go
files, _ := filepath.Glob(spillDir + "/*.jsonl")
gauge.Set(float64(len(files)))
```

正常稳态 = 0；CH 中断时上升；recover 后归零。

## 3. SLO 监控与告警

每条 SLO 对应 alert（详见 spec/15 §6 + ADR-0010 §2.4）。

**Burn rate alert**（Google SRE 实践）：

```yaml
- alert: ErrorBudgetBurnFast
  # 1h 内消耗了 30 天 budget 的 14.4%（30d budget 在 1h 烧完意味着已经不止一倍透支）
  expr: |
    (
      1 - md:availability:5m
    ) > 14.4 * 0.001
  for: 5m
  labels: { severity: page }
```

## 4. 月度复盘流程

每月 1 号自动生成 `docs/handover/slo-YYYY-MM.md`：

```markdown
# SLO Report YYYY-MM

| SLO | 目标 | 实际 | 消耗 budget | 状态 |
|---|---|---|---|---|
| SLO-MD-1 可用性 | 99.9% | 99.95% | 21min / 43.2min | ✅ |
| SLO-MD-2 延迟 P99 | < 500ms | 480ms | - | ✅ |
| SLO-MD-3 完整性 | ≥ 99.9% | 99.85% | 1.5h / 1h | ⚠️ 透支 |
| SLO-MD-4 降级 | spill < 5min×3 | 2 次 | - | ✅ |

## 透支根因
SLO-MD-3 因 2026-05-15 broker A 推送脏数据导致 drop 率短暂飙升至 5%。
**改进**：在 quality.go 加 broker-specific 阈值。
```

由 `cmd/slo-report/main.go` 生成（M10.5 卡片实施）。

## 5. SLO 与卡片硬关联

任何卡片如果会**临时降低** SLO（例如压测），必须在卡片"备注"列声明并经过审批。M10 关闭检查包含：
```bash
# 关闭前 7 天 SLO 全绿
md-doctor all --window 7d --strict
slo-report --window 7d --strict
```

## 6. 反模式

- ❌ 把 SLO 设到现状之上（"我们想做到 99.99%"）→ 永远透支
- ❌ 不区分 user-facing 和 system-internal SLO（行情接入是 system-internal，不直接面向用户）
- ❌ Alert 阈值 = SLO 目标本身 → 触发即透支，没有缓冲

正模式：
- ✅ Alert 阈值 = SLO 目标 + 缓冲（例如 SLO 0.1% drop，alert 0.5% drop 即告警预警）
- ✅ Burn rate 多窗口（fast: 1h × 14.4×；slow: 6h × 6×）

## 7. 验收命令

```bash
# 1. 文档存在 + 4 条 SLO
test -f docs/spec/20-slo.md
grep -E 'SLO-MD-[1-4]' docs/spec/20-slo.md | wc -l | awk '$1>=4'

# 2. recording rule 存在
grep -q 'md:availability:30d' deploy/prometheus/rules.yml

# 3. md_e2e_latency_seconds metric 暴露
curl -s localhost:8080/metrics | grep -q '^md_e2e_latency_seconds_bucket'

# 4. md_spill_pending_files gauge 存在
curl -s localhost:8080/metrics | grep -q '^md_spill_pending_files'

# 5. SLO report CLI 可执行
go build -o /tmp/slo-report ./cmd/slo-report/
/tmp/slo-report --window 1h --output text | grep -E 'SLO-MD-[1-4]'
```
