# 施工交接：实盘报价/价格管线修复（LIVE-PRICE-3 + LIVE-PRICE-1 + LIVE-PRICE-2）

> **审计方根因定论，2026-08-13（用户推翻初版 OPS 诊断后修正）。** 完整根因/证据/验收标准见 `docs/audits/tech-debt-registry.md` §「实盘报价/价格管线审计」。本文件是施工指令。
>
> **触发症状**：Live Strategy Monitor → Active Runs → Run ID `60ca698c`（schedule `599ddaa5`，BTCUSDm/15m/live，账户 `904d14e6`）实盘无法执行策略，Active Runs 无 price。
>
> **⚠️ 用户确认 Exness-Trial 账户没问题（MT 客户端有报价）**——初版"账户 feed 停止"诊断（LIVE-PRICE-OPS）**已撤回**。真实根因在我们这侧的 mtapi 订阅错误被静默吞掉。

---

## 背景因果链（先读懂再动手）—— 真正的断点在 SubscribeMany 响应错误被吞

```
MT4/MT5 gateway.Connect ──► mtapi session(id) ──┬── OnOrderProfit 流：免订阅、连接即推 ──► ✅ 存活（账户 updated_at 0.9s 前）
                                                 └── SubscribeMany(symbols) ──► ❌ 响应 error 未检查，失败被静默吞 → 误记"subscribed"成功
                                                                              └──► OnQuote 流开（"quote stream active"）但零 symbol 订阅 ──► 零报价
                                                                                                                                   └──► HandleTick 从未触发（e2e_latency/gap/drop counter 全 0 实证）
                                                                                                                                        ├──► 无 bar 聚合 → runner.SubscribeBarUpdates 空
                                                                                                                                        └──► 无 tick → runner.SubscribeTickUpdates 空（且 tick 桥本身也缺 = LIVE-PRICE-1）
```

- **真正即时根因 = LIVE-PRICE-3**：SubscribeMany 响应 `error` 字段未检查（MT4+MT5 adapter 共 5 处），mtapi 订阅失败被静默吞 → OnQuote 空。这解释了"profit 活/quote 死"的不对称（profit 免订阅、quote 必须订阅）。
- bar 从 tick 聚合，**无报价 → tick+bar 全无** → 策略完全不执行（OnBar/OnTick 都不触发）。
- **LIVE-PRICE-1（独立 P1）**：即使报价恢复，tick 桥从未接线 → OnTick 不执行 + 价格列空。
- **LIVE-PRICE-2（次要 P2）**：quote recvLoop 无静默死检测——这是为什么我们 31h 没发现/没自愈。

**施工顺序：先 LIVE-PRICE-3（恢复报价，真正即时根因）→ LIVE-PRICE-1（报价流动后 OnTick/价格才工作）→ LIVE-PRICE-2（次要，死检测防再犯）。**

---

## 任务 1：LIVE-PRICE-3 — SubscribeMany 响应 error 未检查（P1，先做，真正即时根因）

### 根因
全代码库 mt4/mt5 adapter 对 mtapi 响应都检查 `resp.GetError()`（`connection_extra.go:32` / `order_history.go:31,106` / `orders.go:73`：`if resp.GetError() != nil && resp.GetError().GetCode() != 0`），**唯独 SubscribeMany 用 `if _, err := sub.SubscribeMany(...)` 丢弃整个响应**，只查 gRPC `err`，不查响应体 `error` 字段。

mtapi proto：`SubscribeManyReply { string result = 1; Error error = 2; }`（见 `reference/grpc/mt4.proto`）。mtapi 可返回 gRPC-OK 但 body 带 `error`（订阅失败）——我们误记 "subscribed symbols" 成功，实际零订阅 → OnQuote 开流但零报价。

