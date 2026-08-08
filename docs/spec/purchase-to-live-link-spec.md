# 施工 Spec：购买→实盘链路打通——调度运行时取码源修正 + 事件型会话化 + 订阅授权闸

> **涉及功能块**：`strategy-marketplace`（购买授权）+ `strategy-runtime`（实盘调度执行）+ `api-gateway`（schedule handler）
>
> 市场价值链的下一个断裂点：购买侧（钱、订阅记录、退款套利防护计数）已正常，但**部署→实盘运行这一段端到端不通**。退款套利防护（`refund.go:144` 把 `strategy_schedules` 当作"购买是否产生实盘部署"的判据）证明 schedule 就是购买→实盘的设计载体，而运行时无法把它跑起来。本 spec 修正调度运行时的三个缺陷，使已购买策略能真正在实盘执行，且源码永不下发前端（守"代码不出平台"承诺）。

---

## 一、验证发现（已查实，含 git 根因）

链路设计意图（从代码反推，是清楚的）：

```
购买  PurchaseStrategy → user_subscriptions(target_strategy_id = marketplace.strategy_id)         ✅
部署  PurchaseTab「Deploy」→ DeployScheduleModal → createSchedule(templateId=strategy_id)
       → strategy_schedules 行（template_id → strategy_templates.id，FK 见 migration 028）         ✅ 存得对
退款防护 refund.go:144  count(strategy_schedules WHERE template_id=strategy_id AND buyer AND active)  ✅
运行  strategy_schedules 行 → 运行时取码 → RunLiveStrategy                                            ❌ 断点
```

### 缺陷 A（致命）— 调度引擎从错误的表读策略代码

`backend/internal/connect/strategy/schedule_engine.go:226` `dispatch`：

```go
tpl, err := e.templateRepo.GetByID(ctx, schedule.TemplateID)   // templateRepo = *AIStrategyTemplatesRepository
...
cfg := LiveStrategyConfig{ ..., Code: tpl.CodeSkeleton, ... }
```

- `templateRepo` 在 `cmd/server/handlers_strategy_runtime.go:144` 注入的是 `NewAIStrategyTemplatesRepository(pool)`，查询 `ai_strategy_templates`（migration 130 创建，**AI 生成用的静态骨架库**，只有系统种子数据，全仓无任何应用代码 INSERT/UPDATE——`grep` 确认仅 migration seed）。
- 而 schedule 的 `template_id` 外键指向 **`strategy_templates`**（migration 028，真实策略 `code` 列；migration 162 把 `marketplace_strategies.strategy_id` 也改指向此表）。
- 两张表、两套 ID 空间，`GetByID` 永远 `ErrNoRows` → `err != nil` → 日志 "dispatch: invalid template / template code is empty" → **任何用户/市场策略的定时调度都跑不起来**。

**根因（git）**：`git log -L` 显示 `dispatch` 的 `templateRepo.GetByID` 自 `0c839142 "implement schedule execution engine"` 首次实现即是如此——调度引擎从诞生起就读错表，从未正确过。这不是回归，是原始实现的缺陷。

### 缺陷 B（致命）— 事件型 schedule 没有任何触发器

市场"Deploy"默认建 `kline_close` 事件型 schedule（`DeployScheduleModal.tsx:68` `triggerMode:'stable_kline'`、`backendType:'event'`），其 `next_run_at` 恒为 nil（`model/strategy_schedule.go:254` 注释明示 event-driven 返回零值）。定时引擎只捞 `next_run_at IS NOT NULL`（`schedule_read_repo.go:145` `GetDueSchedules`）。

`grep` 全仓：**没有任何"bar 收盘 → 查匹配 schedule → 触发"的事件派发器**。schedule 仓库的 `GetByAccountID`/`GetByTemplateID` 等事件导向查询，在生产代码里除引擎/测试外无人调用。

→ 事件型 schedule 建了、存了、退款计数也算了，但**永远不会被执行**。`ToggleSchedule(active=true)` 对事件型只 `engine.Notify()` 唤醒定时器，而定时器不会捞 nil `next_run_at`——激活等于空操作。

### 缺陷 C（安全/商业承诺缺口）— 无购买授权校验

