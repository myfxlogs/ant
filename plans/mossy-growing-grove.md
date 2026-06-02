# K-line 实时推送 + 动态订阅方案

## 问题

当前 K 线管线的三个缺陷：

1. **Gateway 固定订阅 38 个品种** — 用户选了 ENJUSDm 等未订阅品种，无 tick → 无 bar → PriceHistory 返回空
2. **前端 5s 轮询 PriceHistory** — gRPC 的优势（server streaming）完全没用上
3. **新选品种无历史 bar** — bar 聚合器需要等 ticks 积累才能产出 bar，1h bar 最长等 1 小时

## 架构目标

```
用户选 symbol → SubscribeBars RPC → gateway 动态订阅 symbol
  → tick 流入 → bar 聚合 → 当前未闭合 bar 立即 SSE 推送（前端看到实时蜡烛）
                      → bar 闭合 → finalized bar SSE 推送
                      → bar 写入 ClickHouse（持久化）
  → PriceHistory 拉取历史 bar（初始加载）
  → 移除 5s 轮询
```

## 实现步骤

### Step 1: Gateway 动态订阅单个 symbol

**文件**: `backend/internal/mdgateway/adapter/mt5/quotes.go`、`mt4/quotes.go`

新增 `AddSymbols(ctx, symbols []string) error` 方法：
- 调用已有的 `sub.SubscribeMany`（mtapi 的订阅是增量的）
- 不启动新的 recvLoop — 已有的 OnQuote stream 会自动收到新 symbol 的 tick
- MT4 同样模式（已有的 `sub.SubscribeMany`）

```go
func (g *Gateway) AddSymbols(ctx context.Context, symbols []string) error {
    g.mu.RLock()
    sub := g.subCli
    sid := g.sessionID
    g.mu.RUnlock()
    if sub == nil || sid == "" {
        return fmt.Errorf("mt5 AddSymbols: not connected")
    }
    subMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
    subCtx := metadata.NewOutgoingContext(ctx, subMd)
    _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: symbols})
    return err
}
```

### Step 2: MtHubService 暴露动态订阅

**文件**: `backend/internal/mthub/service.go`

新增方法：
```go
func (s *MtHubService) SubscribeSymbols(ctx context.Context, accountID string, symbols []string) error
```

通过 hub.Get(accountID) 获取 executor，调用 `exec.AddSymbols(ctx, symbols)`。

**接口**: 在 `backend/internal/mdgateway/adapter/` 的 Executor 接口新增 `AddSymbols(ctx, symbols) error`。

### Step 3: 新增 SubscribeBars RPC

**文件**: `backend/internal/connect/system/mthub_service_extra.go`

```go
func (s *MtHubServer) SubscribeBars(ctx context.Context, req *connect.Request[antv1.SubscribeBarsRequest]) (*connect.Response[antv1.SubscribeBarsResponse], error)
```

- 校验 `accountId` 属于当前用户
- 调用 `s.svc.SubscribeSymbols(ctx, accountId, []string{symbol})`
- 返回成功

**Proto**: 已有 `mthub_service.proto`，新增 message：
```proto
message SubscribeBarsRequest {
  string account_id = 1;
  string symbol = 2;
}
message SubscribeBarsResponse {}
```

**前端**: `tradingClient.subscribeBars({ accountId, symbol })`

### Step 4: Bar 实时 SSE 推送

**4a. Bar Broker** (`backend/internal/mthub/types.go`)

新增 `BarBroker`（复用已有 broker 模式，keyed by `accountID`）：
```go
type BarUpdate struct {
    AccountID string
    Symbol    string
    Period    string   // "1m", "5m", "1h" ...
    Bar       Bar      // OHLCV + timestamp
    Closed    bool     // true=已闭合, false=进行中
}

type BarBroker struct {
    mu          sync.RWMutex
    subscribers map[string][]chan *BarUpdate  // key: accountID
}
```

Publish/Subscribe 方法复用已有模式（`PositionSnapshotBroker` 同款）。

**4b. Bar 发布点** (`backend/internal/mdgateway/manager.go`)

在 `HandleTick` 中，bar 聚合器的 `onBar` 回调已有 `bars []*mdtick.Bar`。在此处新增：`m.barBroker.Publish(barUpdate)`。

