# ADR-0005 · CircuitBreaker + Spill 故障恢复

- **状态**：Accepted
- **日期**：2026-05-23
- **关联 spec**：`docs/spec/11-mdgateway.md` §"circuit_breaker.go" + §"spill_writer.go"

## 1. 背景

行情链路有两类典型故障：

### A. 单 broker 不可达
- 网络抖动 / mtapi 网关重启 / broker 限流
- 期望：**不影响其他账户**；故障 broker 自动重连

### B. ClickHouse 短暂宕机
- CH 重启 / 磁盘满 / 配置错误
- 期望：**tick 不丢失**；CH 恢复后自动追写

ant v1 都没做。alfq v1 做了 Spill 但**没做 CircuitBreaker**，CH 故障时单文件 spill 无大小/时间旋转。

## 2. 决策

### 2.1 CircuitBreaker（每账户一个）

**位置**：`mdgateway/circuit_breaker.go`

**算法**：滑动窗口
- `failureThreshold = 5`：连续 5 次失败 → 开熔断
- `cooldown = 30s`：开熔断后 30s 内拒绝所有请求
- `successThreshold = 2`：半开后连续 2 次成功 → 关熔断

**集成点**：
- `Manager.HandleTick` 调 adapter 前 `breaker.Allow()`
- adapter 失败 `breaker.OnFailure()`
- adapter 成功 `breaker.OnSuccess()`
- 熔断状态暴露 metric `md_circuit_state{account_id, broker}`

### 2.2 SpillWriter（CH fallback）

**位置**：`mdgateway/spill_writer.go` + `mdgateway/spill_replay.go`

**触发**：
- `CHWriter.tickQ` chan 满 → 立即写 spill
- CH INSERT 返回 error → 整批写 spill

**文件格式**：jsonl，每行一对象，`_kind: tick|bar` 字段区分

**旋转**：
- `MaxFileBytes = 100MB`
- `MaxFileAge = 1h`
- 旋转后命名 `spill-{ts}.jsonl`

**目录**：`/var/lib/ant/spill/`（容器卷）

**Replay**：
- 启动时 `SpillReplay.Run`
- 扫描所有 `*.jsonl`，逐行写 CHWriter
- 成功 → 移到 `processed/{ts}/{filename}`
- 失败 → 移到 `failed/{ts}/{filename}.errors.jsonl`

## 3. 备选方案

### 3.1 CircuitBreaker

| 备选 | 否决 |
|---|---|
| go-resilience4j 库 | 引入大依赖；自研 132 行够用 |
| 全局单 breaker | 单 broker 故障传染所有账户 |
| Token bucket 限流 | 解决问题不同（限速 vs 故障）|

### 3.2 Spill

| 备选 | 否决 |
|---|---|
| Kafka 持久化 | 引入巨大依赖 |
| 写 PG 临时表 | PG 故障时连锁失败 |
| Redis Stream | 需配 AOF/RDB；又一个故障点 |
| 内存 retry 队列 | 进程重启即丢失 |
| 单文件 spill（alfq）| 长时间故障下文件无限大；锁占用 |

## 4. 后果

### 正面
- 单 broker 故障不传染（CircuitBreaker）
- CH 故障 24h 内可恢复（spill 100GB 容量）
- 进程重启自动 replay
- Prometheus 完全可观测

### 负面
- 磁盘要预留 100GB+ spill 空间
- 配置增加：`SPILL_DIR / SPILL_MAX_BYTES / SPILL_MAX_AGE`
- replay 期间 CH 写入压力倍增（需限速）

### 中性
- replay 顺序：`processed/` 之外的所有 `*.jsonl` 按文件名排序
- failed 行不阻塞其他行（continue on error）

## 5. 实施约束

### 5.1 接口

```go
// circuit_breaker.go
type CircuitBreaker struct { /* ... */ }
func (c *CircuitBreaker) Allow() bool
func (c *CircuitBreaker) OnSuccess()
func (c *CircuitBreaker) OnFailure()
func (c *CircuitBreaker) State() State

// spill_writer.go
type SpillWriter struct { /* ... */ }
func (s *SpillWriter) WriteTick(t *mdtick.Tick) error
func (s *SpillWriter) WriteBar(b *mdtick.Bar) error
func (s *SpillWriter) Close() error  // shutdown 时 flush

// spill_replay.go
type SpillReplay struct { /* ... */ }
func (r *SpillReplay) Run(ctx context.Context) (replayed int, err error)
```

