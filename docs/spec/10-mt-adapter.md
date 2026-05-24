# 10 · MT 适配层规范（adapter/mt4 + adapter/mt5）

> 路径：`backend/internal/mdgateway/adapter/`
> 目标 LOC（非测试）：mt4 gateway+executor + mt5 gateway+executor + mdtick 共 ≤ 400 行；含测试 ≤ 800 行
> LOC 预算分配（总 1500 / AGENT.md §5）：mdgateway 800 + adapter 400 + mthub 300 = 1500

## 1. 范围与禁忌

### 1.1 适配层做什么
- 把 mtapi gRPC 的原始 proto 翻译成 ant 内部 DTO
- 维护单个 MT 账户的 gRPC 连接生命周期
- 把 broker 推送的 `OnQuote` 转成 ant `Tick`
- 把 ant `OrderRequest` 转成 mtapi `OrderSendRequest`

### 1.2 适配层不做什么（违反 = 卡片作废）
- ❌ canonical 转换（在 mdgateway/normalizer.go 做）
- ❌ Quality 检查（在 mdgateway/quality.go 做）
- ❌ Bar 聚合（在 mdgateway/bar_aggregator.go 做）
- ❌ Tick 去重（在 mdgateway/tick_dedup.go 做）
- ❌ NATS 发布、CH 写入（在 mdgateway/publisher/clickhouse_writer 做）
- ❌ 业务逻辑（订单状态机、风控）

## 2. 文件结构

```
backend/internal/mdgateway/adapter/
├── mdtick/                  共享 DTO + Normalizer 接口（不依赖 mt4/mt5）
│   ├── mdtick.go           Tick / Bar / Money / AccountConfig 类型
│   ├── normalizer.go       CanonicalResolver 接口
│   └── mdtick_test.go
├── mt4/
│   ├── gateway.go          MT4 gateway（实现 mdgateway.Gateway 接口）
│   ├── executor.go         MT4 OrderExecutor 实现
│   └── gateway_test.go
└── mt5/
    ├── gateway.go          MT5 gateway
    ├── executor.go         MT5 OrderExecutor 实现
    └── gateway_test.go
```

## 3. 核心类型（`adapter/mdtick/mdtick.go`）

```go
// Package mdtick provides shared DTOs for mdgateway adapters.
// This package MUST NOT import mdgateway, mt4, or mt5.
package mdtick

import "github.com/shopspring/decimal"

// Tick is the canonical tick representation flowing into mdgateway.
type Tick struct {
    UserID         string          // ant 用户 ID
    AccountID      string          // ant 账户 UUID
    Broker         string          // broker 唯一标识
    Platform       string          // "mt4" or "mt5"
    SymbolRaw      string          // broker 原生 symbol（如 "BTCUSDm"）
    Canonical      string          // 规范化后（如 "BTCUSD"）；adapter 留空，由 mdgateway 填
    TsUnixMs       int64           // broker 时间戳（毫秒，UTC）
    ArrivedUnixMs  int64           // 本地接收时间（毫秒，UTC）
    Bid            decimal.Decimal
    Ask            decimal.Decimal
    BidVolume      float64
    AskVolume      float64
}

// AccountConfig 来自 PG.mt_accounts_v2 视图；mdgateway.runner 加载并解密后传入 adapter。
// 字段命名与 SQL 列名严格对齐（见 docs/spec/13 §4.1 视图定义）。
type AccountConfig struct {
    AccountID  string  // mt_accounts_v2.id (UUID)
    UserID     string  // mt_accounts_v2.user_id
    Broker     string  // mt_accounts_v2.broker          (来源 broker_company)
    Platform   string  // mt_accounts_v2.platform        ("mt4" / "mt5")
    Login      string  // mt_accounts_v2.login
    Password   string  // password_encrypted 解密后明文（vault.Decrypt）
    Server     string  // mt_accounts_v2.server          (来源 broker_server)
    MtapiHost  string  // mt_accounts_v2.mtapi_host      (来源 broker_host)
    MtapiPort  string  // mt_accounts_v2.mtapi_port
    MtapiToken string  // mtapi_token_encrypted 解密后明文
}

// Bar 由 mdgateway.bar_aggregator 产出，不在 adapter 层
// （此处声明仅供 adapter 单测引用）
type Bar struct {
    UserID         string
    AccountID      string
    Broker         string
    Canonical      string
    Period         string  // "1m"/"5m"/"15m"/"1h"/"4h"/"1d"
    OpenTsUnixMs   int64
    CloseTsUnixMs  int64
    Open, High, Low, Close decimal.Decimal
    Volume         float64
    TickCount      uint32
}
```

**精度规则**：
- Tick 的 Bid/Ask **必须**是 `decimal.Decimal`，**禁止 float64**
- 转换路径：mtapi `string` → `decimal.NewFromString` → ant `decimal.Decimal`
- 失败时丢弃 Tick 并计 `md_tick_dropped_total{reason="parse_error"}`

