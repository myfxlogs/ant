# 15 · 可观测性规范

> 路径：`backend/internal/metrics/` + `backend/internal/monitoring/`
> 端点：`/metrics`（Prometheus）、`/healthz`、`/readyz`
> 账户状态不走 HTTP 端点（决策 RV-C4）：走 ConnectRPC `MtHubService.GetAccountStatus` + Prometheus Gauge `mt_account_connected{account_id, broker, platform}`

## 1. 三支柱

| 支柱 | 工具 | 落点 |
|---|---|---|
| **Metrics** | Prometheus client_golang | `/metrics` |
| **Logs** | zap + ECS-style JSON | stderr → docker logs |
| **Traces** | OpenTelemetry SDK | OTLP exporter（可选；M8 接入）|

## 2. Prometheus 指标命名规范

```
<domain>_<subject>_<verb>_<unit>
```

| 维度 | 必填 |
|---|---|
| Counter 后缀 | `_total` |
| Histogram 后缀 | `_seconds` / `_bytes` / `_ratio` |
| Gauge | 无后缀 |
| Labels | `account_id, broker, canonical, period` 选用；**禁高基数 label**（如 user_id × symbol） |

## 3. 全量指标清单（v2 第一版）

### 3.1 mdgateway

见 `docs/spec/11-mdgateway.md` §12，重复列出关键 5 个：

```
md_tick_total{broker, canonical}                 Counter
md_tick_dropped_total{broker, canonical, reason} Counter   reason ∈ {bid_gt_ask, outlier, parse_error}
md_bar_flushed_total{broker, canonical, period}  Counter
md_ch_write_errors_total{kind}                   Counter
md_circuit_state{account_id, broker}             Gauge     0=closed, 1=open, 2=half_open
```

### 3.2 mthub

```
mthub_orders_placed_total{broker, status}        Counter   status ∈ {ok, rejected, err}
mthub_place_latency_seconds{broker}              Histogram buckets [0.05, 0.1, 0.25, 0.5, 1, 2, 5]
mthub_session_active{account_id, broker}         Gauge
mthub_event_published_total{event_type}          Counter
```

### 3.3 quantengine

```
quant_inference_total{strategy_id, status}       Counter
quant_inference_latency_seconds{strategy_id}     Histogram
quant_signal_total{strategy_id, side}            Counter
quant_signal_rejected_total{strategy_id, reason} Counter
```

### 3.4 oms

```
oms_order_state_transition_total{from, to}       Counter
oms_order_filled_latency_seconds                 Histogram   place→filled 时长
oms_risk_block_total{rule}                       Counter
```

### 3.5 system / runtime

```
go_goroutines, go_memstats_*, process_cpu_seconds_total       (client_golang 自动暴露)
ant_build_info{version, commit, build_time}                   Gauge=1 (启动时设)
ant_pg_pool_active_conns                                       Gauge
ant_redis_pool_active_conns                                    Gauge
ant_ch_pool_active_conns                                       Gauge
ant_nats_connected                                             Gauge   1/0
```

## 4. 健康端点契约

### 4.1 GET /healthz

进程活体探测；最快路径，**不查依赖**。

```
HTTP 200 OK
Content-Type: text/plain
ant ok
```

### 4.2 GET /readyz

接流量前置。**所有关键依赖**通过才返回 200。

检查项（顺序执行，第一个失败即 503）：

```
1. PG: SELECT 1   (timeout 1s)
2. CH: SELECT 1   (timeout 2s)
3. Redis: PING    (timeout 500ms)
4. NATS: 已连接   (即时)
5. mdgateway: 至少 50% active 账户 connected
```

成功响应：

```json
HTTP 200 OK
Content-Type: application/json
{
  "status": "ready",
  "checks": {
    "postgres": "ok",
    "clickhouse": "ok",
    "redis": "ok",
    "nats": "ok",
    "mdgateway": {
      "active_accounts": 8,
      "connected": 7,
      "ratio": 0.875
    }
  }
}
```

