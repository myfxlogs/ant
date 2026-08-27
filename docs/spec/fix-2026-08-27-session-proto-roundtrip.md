# FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP

> **Status**: 🟦open（设计 SSOT，待施工）
> **Priority**: P1（正确性——已导致 1 个 production bug，未来可能复发）
> **Author**: Devin CLI
> **Date**: 2026-08-27

## 1. 问题陈述

### 1.1 症状

`rejectNilRepeatedInLive` 在 live mode 误拒空仓账户（positions=nil），导致策略无法执行。已临时移除 nil 检查修复，但根因未消除。

### 1.2 根因

`VMLiveSession` 是**同进程**的 session 实现，但 `Session` interface 使用 `[]byte` 签名（为跨进程 RPC 设计）。事件循环每个 tick/bar 事件做 **4 次不必要的 proto 序列化**：

```
buildTickContext → 构造 *antv1.TickContext（proto 消息）
  → proto.Marshal(req)                          ← 序列化 #1
  → VMLiveSession.SendEvent(reqBytes)
  → proto.Unmarshal(reqBytes, &req)             ← 序列化 #2
  → vmHandleTick(req.TickContext)
  → VM 执行 → resp (*antv1.ExecuteLiveResponse)
  → proto.Marshal(resp)                         ← 序列化 #3
  → dispatchFromBytes(respBytes)
  → proto.Unmarshal(respBytes, &resp)           ← 序列化 #4
  → dispatch signals
```

proto3 marshal/unmarshal 语义：`repeated` 字段空切片 `[]` marshal 后被省略，unmarshal 后变成 `nil`。这导致 `positions == nil` 无法区分"数据缺失"和"无持仓"——已导致 `rejectNilRepeatedInLive` 误拒。

### 1.3 影响范围

| 文件 | 序列化点 | 说明 |
|------|---------|------|
| `live_runner_events.go:62` | Marshal #1 | handleBar 构造 req 后 marshal |
| `live_runner_events.go:164` | Marshal #1 | handleTick 构造 req 后 marshal |
| `live_runner_events.go:205` | Marshal #1 | handleTrade 构造 req 后 marshal |
| `vm_live_session.go:84` | Unmarshal #2 | Start unmarshal req |
| `vm_live_session.go:127` | Marshal #3 | Start marshal resp |
| `vm_live_session.go:136` | Unmarshal #2 | SendEvent unmarshal req |
| `vm_live_session.go:141` | Marshal #3 | SendEvent marshal resp |
| `live_context.go:149` | Unmarshal #4 | dispatchFromBytes unmarshal resp |

## 2. 修复方案

### 2.1 核心变更：Session interface 改为传结构体指针

**当前**（`vm_live_session.go:16-20`）：
```go
type Session interface {
    Start(ctx context.Context, reqBytes []byte) ([]byte, error)
    SendEvent(ctx context.Context, reqBytes []byte) ([]byte, error)
    Close() error
}
```

**修复后**：
```go
type Session interface {
    Start(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error)
    SendEvent(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error)
    Close() error
}
```

**理由**：
- `VMLiveSession` 是同进程实现，不需要序列化
- 直接传 `*antv1.ExecuteLiveRequest` / `*antv1.ExecuteLiveResponse` 指针，Go 的空切片保持空切片（不会变 nil）
- `antv1.ExecuteLiveRequest` 仍是 proto 消息类型（ConnectRPC 端点 `ExecuteLive` 仍用 proto），但进程内传递不需要 marshal
- 消除 4 次序列化 + proto3 nil 语义问题

### 2.2 不变的部分

- `ExecuteLive` RPC handler（`strategy_execution_handler.go:340`）保持 proto——ConnectRPC 自动处理序列化
- `executeVMLive` / `dispatchVMLive`（`vm_live_dispatch.go`）保持 proto——这是 RPC 端点的内部实现
- `antv1.ExecuteLiveRequest` / `antv1.ExecuteLiveResponse` proto 消息类型不变

### 2.3 不做的

- **不改 posCache 为增量通知**——性能可接受（positions 数量少），重构成本高
- **不删除 `executeVMLive` / `dispatchVMLive`**——RPC 端点可能用于 paper mode 单次测试，单独清理
- **不改 `Session` interface 为泛型**——YAGNI，当前只有 `VMLiveSession` 一个实现

## 3. 施工步骤

### S1: 修改 `Session` interface 签名

**文件**: `internal/connect/strategy/vm_live_session.go:16-20`

**改动**：
```go
type Session interface {
    Start(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error)
    SendEvent(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error)
    Close() error
}
```

### S2: 修改 `VMLiveSession.Start` 签名

