# MT RPC 扩展方案 — 提升用户体验与系统能力

> **背景**：MT4 (49 RPC) + MT5 (75 RPC) = 124 RPC total。当前封装率 ~53%（约 66 个），核心交易/行情/账户管线完整。
> **目标**：将未封装但高价值的 RPC 分三梯队补齐，提升风控预检、前端体验、回测精度。

---

## 已验证的已有能力

| 功能 | 状态 | 位置 |
|------|------|------|
| Investor/Master 密码区分 | ✅ | `mt_account.IsInvestor` 字段，账户同步时自动采集 |
| 经纪商搜索 | ✅ | `adapter/brokersearch/search.go` — 调用 MT `Search` RPC 搜索 broker |
| 核心交易管线 | ✅ | 下单/平仓/改单/查持仓/查历史 全封装 |
| 行情管线 | ✅ | Quote/PriceHistory/Subscribe/OnQuote |
| 账户管线 | ✅ | AccountSummary/BrokerInfo/Connect/Disconnect/Health |

---

## 第一梯队（直接提升产品竞争力）

### 1.1 RequiredMargin — 下单前保证金预检 🔴

**做什么**：传入品种+手数+方向+价格，返回所需保证金。

**为什么重要**：目前 6 门风控管线里没有下单前保证金检查。用户下单被拒后才收到"保证金不足"——这对用户体验很差。预检后前端按钮直接灰掉或显示红色提示。

**实现要点**：
- 封装 MT5 `RequiredMargin` RPC
- 在 `risk-gate` 的 PreCheck 阶段调用（已有位置，只需加规则）
- 前端：`PlaceOrder` 弹窗在手数输入框旁实时显示预估保证金

**文件**：
- `backend/internal/mdgateway/adapter/mt5/account.go` (extend)
- `backend/internal/risksvc/precheck.go` (extend)

### 1.2 IsQuoteSession / IsTradeSession — 市场状态实时展示 🔴

**做什么**：查询品种当前是否在报价时段 / 可交易时段。

**为什么重要**：前端现在没有办法知道"市场开没开门"。交易按钮在所有时间都是亮着的，用户尝试下单被拒后一脸茫然。封装后：
- 市场关闭 → 前端交易按钮灰掉 + tooltip "当前品种已收盘"
- 重大新闻前暂停交易 → 按钮灰掉 + "流动性不足，暂停交易"

**实现要点**：
- 封装 MT5 `IsQuoteSession` + `IsTradeSession`（单个和批量版都封）
- 前端新增 `useMarketSession(symbol)` hook
- 策略市场策略详情页展示品种交易时段

**文件**：
- `backend/internal/mdgateway/adapter/mt5/symbol.go` (extend)
- `frontend/src/hooks/useMarketSession.ts` (new)

### 1.3 TickValueWithSize — 每点价值精确计算 🟡

**做什么**：计算指定手数的 tick 值（每点波动=多少钱）。

**为什么重要**：风控规则"单笔最大亏损"需要知道每点价值才能计算止损距离。目前用的是近似值。

**实现要点**：
- 封装 MT4+MT5 `TickValueWithSize` / `GetTickValueMany`
- 风控 `DailyLoss` 规则使用精确 tick 值计算潜在亏损

---

## 第二梯队（用户体验提升）

### 2.1 SymbolSessionsEx — 品种完整交易时段 🟡

**做什么**：返回品种的完整交易时段（含节假日、夏令时切换）。

**为什么重要**：跨品种策略需要知道流动性窗口。EURUSD 和 XAUUSD 的交易时段不同，策略调度的依据。

**实现要点**：
- 封装 MT5 `SymbolSessionsEx` / `SymbolSessionsExMany`
- 前端策略调度页显示"建议时段"指示

### 2.2 PriceHistoryToday — 当日实时 K 线 🟡

**做什么**：获取品种当日的价格历史（不等到收盘）。

**为什么重要**：前端日线图现在要等到收盘才能更新最后一根 K 线。此 RPC 提供 intraday 数据，图表实时更新。

**实现要点**：
- 封装 MT5 `PriceHistoryToday` / `PriceHistoryTodayMany`
- 补充现有 `FetchPriceHistory` 的当日数据缺口

### 2.3 SubscribeMarketWatch + SubscribeOpenedOrdersTickets 🟡

**做什么**：市场报价实时推送 + 持仓 ticket 实时推送。