**实测证据**（坐实"订阅失败被吞"，非丢弃问题）：metrics 全部 drop counter = 0（bid_gt_ask/non_positive/clock_skew/stuffing/dedup）+ `md_e2e_latency_seconds_count 0` + gap 全 0 = **HandleTick 从未触发**（OnQuote 没把任何 tick 送进 handler，不是丢弃）。profit 流（OnOrderProfit）免订阅、连接即推 → 存活（账户 updated_at 0.9s 前；SubscribeOrderProfit RPC 全代码库从未调用，证 profit 免订阅）。account 用户确认 MT 客户端有报价。

**为什么"以前能用、31h 前停"**：mtapi 以前 SubscribeMany 无错；~31h 前起返回 body 带 error（mtapi 升级/账户状态/symbol 列表变化等），被静默吞。**确切 mtapi 侧触发原因需本修复的日志确认**（补错误检查 = 同时暴露真实 code+message）。

### 改动位置（MT4 + MT5，共 5 处，全部改成检查响应 error）
- `backend/internal/mdgateway/adapter/mt4/quotes.go`：`Subscribe`（:34）+ `reSubscribeSymbols`（:164）
- `backend/internal/mdgateway/adapter/mt5/quotes.go`：`Subscribe`（:34）+ `AddSymbols`（:63）+ `reSubscribeSymbols`（:165）

### 改法（统一成 connection_extra.go:32 的模式）
现状（错误）：
```go
if _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms}); err != nil {
    g.log.Warn("mt4: subscribe symbols failed", zap.Strings("syms", syms), zap.Error(err))
} else {
    g.log.Info("mt4: subscribed symbols", zap.Strings("syms", syms))
}
```
改成（检查响应 error）：
```go
resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms})
if err != nil {
    g.log.Warn("mt4: subscribe symbols RPC failed", zap.Strings("syms", syms), zap.Error(err))
} else if e := resp.GetError(); e != nil && e.GetCode() != 0 {
    g.log.Error("mt4: subscribe symbols rejected by mtapi", zap.Strings("syms", syms),
        zap.Int32("code", e.GetCode()), zap.String("msg", e.GetMessage()))
    // 视为订阅失败：触发重连/重试（return error 或调 handleStreamError，依当前函数签名）
} else {
    g.log.Info("mt4: subscribed symbols", zap.Strings("syms", syms))
}
```
**关键**：响应 error 必须用 `log.Error`（不是 Warn）显式打 code+message，并视作订阅失败（让 recvLoop 重连重订）。`reSubscribeSymbols` 同改（它无返回值，用 log.Error + 触发重连信号即可）。MT5 三处同改（注意 `AddSymbols` 签名返回 error，可直接 `return fmt.Errorf`）。

### 对抗证明（必带，删行必红）
单测：mock SubscriptionsClient 的 `SubscribeMany` 返回 **gRPC nil-error + 响应 `Error{code: 7, message: "symbol not subscribed"}`** → 断言新代码检测到响应错误并 log.Error/return error（GREEN）。**删 `resp.GetError()` 检查分支（改回 `if _, err :=`）→ 误判成功、不报错 → RED**。
> ⚠️ 这个对抗证明同时验证"修复暴露真实 mtapi 错误"——部署后看日志里的真实 code/message 即知 ~31h 前的触发原因。**若部署后日志显示订阅成功但仍无报价**，则转向 mtapi 服务侧（OnQuote 交付问题），但代码侧此项必须先修。

---

## 任务 2：LIVE-PRICE-1 — 接 tick 桥（P1）

### 根因
`mthub.MtHubService.PublishTick` 全 backend 源码零调用方。`RunnerDeps`（`runner.go:38`）有 `OnBar`（→ `pipeline.go:82` PublishBar）无 `OnTick`。`manager_tick.go` HandleTick 无 `onTick(t)`。`git -S PublishTick -- cmd/server` 仅命中 `826539d4`（删 58MB 编译二进制的产物清理，非源码）= **源码层从未实现**（非回归）。

