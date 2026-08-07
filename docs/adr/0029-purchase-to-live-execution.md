# ADR-0029 · 购买→实盘执行链路（Purchase-to-Live Link）

- **状态**：Accepted
- **日期**：2026-08-08
- **决策者**：Windsurf（双角色 agent：审计+施工）× 人类负责人
- **涉及功能块**：`strategy-marketplace`、`strategy-runtime`、`api-gateway`
- **关联**：ADR-0023（AST VM / MQL as source of truth）、ADR-0028（回测可靠性护栏）、`docs/spec/purchase-to-live-link-spec.md`

> 本 ADR 回答：**用户在市场购买策略后，如何确保策略在实盘正确执行、且源码不下发前端、且授权实时有效？**

---

## 1. 背景

市场购买侧（订阅、退款套利防护、冻结结算）已正常运作，但**部署→实盘运行**这一段端到端不通。审计发现三个缺陷：

- **缺陷 A（致命）**：调度引擎 `ScheduleEngine.dispatch` 从 `ai_strategy_templates`（AI 骨架库）读策略代码，而 schedule 的 `template_id` 外键指向 `strategy_templates`（真实策略表）。两表两套 ID 空间，`GetByID` 永远 `ErrNoRows` → 任何用户/市场策略的定时调度都跑不起来。**原始实现缺陷，从未正确过。**
- **缺陷 B（致命）**：事件型 schedule（`kline_close`，市场 Deploy 默认类型）的 `next_run_at` 恒为 nil，定时引擎只捞 `next_run_at IS NOT NULL`。全仓无任何"bar 收盘 → 查匹配 schedule → 触发"的事件派发器。事件型 schedule 建了、存了、退款计数也算了，但**永远不会被执行**。
- **缺陷 C（安全/商业）**：全链路无一处校验"运行某策略前，该用户是否持有有效订阅"。未购买者也能跑发布者策略。且 `StartStrategy` 前端发 code 上来——若买方路径复用它，等于把发布者源码送到前端。

---

## 2. 架构决策

### 决策 1：事件型 schedule = 持久化流式会话（非逐 bar 派发）

事件型 schedule 的正确运行模型是**一个长生命周期的 `RunLiveStrategy` 流式会话**，而非定时器逐 bar 查库派发。

**理由（push-first 第一性）**：实盘策略订阅 bar 流、对每根 bar 做反应——这是长会话，不是逐 bar DB 轮询。`RunLiveStrategy` 本就是订阅 `barBroker` channel 的长驻循环，完美匹配。`sessionRegistry` + `StopSchedule` + `reconcileOnStartup` 基础设施已就绪。

**不采用的备选**：逐 bar 事件派发器（on bar close → 查匹配 schedule → 跑一次）。与 `RunLiveStrategy` 流式模型冲突、重复实现 OnTick/OnTrade 订阅、且违背 push-first。

**实现**：新增 `ScheduleEngine.launchEventSession(schedule)` + `StartSchedule(id)`。触发点：`ToggleSchedule(active=true)`、`reconcileOnStartup`。定时型 schedule 保持原逐 tick `dispatch` 模型不变。

### 决策 2：后端取码，源码永不下发前端

事件型/定时型 schedule 的代码一律由后端在 `dispatch`/`launchEventSession` 内服务端取码：按 `schedule.TemplateID` 从 `strategy_templates.code` 取，直接喂 VM，不经前端、不下发。

**边界**：`StartStrategy`（workspace「Run」）保持现状——它服务用户自己的策略，用户本就持有源码，前端发 code 合法。市场购买→实盘的唯一入口是 schedule 路径（后端取码 + 授权闸），两条路径职责分离、互不污染。

**实现**：`ScheduleEngine` 通过 `TemplateCodeReader` 接口注入 `StrategySvc.GetTemplate`，替换原 `AIStrategyTemplatesRepository`。

### 决策 3：运行时授权闸（纵深防御）

复用 `marketplace.Service.CanAccessCode`（检查 ownership / active subscription / active trial）作为授权闸，在 `dispatch`（定时型）和 `launchEventSession`（事件型）启动会话前调用。未授权 → 不启动 + 记 `last_error` + 日志。不在 `CreateSchedule` 拦截（允许用户先建 schedule、后购买；运行时才闸）。发布者本人对自己的策略免闸。