- `CreateSchedule`（`strategy_schedules.go:67`）只拦 `is_system` 模板。买方拿发布者的 `strategy_id` 建 schedule 时，`GetTemplate(strategy_id, buyer_uid)` 直接 `ErrTemplateNotFound`（发布者模板 `is_public` 默认 false，`publish.go` 不改它），`err==nil && tpl.IsSystem` 不命中 → 不拦截 → schedule 照建（轻微 IDOR：买方能引用任意 `template_id`，只是当前跑不起来）。
- 全链路**无一处**校验"运行某策略前，该用户是否持有有效订阅"。即便 A、B 修好，未购买者也能跑发布者策略。
- "代码不出平台"无执行层保障：当前唯一能跑通的 `StartStrategy`（`strategy_active_handlers.go:205`）是前端把整段 `strategy_code` 发上来——若买方路径复用它，等于把发布者源码送到前端。

### 验证结论

**购买侧 ✅ 正常；部署→运行侧 ❌ 端到端不通。** 不是市场模块的 bug，是 `strategy-runtime` 实盘调度对用户/市场策略从未真正实现。

---

## 二、设计决策（评审重点）

### 决策 1：事件型 schedule = 持久化流式会话（非逐 bar 派发）

事件型 schedule（kline_close / hf_quote）的正确运行模型是**一个长生命周期的 `RunLiveStrategy` 流式会话**，而非定时器逐 bar 查库派发：

- **理由（push-first 第一性）**：实盘策略必须订阅 bar 流、对每根 bar 做反应——这是长会话，不是逐 bar DB 轮询。`RunLiveStrategy` 本就是订阅 `barBroker` channel 的长驻循环（`live_runner.go:90`），完美匹配。
- **基础设施已就绪**：`RunLiveStrategy` + `sessionRegistry`（`session_registry.go`）+ `StopSchedule`（`schedule_engine.go:321` cancel 会话）+ `reconcileOnStartup`（重启恢复）全部存在。
- **定时型 schedule 保持现状**（cron/interval）的逐 tick `dispatch` 模型，只修缺陷 A 的取码源。
- **新增引擎入口** `launchEventSession(schedule)`：取码（服务端）→ 订阅授权闸 → `RunLiveStrategy`。触发点：`ToggleSchedule(active=true)`、`CreateSchedule`（若 active）、`reconcileOnStartup`。

> 备选方案（**不采用**）：逐 bar 事件派发器（on bar close → 查匹配 schedule → 跑一次）。与 `RunLiveStrategy` 流式模型冲突、重复实现 OnTick/OnTrade 订阅、且违背 push-first。本 spec 不走此路。

### 决策 2：源码永不下发前端——后端直接取码运行

- 事件型/定时型 schedule 的代码一律由**后端在 `dispatch`/`launchEventSession` 内服务端取码**：按 `schedule.TemplateID` 从 `strategy_templates.code` 取，直接喂 VM，**不经前端、不下发**。
- 前端 `DeployScheduleModal` 现状已只发 `templateId` + account/symbol/timeframe，不发 code——**前端无需改动**，修复几乎全在后端。
- **边界**：`StartStrategy`（workspace「Run」）保持现状——它服务用户**自己**的策略，用户本就持有源码，前端发 code 合法。**市场购买→实盘的唯一入口是 schedule 路径**（后端取码 + 授权闸），两条路径职责分离、互不污染。

### 决策 3：订阅授权闸放在运行时入口（纵深防御）

- 新增 `marketplace.Service.HasActiveEntitlement(ctx, userID, strategyID) (bool, error)`，复用 `purchase.go:109` 的订阅查询模式（`user_subscriptions WHERE subscriber_user_id=$1 AND target_strategy_id=$2 AND active=true`，含未过期判断），以 `trial.go:71 HasActiveTrial` 为模板。
- 在 `launchEventSession`（事件型）与 `dispatch`（定时型）**启动会话前**调用。未授权 → 不启动 + 记 `last_error` + 日志。**不**在 `CreateSchedule` 拦截（允许用户先建 schedule、后购买；运行时才闸）。
- 发布者本人对自己的策略免闸（`publisher_id == userID`）。

### 决策 4：调度引擎注入 strategy_templates 取码能力