### 改动位置（3 处，对称已有 OnBar 桥）

**① `backend/internal/mdgateway/runner.go`** — RunnerDeps 加字段（照抄 `OnBar` 上一行）：
```go
OnTick               func(tick *mdtick.Tick)                                             // tick 桥：每个 quote 推到 mthub tickBroker（对称 OnBar）
OnBar                 func(bar *mdtick.Bar)                                             // called when a bar is finalized (for bar broker push to strategy runner)
```

**② `backend/internal/mdgateway/manager.go`** — 加 `onTick` 字段 + 构造注入（照抄 `onBar` 模式，:64 / :90 附近）：
- struct 加 `onTick func(*mdtick.Tick)`
- 构造 `onTick: deps.OnTick,`

**③ `backend/internal/mdgateway/manager_tick.go`** — HandleTick publish 阶段调 onTick（**必须在 `:23` normalize 之后**，否则 `t.Canonical` 为空）。当前 publish 阶段在 `:68-80`，`PublishTick` 之后加：
```go
if m.onTick != nil {
    m.onTick(t)   // t.Canonical 已在 :23 由 normalizer.Resolve 设置
}
```

**④ `backend/cmd/server/pipeline.go`** — RunnerDeps 加 OnTick 接线（照抄 `OnBar: func(b *mdtick.Bar) {...}` :82）：
```go
OnTick: func(t *mdtick.Tick) {
    d.mthubSvc.PublishTick(&mthub.TickUpdate{
        AccountID: t.AccountID,
        Symbol:    t.Canonical,   // ⚠️ 用 Canonical（归一化）非 SymbolRaw，与策略 symbol 对齐
        Bid:       t.Bid,
        Ask:       t.Ask,
    })
},
```

### 字段映射核对
- `mdtick.Tick` → `mthub.TickUpdate`：`AccountID→AccountID` / `Canonical→Symbol`（**非 SymbolRaw**）/ `Bid→Bid` / `Ask→Ask`
- `mthub.TickUpdate` struct（`tick_broker.go:15-20`）：`{AccountID, Symbol, Bid, Ask}`（decimal.Decimal）

### 对抗证明（必带，删行必红）
- 注入 mock `OnTick` callback + 喂一个 `*mdtick.Tick`（Canonical 已设）→ 断言 callback 被调用且收到正确的 `TickUpdate{Symbol: canonical, Bid, Ask}`（GREEN）。
- **删 `manager_tick.go` 的 `m.onTick(t)` 调用** → callback 不触发 → RED。
- **删 `pipeline.go` 的 `OnTick` 接线** → `mthubSvc.PublishTick` 不被调 → 断言 tickBroker 收不到事件 → RED。

---

## 任务 3：LIVE-PRICE-2 — quote 流加静默死检测（P2，次要）

> ⚠️ **已排除：报价订阅的 API 用法无 bug**（`reference/grpc/mt4.proto`+`mt4example.go` 证实 `SubscribeMany{id,symbols,interval=0}`→`OnQuote{id}`→`Recv` 读 `QuoteEventArgs` 是 mtapi 标准正确模式）。**LIVE-PRICE-3 是订阅失败被吞，不是 API 用错。本任务只加 recvLoop 死检测超时，禁止改 SubscribeMany/OnQuote 的 API 调用方式。**

### 根因
MT4/MT5 quote `recvLoop`（`mt4/quotes.go:122-146` / `mt5/quotes.go` 同结构）内层 `for { quote, err := stream.Recv() }` **无超时**。broker/mtapi 停推报价时，流日志显 `quote stream active` 却零数据、零错误、零重连。**profit 流已有 90s 无数据超时 + 重连**（`quotes.go:226-253`，commit `98d5e03e`），quote 流未获同等待遇。

