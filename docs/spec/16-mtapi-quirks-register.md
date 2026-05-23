# 16 · mtapi 暗坑寄存器（quirks register）

> **强制阅读**：实现 `adapter/mt[45]/` 与 `mdgateway/` 之前必须读完本文档。
> 每条 quirk 都来自 alfq 或 anttrader 的真实生产 fix，不是猜测。
> **代码引用 quirk 必须用注释 `// QUIRK Q-NNN: <一句话> See docs/spec/16-mtapi-quirks-register.md#q-nnn`**。

## 索引

| ID | 标题 | 平台 | 严重度 | 来源 |
|---|---|---|---|---|
| Q-001 | OnQuote.Time 在部分 broker 不实时推进 | MT4 | 🔴 高 | alfq `06d81be` |
| Q-002 | mtapi gRPC metadata 必须含 Bearer token | MT4/5 | 🔴 高 | alfq `76235b5` |
| Q-003 | TradeMode 默认 0 阻塞 quote stream | MT4 | 🔴 高 | alfq `b388618` |
| Q-004 | 跨 broker symbol 别名混合 → SubscribeMany 静默丢弃 | MT4/5 | 🔴 高 | alfq `b388618` |
| Q-005 | broker symbol 后缀差异（.m/.pro/.x/.c/_i 等） | MT4/5 | 🟠 中 | alfq `b8c3cb8` `8387153` |
| Q-006 | broker symbol 大小写不一致 | MT4/5 | 🟠 中 | 经验 |
| Q-007 | OHLCV 数值写 ClickHouse Decimal 必须包装 | MT4/5 | 🟠 中 | alfq `c859b65` |
| Q-008 | MT4 OnQuote 时间戳是秒，MT5 是毫秒 | MT4/5 | 🟠 中 | mtapi 文档 |
| Q-009 | MT5 QuoteHistory TimeFrame 用分钟数而非 PERIOD enum | MT5 | 🟠 中 | mtapi 文档 |
| Q-010 | broker 重连后会重发最近 N 条历史 quote | MT4/5 | 🟠 中 | 经验 |
| Q-011 | 网关搜索 broker 要求关键词 ≥2 字符 | MT4/5 | 🟡 低 | alfq `4c73728` |
| Q-012 | broker_host 切换需触发 reconnect | MT4/5 | 🟡 低 | anttrader `8d7aa32` |
| Q-013 | mtapi.io max gRPC msg size 默认 4MB | MT4/5 | 🟡 低 | 经验 |
| Q-014 | session token 不主动续期会过期 | MT4/5 | 🟠 中 | 经验 |
| Q-015 | bid/ask 字符串可能为 "0" 或空（symbol 未交易时段） | MT4/5 | 🟠 中 | 经验 |

---

## Q-001 · MT4 OnQuote.Time 在部分 broker 不实时推进 🔴

**症状**：
- 用 broker 返回的 `quote.Time` 做 bar 分桶
- 多个连续 tick 的 Time 完全相同
- 1m bar 永远过不了分钟边界，bar 不 flush

**根因**：MT4 部分 broker 的 OnQuote 推送返回缓存时间戳，不在每 tick 更新。

**解决**：
- bar 分桶**必须**用本地接收时间 `arrived_unix_ms = time.Now().UTC().UnixMilli()`
- broker 时间 `ts_unix_ms` 仍写 CH，但**不参与逻辑**（仅供审计）

**代码引用**：
```go
// QUIRK Q-001: MT4 OnQuote.Time may be stale; use local arrival time for bucketing.
// See docs/spec/16-mtapi-quirks-register.md#q-001
arrivedMs := time.Now().UTC().UnixMilli()
```

**影响位置**：
- `adapter/mt4/gateway.go:Subscribe`（设置 `ArrivedUnixMs`）
- `mdgateway/bar_aggregator.go:AddTick`（用 `ArrivedUnixMs` 分桶）

---

## Q-002 · mtapi gRPC metadata 必须含 Bearer token 🔴

**症状**：mtapi.io 返回 `UNAUTHENTICATED`。

**根因**：mtapi.io 用 gRPC metadata `authorization: Bearer <token>` 鉴权，不是 TLS 双向证书。

**解决**：

```go
md := metadata.New(map[string]string{
    "authorization": "Bearer " + cfg.MtapiToken,
})
ctx = metadata.NewOutgoingContext(ctx, md)
```

每个 mtapi 调用都要带；不能只在 Connect 时带一次。

**影响位置**：
- `adapter/mt[45]/gateway.go:Connect/Subscribe/Disconnect`
- `adapter/mt[45]/executor.go:PlaceOrder/CloseOrder/...`

---

## Q-003 · MT4 TradeMode 默认 0 阻塞 quote stream 🔴

**症状**：
- MT4 SubscribeMany 调用成功
- `OnQuote` 永远不触发，0 ticks

