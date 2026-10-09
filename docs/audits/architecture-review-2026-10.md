# 架构与方法评估（2026-10-09，Devin CLI）

> 背景：FILE-SPLIT T1/T2 复审期间的结构性观察。结论按杠杆排序——机械折分是"地板"，本文档记"天花板"。

## 方法级（即刻可做，成本低）

### M1 门禁固化：复审发现的洞 → CI 的门

T2 复审实证：`go test -list` 不带 `-tags integration` 不编译 `//go:build integration` 文件，拆分层撞名逃逸守恒门禁（connect/system `events_test.go` 残余层、connect/strategy `ptr` 重名两处实证）。写进派工文档的规则依赖执行自觉，迟早再漏。

**解**：`go vet -tags integration ./...` + `git diff --check` 固化进 `ci.yml` backend-lint job——一次发现永久免疫。派单 `docs/plan/2026-10-ci-gate-integ-1.md`。

### M2 竞态测试是真债不是运气

`TestSubmitOrder_CommentAndDeviationReachExecutor`（vm_live_parity_test.go:83）在 T1/T2 两轮复审各败一次——根因是 `TradeBarrier.WaitState` 非锁存现态等待（trade_barrier.go:392），waiter goroutine 与同步 submit 路径抢 µs 窗口，负载高即输。

**解**：`TradeBarrier` 加 visited 位图锁存（每迁移记位，Acquire 重置生命周期），`WaitState` 语义升级为"曾到达即命中"——测试同步从抢窗口变查历史。8 处调用点全在测试文件，负向等待语义经推演保持（latch 命中恰是负向断言要抓的信号）。派单 `docs/plan/2026-10-barrier-waitstate-latch-1.md`。

## 结构级（S1-S3 完成后排期）

### S-A god package 分解

折分目标病灶高度集中：`connect/strategy` 单包 399 测试、≥6 职责域（execution/schedule/mutation/barrier/live_runner/diag）；`connect/system`、`mthub` 同型。文件折分不降低包内耦合——文件数翻倍而依赖图不变，明年再超。**S1-S3 verbatim 折分是必要前置**（先变小才可审查地移动），其后对 `connect/strategy` 做子包化：`strategy/execution`、`strategy/schedule`、`strategy/mutation`——import 图成为可强制模块边界，测试集按域归属。

### S-B SSE 多路复用

现状每功能一条流（WatchPaperAccount/SubscribeJob/ChatStream/SubscribeMetrics/WatchSchedules/SubscribeOrderUpdates/SubscribeProfitUpdates/WatchActiveStrategies）。STREAM-FREEZE-1 的逐流补 watchdog 证明：同样的坑每条流都会再踩；G-POST2-1 实证 limiter 未覆盖 ConnectRPC 流形态。**解**：单客户端一条多路复用流，后端按 event-type 扇出，心跳/watchdog/限流收口单一 chokepoint。代价=前端订阅协议改版，定为架构级方向不即刻动工。

### S-C VM 实盘 parity 回执改造（已立项 F1-F3，继续推）

最深正确性轴：`OrderExecutor.PlaceOrder` 接口丢弃 broker 回执，VM 层回显请求值冒充成交事实——造假点长在签名里。F1 已定正确方向（返回 `*OrderRecord` 真回执）。正确性债优先级高于卫生债。

## 不动清单

- 单仓 monolith + compose 双项目共机——现阶段正解，不碰微服务
- PG 作配置唯一真相（ADR-0031）、push-first、前端零信任——不变量立得住
- mt4/mt5 adapter 不共享代码——刻意隔离决策，不为省码合并