## 4. Gateway 接口（`mdgateway/manager.go` 定义）

```go
package mdgateway

import (
    "context"
    "anttrader/internal/mdgateway/adapter/mdtick"
)

// Gateway 是 adapter 必须实现的契约。
type Gateway interface {
    // Platform 返回 "mt4" 或 "mt5"。
    Platform() string

    // AccountID 返回此 gateway 绑定的 ant 账户 ID。
    AccountID() string

    // Connect 建立 mtapi gRPC 连接 + 登录 broker。
    // 失败时不要 retry（外层 CircuitBreaker 决策）。
    Connect(ctx context.Context) error

    // Disconnect 关闭连接。idempotent。
    Disconnect(ctx context.Context) error

    // Subscribe 订阅 symbols 的实时报价。
    // 每个 tick 通过 handler 回调（同步调用，handler 不许阻塞）。
    Subscribe(ctx context.Context, symbols []string, handler TickHandler) error

    // HealthCheck 返回 nil 表示账户在线且可用。
    // 用于 MtHubService.GetAccountStatus（决策 RV-C4 后替代原 /livez/account）。
    HealthCheck(ctx context.Context) error

    // SessionID 返回当前 broker session token；未连接时返回空。
    SessionID() string
}

// TickHandler 由 mdgateway.Manager 提供，adapter 在 Subscribe 内调用。
type TickHandler func(t *mdtick.Tick)
```

## 5. MT4 实现规范（`adapter/mt4/gateway.go`）

### 5.1 文件骨架

```go
// Package mt4 provides the MT4 gateway adapter for mdgateway.
package mt4

import (
    "context"
    "fmt"
    "sync"
    "time"

    pb "anttrader/mt4"  // mtapi proto (v2: separate module)
    "anttrader/internal/mdgateway/adapter/mdtick"
    "github.com/shopspring/decimal"
    "go.uber.org/zap"
    "google.golang.org/grpc"
)

type Gateway struct {
    cfg mdtick.AccountConfig
    log *zap.Logger

    mu        sync.RWMutex
    conn      *grpc.ClientConn
    client    pb.MT4APIClient
    sessionID string
    cancelSub context.CancelFunc
}

func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway {
    return &Gateway{cfg: cfg, log: log}
}

func (g *Gateway) Platform() string  { return "mt4" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }

func (g *Gateway) Connect(ctx context.Context) error { /* ... */ }
func (g *Gateway) Disconnect(ctx context.Context) error { /* ... */ }
func (g *Gateway) Subscribe(ctx context.Context, syms []string, h mdgateway.TickHandler) error { /* ... */ }
func (g *Gateway) HealthCheck(ctx context.Context) error { /* ... */ }
func (g *Gateway) SessionID() string { /* ... */ }
```

### 5.2 Subscribe 实现要点（**必须**遵守 quirks register）

```go
func (g *Gateway) Subscribe(ctx context.Context, syms []string, h mdgateway.TickHandler) error {
    subCtx, cancel := context.WithCancel(ctx)
    g.mu.Lock()
    g.cancelSub = cancel
    g.mu.Unlock()

    stream, err := g.client.OnQuoteStream(subCtx, &pb.OnQuoteRequest{
        SessionId: g.sessionID,
        Symbols:   syms,
    })
    if err != nil {
        return fmt.Errorf("mt4 subscribe: %w", err)
    }

    go func() {
        defer stream.CloseSend()
        for {
            quote, err := stream.Recv()
            if err != nil {
                g.log.Warn("mt4 stream recv", zap.Error(err))
                return
            }
            // QUIRK Q-001: MT4 OnQuote.Time may not advance in real-time.
            // Use local arrival time, NOT broker time, for ArrivedUnixMs.
            // See docs/spec/16-mtapi-quirks-register.md#q-001
            arrivedMs := time.Now().UTC().UnixMilli()

            bid, errB := decimal.NewFromString(quote.Bid)
            ask, errA := decimal.NewFromString(quote.Ask)
            if errB != nil || errA != nil {
                // metric increment + drop
                continue
            }

            t := &mdtick.Tick{
                UserID:        g.cfg.UserID,
                AccountID:     g.cfg.AccountID,
                Broker:        g.cfg.Broker,
                Platform:      "mt4",
                SymbolRaw:     quote.Symbol,
                Canonical:     "",  // mdgateway will fill
                TsUnixMs:      quote.Time * 1000,  // mt4 returns seconds
                ArrivedUnixMs: arrivedMs,
                Bid:           bid,
                Ask:           ask,
                BidVolume:     0,  // MT4 不提供
                AskVolume:     0,
            }
            h(t)
        }
    }()

    return nil
}
```

