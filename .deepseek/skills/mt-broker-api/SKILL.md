---
name: mt-broker-api
description: Use when connecting to MT4/MT5 brokers via mtapi.io gRPC for quotes, symbols, or trading. Covers the correct connection flow, symbol fetching, quote retrieval, and common pitfalls based on production debugging experience.
---

# MT4/MT5 Broker API via mtapi.io

> 基于 ant 项目 Exness 真实账户验证的 mtapi 最佳实践。

## 1. 架构关键：mtapi 网关 ≠ broker 服务器

mtapi.io 作为 gRPC 网关代理，不直连 broker。

```
ant adapter → TLS dial mt4grpc3.mtapi.io:443 → Connect RPC(host=broker_ip) → broker
```

- **拨号地址**：`mt4grpc3.mtapi.io:443` 或 `mt5grpc3.mtapi.io:443`（TLS）
- **broker 地址**：数据库 `broker_host` 字段（如 `18.163.85.196`），通过 Connect RPC 的 `Host` 参数传递
- broker_host 可能含端口后缀（如 `18.163.85.196:443`），需剥离后分别传 Host 和 Port

## 2. 连接流程（必须严格遵守）

### 2.1 拨号
```go
conn, err := grpc.DialContext(ctx, "mt4grpc3.mtapi.io:443",
    grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
    grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
)
```

### 2.2 登录（获取 Session Token）
```go
// 生成唯一连接 ID
sessionUUID := uuid.NewString()

// metadata 必须同时包含 authorization 和 id
md := metadata.New(map[string]string{
    "authorization": "Bearer " + mtapiToken,
    "id":            sessionUUID,
})
loginCtx := metadata.NewOutgoingContext(ctx, md)

// Host 和 Port 分离传（不要传 "host:port" 格式）
brokerHost := brokerHostRaw  // 如 "18.163.85.196"
if idx := strings.LastIndex(brokerHost, ":"); idx > 0 {
    brokerHost = brokerHostRaw[:idx]
}

resp, err := connCli.Connect(loginCtx, &pb.ConnectRequest{
    Host:     brokerHost,   // 不含端口
    Port:     443,          // broker MT 端口
    User:     loginNumber,  // MT4: int32, MT5: uint64
    Password: brokerPassword,
    Id:       &sessionUUID, // 必填，与 metadata 中的 id 一致
})
// 检查 resp.GetResult() 非空
token := resp.GetResult()
if token == "" { return error("empty token") }
```

**关键点**：
- `Id` 字段**必须**同时出现在 metadata header 和 ConnectRequest 中
- `ConnectReply.Result` 返回的才是真正的 session token
- 空 token = 登录失败

## 3. 获取 Symbol 列表

```go
// metadata 中必须带 id (session token) 和 authorization
mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
    "id",            sessionToken,
    "authorization", "Bearer "+mtapiToken,
))
resp, err := client.Symbols(mdCtx, &pb.SymbolsRequest{Id: sessionToken})
// MT4: resp.GetResult() → []string  (直接就是 symbol 名称)
// MT5: resp.GetResult() → []*SymbolInfo
//   → 使用 si.GetCurrency() 获取 symbol 代码（如 "BTCUSDm"）
//   → si.GetDescription() 是人类可读标签（如 "Bitcoin vs US Dollar"），不能用于 QuoteMany
```

**关键点**：
- `SymbolsRequest.Id` **和** metadata header 中的 `id` **都必须**传 session token
- 缺少任一个 → 返回空列表

## 4. 获取实时报价

```go
// QuoteMany（批量报价，推荐）
symbols := []string{"BTCUSDm", "ETHUSDm", "XAUUSDm"}  // 注意 broker 实际名称
resp, err := client.GetQuoteMany(mdCtx, &pb.GetQuoteManyRequest{Symbols: symbols})
// resp.GetResult() → []*QuoteEventArgs (MT4) / []*Quote (MT5)
// 每个 quote: Symbol, Bid(float64), Ask(float64), Time(*Timestamp)
```

**OnQuote 流式订阅**：
```go
stream, err := streamCli.OnQuote(mdCtx, &pb.OnQuoteRequest{Id: sessionToken})
// 持续接收所有报价，不在 request 中过滤 symbol
```

## 5. 常见陷阱

| 陷阱 | 错误做法 | 正确做法 |
|---|---|---|
| **网关地址** | 直连 broker IP | 拨号 mtapi.io 网关，Connect RPC 传 broker 地址 |
| **端口重复** | `host:443:443` | broker_host 剥离端口后缀 |
| **session token 为空** | ConnectRequest 不传 Id | Id 必须 = UUID，且与 metadata header 一致 |
| **Symbols 返回空** | 只传 Request.Id | metadata header 也必须含 "id" |
| **symbol 名称** | 用 `BTCUSD` | broker 可能用 `BTCUSDm`（先 Symbols 获取真实名称再使用） |
| **OnQuote 无数据** | 只传 Bearer token | metadata 还需 `"id"` header |
| **QuoteMany 全失败** | 列表含错误 symbol | 一个错误 symbol 会导致整个请求失败 |

## 6. 验证命令

```bash
# 1. 确认连接
ANT_MASTER_KEY=... DB_PASSWORD=... go run ./cmd/live-test/

# 2. 预期输出
# Connected! Session=xxx
# BTCUSDm: bid=76707.91 ask=76723.31 ← 实时价格
# ✅ PIPELINE VERIFIED
```
