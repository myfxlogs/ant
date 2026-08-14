# 施工交接：流活性检测纠偏（删 8 处"无数据超时"反模式 → gRPC keepalive）+ 成交补账

> **审计方定论 2026-08-15（用户抓到第 4 类系统性违规后自审修正）。** "90s 无数据=死"是**反向轮询**——用定时器启发式猜死亡，违 Push-First 精神（形式上用了流，行为被 time.After 治理）。8 处中 5 处是审计方本会话修"流僵死"时引入——修症状用错工具，制度化反模式。
>
> **铁律**：对抗证明 + 实测回填 + commit + 部署 + 回填 registry。

---

## 背景：违规模式与实测损害

```
轮询（被禁）：每 5s 问"有数据吗？"            ← 定时器驱动取数
无数据超时（现状）：90s 检查"没数据？→判死重连" ← 定时器驱动判死
```

**实测损害**：order-update 流是事件驱动（订单变更才推），空闲=正常 → 90s 超时必然误判 → **无限重连循环**（实测 337 次"stream active"，间隔 5-30s）→ 用户两笔真实订单（ticket 344012976/344010713）的成交事件落在重连缝隙里**永久丢失** → OMS 卡 SUBMITTED、无持仓、无 trade_record。

## 受影响全清单（8 处，全删）

| # | 流 | mt4 位置 | mt5 位置 | 引入者 |
|---|---|---|---|---|
| 1 | quote 流 | quotes.go:164 `case <-time.After(g.quoteTimeoutOrDefault())` | quotes.go:165 | 审计方（MDGATEWAY-3）|
| 2 | profit 流 | quotes.go:284 `case <-time.After(90*time.Second)` | 对应处 | commit 98d5e03e |
| 3 | order-update 流 | order_stream.go:113 `case <-time.After(g.orderUpdateTimeoutOrDefault())` | order_stream.go:113 | 审计方（MDGATEWAY-3）——**本次事故根因** |
| 4 | hub 订单事件 | orders.go:375 `case <-time.After(g.orderUpdateTimeoutOrDefault())` | orders.go:对应处 | 审计方（MDGATEWAY-4）|

**注意**：删的是"判死分支"；`select{recv/err}` 的 **Recv error 驱动重连保留**（那是真事件驱动）。删后如果 select 只剩 recv/err 两 case——直接改回阻塞 `for { Recv() }`（err→handleStreamError 重连），结构更简单。

## §1 gRPC keepalive（传输层阳性死亡信号）

mtapi 连接建立处（mt4/mt5 connection.go 的 grpc.Dial/NewClient）加：

```go
grpc.WithKeepaliveParams(keepalive.ClientParameters{
    Time: 30 * time.Second,                // 空闲 30s 后发 PING
    Timeout: 20 * time.Second,             // PING 20s 无应答 = 连接死
    PermitWithoutStream: true,             // 无活跃流时也保活
})
```

- 连接真死（网络分区/服务端僵死）→ keepalive 超时 → **Recv() 返回错误** → 错误处重连（事件驱动）。
- **先实测 mtapi 是否应答 PING**：连上一个账户后观察 5 分钟——若连接被 mtapi 误断（服务端有 ping 限制策略，表现为立刻/反复报 keepalive 错误），记录实测结果并调整（加长 Time 或 PermitWithoutStream 实验）；**实测结果必须回填 registry**。
- **连带**：`reconnect_test.go`/`mt4_test.go` 里依赖 no-data 超时的既有测试（TestMDGATEWAY3_OrderUpdateTimeout_FiresReconnect、TestQuoteTimeout_FiresReconnect、TestMDGATEWAY4_* 等）需**重写为 Recv-error 驱动**（mock stream 返回 error → 断言重连），不是删除测试。

## §2 重连补账（reconcile-repair，事件驱动兜底）

order 流（重）建立后跑一次 OpenedOrders 对账：
- 本地卡 SUBMITTED + broker 有该持仓 → 转 FILLED + 补持仓快照。
- 本地 SUBMITTED + broker 无（已平）→ 查历史 → 转相应终态。
- 基建：`mthub/reconciliation.go` 现只检测不修复（旧审计 P2#8）→ 升级为修复型，**由 order 流建立事件触发**（非定时轮询）。

**现成终验用例**：orders 表卡着的 ticket **344012976 / 344010713**（07:03/07:04 真实下到 Exness demo）——修复部署后对账应把这两单正确转态。

## 对抗证明（双向）

- **① 删超时**：mock stream Recv 永久阻塞（不返回数据也不返回错误）+ 事件流空闲 >90s → 新代码**不重连**（静默正常，GREEN）；旧代码重连（RED）。可用 verify-adversarial.sh 突变验证。
- **② keepalive**：mock 连接层 keepalive 超时 → Recv 报错 → 重连（GREEN）；删 keepalive 参数 → Recv 永不报错（RED）。若用真实连接难 mock，至少单测 Dial 参数存在（断言 keepalive 配置注入）。
- **③ 补账**：本地造一单 SUBMITTED + mock broker OpenedOrders 含该 ticket → 对账后转 FILLED（GREEN）；删对账调用 → 永卡 SUBMITTED（RED）。

## 红队自审
- [ ] 删超时≠删重连：Recv error 重连路径必须在（否则真死了不自愈）。
- [ ] keepalive 参数先实测 mtapi 行为（PING 限制策略），实测结果回填。
- [ ] 既有依赖 no-data 超时的测试重写（Recv-error 驱动），不是删掉——防测试覆盖倒退。
- [ ] reconcile-repair 由 order 流建立事件触发，别引入定时器（否则又一个变相轮询）。
- [ ] mt4+mt5 双端 8 处全改，漏一处=该流仍循环。
- [ ] 本文件设计已审计方自审（反向轮询定性 + keepalive 实测风险 + 测试重写要求），非口头版。