同时，对于**当前未闭合的 bar**，也在每次 tick 到达时发布实时更新：
- 在 `HandleTick` 中 tick 处理完后，将当前 openBar 的状态转换为 `BarUpdate{Closed: false}` 并 publish。

**4c. SSE handler** (`backend/internal/connect/system/stream_handler.go`)

在 `SubscribeEvents` 的 select loop 中新增 bar 事件处理：
```go
case bar := <-barCh:
    sendEvent("bar_update", &BarUpdatePayload{...})
```

**4d. Proto** (`proto/ant/v1/stream.proto`)

`StreamEvent.payload` oneof 新增：
```proto
BarUpdate bar_update = 5;
```
`BarUpdate` message 包含 account_id, symbol, period, open_time, open, high, low, close, volume, closed(bool)。

### Step 5: 前端 SSE 订阅 bar

**5a. Bar 事件订阅** (`frontend/src/client/stream.ts`)

`subscribeEvents` 新增 `onBar` 回调参数。在 switch case 中新增 `"bar_update"` 处理。

**5b. PriceChart 改造** (`frontend/src/components/chart/PriceChart.tsx`)

- 移除 5s 轮询 (`pollRef` / `setInterval`)
- 挂载时：`marketApi.subscribeBars({ accountId, symbol })` + `marketApi.getKlines({...})` 拉初始历史
- symbol 切换时：重新订阅 + 拉历史
- 收到 `onBar` 回调：
  - `Closed: true` → 合并到 bars（已有 dedup 逻辑）
  - `Closed: false` → 更新 klinecharts 的当前蜡烛（`chart.updateCurrentBar()`）
- `handleLoadMore` 保持不变（走 PriceHistory 分页）

**5c. 前端 subscribeBars API** (`frontend/src/client/market.ts`)

```ts
subscribeBars: async (params: { accountId: string; symbol: string }) => {
    await tradingClient.subscribeBars({ accountId: params.accountId, symbol: params.symbol });
}
```

### Step 6: 清理

- 移除 `PriceChart.tsx` 中的 `pollRef` 和轮询逻辑
- 移除 `market.ts` 中不再需要的轮询相关代码
- MT5 quotes.go 中的 debug 日志降级或移除

## 文件改动清单

| 步骤 | 文件 | 改动 |
|------|------|------|
| 1 | `backend/internal/mdgateway/adapter/mt5/quotes.go` | 新增 `AddSymbols` |
| 1 | `backend/internal/mdgateway/adapter/mt4/quotes.go` | 新增 `AddSymbols` |
| 1 | `backend/internal/mdgateway/adapter/` executor interface | 新增 `AddSymbols` |
| 2 | `backend/internal/mthub/service.go` | 新增 `SubscribeSymbols` |
| 3 | `proto/ant/v1/mthub_service.proto` | 新增 `SubscribeBarsRequest/Response` |
| 3 | `backend/internal/connect/system/mthub_service_extra.go` | 新增 `SubscribeBars` handler |
| 4a | `backend/internal/mthub/types.go` | 新增 `BarBroker`, `BarUpdate` |
| 4b | `backend/internal/mdgateway/manager.go` | `HandleTick` 中 publish bar |
| 4c | `backend/internal/connect/system/stream_handler.go` | SSE bar 事件 |
| 4d | `proto/ant/v1/stream.proto` | `BarUpdate` message + oneof |
| 5a | `frontend/src/client/stream.ts` | `subscribeEvents` 加 onBar |
| 5b | `frontend/src/components/chart/PriceChart.tsx` | 移除轮询，接入 SSE bar |
| 5c | `frontend/src/client/market.ts` | 新增 `subscribeBars` |
| 6 | `buf generate` | 重新生成 Go + TS |

## 验证

1. 选 MT5 账号中一个不在固定订阅列表的品种（如 ENJUSDm）→ K 线能出来
2. 选后几秒内看到当前未闭合蜡烛在图表上实时跳动
3. bar 闭合后自动更新，无需手动刷新
4. 向左滚动加载历史 bar 正常
5. 切换 symbol → 旧 symbol 停止推送，新 symbol 立即开始
6. 切换 account → 自动切换对应 broker 的数据