### 改法（镜像同文件 profit 流 `profitRecvLoop` 的 select 模式，`mt4/quotes.go:226-253`）
```go
for {
    recvCh := make(chan *pb.OnQuoteReply, 1)
    errCh := make(chan error, 1)
    go func() {
        resp, err := stream.Recv()
        if err != nil { errCh <- err; return }
        recvCh <- resp
    }()
    select {
    case resp := <-recvCh:
        // 现有 handler(&mdtick.Tick{...}) 逻辑
    case err := <-errCh:
        g.log.Warn("mt4 recv", zap.Error(err))
        cancel()
        g.handleStreamError(ctx, err, &backoff)
        goto quoteLoopEnd
    case <-time.After(90 * time.Second):
        g.log.Warn("mt4 quote stream: no data for 90s — treating as dead", zap.String("account", g.cfg.AccountID))
        cancel()
        g.handleStreamError(ctx, fmt.Errorf("quote stream: no data timeout"), &backoff)
        goto quoteLoopEnd
    }
}
quoteLoopEnd:
```
MT5 同改。`goto` label 名不能与 `profitLoopEnd` 冲突。**超时阈值做成可注入参数**（测试勿真等 90s）。

### 对抗证明（必带，删行必红）
mock `OnQuote` stream 的 `Recv()` 阻塞 → 断言超时后（注入短超时）触发 `handleStreamError` + reconnecting（GREEN）。**删 `case <-time.After` 分支 → 永不重连 → RED**。

---

## 任务 5：LIVE-PRICE-4 — 删硬编码 symbol 列表，改「需求驱动 × broker 校验 × 逐 symbol」（P1，真正让报价流起来）

> **LIVE-PRICE-1/2/3 已 ✅done 但报价仍不流**——LIVE-PRICE-3 只让订阅错误**可见**（日志现打 `code:257 "XAUJPYm not exist"`），没让报价起来。真正根因 = 硬编码 `defaultQuoteSymbols()` 喂原子 `SubscribeMany`。**本任务才是让 OnQuote 真正通的关键。**

### 第一性原理（为什么这是 bug 的根）
系统把三个本该分开的关注点混进一个硬编码列表：
- ① **broker 有哪些 symbol**（外部事实）——应查 `FetchAllSymbols`，不该硬编码。
- ② **我们要订哪些**（需求）——应来自策略声明 symbol + 账户配置，不该猜。
- ③ **怎么订**（机制）——应逐 symbol（失败隔离），不该原子批量（一个坏全挂）。
**原则：系统能"知道"时绝不能"猜"。** broker 知道自己有什么、策略知道要什么，硬编码列表是个跨 broker 的猜测，必然在某些 broker 错（XAUJPYm/EURUSDm 在 Exness-Trial 不存在），原子 SubscribeMany 又把失败放大成全军覆没 → 连 100% 存在的 XAUUSDm 都订不上。

### 根因（实测铁证）
`runner_gateway.go:125-128` 账户无 `cfg.Symbols` 时 fallback 到硬编码 `defaultQuoteSymbols()`（runner_health.go:156，37 symbol，含 broker 不存在的 XAUJPYm/EURUSDm）。mtapi `SubscribeMany` **原子** → 一个不存在 → 整批 37 全拒（`code 257`）→ OnQuote 零交付 → 实盘无法开仓。用户证实 XAUUSDm 绝对存在（建调度选 symbol 实时拉的 broker 列表里有）→ 存在却零交付 = 原子连坐。