- `ScheduleEngine` 当前只持有 `*AIStrategyTemplatesRepository`（缺陷 A 根因）。新增一个 `strategy_templates` 的只读取码依赖（复用 `service.StrategySvc.GetTemplate` 或新增仓库方法 `GetTemplateCode(ctx, id) (string, error)`），注入引擎，替换 `dispatch` 中的 `ai_strategy_templates` 查询。
- `AIStrategyTemplatesRepository` 保留（AI 生成骨架库仍有用），仅从调度运行时路径移除。

---

## 三、实现任务（文件/函数级）

### 任务 1：取码源修正（缺陷 A）

- `backend/internal/connect/strategy/schedule_engine.go`
  - `ScheduleEngine` 结构新增字段 `codeRepo`（`strategy_templates` 取码依赖）。
  - `NewScheduleEngine` 签名新增该依赖参数。
  - `dispatch`：`tpl, err := e.codeRepo.GetTemplateCode(ctx, schedule.TemplateID)` → `Code: code`。删除对 `AIStrategyTemplatesRepository` 的引用。
- `backend/cmd/server/handlers_strategy_runtime.go:144`：`NewScheduleEngine` 调用补传新依赖。
- 新增取码方法（择一）：
  - `NEW`：`repository` 层 `GetTemplateCode(ctx, id uuid.UUID) (string, error)` 查 `strategy_templates.code`；或
  - `REUSE`：复用 `service.StrategySvc.GetTemplate`（但 service 依赖较重，引擎目前不持有 `*StrategySvc`，建议走仓库方法保持依赖轻量）。

### 任务 2：事件型会话化（缺陷 B）

- `backend/internal/connect/strategy/schedule_engine.go`
  - 新增 `launchEventSession(ctx, schedule)`：**配额闸**（任务 5）→ 取码 → 授权闸 → 预建 run 记录 → `sessionRegistry.Register` → `go RunLiveStrategy`，纳入 `activeRuns`。
  - `dispatch` 内按 `schedule.ScheduleType`/`triggerMode` 分流：定时型走原逐 tick 逻辑；事件型走 `launchEventSession`（仅启动一次，非每 tick）。
  - `reconcileOnStartup`：对 active 事件型 schedule 也启动会话（当前只补 `next_run_at`）。
- `backend/internal/connect/strategy/strategy_schedules.go`
  - `ToggleSchedule(active=true)`：事件型 schedule 调 `engine.StartSchedule(id)`（新增），而非仅 `Notify()`。
  - `CreateSchedule`：若新建即 active 且为事件型，`engine.StartSchedule(id)`。
- 新增 `ScheduleEngine.StartSchedule(id uuid.UUID)`：加载 schedule → 事件型则 `launchEventSession`；与 `StopSchedule` 对称。
- **同 account 冲突（已决，见 §七-Q2）**：`launchEventSession` 在 `Register` 返回 nil（账户已有在跑会话）时，**不顶替、不重试**——标记预建 run 为 error + 友好报错"账户 X 已有运行中策略，请先停止"。绝不静默杀掉可能持仓的在跑会话。

### 任务 3：订阅授权闸（缺陷 C）

- `backend/internal/marketplace/`（新方法）
  - `func (s *Service) HasActiveEntitlement(ctx, userID, strategyID string) (bool, error)`：复用 `purchase.go:109` 订阅查询 + 过期判断；发布者本人免闸（`publisher_id == userID`）。以 `trial.go:71 HasActiveTrial` 为模板。**NEW**（已搜：仅有 `HasActiveTrial`，无购买态授权方法）。
- `backend/internal/connect/strategy/schedule_engine.go`
  - 引擎注入 `entitlement` 依赖（`func(ctx, userID, strategyID) bool`）。
  - `dispatch`/`launchEventSession` 启动前调用（**启动闸**）；未授权 → `UpdateLastRun(... "unauthorized: no active entitlement")` + return。
- `backend/cmd/server/handlers_strategy_runtime.go`：注入 entitlement 依赖。

### 任务 4：在跑会话的授权复验 + 撤销只停信号（缺陷 C 运行时闭环）

**问题**：启动闸只挡新增；已启动的市场策略会话在用户**退款/订阅过期/Admin 禁用**后仍会继续交易——收入泄漏 + 合规风险。