**根因**：
- mtapi.io 返回的 `GroupParams.TradeMode` 在 MT4 不可靠，常为 0
- ant 内部把 `TradeMode=0` 解释为"该 symbol 禁止交易/订阅"
- 整条 quote stream 因为单 symbol filter 失败而阻塞

**解决**：
- MT4 fetcher 在拉 SymbolParams 时，对满足 `Digits>0 AND Point>0 AND ContractSize>0` 的 symbol **强制设置 `TradeMode=4`** (`SYMBOL_TRADE_MODE_FULL`)
- 不依赖 broker 返回的 TradeMode

**代码位置**：
- `adapter/mt4/executor.go:FetchSymbolParams`
- `mdgateway/runner.go` 启动时同步 broker_symbols

---

## Q-004 · 跨 broker symbol 别名混合 → SubscribeMany 静默丢弃 🔴

**症状**：
- 多 broker 部署，broker A 用 `EURUSDm`、broker B 用 `EURUSD`、broker C 用 `EURUSD.`
- 把所有别名都丢给某个 broker 的 SubscribeMany
- 该 broker 只识别自己的别名，其他**静默丢弃**，无 error 返回

**根因**：mtapi.io 对未知 symbol 不报错，仅不订阅。

**解决**：
- `mdgateway/runner.go` 的 `loadBrokerSymbols` 必须**按 broker 过滤**，**只**给该账户的 broker 传它认识的 raw symbol
- canonical → raw 反查走 `broker_symbols` 表 `(broker, canonical)`

**代码位置**：
- `mdgateway/runner.go:loadAccountSymbols`
- `mdgateway/normalizer.go` 反向查询接口

**反例（v1 错误代码）**：

```go
// ❌ 把所有 symbol 都给所有 broker
allSymbols := []string{"EURUSD", "EURUSDm", "EURUSD."}
for _, gw := range gateways {
    gw.Subscribe(ctx, allSymbols, h)  // 各 broker 都丢弃自己不认识的
}
```

**正确**：

```go
// ✅ 每个账户拿自己 broker 的 raw symbols
for _, acc := range accounts {
    rawSyms := lookupRawSymbols(acc.Broker, canonicals)
    gw := gateways[acc.ID]
    gw.Subscribe(ctx, rawSyms, h)
}
```

---

## Q-005 · broker symbol 后缀差异 🟠

不同 broker 同一品种的 symbol：

| Canonical | broker A | broker B | broker C | broker D |
|---|---|---|---|---|
| BTCUSD | BTCUSD | BTCUSDm | BTCUSD.pro | BTCUSD.c |
| EURUSD | EURUSD | EURUSDm | EURUSD. | EURUSD_i |
| XAUUSD | XAUUSD | XAUUSDm | XAU/USD | XAUUSD.c |

**算法 fallback**（在 `mdgateway/normalizer.go`）：
1. 去除 `.` 之后所有内容
2. 去除尾部小写 `m/c/x/i/r/o/p`
3. 去除 `_i / _r / _institutional / _retail` 后缀
4. 大写

**优先级**：
1. 查 PG `broker_symbols` 表（最权威）
2. 查内存缓存
3. 算法 fallback
4. 缓存 fallback 结果（带 1h TTL，便于 PG 后续覆盖）

---

## Q-006 · broker symbol 大小写不一致 🟠

部分 broker 返回 `eurusd` / `Eurusd` / `EURUSD`，但订阅时**严格大小写匹配**。

**解决**：
- 所有 SubscribeMany 调用前，**先用 broker 实际返回的大小写**（从 SymbolList 拉，不要凭 canonical 大写猜）
- canonical 内部统一大写
- raw symbol 保留 broker 原始大小写

---

## Q-007 · OHLCV 数值写 ClickHouse Decimal 必须包装 🟠

**症状**：`clickhouse: code 53: cannot cast Float64 to Decimal(18, 6)`

**根因**：clickhouse-go v2 的批量 INSERT 对 Decimal 列不接受 `float64`，必须传 `decimal.Decimal`。

**解决**：

```go
import "github.com/shopspring/decimal"

batch.Append(
    bar.UserID,
    bar.AccountID,
    // ...
    decimal.NewFromFloat(bar.Open.InexactFloat64()),  // 或保持 decimal 全程不转 float
    decimal.NewFromFloat(bar.High.InexactFloat64()),
    // ...
)
```

更佳：bar 内部全程 `decimal.Decimal`，**永远不进 float64**。

---

## Q-008 · MT4 OnQuote.Time 是秒、MT5 是毫秒 🟠

| 平台 | quote.Time 单位 | 转换 |
|---|---|---|
| MT4 | 秒（int64）| `tsMs = quote.Time * 1000` |
| MT5 | 毫秒（int64）| `tsMs = quote.Time` |