### 5.3 Connect 实现要点

```go
func (g *Gateway) Connect(ctx context.Context) error {
    addr := g.cfg.MtapiHost + ":" + g.cfg.MtapiPort
    conn, err := grpc.DialContext(ctx, addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),  // mtapi.io 用 token 鉴权
        grpc.WithBlock(),
        grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
    )
    if err != nil {
        return fmt.Errorf("mt4 dial %s: %w", addr, err)
    }

    client := pb.NewMT4APIClient(conn)

    // QUIRK Q-002: mtapi.io gRPC metadata must include token for auth.
    // See docs/spec/16-mtapi-quirks-register.md#q-002
    md := metadata.New(map[string]string{"authorization": "Bearer " + g.cfg.MtapiToken})
    callCtx := metadata.NewOutgoingContext(ctx, md)

    resp, err := client.Connect(callCtx, &pb.ConnectRequest{
        Login:    g.cfg.Login,
        Password: g.cfg.Password,
        Server:   g.cfg.Server,
    })
    if err != nil {
        conn.Close()
        return fmt.Errorf("mt4 login: %w", err)
    }

    g.mu.Lock()
    g.conn = conn
    g.client = client
    g.sessionID = resp.SessionId
    g.mu.Unlock()

    return nil
}
```

## 6. MT5 实现规范（`adapter/mt5/gateway.go`）

与 mt4 同构，差异：
- proto: `anttrader/mt5`（v2：独立 Go module）
- `OnTick` 返回 `Bid/Ask/Last/Volume`（含 volume）
- 时间戳已是毫秒（无需 `*1000`）
- `BidVolume/AskVolume` 可填实际值

**禁止**与 mt4 共享代码（除 mdtick 包）。理由：mtapi proto 类型完全独立，强制共享会导致泄漏抽象。

## 7. OrderExecutor 接口（`mthub/executor.go` 定义，adapter 实现）

```go
// adapter/mt4/executor.go
type Executor struct {
    gateway *Gateway  // 复用 gateway 的 conn + sessionID
}

func (e *Executor) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
    // mtapi.OrderSend → 返回 ticket
}

func (e *Executor) CloseOrder(ctx context.Context, ticket int64, lots float64) error { ... }
func (e *Executor) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*mthub.OrderRecord, error) { ... }
func (e *Executor) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) { ... }
func (e *Executor) FetchSymbolParams(ctx context.Context, syms []string) ([]*mthub.SymbolParam, error) { ... }
```

详见 `docs/spec/12-mthub.md` §"OrderExecutor"。

## 8. 测试要求

### 8.1 单元测试（`*_test.go`）

每个 gateway 必须覆盖：
- ✅ Connect 成功 → sessionID 非空
- ✅ Connect 失败 → 返回 wrapped error，conn 已清理
- ✅ Subscribe 收到 quote → 调用 handler
- ✅ Subscribe 收到非法 Bid/Ask → drop 且不调用 handler
- ✅ Disconnect 后再 Connect → idempotent
- ✅ HealthCheck 在断开时返回 error

### 8.2 集成测试

`adapter/mt4/integration_test.go`（build tag `integration`）：
- 启动 dockertest mtapi mock server
- 真实跑 Connect → Subscribe → Disconnect 全流程

```go
//go:build integration
// +build integration

package mt4_test
// ...
```

## 9. 验收命令（卡片完成必跑）

```bash
# 编译 + 单测
( cd backend && go build ./internal/mdgateway/adapter/... \
                && go test -race ./internal/mdgateway/adapter/... )

# LOC 上限
LOC=$(find backend/internal/mdgateway/adapter -name "*.go" -not -name "*_test.go" \
       | xargs wc -l | tail -1 | awk '{print $1}')
test "$LOC" -le 400 || { echo "LOC=$LOC > 400"; exit 1; }

# 不许 import mt4client/mt5client
! grep -r "mt4client\|mt5client" backend/internal/mdgateway/adapter/

# 不许 import mdgateway 的非接口部分
! grep -rE "mdgateway\.(Manager|Quality|BarAggregator|Publisher|CHWriter)" \
    backend/internal/mdgateway/adapter/
```

## 10. 与 quirks register 的强关联

实现 adapter 时必须读完 `docs/spec/16-mtapi-quirks-register.md`，每条 quirk 在代码里**精确引用** quirk ID（注释 `// QUIRK Q-NNN: ...`）。

CI 会做这一项检查：
```bash
# 每个引用 quirk 的注释必须能在 quirks register 找到对应 ID
grep -hoE 'QUIRK Q-[0-9]+' backend/internal/mdgateway/adapter -r \
    | sed 's/QUIRK //' | sort -u | while read q; do
    grep -q "^## $q" docs/spec/16-mtapi-quirks-register.md || { echo "Unknown $q"; exit 1; }
done
```
