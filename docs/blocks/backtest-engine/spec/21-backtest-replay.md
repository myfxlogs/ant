# 21 · 回测/实盘统一代码路径规范

> **关联 ADR**：ADR-0012
> **关联 spec**：`docs/spec/11-mdgateway.md`、`docs/spec/24-paper-trading.md`

## 1. 设计原则

回测和实盘必须走**完全相同的** factor → signal → execution 代码路径。唯一差异：数据源（历史回放 vs 实时流）和执行终端（仿真 vs 实盘）。

## 2. Source 接口

```go
// internal/factorsvc/source.go

type BarSource interface {
    // StreamBars 返回 bar 通道。From/To 都为 0 = 实时模式。
    StreamBars(ctx context.Context, params StreamParams) (<-chan *mdtick.Bar, error)
}

type StreamParams struct {
    Broker    string
    Canonical string
    Period    string
    From, To  int64  // unix_ms
}
```

```go
// internal/quantengine/source.go

type FactorSource interface {
    StreamFactors(ctx context.Context, params StreamParams) (<-chan *FactorValue, error)
}
```

## 3. LiveSource 实现

```go
// internal/factorsvc/live_source.go

type LiveSource struct {
    natsConn *nats.Conn
}

func (s *LiveSource) StreamBars(ctx context.Context, p StreamParams) (<-chan *mdtick.Bar, error) {
    subject := fmt.Sprintf("md.bar.%s.%s.%s", p.Broker, p.Canonical, p.Period)
    sub, _ := s.natsConn.ChanSubscribe(subject, 256)
    ch := make(chan *mdtick.Bar, 256)
    go func() {
        defer close(ch)
        for {
            select {
            case <-ctx.Done():
                return
            case msg := <-sub:
                var bar mdtick.Bar
                bar.IsReplay = false
                // proto unmarshal → ch
                ch <- &bar
            }
        }
    }()
    return ch, nil
}
```

## 4. ReplaySource 实现

```go
// internal/factorsvc/replay_source.go

type ReplaySource struct {
    chConn *clickhouse.Conn
}

func (s *ReplaySource) StreamBars(ctx context.Context, p StreamParams) (<-chan *mdtick.Bar, error) {
    query := `
        SELECT broker, canonical, period, close_ts_unix_ms, open, high, low, close, volume
        FROM md_bars FINAL
        WHERE broker = ? AND canonical = ? AND period = ?
          AND close_ts_unix_ms >= ? AND close_ts_unix_ms <= ?
        ORDER BY close_ts_unix_ms ASC
    `
    rows, _ := s.chConn.Query(ctx, query, p.Broker, p.Canonical, p.Period, p.From, p.To)
    ch := make(chan *mdtick.Bar, 256)
    go func() {
        defer close(ch)
        defer rows.Close()
        for rows.Next() {
            var bar mdtick.Bar
            bar.IsReplay = true  // 标记为回放数据
            // scan → ch
            ch <- &bar
        }
    }()
    return ch, nil
}
```

## 5. 与 factorsvc 集成

```go
type FactorService struct {
    source  BarSource   // 构造时注入
    dsl     *dsl.Engine
    // ...
}

func (s *FactorService) OnBar(ctx context.Context, bar *mdtick.Bar) {
    // 与 Live/Replay 完全相同的逻辑，不检查 bar.IsReplay
    value := s.dsl.Eval(ctx, bar, s.window.Get(bar.Canonical, bar.Period))
    if !math.IsNaN(value) {
        s.publisher.Publish(ctx, &FactorValue{...})
        s.chWriter.Enqueue(ctx, &FactorValue{...})
    }
}
```

## 6. 与 quantengine 集成

```go
type QuantEngine struct {
    factorSource FactorSource
    // ...
}

func (e *QuantEngine) OnFactor(ctx context.Context, fv *FactorValue) {
    signal := e.infer(ctx, fv)  // ONNX / DSL — 同样的逻辑
    signal.Source = e.mode      // "live" | "replay"
    e.signalRouter.Route(ctx, signal)
}
```

## 7. OMS 路由

```go
func (r *SignalRouter) Route(ctx context.Context, signal *Signal) {
    if err := r.risk.PreCheck(ctx, signal); err != nil {
        r.auditLog.Deny(ctx, signal, err)
        return
    }
    switch signal.Source {
    case "live":
        r.liveExecutor.Place(ctx, signal)
    case "replay":
        r.paperExecutor.Place(ctx, signal)  // ADR-0015
    }
}
```

## 8. 配置

```bash
ANT_MODE=live              # 实时模式
ANT_MODE=backtest          # 回测模式
ANT_BACKTEST_FROM=1714867200000  # unix_ms
ANT_BACKTEST_TO=1714953600000
```

## 9. 指标

| 指标 | 说明 |
|------|------|
| `replay_bar_total` | 回放 bar 计数 |
| `replay_signal_total` | 回放产生的信号数 |
| `replay_signal_divergence_ratio` | 回测信号与实盘日志的偏差率（回归测试） |

## 10. Determinism Contract (M10-BASE-A5)

All time-dependent code paths MUST use `clock.Clock` instead of raw `time.Now`/`time.Sleep`/`time.NewTicker`/`time.AfterFunc`. The `clock.SimulatedClock` enables deterministic backtest replay.

**Invariant**: Running the same backtest twice with identical input bars and the same `SimulatedClock` start time MUST produce identical PnL, signals, and trade records (byte-for-byte hash match).

**Violations to avoid**:
- `time.Now()` outside clock package — forbidden by `.golangci.yml` forbidigo
- `rand.Seed(time.Now().UnixNano())` — use fixed seed or clock-derived seed
- `map` iteration order — sort keys before iterating
- Goroutine ordering — use deterministic sequencing via simulated clock events

**Verification**: `TestBacktestDeterminism` runs the same 100-bar dataset through the backtest engine twice with a simulated clock and asserts hash identity.

## 11. 验收命令

```bash
# 1. Source 接口定义
grep -q "BarSource" backend/internal/factorsvc/source.go

# 2. factorsvc 不检查 IsReplay（代码无分支）
grep -c "IsReplay" backend/internal/factorsvc/ | grep -q ":0$"

# 3. 回归测试：回测过去 7 天 vs 同期实盘日志
go test -tags=regression ./tests/regression/ -run TestBacktestLiveParity -v
```