### 改法（即时修复 A，推荐先做）
`postConnectSetup`（`runner_gateway.go:125-133`）：
```go
// 旧：syms := cfg.Symbols; if len(syms)==0 { syms = defaultQuoteSymbols() }; gw.Subscribe(ctx, syms, ...)
// 新：
available, _ := gw.FetchAllSymbols(ctx)          // broker 权威清单
availSet := toSet(available)
desired := cfg.Symbols                            // 账户配置优先
if len(desired) == 0 { desired = available }      // 无配置 → 订 broker 全量真实清单（非硬编码）
// 逐 symbol 订阅（非原子，失败隔离）
for _, sym := range desired {
    if availSet != nil && !availSet[sym] { g.log.Info("skip symbol not on broker", sym); continue }
    if err := perSymbolSubscribe(ctx, sym, handler); err != nil { g.log.Warn("subscribe failed", sym, err); continue }
}
```
- **删 `defaultQuoteSymbols()`**（+ frontend COMMON_SYMBOLS 耦合注释）。
- 订阅改逐 symbol（循环单 `Subscribe`，镜像 `mt4example.go`），**不用 `SubscribeMany`**。`reSubscribeSymbols` 同改。
- MT4 + MT5 都改。

### 结构最优 B（演进，可后续）
按需订阅：connect 只订账户配置集（或空）；**策略启动时**调 `AddSymbols([策略symbol])`（quotes.go:47 已存在）按需加。gateway 永远只订"有人在用的 symbol"，不猜不浪费（避免真账户上千 symbol 全订）。A 是 B 的安全子集。

### 对抗证明（必带）
mock `FetchAllSymbols` 返回含 `XAUUSDm`/`BTCUSDm`、不含 `XAUJPYm`：
- 旧代码（硬编码 SubscribeMany 整批）→ `XAUJPYm not exist` 整批失败 → OnQuote 零交付 **RED**。
- 新代码（FetchAllSymbols 过滤 + 逐 symbol）→ `XAUJPYm` 跳过、`XAUUSDm`/`BTCUSDm` 订成功 → OnQuote 推 tick **GREEN**。
+ 部署后实测 `md_e2e_latency_count > 0` + Active Runs 价格列 + 策略产生信号。

### ⚠️ Reuse Preflight
- 复用 `FetchAllSymbols`（mt4/orders.go:258 / mt5/orders.go:289，"returns all available symbol names from the broker"）——建调度选 symbol 早已用它，gateway 订阅用同一权威源。
- 复用单 `Subscribe` RPC（reference/grpc/mt4example.go 模式）。

---

## 任务 4：LIVE-SSE-HEARTBEAT — watchActive 流加心跳保活（P2，顺手修，独立缺陷）

### 症状
Live Strategy Monitor → Active Runs tab **不定时出现 "Connection interrupted, reconnecting…"** 横幅（前端 `LiveStrategyPage.tsx:183` `streamError`，2s 自动重连）。功能自愈但横幅闪烁。

### 根因（审计方实测定论）
- **后端健康**：2h38m 未重启 / 0 restarts / 30min 内零 panic·rpc error·context canceled（**非后端崩溃或重部署**；Windsurf 改动未 build）。
- **nginx 非元凶**：`/api/` `proxy_read_timeout 3600s` + `proxy_buffering off`（不会 1h 内关）。
- **真因 = handler 无心跳**：`WatchActiveStrategies`（`strategy_active_handlers.go:448-472`）select 只在 `notifCh`（session 增删）/ `tickCh`（500ms 节流）触发时 `stream.Send`，**无周期性心跳**。当 tick 不流动（LIVE-PRICE-3 bug）且无 session 变化时流**完全静默** → HTTP/2 GOAWAY / 浏览器空闲 / 网络抖动任一层掐断静默流 → 前端 catch → 横幅。"不定时"= 取决于中间层何时下手。LIVE-PRICE-3 修好后 tick 流动会**碰巧保活**，但正解是显式心跳。

### 改法（标准 SSE 保活，~5 行）
`WatchActiveStrategies` select 循环加心跳 ticker：
```go
heartbeat := time.NewTicker(20 * time.Second)
defer heartbeat.Stop()
for {
    select {
    case <-ctx.Done():
        return nil
    case <-heartbeat.C:
        if err := stream.Send(&antv1.WatchActiveStrategiesEvent{}); err != nil { // 空事件=纯保活
            return err
        }
    case <-notifCh:
        if err := sendList(); err != nil { return err }
    // ... tickCh / tickTimer.C 不变 ...
    }
}
```
**20s < 任何中间层空闲超时**；空 `WatchActiveStrategiesEvent{}` 前端收到只是 `setActiveStrategies([])`（无害空刷新，**或前端忽略空 strategies 保现状**——见红队自审）。