### 5.2 启动顺序（runner.go）

```go
1. SpillReplay.Run(ctx)              // 优先回放
2. CHWriter.Start(ctx)               // 启动后台 flush
3. for each account: Manager.AddGateway(...)  // 启动行情订阅
```

如果 step 1 在某文件失败（CH 仍不可达）→ 不阻塞 step 2/3，新数据继续走 spill。下次启动时再尝试。

### 5.3 Prometheus

| 指标 | Labels |
|---|---|
| `md_circuit_state` | account_id, broker（0=closed, 1=open, 2=half_open）|
| `md_circuit_transitions_total` | account_id, broker, from, to |
| `md_spill_writes_total` | reason={ch_error, queue_full} |
| `md_spill_replay_total` | status={ok, err} |
| `md_spill_replay_lag_seconds` | — (启动时单次) |
| `md_spill_dir_bytes` | — (Gauge, periodic) |

### 5.4 alerts

```yaml
- alert: CircuitOpenLong
  expr: md_circuit_state == 1
  for: 5m
  annotations:
    summary: "CircuitBreaker open >5min on {{ $labels.account_id }}/{{ $labels.broker }}"

- alert: SpillDirBigGrowing
  expr: increase(md_spill_dir_bytes[10m]) > 1e9   # >1GB / 10min
  for: 0s
  annotations:
    summary: "Spill directory growing fast; CH likely down"
```

## 6. 验证方式

### 6.1 单元测试

```go
// circuit_breaker_test.go
func TestBreaker_FiveFailures_Opens(t *testing.T) { ... }
func TestBreaker_OpenAfterCooldown_HalfOpen(t *testing.T) { ... }
func TestBreaker_TwoSuccess_Closes(t *testing.T) { ... }
func TestBreaker_OneFailureInHalfOpen_ReOpens(t *testing.T) { ... }

// spill_writer_test.go
func TestSpill_WriteThenRotate(t *testing.T) { ... }
func TestSpill_RotateOnSize(t *testing.T) { ... }
func TestSpill_RotateOnAge(t *testing.T) { ... }
func TestSpill_ConcurrentWrites(t *testing.T) { ... }

// spill_replay_test.go
func TestReplay_AllFilesProcessed(t *testing.T) { ... }
func TestReplay_FailedRowsIsolated(t *testing.T) { ... }
func TestReplay_EmptyDir_NoOp(t *testing.T) { ... }
```

### 6.2 chaos 测试

`backend/internal/mdgateway/chaos_test.go`（build tag `chaos`）：

```go
// 模拟 CH 中断 30s，验证：
// 1. spill 文件出现
// 2. CH 恢复后自动 replay
// 3. md_ticks 最终行数 = 期望行数
func TestChaos_CHDownAndUp_NoDataLoss(t *testing.T) { ... }
```

### 6.3 生产验证

```bash
# 模拟 CH 重启 60s
docker stop ant-clickhouse
sleep 60
docker start ant-clickhouse

# 等待 30s 让 SpillReplay 处理
sleep 30

# 验证：spill 目录已清空 / processed 子目录有归档
SPILL_FILES=$(docker exec ant-backend find /var/lib/ant/spill -maxdepth 1 -name "*.jsonl" | wc -l)
test "$SPILL_FILES" -eq 0

PROCESSED=$(docker exec ant-backend find /var/lib/ant/spill/processed -name "*.jsonl" | wc -l)
test "$PROCESSED" -gt 0

# 验证 CH 数据连续
docker exec ant-clickhouse clickhouse-client --query "
  SELECT
    count() AS total,
    countIf(arrived_unix_ms BETWEEN $START AND $END) AS during_outage
  FROM md_ticks
" | awk '$2>0{print "Replay OK"} $2==0{print "FAIL: no data during outage"; exit 1}'
```