**为什么重要**：
- MarketWatch 推送 → 前端自选列表实时更新（不用页面刷新）
- OpenedOrdersTickets 推送 → 持仓变化实时反映（开仓/平仓瞬间看到）

**实现要点**：
- 封装 MT5 `SubscribeMarketWatch` + `SubscribeOpenedOrdersTickets`
- 前端已有 SSE 管道，后端订阅 NATS → SSE 推前端

### 2.4 Search — 品种搜索 🟢

**做什么**：按关键词搜索品种（如输入"EUR"返回 EURUSD, EURGBP, EURJPY…）。

**为什么重要**：前端品种选择器目前只能下拉滚动。搜索让用户体验好一个量级。

**实现要点**：
- 封装 MT4 + MT5 `Search` RPC（品种搜索，非经纪商搜索）
- 前端 `SymbolSelector` 组件加入搜索框

### 2.5 Events — 经济事件日历 🟢

**做什么**：MT 终端事件流（央行利率决议、非农、节假日等）。

**为什么重要**：前端策略调度页显示"今天有非农"——策略是否在重大新闻前暂停，用户自己判断。

**实现要点**：
- 封装 MT5 `Events` RPC
- 前端经济日历组件（可选，优先级低）

---

## 第三梯队（锦上添花 · 低频但有用）

### 3.1 GetLogs — MT 终端日志 🟢

**做什么**：获取 MT 终端运行日志。

**为什么重要**：用户反馈"连不上"时 → Admin 远程查 MT 日志，不用让用户截图。

**实现要点**：封装 MT4 `GetLogs` + `GetLogsByUser`，Admin 面板调用。

### 3.2 PriceHistoryMonth / PriceHistoryHighLow 🟢

**做什么**：月度 K 线 / 区间高低价。

**为什么重要**：长周期策略回测时减少 API 调用次数（一月一根 K 线 vs 逐日请求）。

### 3.3 Mails / OnMail 🟢

**做什么**：MT 终端内部邮件系统。

**为什么重要**：系统通知可直接推送到 MT 终端（如"你的策略已触发止损"）。

---

## 优先级汇总

| 优先级 | RPC | 平台 | 改动量 | 用户感知 |
|--------|-----|------|--------|---------|
| 🔴 P0 | RequiredMargin | MT5 | 小 | 中（下单体验改善） |
| 🔴 P0 | IsQuoteSession / IsTradeSession | MT5 | 小 | 高（市场状态一目了然） |
| 🟡 P1 | TickValueWithSize | MT4+5 | 小 | 中（风控更精确） |
| 🟡 P1 | SymbolSessionsEx | MT5 | 中 | 中（策略调度优化） |
| 🟡 P1 | PriceHistoryToday | MT5 | 小 | 中（图表实时更新） |
| 🟡 P1 | SubscribeMarketWatch / OpenedOrdersTickets | MT5 | 中 | 高（推送实时化） |
| 🟢 P2 | Search | MT4+5 | 小 | 高（搜索体验） |
| 🟢 P2 | Events | MT5 | 中 | 低 |
| 🟢 P3 | GetLogs | MT4 | 小 | 低（运维用） |
| 🟢 P3 | PriceHistoryMonth/HighLow | MT5 | 小 | 低 |
| 🟢 P3 | Mails/OnMail | MT5 | 中 | 低 |

---

## 关于 TickHistory — 不做，原因明确

**为什么改为了 bar 级**：不是 bug，是有意设计。

1. **VM 执行模型是 bar 级的**——MQL EA 的 `OnTick()` 被编译为 bar 边界执行。每根 bar 触发一次策略逻辑。改为 tick 级重放需要重做 VM，性能下降 200x。
2. **Tick 数据来自 TickHistoryRequest/Stop**——可以拿到，但回测引擎的 `Engine.Run()` 遍历 `[]sdk.Bar`，不支持 tick 粒度的撮合。
3. **bar 级对 95% 的策略足够**——只有高频/剥头皮策略真正需要 tick 级。这类策略也不是平台的目标用户（他们用 C++ 直连，不用 MQL VM）。

**如果未来需要**：正确的做法不是改 backtest-engine，而是用 `PriceHistory` 取 M1 级别数据（最小 bar 粒度）。M1 的回测精度对 99% 的策略够用，且不需要改架构。

---

---

## 🔴 前置修复：熔断器接入

