# 14 · ConnectRPC 契约（mthub.v1 + market.v1）

> 路径：`proto/ant/v1/`
> 生成：`make proto` → `backend/gen/proto/ant/v1/` + `frontend/src/gen/ant/v1/`
> 目标：v2 新增 2 个 service，前端可直接调用
>
> **前置依赖**：本 spec 假设 `proto/ant/v1/` 目录已存在（v1 是平铺 `proto/*.proto`）。完成 ROADMAP M7.0-7 后再开工本 spec 涉及的 M7.2-* 卡片。

## 1. 文件清单

```
proto/ant/v1/
├── mthub_service.proto      会话与下单
├── market_service.proto     行情查询（CH 读路径）
├── common.proto             共享类型（已有，扩展即可）
└── ...（其他已有）
```

## 2. mthub_service.proto

```proto
syntax = "proto3";
package ant.v1;

option go_package = "anttrader/gen/proto/ant/v1;antv1";

import "ant/v1/common.proto";
import "google/protobuf/timestamp.proto";

service MtHubService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc CloseOrder(CloseOrderRequest) returns (CloseOrderResponse);
  rpc OpenedOrders(OpenedOrdersRequest) returns (OpenedOrdersResponse);
  rpc OrderHistory(OrderHistoryRequest) returns (OrderHistoryResponse);
  rpc SymbolParams(SymbolParamsRequest) returns (SymbolParamsResponse);
  rpc PriceHistory(PriceHistoryRequest) returns (PriceHistoryResponse);

  // SSE：用户所有 MT 账户的订单事件流
  rpc StreamOrderEvents(StreamOrderEventsRequest) returns (stream OrderEvent);
}

// ----- Place / Close -----

message PlaceOrderRequest {
  string account_id   = 1;  // ant 账户 UUID
  string canonical    = 2;  // 例如 "BTCUSD"
  Side   side         = 3;
  OrderType order_type = 4;
  string volume       = 5;  // decimal string，如 "0.10"
  string price        = 6;  // decimal string；market 单留空
  string stop_loss    = 7;
  string take_profit  = 8;
  string comment      = 9;
  string client_id    = 10; // 业务侧去重用
  int32  magic        = 11;
}

message PlaceOrderResponse {
  int64  ticket = 1;        // broker 返回的 ticket
  string status = 2;        // "submitted" | "filled" | "rejected"
}

message CloseOrderRequest {
  string account_id = 1;
  int64  ticket     = 2;
  string lots       = 3;    // decimal；空 = 全平
}

message CloseOrderResponse {
  string status = 1;
}

// ----- Query -----

message OpenedOrdersRequest {
  string account_id = 1;
}

message OpenedOrdersResponse {
  repeated OrderRecord orders = 1;
}

message OrderHistoryRequest {
  string account_id = 1;
  google.protobuf.Timestamp from = 2;
  google.protobuf.Timestamp to   = 3;
}

message OrderHistoryResponse {
  repeated OrderRecord orders = 1;
}

message SymbolParamsRequest {
  string account_id = 1;
  repeated string canonicals = 2;  // 空 = 返回该账户所有
}

message SymbolParamsResponse {
  repeated SymbolParam params = 1;
}

message PriceHistoryRequest {
  string account_id = 1;
  string canonical  = 2;
  string period     = 3;            // "1m"/"5m"/...
  google.protobuf.Timestamp from = 4;
  google.protobuf.Timestamp to   = 5;
  int32  limit      = 6;            // 0 = 默认 1000
}

message PriceHistoryResponse {
  repeated OHLCV bars = 1;
}

// ----- Stream -----

message StreamOrderEventsRequest {
  // user_id 从 ConnectRPC interceptor 取，不在 request 里
}

message OrderEvent {
  string account_id = 1;
  int64  ticket     = 2;
  string event_type = 3;       // "open" | "close" | "modify" | "delete"
  OrderRecord order = 4;
  google.protobuf.Timestamp timestamp = 5;
}

// ----- Shared -----

enum Side {
  SIDE_UNSPECIFIED = 0;
  SIDE_BUY  = 1;
  SIDE_SELL = 2;
}

enum OrderType {
  ORDER_TYPE_UNSPECIFIED = 0;
  ORDER_TYPE_MARKET      = 1;
  ORDER_TYPE_LIMIT       = 2;
  ORDER_TYPE_STOP        = 3;
  ORDER_TYPE_STOP_LIMIT  = 4;
}

enum OrderState {
  ORDER_STATE_UNSPECIFIED = 0;
  ORDER_STATE_PENDING     = 1;
  ORDER_STATE_OPEN        = 2;
  ORDER_STATE_CLOSED      = 3;
  ORDER_STATE_CANCELLED   = 4;
  ORDER_STATE_REJECTED    = 5;
}

message OrderRecord {
  int64  ticket        = 1;
  string account_id    = 2;
  string symbol_raw    = 3;
  string canonical     = 4;
  Side   side          = 5;
  OrderType order_type = 6;
  string volume        = 7;
  string open_price    = 8;
  google.protobuf.Timestamp open_time = 9;
  string close_price   = 10;
  google.protobuf.Timestamp close_time = 11;
  string profit        = 12;
  string commission    = 13;
  string swap          = 14;
  string comment       = 15;
  int32  magic         = 16;
  OrderState state     = 17;
}

message SymbolParam {
  string canonical    = 1;
  string symbol_raw   = 2;
  int32  digits       = 3;
  string point_value  = 4;
  string lot_size     = 5;
  string lot_step     = 6;
  string lot_min      = 7;
  string lot_max      = 8;
  bool   spread_float = 9;
  int32  trade_mode   = 10;
  int32  stop_level   = 11;
}

message OHLCV {
  google.protobuf.Timestamp open_time  = 1;
  google.protobuf.Timestamp close_time = 2;
  string open   = 3;
  string high   = 4;
  string low    = 5;
  string close  = 6;
  double volume = 7;
  uint32 tick_count = 8;
}
```