- `backend/internal/connect/strategy/live_runner.go`（事件循环，每个 bar 周期）
  - 对**市场购买型**会话（非用户自有策略），每 bar 复验授权：命中 TTL 缓存（~30s，仿 `schedule_engine.go:204 autoTradeCache`）则零 DB 开销；未命中查 `HasActiveEntitlement`。
  - **这是搭车 bar 驱动事件循环，非 `time.Ticker` 轮询——符合 push-first**。
  - 撤销 → 会话优雅自终止（deregister + run 记录标 stopped/revoked）。
  - 实现方式：`LiveStrategyConfig` 增加 `EntitlementCheck func(ctx) bool`（自有策略传 nil = 不校验）；事件循环每 bar 调用。
- **撤销时绝不自动平仓（已决，见 §七-Q3）**：只停新信号 + 强通知用户 + UI 提供"一键平仓"按钮（用户显式点击）。平台**不**主动平用户头寸——守住"不代客交易、不碰资金"牌照边界。
- 退款分支天然已处理：`refund.go:153` 当前拒绝在有 active schedule 时退款，不存在"边跑边退"。

### 任务 5：配额闸复用（缺陷 C + 风控）

- `launchEventSession` 取码后、`RunLiveStrategy` 前调 `checkStrategyQuota`（含 `CheckLiveStrategyLimit` live 子限额）。
- 超额 → 不启动 + `last_error` 记录 "live strategy limit reached"。
- schedule 启动的会话与手动 `StartStrategy` 在风控语义上等价（占 broker 连接 + 持仓敞口），**必须共用同一道配额闸**——既防 Free 用户借 schedule 绕过配额，也防风控漏洞。

### 任务 6：集成测试（固化链路）

- `backend/internal/connect/strategy/schedule_engine_test.go`（新建或扩充）：
  - 事件型 schedule 激活 → 启动会话 → 取到正确 code（来自 strategy_templates，非 ai_strategy_templates）。
  - 未订阅用户激活发布者策略 schedule → 被授权闸拦截、不启动。
  - 已订阅用户 → 正常启动。
  - 复用 mthub 的 PG 集成测试模式（`idempotency_integration_test.go`）。

### 任务 7：ADR-0029 留痕

- 新建 `docs/adr/0029-purchase-to-live-execution.md`，记录四个架构决策（事件型 schedule=持久流式会话 / 后端取码代码不出平台 / 运行时授权闸 / 撤销只停信号不平仓）+ 备选方案（逐 bar 派发为何不采用）+ 与 ADR-0023/0028 关系。决策已在本 spec §七定档，ADR 仅作留痕。

---

## 四、边界（本次不做）

- `ProtectedBacktestPanel`（买方受保护回测）是独立链路，不在本 spec。但取码/授权模式应保持一致，后续对齐。
- `StartStrategy` 手动路径不改（服务用户自有策略）。
- 多账户执行、实盘战绩不可篡改、AI 迭代闭环等均不在本 spec。
- `AIStrategyTemplatesRepository` 不删除（AI 生成仍用）。

---

## 五、约束

- 后端 Go +（如需）SQL migration。proto 不变（schedule 现有字段足够）。
- push-first：事件型走流式会话，**禁止**逐 bar 轮询查库派发。
- 金额相关无变化（本 spec 不碰钱包/结算）。
- 文件/函数规模遵守 `complexity-limits.md`；`go build ./...` + `go run ./tools/check-file-lines --strict` + `go test` 全绿。
- 复用核对：取码方法、授权方法在 PR 描述给 `REUSE:`/`NEW:` 结论。

---

## 六、验收

1. 已购买策略 → Deploy → 激活 → **实盘会话真正启动**（run 记录 running，sessionRegistry 有会话，日志取到 strategy_templates.code）。
2. 事件型 schedule（kline_close）激活后持续运行、bar 到达时执行策略；停用则会话停止。
3. 定时型 schedule（cron/interval）到点执行，代码来自 strategy_templates（非 ai_strategy_templates）。
4. **未订阅**用户激活发布者策略 → 被授权闸拦截，不启动，`last_error` 记录原因。
5. 发布者本人激活自己的策略 → 免闸，正常启动。
6. 源码全程不下发前端（抓 schedule/CreateSchedule 请求体无 code 字段）。
7. 服务重启 → active 事件型 schedule 自动恢复会话（`reconcileOnStartup`）。
8. `go build` + `check-file-lines --strict` + `go test` 全绿。
9. **配额**：超额用户激活事件型 schedule → 被 `CheckLiveStrategyLimit` 拦截，不启动。
10. **同 account 冲突**：账户已有在跑会话时，再次激活 → 被拒（不顶替原会话）。
11. **授权撤销**：会话运行中退款/订阅过期 → 每 bar 复验命中 → 会话自终止；**未平仓持仓保留**（用户自行一键平仓），不自动平仓。