**adapter 内必须做这一转换**，下游 mdgateway 收到的 Tick.TsUnixMs 永远是毫秒。

---

## Q-009 · MT5 QuoteHistory TimeFrame 用分钟数 🟠

mtapi.io MT5 的 QuoteHistory `PriceHistory` 接口的 `TimeFrame` 字段：
- ❌ 不是 MT5 的 `PERIOD_M1=1, PERIOD_M5=5, PERIOD_M15=15, PERIOD_M30=30, PERIOD_H1=16385...` enum
- ✅ 直接是分钟数 `1=1m, 5=5m, 15=15m, 30=30m, 60=1h, 240=4h, 1440=1d, 10080=1w, 43200=1M`

**代码位置**：`adapter/mt5/executor.go:FetchPriceHistory` + `mdgateway/backfill/backfill.go`

```go
var periodToMinutes = map[string]int32{
    "1m": 1, "5m": 5, "15m": 15, "30m": 30,
    "1h": 60, "4h": 240, "1d": 1440, "1w": 10080, "1M": 43200,
}
```

---

## Q-010 · broker 重连后会重发最近 N 条历史 quote 🟠

**症状**：网络抖动后 mtapi 重连，`OnQuote` stream 收到 N 条**已经处理过**的 tick。

**影响**：
- CH `md_ticks` 出现重复行（即使 ReplacingMergeTree 也增加 merge 压力）
- bar 聚合器把同一 tick 算两次

**解决**：`mdgateway/tick_dedup.go` 100 条窗口去重（hash by `ts_ms || bid || ask`）。

---

## Q-011 · 网关搜索 broker 要求关键词 ≥2 字符 🟡

`adminapi/broker_handler.go` 在前端搜索经纪商时，关键词长度 < 2 → mtapi.io 网关返回 INVALID_ARGUMENT。

**解决**：
- 关键词为空时用 `tr` 前缀（取常见经纪商前缀）
- 关键词长度 < 2 时拒绝（前端校验 + 后端兜底）

**代码位置**：`backend/internal/connect/broker_search_service.go`

---

## Q-012 · broker_host 切换需触发 reconnect 🟡

**症状**：admin 改了 `mtapi_host` 字段，但运行中的 gateway 还连着老地址。

**解决**：
- `mt_accounts` UPDATE host/port 时，`runner.go` 监听 PG `LISTEN/NOTIFY` 或定时同步
- 检测到变更 → `manager.RemoveGateway(accountID); manager.AddGateway(...)`

---

## Q-013 · mtapi.io max gRPC msg size 默认 4MB 🟡

**症状**：拉历史 K 线 limit=100000 时 `RESOURCE_EXHAUSTED`。

**解决**：

```go
grpc.WithDefaultCallOptions(
    grpc.MaxCallRecvMsgSize(16*1024*1024),  // 16MB
    grpc.MaxCallSendMsgSize(16*1024*1024),
)
```

---

## Q-014 · session token 不主动续期会过期 🟠

**症状**：账户连接了几小时后下单返回 `INVALID_SESSION`。

**解决**：
- `mthub/session.go` 维护 session 创建时间
- 超过 `cfg.SessionMaxAge`（默认 4h）→ 主动 Disconnect + Reconnect
- 或每次 PlaceOrder 失败为 INVALID_SESSION → Reconnect 后重试 1 次

---

## Q-015 · bid/ask 可能为 "0" 或空 🟠

**症状**：周末/休市时段 OnQuote 推 `bid="0", ask="0"` 或空字符串。

**解决**：
- `mdgateway/quality.go` 把 `bid <= 0 || ask <= 0` 计入 `dropped{reason="parse_error"}`
- adapter 层不主动过滤（保留 quirks 给 quality 统一处理）

---

## 暗坑发现/录入流程

新发现的暗坑必须：

1. 在本文件追加新 `## Q-NNN`，编号单调递增
2. 包含：**症状 / 根因 / 解决 / 代码位置**
3. 解决代码必须用注释 `// QUIRK Q-NNN:` 引用
4. PR 标题加 `quirk(Q-NNN):` 前缀
5. 同步更新本节顶部"索引"表

CI 校验：

```bash
# 索引表 ID 与正文 ID 一致
diff <(grep -oE '^\| Q-[0-9]+' docs/spec/16-mtapi-quirks-register.md | tr -d '| ' | sort) \
     <(grep -oE '^## Q-[0-9]+' docs/spec/16-mtapi-quirks-register.md | awk '{print $2}' | sort)

# 代码注释 ID 必须在文档中存在
grep -hoE 'QUIRK Q-[0-9]+' backend/internal/ -r 2>/dev/null | sort -u | while read ref; do
    qid=${ref#QUIRK }
    grep -q "^## $qid" docs/spec/16-mtapi-quirks-register.md || { echo "Missing: $qid"; exit 1; }
done
```
