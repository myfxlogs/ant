# 12 · mthub 规范（会话与下单中心）

> 路径：`backend/internal/mthub/`
> 目标 LOC：≤ 600 行（含测试 ≤ 1200）
> 上游：业务层（oms/marketplace/ai）；下游：`adapter/mt[45]/executor.go`

## 1. 职责

mthub 是 ant 唯一的"会话注册中心"。所有 MT 账户的：
- session 缓存（避免每次下单重新登录）
- 下单（PlaceOrder）
- 撤单（CloseOrder）
- 历史/持仓查询（FetchOrderHistory / FetchOpenedOrders）
- broker 推送的订单事件 fan-in（OrderEventBroker）

**不在 mthub 范围**：
- ❌ 行情订阅（mdgateway 负责）
- ❌ 风控判断（risk 包负责）
- ❌ 订单状态机（oms 包负责）
- ❌ symbol 规范化（mdgateway/normalizer 已做）

## 2. 文件结构

```
backend/internal/mthub/
├── service.go         ≤200 lines  MtHubService（业务层调用入口）
├── hub.go             ≤150 lines  Hub（session 注册 + 复用）
├── events.go          ≤180 lines  OrderEventBroker（fan-in/fan-out）
├── executor.go        ≤80 lines   OrderExecutor 接口定义
├── session.go         ≤100 lines  Session 类型 + 续期逻辑
├── types.go           ≤150 lines  OrderRequest/OrderRecord/SymbolParam 等
├── metrics.go         ≤80 lines
└── *_test.go
```

## 3. 类型契约（`types.go`）

```go
package mthub

import (
    "time"
    "github.com/shopspring/decimal"
)

type OrderRequest struct {
    AccountID  string
    Canonical  string  // 规范化后的 symbol；mthub 内部转 broker raw
    Side       Side    // Buy / Sell
    OrderType  OrderType  // Market / Limit / Stop / StopLimit
    Volume     decimal.Decimal  // lots
    Price      decimal.Decimal  // limit/stop price (may be zero for market)
    StopLoss   decimal.Decimal
    TakeProfit decimal.Decimal
    Comment    string
    ClientID   string  // ant 内部唯一 ID（用于去重）
    Magic      int32
}

type OrderRecord struct {
    Ticket       int64
    AccountID    string
    SymbolRaw    string
    Canonical    string
    Side         Side
    OrderType    OrderType
    Volume       decimal.Decimal
    OpenPrice    decimal.Decimal
    OpenTime     time.Time
    ClosePrice   decimal.Decimal
    CloseTime    time.Time  // zero if open
    Profit       decimal.Decimal
    Commission   decimal.Decimal
    Swap         decimal.Decimal
    Comment      string
    Magic        int32
    State        OrderState
}

type SymbolParam struct {
    Canonical    string
    SymbolRaw    string  // broker 的原生名
    Digits       int32
    PointValue   decimal.Decimal
    LotSize      decimal.Decimal
    LotStep      decimal.Decimal
    LotMin       decimal.Decimal
    LotMax       decimal.Decimal
    SpreadFloat  bool
    TradeMode    int32  // 0=disabled, 1=long_only, 2=short_only, 3=close_only, 4=full
    StopLevel    int32  // 最小止损/止盈距离（点）
}

type Side int8
const (
    SideBuy Side = 1
    SideSell Side = -1
)

type OrderType int8
const (
    OrderMarket OrderType = iota
    OrderLimit
    OrderStop
    OrderStopLimit
)

type OrderState int8
const (
    OrderStatePending OrderState = iota
    OrderStateOpen
    OrderStateClosed
    OrderStateCancelled
    OrderStateRejected
)
```

## 4. OrderExecutor 接口（`executor.go`）

```go
// OrderExecutor 由 adapter/mt4 与 adapter/mt5 实现。
// mthub 只持有此接口，不知道具体平台。
type OrderExecutor interface {
    Platform() string  // "mt4" / "mt5"

    PlaceOrder(ctx context.Context, req *OrderRequest) (ticket int64, err error)
    CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error
    ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error

    FetchOpenedOrders(ctx context.Context) ([]*OrderRecord, error)
    FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*OrderRecord, error)
    FetchSymbolParams(ctx context.Context, canonicals []string) ([]*SymbolParam, error)
    FetchPriceHistory(ctx context.Context, canonical, period string, from, to time.Time) ([]*OHLCV, error)

    SubscribeOrderEvents(ctx context.Context, h OrderEventHandler) error
}

type OrderEventHandler func(*OrderEvent)

type OrderEvent struct {
    AccountID  string
    Ticket     int64
    EventType  string  // "open" | "close" | "modify" | "delete"
    Order      *OrderRecord  // snapshot at event
    Timestamp  time.Time
}
```

## 5. Hub（`hub.go`）

```go
type Hub struct {
    mu        sync.RWMutex
    sessions  map[string]*Session  // accountID → Session
    executors map[string]OrderExecutor  // accountID → executor
    log       *zap.Logger
}

// Register 在 mdgateway 启动账户后注册到 hub。
func (h *Hub) Register(accountID string, sess *Session, exec OrderExecutor)

// EnsureSession 由业务层调用：返回有效 session，过期则续期。
func (h *Hub) EnsureSession(ctx context.Context, accountID string) (*Session, error)

// CloseSession 主动登出。
func (h *Hub) CloseSession(ctx context.Context, accountID string) error

// Get returns the executor for accountID; nil if not registered.
func (h *Hub) Get(accountID string) OrderExecutor
```