---

## 七、已决议（评审通过 2026-08-06）

> 四个原"未决"项已定。统一原则：**平台是"付费授权"与"真金白银执行"之间的唯一咽喉——必须在每个执行入口强制校验、绝不可自动碰持仓、且决策可审计。** 风控与牌照边界优先级高于功能灵活性。

**Q1 配额（→ 任务 5）**：schedule 启动的会话**必须**走同一道 `CheckLiveStrategyLimit` 配额闸。理由：事件型会话 = broker 连接 + 持仓敞口，与手动 `StartStrategy` 风控语义等价；不闸 = Free 用户借 schedule 绕过配额 + 风控漏洞。

**Q2 同 account 冲突**：**先跑者赢、后来者拒、绝不静默替换**。`launchEventSession` 在 `Register` 返回 nil（账户占用）时标记 error + 友好报错，不顶替、不重试。理由：绝不静默杀掉可能持有未平仓的在跑会话——孤儿/冲突持仓是交易系统最糟故障。
> ⚠️ 顺带挖出的商业 bug（见 §八 衍生项 P1-MKT-1）：`sessionRegistry` 一账户一会话，与 Pro 档"5 账户/20 实盘策略"售卖档位冲突——本 spec 守住现状，多策略共账户另行立项。

**Q3 授权撤销实时性（→ 任务 4）**：**启动闸 + 每 bar 复验（TTL 缓存，搭车 bar 事件循环、非 Ticker 轮询）+ 撤销只停信号不平仓**。撤销 → 会话优雅自终止。理由：复验统一覆盖退款/过期/Admin 禁用；**不自动平仓**是"不代客交易、不碰资金"牌照硬边界——平台提供一键平仓按钮（用户显式点），不行使权力。退款分支天然已处理（`refund.go:153` 拒绝在有 active schedule 时退款）。

**Q4 ADR-0029（→ 任务 7）**：**写**。这几个决策（事件型 schedule=持久流式会话、后端取码、运行时授权闸、撤销不平仓）是耐用架构决策、触碰核心运行时，单独立 ADR 留痕；本 spec 作实现规格。

---

## 八、衍生立项（已登记，不可遗忘）

> 本 spec 守住"购买→实盘链路打通"边界，下列项是验证中浮现、但超出本 spec 范围的工作。**已登记到 memory 遗留项总登记 + `GLM-master-task-list.md`，完工即销账。**

| ID | 项 | 为什么延期 | 触发时机 | 登记 |
|----|----|-----------|---------|------|
| **P1-MKT-1** | 多策略共账户（Magic Number 归因）→ 解开 Pro 档"20 实盘策略"容量 | **决策已定（2026-08-08）**：允许多策略共账户（决策 A）；**风控按 account 级聚合，不按策略**（决策 B——旧表述"按策略风控聚合"已废弃，改 magic 级会削弱安全）；magic 仅用于归因。**①-⑤ 已落地验收**（commit `e47ea7bb`：magic 打标 / close_all 隔离 / 多 session）；**step⑥ 归因闭环待施工**（trade_records.schedule_id 按 magic 回填），施工 spec 见 `docs/spec/multi-strategy-attribution-spec.md` | step⑥ 施工 | `GLM-master-task-list.md` P1-6 + `tech-debt-registry.md` ARCH-4 + memory |
| **P2-MKT-2** | `ProtectedBacktestPanel`（买方受保护回测）取码/授权模式与本 spec 对齐 | 独立链路，本 spec 不碰；但取码+授权应一致，避免两套分叉 | 下次触及受保护回测代码时 | memory |
| **ADR-0028 §7 对账** | §7"剩余"状态表与 commit `30668f64`（参数链 E2E）矛盾，需核对"端到端测试(参数链)"是否真闭环、并刷新 §7 | 文档漂移，非代码问题 | 下次触及 ADR-0028 时 | memory |
