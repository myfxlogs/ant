# post2-capacity-baseline-2026-09 — POST-2 容量基线（探针批实测）

> POST-2「性能/容量压测（下单/回测/SSE）」探针批交付。**本机即生产宿主，全程 in-process、零触运行中容器/生产端口/mtapi 真路径/DB**（派工单 `docs/audits/builder-handoff-post2-capacity-probe.md`）。
> 测量命令（可复现）：
> ```bash
> cd backend && go test -bench=. -benchtime=1s -run='^$' ./tools/mql2go/ ./internal/paper/
> cd backend && go test -run='TestSSEFanoutCostCurve|TestPlacePaperOrderLatency_Concurrency' -v -count=1 ./internal/paper/
> ```

## 1. 环境

| 项 | 值 |
|----|-----|
| CPU | Intel(R) Xeon(R) Gold 6138 @ 2.00GHz（4 vCPU 配额，`-4`） |
| OS/Arch | linux/amd64 |
| Go | go1.26.5 |
| 代码点 | commit `981ec6ad` 之后工作树（派工单落档点） |
| 口径总注 | 探针均为进程内测量：单机数字**不代表多容器部署容量**；每轴专属口径见各节 |

## 2. 轴一：回测 VM 执行并发退化（S1）

**口径**：VM 纯执行——同一编译产物（MA 交叉形态策略，每 tick 40 次 Close 访问 + decimal 除法/比较，QS-3-B4 同源）跑 N 个独立 VM 实例（共享只读 `*Bytecode` + 无状态 `noopContext`）。**不含** claim/lease/PG 往返（worker 侧有界常数，非本轴）；不含与实盘 dispatch 的宿主争抢（进程内隔离）。

```text
BenchmarkVMExec_Concurrency/N=1-4     17809   68361 ns/op   14628 runs/s
BenchmarkVMExec_Concurrency/N=4-4     31262   37294 ns/op   26814 runs/s
BenchmarkVMExec_Concurrency/N=8-4     35106   32919 ns/op   30378 runs/s
BenchmarkVMExec_Concurrency/N=16-4    35008   33730 ns/op   29647 runs/s
```

**读数**：

- 单 VM 单 tick（重策略）≈ **68µs**（1000-tick 回测单独跑 ≈ 68ms）。
- 4 vCPU 上 N=4 吞吐 26.8k runs/s（**1.83× 单核**），N=8/16 平台期 ~30k（2.0–2.1×）——非线性封顶，归因 decimal 分配/GC + 调度争用（QS-3-B4：每事件 ~310 allocs/~10.8KB）。
- **排队代价量化**：3 worker（`backtest_worker.go:23` `defaultWorkers=3`）满占时宿主已在 N≈4 进平台期；第 4 个并发回测负载不再降低吞吐（仍在 ~30k runs/s 池子里排队均摊 ~33µs/tick），但单个请求的墙钟时间随排队深度线性变长——N=16 并发时每个 1000-tick 回测摊到 ~1.9k ticks/s → **~0.53s**（单独跑 68ms，~8× 退化）。有界 worker 把退化变成排队而非雪崩，形态健康。
- 宿主余量参考：N=3（3 worker 满跑）吞吐内插 ~22–24k ticks/s ≈ **1.5–1.6 核当量**，距 4 核饱和（~30k ticks/s 平台期）仍有约 2× 余量给实盘 dispatch/Web 服务。

## 3. 轴二：SSE/流扇出成本（S2）

**口径**：httptest 起真 HTTP 服务，handler 复刻 `WatchPaperAccount` select 循环（`internal/connect/paper/handler.go:121`）+ **真 `PaperEngine` Subscribe/broadcast 原语**（chan(8) 非阻塞投递，drop-slow-consumer 语义），N∈{10,100,500} 并发长连流保持，200 次广播按 5ms 步进（贴近真实 paper 成交节奏）。
**不含**：ConnectRPC 帧+proto marshal（探针载荷为最小 SSE 行）；每流 PG LISTEN/NATS 订阅加项（DB-free）；生产 `SSEKeepaliveMiddleware` 加项（见 §5）。
**内存/goroutine 口径注**：探针 server+client 同进程——每 conn 5 goroutine ≈ server 侧 2（conn+handler）+ client 侧 3（transport readLoop/writeLoop+scanner）；heap/conn 折半估 server 侧 ≈ **2 goroutine + ~45–50KB/流**，生产再加 keepalive +1 goroutine（`sse_keepalive.go` Write once.Do → `go keepaliveLoop()`）。

```text
N=10:   goroutines/conn=5.0  heap/conn=91203B  deliveries=2000/2000(100%)  p50=132µs   p95=366µs   p99=601µs    max=1.4ms    publish-avg=21.6µs
N=100:  goroutines/conn=5.0  heap/conn=95229B  deliveries=20000/20000(100%) p50=376µs   p95=1.74ms  p99=2.95ms   max=5.0ms    publish-avg=123µs
N=500:  goroutines/conn=5.0  heap/conn=95630B  deliveries=100000/100000(100%) p50=2.68ms p95=9.03ms  p99=19.9ms   max=34.0ms   publish-avg=434µs
```

**读数**：