**文件**: `internal/connect/strategy/vm_live_session.go:78-128`

**当前**：
```go
func (s *VMLiveSession) Start(ctx context.Context, reqBytes []byte) ([]byte, error) {
    var req antv1.ExecuteLiveRequest
    if err := proto.Unmarshal(reqBytes, &req); err != nil {
        return nil, fmt.Errorf("unmarshal request: %w", err)
    }
    // ... 逻辑 ...
    resp := s.dispatch(ctx, &req)
    return proto.Marshal(resp)
}
```

**修复后**：
```go
func (s *VMLiveSession) Start(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
    if s.started {
        return nil, fmt.Errorf("vm live session already started")
    }
    bctx := req.GetBarContext()
    if bctx == nil {
        return nil, fmt.Errorf("first request must have bar_context for initialization")
    }
    // ... 其余逻辑不变，直接用 req ...
    // ... 省略 runner.New / SetStrategy / validateFirstBarContext / SetLogin / SetAccountStatus / Init ...
    s.started = true
    resp := s.dispatch(ctx, req)
    return resp, nil
}
```

**消除**：`proto.Unmarshal`（:84）+ `proto.Marshal`（:127）

### S3: 修改 `VMLiveSession.SendEvent` 签名

**文件**: `internal/connect/strategy/vm_live_session.go:130-142`

**当前**：
```go
func (s *VMLiveSession) SendEvent(ctx context.Context, reqBytes []byte) ([]byte, error) {
    if !s.started {
        return nil, fmt.Errorf("vm live session not started")
    }
    var req antv1.ExecuteLiveRequest
    if err := proto.Unmarshal(reqBytes, &req); err != nil {
        return nil, fmt.Errorf("unmarshal request: %w", err)
    }
    resp := s.dispatch(ctx, &req)
    return proto.Marshal(resp)
}
```

**修复后**：
```go
func (s *VMLiveSession) SendEvent(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
    if !s.started {
        return nil, fmt.Errorf("vm live session not started")
    }
    return s.dispatch(ctx, req), nil
}
```

**消除**：`proto.Unmarshal`（:136）+ `proto.Marshal`（:141）

### S4: 修改 `handleBar` 调用点

**文件**: `internal/connect/strategy/live_runner_events.go:56-96`

**当前**：
```go
req := &antv1.ExecuteLiveRequest{...}
reqBytes, marshalErr := proto.Marshal(req)
if marshalErr != nil { ... }
var respBytes []byte
if *firstBar {
    vmSess, vmErr := s.initVMSession(ctx, cfg, activeSess)
    ...
    respBytes, err = (*session).Start(ctx, reqBytes)
    *firstBar = false
} else {
    respBytes, err = (*session).SendEvent(ctx, reqBytes)
}
...
s.dispatchFromBytes(ctx, cfg, bar, respBytes, activeSess)
```

**修复后**：
```go
req := &antv1.ExecuteLiveRequest{...}
var resp *antv1.ExecuteLiveResponse
if *firstBar {
    vmSess, vmErr := s.initVMSession(ctx, cfg, activeSess)
    ...
    resp, err = (*session).Start(ctx, req)
    *firstBar = false
} else {
    resp, err = (*session).SendEvent(ctx, req)
}
...
s.dispatchResponse(ctx, cfg, bar, resp, activeSess)
```

**消除**：`proto.Marshal`（:62）

### S5: 修改 `handleTick` 调用点

**文件**: `internal/connect/strategy/live_runner_events.go:158-180`

同 S4 模式：`reqBytes` → `req`，`respBytes` → `resp`，`dispatchFromBytes` → `dispatchResponse`。

**消除**：`proto.Marshal`（:164）

### S6: 修改 `handleTrade` 调用点

**文件**: `internal/connect/strategy/live_runner_events.go:199-221`

同 S4 模式。

**消除**：`proto.Marshal`（:205）

### S7: 重命名 `dispatchFromBytes` → `dispatchResponse`

**文件**: `internal/connect/strategy/live_context.go:146-184`

**当前**：
```go
func (s *StrategyExecutionServer) dispatchFromBytes(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, respBytes []byte, activeSess *ActiveSession) {
    var resp antv1.ExecuteLiveResponse
    if err := proto.Unmarshal(respBytes, &resp); err != nil {
        ...
    }
    if !resp.GetSuccess() { ... }
    ...
}
```

**修复后**：
```go
func (s *StrategyExecutionServer) dispatchResponse(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, resp *antv1.ExecuteLiveResponse, activeSess *ActiveSession) {
    if resp == nil {
        s.log.Error("LiveStrategyRunner: nil response from VM")
        if activeSess != nil {
            activeSess.RecordError("nil response from VM")
        }
        return
    }
    if !resp.GetSuccess() { ... }
    ...
}
```