**运行时复验**：对市场购买型会话（非用户自有策略），每 bar 复验授权（`LiveStrategyConfig.EntitlementCheck`）。搭车 bar 驱动事件循环，非 `time.Ticker` 轮询——符合 push-first。撤销 → 会话优雅自终止。

### 决策 4：撤销只停信号，绝不自动平仓

授权撤销时，会话自终止（deregister + run 记录标 stopped/revoked），但**不自动平仓**。平台提供一键平仓按钮（用户显式点击），不行使权力。

**理由**：自动平仓是"代客交易、碰资金"行为，触碰牌照硬边界。退款分支天然已处理（`refund.go:153` 拒绝在有 active schedule 时退款），不存在"边跑边退"。

### 决策 5：配额闸复用

schedule 启动的会话必须走同一道 `checkStrategyQuota`（含 `CheckLiveStrategyLimit` live 子限额）。事件型会话 = broker 连接 + 持仓敞口，与手动 `StartStrategy` 风控语义等价；不闸 = Free 用户借 schedule 绕过配额 + 风控漏洞。

### 决策 6：同 account 冲突——先跑者赢，绝不静默替换

`launchEventSession` 在 `sessionRegistry.Register` 返回 nil（账户已有在跑会话）时，标记 error + 友好报错，不顶替、不重试。绝不静默杀掉可能持有未平仓的在跑会话——孤儿/冲突持仓是交易系统最糟故障。

---

## 3. 与其他 ADR 的关系

- **ADR-0023（AST VM / MQL as source of truth）**：本 ADR 确认 schedule 路径的代码源是 `strategy_templates.code`（用户策略源码），与 ADR-0023 的"MQL 源码即真相"一致。`ai_strategy_templates` 仅服务 AI 生成骨架，不参与实盘执行。
- **ADR-0028（回测可靠性护栏）**：本 ADR 的授权闸 + 配额闸是实盘侧的"执行入口护栏"，与 ADR-0028 回测侧的"防线B 不变量"形成对称——回测防假成功，实盘防未授权。
- **spec `purchase-to-live-link-spec.md`**：本 ADR 的实现规格。spec §七 四个已决议项（Q1~Q4）对应本 ADR 决策 5/6/3+4/1。

---

## 4. 实现摘要

| 任务 | 内容 | 文件 | 状态 |
|------|------|------|------|
| 1 (ARCH-3) | 取码源修正：`AIStrategyTemplatesRepository` → `TemplateCodeReader`/`StrategySvc.GetTemplate` | `schedule_engine.go`, `handlers_strategy_runtime.go` | ✅done |
| 2 | 事件型会话化：`StartSchedule` + `launchEventSession` | `schedule_event.go` (NEW), `schedule_engine.go`, `strategy_schedules.go` | ✅done |
| 3 | 订阅授权闸：复用 `CanAccessCode` | `schedule_engine.go`, `schedule_event.go`, `handlers_strategy_runtime.go` | ✅done |
| 4 | 每 bar 授权复验：`EntitlementCheck` | `live_runner.go`, `schedule_engine.go`, `schedule_event.go` | ✅done |
| 5 | 配额闸：复用 `checkStrategyQuota` | `schedule_engine.go`, `schedule_event.go` | ✅done |
| 6 | 集成测试 | `schedule_engine_test.go` | 🟦pending |
| 7 | ADR-0029 留痕 | 本文件 | ✅done |

**复用核对**：
- `REUSE: CanAccessCode` @ `service_subscription.go:352`
- `REUSE: checkStrategyQuota` @ `strategy_active_handlers.go:271`
- `REUSE: StrategySvc.GetTemplate` @ `template_svc.go`
- `REUSE: SessionRegistry.Register` @ `session_registry.go:101`
- `NEW: schedule_event.go`（`StartSchedule` + `launchEventSession`）— 无现成能力

---

## 5. 衍生项（已登记，超出本 ADR 范围）

| ID | 项 | 触发时机 |
|----|----|---------|
| P1-MKT-1 | 多策略共账户（持仓归因 + 按策略风控聚合）→ 解开 Pro 档容量 | Pro 用户量起来 |
| P2-MKT-2 | `ProtectedBacktestPanel` 取码/授权模式对齐 | 下次触及受保护回测 |
