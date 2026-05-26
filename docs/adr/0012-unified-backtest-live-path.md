# ADR-0012 · 统一回测/实盘代码路径

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/21-backtest-replay.md`、`docs/spec/24-paper-trading.md`
- **关联 ADR**：ADR-0009

## 1. 背景

量化交易系统的基本要求：回测结果必须能预测实盘表现。如果回测和实盘走不同的代码路径，任何回测结论都不可信。

当前 v2 架构中：
- 实盘路径：NATS `md.bar.*` → factorsvc → quantengine → oms → mthub → broker
- 回测路径：**未定义**

如果回测时另写一套 Python 脚本或独立 Go 工具读取 CH 数据、重算因子、模拟信号，则：
- 因子计算逻辑可能与实盘 factorsvc 不一致（DSL 版本、参数默认值、NaN 处理）
- 信号触发时序可能不同（bar 边界判定、窗口大小）
- 回测赚钱的策略上线后表现完全不可预测

## 2. 决策

**factorsvc 和 quantengine 必须通过 `Source` interface 接收数据**，仅两种实现：

```go
type BarSource interface {
    StreamBars(ctx context.Context, params StreamParams) (<-chan *mdtick.Bar, error)
}

type StreamParams struct {
    Broker    string
    Canonical string
    Period    string
    From, To  int64  // unix_ms; 两者都为零 = 实时流
}
```

- `LiveSource`：NATS JetStream 订阅，无结束时间
- `ReplaySource`：CH `md_bars` 范围查询，游标遍历，标记 `IsReplay=true`

下游逻辑（factorsvc.Eval → quantengine.Infer → oms.SignalRouter）零差异。

OMS 根据 `Signal.Source` 字段（`live` | `replay`）选择 executor：
- `live` → LiveExecutor（mthub → mtapi OrderSend）
- `replay` → PaperExecutor（仿真成交，见 ADR-0015）

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| 独立回测引擎重写因子评估 | 代码重复 + 漂移风险；回测结果不可信 |
| Python 回测框架（backtrader/zipline）| 违反 invariant #2（生产路径零 Python）|
| 不建回测路径，直接实盘 | 零售用户不可接受；资金风险 |

## 4. 后果

- **正面**：回测与实盘信号偏差 < 0.1%；策略上线后可预期
- **负面**：factorsvc 需重构为接受 `BarSource` interface；quantengine 同理
- **中性**：`ReplaySource` 查询 CH 的性能取决于时间窗口大小；大窗口需分页

## 5. 实施约束

1. `internal/factorsvc/source.go`：定义 `BarSource` interface
2. `internal/factorsvc/live_source.go`：NATS JetStream 实现
3. `internal/factorsvc/replay_source.go`：CH 历史查询实现
4. `internal/quantengine/source.go`：定义 `FactorSource` interface（同样模式）
5. `config.LiveMode` vs `config.ReplayMode`：启动时选择，禁止运行时切换
6. spec/21 详细规范

## 6. 验证方式

```bash
# 自动回归：回测过去 7 天 → 对比同期实盘日志 → 信号偏差 < 0.1%
go test -tags=integration ./tests/regression/ -run TestBacktestLiveParity -v
```