### 对抗证明（必带）
单测：handler 跑 25s（注入短心跳间隔 1s）无 notifCh/tickCh 触发 → 断言至少 1 次 `stream.Send`（心跳）（GREEN）。**删 `case <-heartbeat.C` 分支 → 25s 内零 Send → RED**。

---

## 红队自审（动手前自查 edge cases，不只是对抗证明）

- [ ] **LIVE-PRICE-3 onTick 调用位置**（若同批做）：必须在 normalize（`manager_tick.go:23`）之后，否则 Symbol 空 → runner 全过滤。
- [ ] **LIVE-PRICE-3 重试副作用**：订阅失败触发重连 → `reSubscribeSymbols` 重订。确认 `subscribedSymbols` 已持久化（已持久化，`reSubscribeSymbols` 会重订），不会丢。
- [ ] **LIVE-PRICE-3 MT5 同构**：5 处全改（MT4 两处 + MT5 三处），漏一个 = 该平台仍静默吞。
- [ ] **LIVE-PRICE-3 不要改 API 调用方式**：SubscribeMany/OnQuote 的 RPC 用法是对的（reference 背书），只补响应 error 检查。
- [ ] **tick 频率**：`TickBroker` 已有 buffered channel + slow-consumer drop，onTick 不要改成阻塞发送。
- [ ] **不引入 Push-First 违规**：本修复接通已有 push 流 + 补错误检查 + 加死检测，不新增 polling。
- [ ] **任务 4 心跳空事件前端处理**：后端发空 `WatchActiveStrategiesEvent{}` 保活时，前端 `setActiveStrategies(event.strategies || [])` 会把列表清空成 `[]`（闪烁空表）。**两种修法二选一**：① 后端心跳发**带当前列表**的事件（复用 sendList，多一次查询但前端不闪）；② 前端忽略 `event.strategies` 为空/undefined 的事件（保留现状）。推荐 ②（后端心跳保持纯空 Send 最轻量，前端 `if (!event.strategies || event.strategies.length === 0) return;` 跳过）。**别让心跳把 Active Runs 表闪空**。

---

## 回填纪律（不做 = 任务失败）

1. `docs/audits/tech-debt-registry.md`：LIVE-PRICE-3 / 1 / 2 状态 `🟦open → ✅done`（标日期 + commit）+ 追加**真实根因/修复方式/对抗证明结果**。**特别重要**：LIVE-PRICE-3 部署后日志里的真实 mtapi error code/message 必须回填（那是 ~31h 前触发原因的真相）。只改状态列 + 追加，**不删条目、不改审计方事实陈述**。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. **不自行宣告完成**——等审计方核对状态 + 实测（删行复测 + 报价恢复后 Active Runs 价格列/OnTick 实测）。

## 门禁（Before Commit）
```bash
go build ./...
go test ./internal/mdgateway/... ./internal/connect/strategy/...
cd backend && go run ./tools/check-file-lines --strict
bash scripts/gen_capability_map.sh
```
部署：`docker compose build backend && docker compose up -d backend`（唯一方式，禁宿主 go build → docker cp）。

## Reuse Preflight（动工前必做）
`bash scripts/cap.sh <动词>` 查能力。LIVE-PRICE-3 复用 `connection_extra.go:32` 的 `resp.GetError()` 检查模式；LIVE-PRICE-1 复用 OnBar 桥模式；LIVE-PRICE-2 复用 profit 流 select 超时模式。**不新造**。PR 描述逐条给 `REUSE:` / `NEW:`。