**现状**：`mdgateway/circuit_breaker.go` 实现了完整的三态熔断器（滑动窗口、半开探测），`Manager.breakers` map 已初始化。但订单/流处理路径中从未调用 `Allow()`/`OnSuccess()`/`OnFailure()`——代码存在但不工作。

**为什么必须先修**：新增 RPC 会增加对 MT 服务器的调用频率。没有熔断器保护，MT 服务器故障或过载时平台会雪崩。

**任务**：
- [ ] 在 `PlaceOrder` 调用前接入 `breaker.Allow()` → 熔断时返回 `ErrCircuitOpen`
- [ ] 在 `PlaceOrder` 成功后调用 `breaker.OnSuccess()`，失败后调用 `breaker.OnFailure()`
- [ ] 在 `recvLoop` / `profitRecvLoop` / `orderUpdateRecvLoop` 的重连逻辑中接入熔断——连续重连失败 N 次后触发熔断，避免死循环重连
- [ ] 每个 broker endpoint 一个 breaker（按 broker host:port 建 key），不是全局一个
- [ ] Admin 面板展示熔断器状态（Open/Closed/HalfOpen + 失败计数）

**文件**：
- `backend/internal/mdgateway/adapter/mt4/orders.go` (extend)
- `backend/internal/mdgateway/adapter/mt5/orders.go` (extend)
- `backend/internal/mdgateway/adapter/mt4/quotes.go` (extend)
- `backend/internal/mdgateway/adapter/mt5/quotes.go` (extend)
- `backend/internal/mdgateway/manager.go` (已有 breaker map，接入)

---

## 🟡 系统韧性增强（熔断器接入后补充）

> 熔断器只是"停止伤害"。完整的系统韧性需要"停止 + 通知 + 降级"三层。

### 韧-1 告警通知

**做什么**：熔断器触发（Open/HalfOpen 状态变化）时，通过 SSE 推送 Admin 告警 + 写 `admin_audit_logs`。

**为什么重要**：熔断器默默断开连接 = 策略停止执行但没人知道。Admin 需要在第一时间看到"broker X 熔断已触发"。

**实现要点**：
- `CircuitBreaker.StateChange` 时发布 NATS 事件
- 前端 Admin 面板实时显示熔断器状态（已有 MonitoringPage，扩展即可）
- 不需要新增 RPC——复用 SSE 管道

### 韧-2 前端降级展示

**做什么**：MT 服务不可用时，前端订单面板显示"broker 暂不可用"而非报错或卡住。

**为什么重要**：用户不知道是平台挂了还是 broker 挂了。明确告诉他是 broker 的问题，保护平台信誉。

**实现要点**：
- 封装 `IsQuoteSession` / `IsTradeSession` 时一并返回 broker 连通性状态
- 前端交易按钮根据连通性状态灰掉，tooltip 显示原因

### 韧-3 实盘策略有序降级

**做什么**：broker 不可用时，实盘策略自动暂停并等待恢复（而非 crash 或疯狂重连）。

**为什么重要**：MtHub 的重连循环已经做了指数退避。但策略层面不知道 broker 不可用——它只会看到"订单被拒绝"。通知策略层"broker 不可用"可以让策略自己决定暂停还是继续。

**实现要点**：
- `LivePerformanceCollector` 已有 NATS 订阅基础设施——监听 `mt.<broker>.status` 事件
- broker 熔断 → 发布 `broker.unavailable` NATS 消息 → LiveRunner 收到 → 暂停策略信号发送，进入等待恢复模式
- 恢复时自动重连，不需要手动介入

**文件**：
- `backend/internal/mdgateway/circuit_breaker.go`（extend——加 NATS 发布）
- `backend/internal/mthub/service.go`（extend——加 broker status 查询接口）
- `backend/strategy/runner/live_runner.go`（extend——加暂停/恢复逻辑）

---

## 建议执行

- **前置**：熔断器接入 → 必须先修
- **P0 补充（韧-1 + 韧-2）** → 熔断接入后立即做，改动小，保护平台信誉
- **P0（RequiredMargin + IsQuoteSession）** → 嵌入 Phase 1 或作为独立的小模块先做
- **P1（TickValue + SymbolSessions + PriceHistoryToday + Subscribe 系列 + 韧-3）** → 嵌入 Phase 3
- **P2-P3** → 嵌入 Phase 4 或独立小任务，有空就做。