## 3. market_service.proto

```proto
syntax = "proto3";
package ant.v1;

option go_package = "anttrader/gen/proto/ant/v1;antv1";

import "ant/v1/common.proto";
import "google/protobuf/timestamp.proto";

service MarketService {
  // 从 ClickHouse md_bars 读取（取代 v1 的 kline_service）
  rpc GetKlines(GetKlinesRequest) returns (GetKlinesResponse);

  // 聚合统计：tick rate / 涨跌幅 / spread
  rpc GetSymbolStats(GetSymbolStatsRequest) returns (GetSymbolStatsResponse);

  // SSE：实时报价订阅
  rpc StreamTicks(StreamTicksRequest) returns (stream TickMsg);
}

message GetKlinesRequest {
  string canonical = 1;
  string broker    = 2;            // 空 = 用户首选 broker
  string period    = 3;            // "1m","5m","15m","1h","4h","1d"
  google.protobuf.Timestamp from = 4;
  google.protobuf.Timestamp to   = 5;
  int32  limit     = 6;
}

message GetKlinesResponse {
  repeated OHLCV bars = 1;
}

message GetSymbolStatsRequest {
  string canonical = 1;
  string broker    = 2;
}

message GetSymbolStatsResponse {
  string current_bid = 1;
  string current_ask = 2;
  string change_24h_pct = 3;
  string spread = 4;
  uint64 tick_rate_1m = 5;
}

message StreamTicksRequest {
  repeated string canonicals = 1;
  string broker = 2;
}

message TickMsg {
  string canonical = 1;
  string broker    = 2;
  int64  ts_unix_ms = 3;
  string bid = 4;
  string ask = 5;
}
```

## 4. ConnectRPC handler 模式（参考实现）

`backend/internal/connect/mthub_service.go`：

```go
package connect

import (
    "context"
    "connectrpc.com/connect"
    "github.com/shopspring/decimal"
    antv1 "anttrader/gen/proto/ant/v1"
    antv1c "anttrader/gen/proto/ant/v1/antv1connect"
    "anttrader/internal/mthub"
    "anttrader/internal/errs"
)

type MtHubServer struct {
    svc *mthub.MtHubService
}

var _ antv1c.MtHubServiceHandler = (*MtHubServer)(nil)

func (s *MtHubServer) PlaceOrder(
    ctx context.Context,
    req *connect.Request[antv1.PlaceOrderRequest],
) (*connect.Response[antv1.PlaceOrderResponse], error) {
    msg := req.Msg

    if msg.AccountId == "" {
        return nil, errs.New(errs.CodeInvalidArgument, "account_id required")
    }
    vol, err := decimal.NewFromString(msg.Volume)
    if err != nil {
        return nil, errs.Wrap(err, errs.CodeInvalidArgument, "invalid volume")
    }

    rec, err := s.svc.PlaceOrder(ctx, &mthub.OrderRequest{
        AccountID: msg.AccountId,
        Canonical: msg.Canonical,
        Side:      sideFromProto(msg.Side),
        OrderType: orderTypeFromProto(msg.OrderType),
        Volume:    vol,
        // ...
    })
    if err != nil {
        return nil, errToConnect(err)
    }

    return connect.NewResponse(&antv1.PlaceOrderResponse{
        Ticket: rec.Ticket,
        Status: stateString(rec.State),
    }), nil
}

// 其他方法略
```

## 5. SSE 实现（StreamOrderEvents）

```go
func (s *MtHubServer) StreamOrderEvents(
    ctx context.Context,
    req *connect.Request[antv1.StreamOrderEventsRequest],
    stream *connect.ServerStream[antv1.OrderEvent],
) error {
    userID := userIDFromCtx(ctx)  // interceptor 注入
    if userID == "" {
        return errs.New(errs.CodeUnauthenticated, "")
    }

    ch, cancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
    defer cancel()

    for {
        select {
        case <-ctx.Done():
            return nil
        case ev, ok := <-ch:
            if !ok {
                return nil
            }
            if err := stream.Send(toProtoOrderEvent(ev)); err != nil {
                return err
            }
        }
    }
}
```

## 6. 鉴权 / interceptor

ConnectRPC interceptor 链（从外到内）：
1. `RecoverInterceptor` — panic → connect.Internal
2. `TraceInterceptor` — 注入 trace_id 到 ctx
3. `AuthInterceptor` — 校验 JWT → 注入 user_id
4. `LoggingInterceptor` — 结构化日志
5. `MetricsInterceptor` — Prometheus 计数

## 7. proto 改动规则

- 严禁删除 field（用 `reserved`）
- 禁止改 field number
- 新增 enum value 必须放在末尾
- 新增 RPC 必须 `make proto-breaking` 通过

## 8. 验收命令

```bash
# proto 文件存在
test -f proto/ant/v1/mthub_service.proto
test -f proto/ant/v1/market_service.proto

# 生成
make proto

# 编译
( cd backend && go build ./gen/proto/... && go build ./internal/connect/... )

# 前端 client 类型
test -f frontend/src/gen/ant/v1/mthub_service_pb.ts
test -f frontend/src/gen/ant/v1/market_service_pb.ts

# breaking
make proto-breaking || exit 1
```