**关键约束**：mthub 与 mdgateway 共享 session（同一个账户只有一份连接）。在 mdgateway.Manager.AddGateway 后必须立即调用 `Hub.Register`。

## 6. OrderEventBroker（`events.go`）

```go
type OrderEventBroker struct {
    mu          sync.RWMutex
    subscribers map[string][]chan *OrderEvent  // userID → channels（多订阅）
    log         *zap.Logger
}

// PublishEvent 由 adapter 通过 OrderEventHandler 调用，
// 内部根据 accountID → userID 映射 fan-in 到所有订阅该 user 的 channel。
func (b *OrderEventBroker) PublishEvent(ev *OrderEvent)

// Subscribe 业务层订阅某 user 的所有账户事件。
// 返回 channel 与取消函数。
// channel 容量 = 64；满了丢弃最旧（with metric）。
func (b *OrderEventBroker) Subscribe(userID string) (<-chan *OrderEvent, func())
```

**fan-in 关键**：用户绑定多个 MT 账户（多 broker），订阅 oms.events.{user_id}（SSE）能拿到全部账户事件。

## 7. MtHubService（`service.go`，业务层 facade）

```go
type MtHubService struct {
    hub    *Hub
    events *OrderEventBroker
    log    *zap.Logger
}

// 业务层只用这一个 service，不直接接触 Hub/OrderEventBroker。

func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error)
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error
func (s *MtHubService) FetchOpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error)
func (s *MtHubService) FetchOrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error)
func (s *MtHubService) FetchSymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error)
func (s *MtHubService) PriceHistory(ctx context.Context, accountID, canonical, period string, from, to time.Time) ([]*OHLCV, error)

// SubscribeUserOrderEvents 返回该用户所有 MT 账户的订单事件流。
// 业务层（SSE）持有此 channel；ctx 取消时自动 unsubscribe。
func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, error)
```

## 8. 与 ConnectRPC 的映射

`proto/ant/v1/mthub_service.proto`（M7-rewrite 卡片产出）定义如下 RPC：

```proto
service MtHubService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc CloseOrder(CloseOrderRequest) returns (CloseOrderResponse);
  rpc OpenedOrders(OpenedOrdersRequest) returns (OpenedOrdersResponse);
  rpc OrderHistory(OrderHistoryRequest) returns (OrderHistoryResponse);
  rpc SymbolParams(SymbolParamsRequest) returns (SymbolParamsResponse);
  rpc PriceHistory(PriceHistoryRequest) returns (PriceHistoryResponse);
  rpc StreamOrderEvents(StreamOrderEventsRequest) returns (stream OrderEvent);  // SSE
}
```

`internal/connect/mthub_handler.go` 仅做：
1. 参数校验（required fields）
2. 调用 `MtHubService` 对应方法
3. proto ↔ 内部类型转换
4. error → connect.Code 映射

## 9. 错误处理

所有 mthub 错误必须用 `internal/errs/`，**禁裸字符串**。错误码清单：

| Code | 含义 | HTTP/Connect Code |
|---|---|---|
| `MTHUB_NOT_REGISTERED` | accountID 未在 hub 注册 | NotFound |
| `MTHUB_SESSION_EXPIRED` | session 续期失败 | Unauthenticated |
| `MTHUB_BROKER_REJECTED` | broker 返回拒单 | FailedPrecondition |
| `MTHUB_INVALID_VOLUME` | volume 不符合 LotMin/LotMax/LotStep | InvalidArgument |
| `MTHUB_TRADE_DISABLED` | symbol TradeMode != 4 | FailedPrecondition |
| `MTHUB_INSUFFICIENT_MARGIN` | broker margin 不足 | FailedPrecondition |
| `MTHUB_TIMEOUT` | mtapi 请求超时 | DeadlineExceeded |

## 10. Metrics（`metrics.go`）

| 指标 | 类型 | Labels |
|---|---|---|
| `mthub_orders_placed_total` | Counter | broker, status={ok,rejected,err} |
| `mthub_orders_closed_total` | Counter | broker, status |
| `mthub_place_latency_seconds` | Histogram | broker |
| `mthub_session_active` | Gauge | account_id, broker |
| `mthub_event_published_total` | Counter | event_type |
| `mthub_event_subscriber_count` | Gauge | user_id |
| `mthub_event_dropped_total` | Counter | reason={chan_full} |

## 11. 验收命令

```bash
# 编译 + 测试
( cd backend && go build ./internal/mthub/... \
                && go test -race -cover ./internal/mthub/... )

# LOC
LOC=$(find backend/internal/mthub -name "*.go" -not -name "*_test.go" \
       | xargs wc -l | tail -1 | awk '{print $1}')
test "$LOC" -le 600 || { echo "LOC=$LOC > 600"; exit 1; }

# 必有文件
for f in service.go hub.go events.go executor.go session.go types.go metrics.go; do
  test -f "backend/internal/mthub/$f" || exit 1
done

# 不许 import mtapi proto（应通过 OrderExecutor 接口隔离）
! grep -rE 'anttrader/gen/proto/(mt4|mt5)' backend/internal/mthub/ \
  || { echo "FAIL: mthub imports mtapi proto directly"; exit 1; }
```