失败响应：

```json
HTTP 503 Service Unavailable
{
  "status": "not_ready",
  "checks": {
    "postgres": "ok",
    "clickhouse": "FAIL: connection refused",
    ...
  }
}
```

### 4.3 账户健康查询（不走 HTTP）

**废弃**：原计划的 `GET /livez/account/{id}` HTTP 端点已废弃（决策 RV-C4：k8s liveness 哲学只关心进程级；账户级是观测问题不是健康检查问题）。

**替代方案**：

1. **前端查询** → ConnectRPC `MtHubService.GetAccountStatus(account_id)` 返回：
   ```
   { account_id, broker, platform, state, last_tick_at, circuit_state, tick_rate_1m, subscribed_symbols }
   ```
2. **SRE / Grafana** → Prometheus Gauge：
   - `mt_account_connected{account_id, broker, platform}` ∈ {0,1}
   - `md_tick_total{broker, canonical}` 以 rate(1m) 求 tick rate
   - `md_circuit_state{broker, mtapi_endpoint}` 看熝断
3. **告警**：在 Grafana 设 `mt_account_connected == 0 for 60s` 规则

`state` 状态机：

```
disconnected → connecting → connected → degraded → disconnected
                                ↑          │
                                └──────────┘ (degraded → connected on success)
```

`degraded` 触发：60s 内无 tick / 熔断 half-open。

## 5. 日志规范

### 5.1 字段（ECS 风格）

每条日志**必带**：

```json
{
  "timestamp": "2026-05-23T14:23:45.123Z",
  "level": "info",
  "message": "tick processed",
  "trace_id": "...",
  "request_id": "...",
  "user_id": "...",
  "account_id": "...",        // 涉及账户时
  "broker": "...",
  "canonical": "...",
  "service": "mdgateway",
  "version": "v2.0.1"
}
```

### 5.2 级别约定

| 级别 | 用途 | 频率 |
|---|---|---|
| `debug` | 单 tick 处理细节、SQL 参数 | 默认关闭，启用时按账户开关 |
| `info` | 启动/关闭、连接/断开、批量 flush | 中等 |
| `warn` | 重试、降级（spill、circuit half-open）| 低 |
| `error` | 操作失败但未崩溃 | 极低；每条必有 `errs.Code` |
| `fatal` | 启动失败、不可恢复 | 极少 |

### 5.3 反模式（禁止）

- ❌ `log.Info("tick: " + tick.String())` → 必须 zap field
- ❌ `log.Error("err: " + err.Error())` → 必须 `zap.Error(err)`
- ❌ 在 hot path 用 `Sprintf` 拼字符串
- ❌ 不带 trace_id 的日志

## 6. 告警规则（Prometheus）

`deploy/prometheus/alerts.yml`：

```yaml
groups:
- name: mdgateway
  rules:
  - alert: MdGatewayAccountDisconnected
    expr: md_gateway_connected == 0
    for: 60s
    annotations:
      summary: "Account {{ $labels.account_id }} disconnected for >60s"

  - alert: MdGatewayTickDropRateHigh
    expr: rate(md_tick_dropped_total[5m]) > 0.05 * rate(md_tick_total[5m])
    for: 2m
    annotations:
      summary: "Drop rate >5% for {{ $labels.broker }}/{{ $labels.canonical }}"

  - alert: MdGatewayCircuitOpen
    expr: md_circuit_state == 1
    for: 30s
    annotations:
      summary: "CircuitBreaker open for account {{ $labels.account_id }}"

  - alert: MdGatewayCHWriteErrors
    expr: rate(md_ch_write_errors_total[1m]) > 0.1
    for: 1m
    annotations:
      summary: "CH write errors >0.1/s, spill is active"

- name: mthub
  rules:
  - alert: MtHubOrderRejectRateHigh
    expr: rate(mthub_orders_placed_total{status="rejected"}[5m])
        / rate(mthub_orders_placed_total[5m]) > 0.10
    for: 2m
    annotations:
      summary: "Order reject rate >10%"

  - alert: MtHubPlaceLatencyP99High
    expr: histogram_quantile(0.99, rate(mthub_place_latency_seconds_bucket[5m])) > 2.0
    for: 5m
    annotations:
      summary: "PlaceOrder p99 >2s"
```