**消除**：`proto.Unmarshal`（:149）

### S8: 清理 `proto` import

**文件**: `internal/connect/strategy/vm_live_session.go` / `live_runner_events.go` / `live_context.go`

移除不再使用的 `"google.golang.org/protobuf/proto"` import（如果文件中无其他 proto 用法）。

**注意**：`live_context.go` 可能仍有其他 proto 用法，需检查。`vm_live_session.go` 的 `dispatch` 方法不需要 proto（直接传 `*antv1.ExecuteLiveRequest`）。

### S9: 更新测试

**文件**: 以下测试文件直接调用 `Session.Start` / `SendEvent`，需更新签名：
- `live_harness_parity_test.go:318,336`
- `live_indicator_freeze_test.go:95,111,131`
- `live_integration_test.go:130`

**模式**：
```go
// 当前
reqBytes, _ := proto.Marshal(req)
respBytes, err := vmSess.Start(ctx, reqBytes)
var resp antv1.ExecuteLiveResponse
proto.Unmarshal(respBytes, &resp)

// 修复后
resp, err := vmSess.Start(ctx, req)
```

### S10: 对抗证明

**目标**：验证修复后空切片 positions 不再变 nil。

**测试**：在 `vm_live_session_test.go` 新增 `TestVMLiveSession_NilPositionsSurviveRoundTrip`：
```go
func TestVMLiveSession_NilPositionsSurviveRoundTrip(t *testing.T) {
    // 构造 req with Positions = []*antv1.LivePosition{}（空切片，非 nil）
    // 调用 SendEvent
    // 验证 vmHandleTick 收到的 tctx.Positions 是 nil 或空切片（都表示无持仓）
    // 验证 resp.Success == true（不再被 rejectNilRepeatedInLive 误拒）
}
```

**Mutation**：如果恢复 proto marshal/unmarshal round-trip，空切片会变 nil，测试应 RED。

## 4. 验收标准

### 4.1 机检五件套

- `go build ./...` 通过
- `go vet ./...` 通过
- `go test ./internal/connect/strategy/ -count=1` 通过
- `go test -race ./internal/connect/strategy/ -count=3` 通过
- `go run ./tools/check-file-lines --strict` 零警告

### 4.2 对抗证明

- S10 测试 GREEN
- 恢复 proto round-trip 后 S10 测试 RED

### 4.3 无回归

- 所有现有测试通过（包括 `vm_trade_context6_batch2_test.go` / `vm_api_truth3_batch3_test.go`）
- `executeVMLive` / `dispatchVMLive` 路径不受影响（RPC 端点仍用 proto）

## 5. 影响评估

### 5.1 性能

消除 4 次 proto 序列化/tick 事件。tick 频率高（每秒多次），序列化成本包括 allocations + reflection。修复后零序列化成本。

### 5.2 正确性

- proto3 nil/空切片不可区分问题消除（Go 结构体指针传递保持空切片语义）
- 未来不会在其他 repeated 字段复发同类 bug

### 5.3 兼容性

- `ExecuteLive` RPC 端点不变（ConnectRPC 自动序列化）
- `executeVMLive` / `dispatchVMLive` 不变（RPC 内部实现）
- proto 消息类型不变（`antv1.ExecuteLiveRequest` / `antv1.ExecuteLiveResponse`）

### 5.4 风险

- `Session` interface 签名变更影响所有实现和 mock
- 当前只有 `VMLiveSession` 一个实现，无 mock session（grep 确认）
- 测试文件直接调用 `Start` / `SendEvent`，需同步更新（S9）

## 6. 文件清单

| 文件 | 改动类型 |
|------|---------|
| `internal/connect/strategy/vm_live_session.go` | S1 interface + S2 Start + S3 SendEvent + S8 import |
| `internal/connect/strategy/live_runner_events.go` | S4 handleBar + S5 handleTick + S6 handleTrade + S8 import |
| `internal/connect/strategy/live_context.go` | S7 dispatchResponse + S8 import |
| `internal/connect/strategy/live_harness_parity_test.go` | S9 测试更新 |
| `internal/connect/strategy/live_indicator_freeze_test.go` | S9 测试更新 |
| `internal/connect/strategy/live_integration_test.go` | S9 测试更新 |
| `internal/connect/strategy/vm_live_session_test.go` | S10 对抗证明（新增） |

## 7. 不做

- 不改 posCache 为增量通知
- 不删除 `executeVMLive` / `dispatchVMLive`
- 不改 `Session` interface 为泛型
- 不改 proto 消息类型
- 不部署（施工完成后停手等 Devin CLI 复审）