- **每流成本恒定，无退化拐点**（N=10→500 每 conn 成本不变）；退化体现在延迟随扇出宽度增长：同账号 500 订阅时单次广播 ~0.43ms、投递 p99 ~20ms。生产中 broadcast 按账号分片（只扇给同账号订阅者），单账号订阅数通常 1–5 → 延迟轴余量大；**风险轴在总流数**（内存/goroutine 随总流数线性，见 §6 缺口）。
- 外推（server 侧口径）：5 万并发流 ≈ 10 万 goroutine + ~2.3–2.5GB heap。4 vCPU 宿主上 goroutine 调度先于内存成为约束。
- **突发丢弃语义（设计行为，非债）**：无步进瞬时突发 200 次广播时，每 conn 仅收到 ~9–10 条（chan 深度 8 + 在途），其余被 drop-slow-consumer 丢弃。`PaperAccountUpdate` 载荷是全量账户快照（`paperAccountToProto`），丢中间事件无害（下一事件自愈）——语义自洽。

## 4. 轴三：paper 下单路径（S3）

**口径**：`PaperEngine.PlacePaperOrder` 进程内全链（fill-price 计算 + nil-guard skip + 内存 stub repo 三写 + broadcast 空转）——**paper 基线，非真经纪商路径**（mtapi RTT 不在本探针范围，见 §7）。

```text
BenchmarkPlacePaperOrder-4    168453   6241 ns/op          （串行吞吐基线）
C=1:  ops/s=159041  mean=6.2µs   p50=4.1µs   p95=13.7µs  p99=30µs    max=726µs
C=8:  ops/s=288791  mean=17.0µs  p50=4.8µs   p95=13.9µs  p99=165µs   max=3.2ms
C=32: ops/s=337358  mean=16.8µs  p50=4.3µs   p95=14.4µs  p99=204µs   max=5.9ms
```

**读数**：串行 ~16 万单/s（6.2µs/单）；并发封顶 ~34 万单/s（~2.1×，mutex+调度封顶）。**结论：进程内 paper 引擎距瓶颈三个数量级——paper 轴的真实容量约束在下游（DB 写放大、每单 3 次 repo 往返），不在引擎本身。** p99/max 长尾为 GC 停顿（QS-3 已知 ~310 allocs/事件的同源放大）。

## 5. 容量模型表（有界/无界枚举 · 含对派工单结论的修正）

| 轴 | 界 | 证据 |
|----|----|------|
| 回测并发 | **有界 3 worker** | `backtest_worker.go:23` `defaultWorkers=3` + SKIP LOCKED 原子认领 + 60s lease + LISTEN/NOTIFY |
| 回测日配额 | 有界（per-plan） | `quota_checker.go:118-121` `MaxBacktestsDaily`（派工单指针 `subscription.go:18` 已漂移，以此为准） |
| 登录 | 有界 | `main.go:101` `RateLimitLoginPerMinute` |
| 实盘策略数 | 有界（per-plan） | `quota_checker.go:127-130` `MaxLiveStrategies` |
| AI tokens 月配额 | 有界（per-plan） | `quota_checker.go:71` `MaxAITokensMonthly` |
| **SSE（text/event-stream 形态）** | 有界 per-user 5 | `main.go:276` `SSEStreamLimitMiddleware(5)` |
| **ConnectRPC server-streaming（前端主形态）** | **无界（per-user/global 均无上限）** | limiter 只匹配 `text/event-stream`（`sse_limiter.go:65-72` `isSSERequest`）；前端走 binary ConnectRPC（`frontend/src/client/connect.ts:97` `useBinaryFormat` → `application/connect+proto`）→ **不过 limiter**。WatchPaperAccount/SubscribeJob/ChatStream/SubscribeMetrics 等全部 server-streaming RPC 皆此形态 |

**对派工单设计表的修正**：派工单记「SSE 无界——未见 per-user/per-global 连接上限」。实查结论是**形态错位**而非缺失：per-user=5 的 limiter 存在且已接线，但 `isSSERequest` 的匹配规则覆盖不到前端实际使用的 ConnectRPC binary 流。净效果与「无界」等价，但修法是**把 ConnectRPC 形态纳入 limiter 匹配**（或对 server-streaming RPC 加等价闸），不是新增一套限制。

## 6. 硬缺口（建议登记新债；本批只登记不修）

1. **G-POST2-1（P2，容量/滥用面）ConnectRPC 流无连接上限**：N tabs × M users 线性堆积流 goroutine+~50KB/流内存，无限流/全局保护；与 §5 修正同源。配合 §3 外推：1 万流 ≈ 2 万 goroutine + ~0.5GB。
2. **G-POST2-2（P3，观测）**：stream 类 RPC 无活跃流数 metric；无 metric 则 §3 的外推无法在生产侧验证（QS-3「生产 p99 待 metric」同族）。

## 7. 待 staging 清单（本机不可测残余）

| # | 项 | 原因 |
|---|----|------|
| 1 | 下单真路径延迟（mtapi gRPC → 真经纪商 RTT） | 本机即生产，真路径打不得 |
| 2 | claim/lease/PG 往返吞吐（worker 认领轴） | 探针 DB-free（PG 在容器内网） |
| 3 | 每流 PG LISTEN / NATS 订阅成本加项 | 同上 |
| 4 | ConnectRPC 帧+proto marshal 每流成本 | 探针用最小 SSE 行口径 |
| 5 | 多容器水平扩展容量 | 单机数字不代表部署容量 |
| 6 | 生产 p99 观测（回测/下单/流投递） | 依赖 metric 上线（QS-3-BASELINE 同族） |
| 7 | 单账号极端订阅数（broadcast O(N)，~0.43ms@500） | 公开策略页潜在热点，需 marketplace 场景压测 |