## 6.x M10 新增告警（ADR-0010）

```yaml
- name: mdgateway-m10
  rules:
  - alert: BrokerClockSkewHigh
    expr: abs(md_clock_skew_seconds) > 30
    for: 2m
    annotations: { summary: "Broker {{ $labels.broker }} clock skew > 30s" }

  - alert: TickLatencyP99High
    expr: histogram_quantile(0.99, rate(md_e2e_latency_seconds_bucket[5m])) > 0.5
    for: 5m
    annotations: { summary: "Tick e2e P99 > 500ms" }

  - alert: SpillBacklog
    expr: md_spill_pending_files > 0
    for: 5m
    annotations: { summary: "Spill backlog > 5min ({{ $value }} files)" }

  - alert: SpillUnwritable
    expr: rate(md_tick_dropped_total{reason="spill_failed"}[5m]) > 0
    for: 1m
    labels: { severity: page }
    annotations: { summary: "Spill writer failing — data loss" }

  - alert: DLQSpike
    expr: rate(md_dlq_sampled_total{reason="parse_error"}[5m]) > 1
    for: 2m
    annotations: { summary: "DLQ parse_error spike — broker corrupting data" }

  - alert: NormalizerFallbackHigh
    expr: rate(md_canonical_fallback_total[10m]) > 0.05 * rate(md_tick_total[10m])
    for: 10m
    annotations: { summary: "Normalizer algorithmic fallback > 5% — PG broker_symbols incomplete" }
```

## 6.y M10 新增 metric

| 指标名 | 类型 | Labels | 用途 |
|---|---|---|---|
| `md_e2e_latency_seconds` | Histogram | kind={tick,bar} | SLO-MD-2 端到端延迟 |
| `md_spill_pending_files` | Gauge | — | SLO-MD-4 降级 |
| `md_dlq_sampled_total` | Counter | reason | DLQ 采样写入量 |
| `md_bar_skipped_finalized_total` | Counter | broker, canonical, period | bar finality 跳过计数 |
| `md_backfill_*` | 多 | — | 见 spec/18 §7 |

## 7. Grafana dashboard（必要面板）

`deploy/grafana/dashboards/mt-foundation.json` 包含：

1. **行情接入**：tick rate per (broker, canonical) | drop reasons | bar flushes
2. **CH 健康**：write errors | spill writes | replay status
3. **熔断**：per account state matrix
4. **mthub**：order placed rate / reject rate / latency p50/p95/p99
5. **资源**：goroutines / memory / pool conns
6. **NATS**：subjects rate / pending messages

## 8. 验收命令

```bash
# /metrics 暴露
curl -sf http://localhost:8080/metrics | grep -c "^md_tick_total" | grep -q "^1$"

# /readyz 200
curl -sfo /dev/null -w "%{http_code}" http://localhost:8080/readyz | grep -q "^200$"

# 至少有一个账户 connected（走 metric 代替原 livez/account）
curl -sf http://localhost:8080/metrics \
  | grep -E '^mt_account_connected\{[^}]+\} 1$' \
  | head -1 | grep -q 'mt_account_connected'

# 日志含必填字段
docker logs ant-backend 2>&1 | head -100 | jq -e \
  'select(.trace_id != null and .level != null and .timestamp != null)' \
  | head -1 | grep -q .

# 告警规则语法
promtool check rules deploy/prometheus/alerts.yml
```
