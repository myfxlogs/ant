# 施工交接批次 4：DEPLOY-LIVE-8（P1 执行链断裂）+ Live 页 UI 重设计（监视主视图）

> 审计方（Claude Code）2026-08-12 用户端到端实测暴露 P1 后发出（用户已确认 UI 布局方向 = **监视主视图**）。
> 施工方（Windsurf）任务：**one task = one scope，只做本批次三件事**。**不重新审计、不自由发挥、不扩大范围。**
> 审计方角色提醒：**本批次全部由你实现**，审计方只做验收（独立删行复测 + 冒烟实测）。

---

## 一、批次范围

| # | 任务 | 级别 | 位置（审计方已逐行核对） |
|---|------|------|--------------------------|
| 1 | **DEPLOY-LIVE-8**：调度启用即死（执行链断裂）| **P1** | `strategy_schedules.go:221` + `schedule_engine.go:326`（buildLiveRun）+ `schedule_engine.go` Start |
| 2 | **ActiveStrategy 增强**（UI 前置数据契约）| P1 配套 | proto `strategy_runtime.proto:475` + `strategy_active_handlers.go:158` |
| 3 | **Live 页 UI 重设计**（监视主视图，用户已确认方案）| P1 配套 | `LiveStrategyPage.tsx` + `LiveSchedulesTab.tsx` + `DeployScheduleModal.tsx` |

## 二、任务 1：DEPLOY-LIVE-8（P1，启用即死）

**根因（审计方实测证据链完整，勿重新排查）**：
- `strategy_schedules.go:221` `ToggleSchedule` 调 `_ = s.engine.StartSchedule(ctx, id)` 传 **ConnectRPC handler ctx**（原 :217，LIVE-7b 施工后 +4）
- `schedule_engine.go:326`（`buildLiveRun` 内，LIVE-6 `cd3416b1` 抽公共函数后的 **P1 当前点**）`runCtx, cancel := context.WithCancel(ctx)` —— runCtx 继承 handler ctx
- handler 返回 → ConnectRPC 框架 cancel 请求 ctx → `live_runner.go:270` `runLiveEventLoop` 收到 `runCtx.Done()` 退出
- **实测**（用户 2026-08-12 23:55 端到端）：run `fbef8bfc` **28ms 即死**（started 15:55:42.761 → stopped 15:55:42.789，0 信号 0 错误）；日志 `LiveStrategyRunner: starting` → 2.3ms 后 `context cancelled, exiting` → `run completed`（静默假成功）
- **⚠️ 位置说明（勿困惑）**：批次3 施工（LIVE-6 抽 `buildLiveRun`，commit `cd3416b1`）已把四道门 + runCtx 构造移入 `schedule_engine.go:276-378` 公共函数，`launchEventSession`（schedule_event.go:52-76）只调它。**抽函数时行为保持不变，P1 未修**：dispatch 路径（executeLoop ← Start 引擎 ctx）正确，launchEventSession 路径（StartSchedule ← handler ctx）仍错——两路径共用 `buildLiveRun:326`，修复点在公共函数内

**修复（契约）**：
1. `ScheduleEngine` struct 加字段 `lifecycleCtx context.Context`（schedule_engine.go:43-57）
2. `func (e *ScheduleEngine) Start(ctx context.Context) error` 开头保存 `e.lifecycleCtx = ctx`（:98，`reconcileOnStartup` 之前）
3. `buildLiveRun` 里 `runCtx, cancel := context.WithCancel(ctx)`（**schedule_engine.go:326**）改为 `context.WithCancel(e.lifecycleCtx)`（run 生命周期 = 引擎生命周期；`Stop()`/`StopSchedule` 仍走 handle.cancel() 双保险 :425/:436）
4. **nil 守卫（必做，审计方已核对 main.go 启动顺序，非可选）**：`main.go:223` 是 `go func() { _ = scheduleEngine.Start(ctx) }()` **goroutine 启动**——与 handler 请求完全并发、无顺序保证 → 首个 ToggleSchedule 请求可能先于 `e.lifecycleCtx = ctx` 赋值到达 → `context.WithCancel(nil)` **panic**（context 包直接 panic）。取 lifecycleCtx 时 nil → 退化 `context.Background()`（或 Start 内互斥保证原子性）
5. `StartSchedule` 内 `GetByID` 等快路径 DB 查询保留 handler ctx（无 goroutine 泄漏，无需改）
6. `ToggleSchedule` 调用不用改（ctx 只用于快查询）

**对抗证明（必做，删行必红）**：
- 集成测试：构造**已 cancel 的 ctx** 调 `launchEventSession`（入口 schedule_event.go:52 不变，内部走 buildLiveRun）→ run 仍启动且持续（断言 `activeRuns` 含该 schedule + run 记录 status=running）——证明 run 不再依赖调用方 ctx
- 删行实验：`schedule_engine.go:326` `context.WithCancel(e.lifecycleCtx)` 还原为 `context.WithCancel(ctx)` → 测试 **RED**（run 立即退出）→ 还原 → GREEN
- 守卫测试：engine 从未 Start（lifecycleCtx nil）时调 launchEventSession → 不 panic（run 用 background ctx 兜底）
- 冒烟：启用调度 → run 持续 running 超过 1 分钟（原 28ms 死）→ 等 15m bar 收盘观察信号（可先看 `session registered` 后不再 `context cancelled`）

## 三、任务 2：ActiveStrategy 增强（运行监视 UI 的数据前置）

**现状**：`ActiveStrategy`（`proto/ant/v1/strategy_runtime.proto:475-488`）无 schedule_id/策略名 → 运行监视无法显示策略名、无法跳日志/健康页。

**契约**：
1. proto 加字段（append，不动已有编号）：`string schedule_id = 13;` + `string strategy_name = 14;`
2. `activeSessionToProto`（`strategy_active_handlers.go:158-176`）填 `ScheduleId: sess.ScheduleID.String()`（ActiveSession 已有 ScheduleID，纯字段无 repo）
3. `strategy_name`：在**调用点**（watchActive 组装 / GetActiveStrategy）批量查 `strategy_schedules.name`（按 schedule_id；用 `repository.NewStrategyScheduleRepository(pool)` 或 srv 已注入的 schedule 能力，注入方式以现有模式为准）。查不到（非调度启动的手动 run）→ name 留空，前端回退显示 runId
4. 重新生成 proto（前端 `gen/` + 后端 `gen/` 同步）

## 四、任务 3：Live 页 UI 重设计（监视主视图，用户已确认）

**设计决策（审计方定稿，照做）**：

1. **tab 顺序**：`active`（运行监视，默认）→ `schedules`（调度管理）→ `history`（运行历史）
2. **tab1 运行监视增强**（`LiveStrategyPage.tsx` activeColumns）：
   - 新增"策略名"列：`strategyName`（新字段；空则回退 `shortId(runId)`）
   - 行操作补齐：信号流（已有 Watch Signals）+ **日志**（跳 `/strategy/schedules/${scheduleId}/logs`，scheduleId 为空 disabled）+ **健康**（跳 `?tab=schedules&healthId=${scheduleId}`，同上 disabled）+ 停止（已有）
   - 空态：`Empty` 描述改为引导文案 + "去调度管理"按钮（切 tab）
3. **创建闭环**：
   - `DeployScheduleModal` 创建成功 → 跳 `?tab=schedules&scheduleId=xxx`（现状保留：高亮新调度 + 引导 Enable）
   - `LiveSchedulesTab` Enable 成功 → **自动跳 tab1 运行监视**（`?tab=active`，回调/URL 方式以现有路由模式为准）
4. **状态诚实（调度列表 `ScheduleTable.tsx:143-147`）**：
   - ❌ 删除误导性绿色"运行中" Tag（= is_active 开关，与 run 实际存活脱节——P1 即死时仍显示运行中）
   - ✅ 改为：`已启用/未启用` 文本 + `last_run_at`（上次运行时间）+ `last_error` 非空 → 红色错误显示（现有字段，无新后端依赖）
   - 实时存活状态由 tab1 运行监视（watchActive 流）承担——第1性原则：实时状态在主视图，配置态在调度管理
5. **健康检查联动**：`LiveStrategyPage` 读 `?tab=schedules&healthId=xxx` → 传 `LiveSchedulesTab` → 挂载时自动打开该调度 ScheduleHealthModal（复用现有 modal，勿新建）

**对抗证明（必做）**：
- Enable 成功自动跳 tab1：e2e 或组件测试断言（删联动行 → 测试 RED）
- 状态诚实：构造 last_error 数据 → 红色错误显示（删渲染行 → RED）
- 日志/健康按钮 scheduleId 空时 disabled：断言
- 回归：tsc 0err / vitest / npm build / go build / proto 生成同步

## 五、对抗证明汇总

| # | 场景 | 红（删行/修复前） | 绿（修复后） |
|---|------|------------------|-------------|
| 1 | cancel ctx 调 launchEventSession | run 立即退出 | run 持续 running（断言级）|
| 2 | 冒烟：启用调度 | run 28ms 死 | run 持续 >1min |
| 3 | 删 Enable→tab1 联动行 | 不跳转 | 自动跳运行监视 |
| 4 | 删 last_error 渲染 | 错误不可见 | 红色错误显示 |
| 5 | 回归：全量门禁 | — | 全绿 |

## 六、红队自审（逐条给出结论）

- [x] P1：**`lifecycleCtx` nil 守卫必须做（审计方已核对启动顺序，非可选）**：`main.go:223` 是 `go func() { _ = scheduleEngine.Start(ctx) }()` **goroutine 启动**——与 handler 服务请求完全并发、无顺序保证 → 首个 ToggleSchedule 请求可能先于 `e.lifecycleCtx = ctx` 赋值到达 → `context.WithCancel(nil)` **panic**（context 包 `WithCancel(nil)` 直接 panic "cannot create context from nil parent"）。**必须**：`buildLiveRun`（schedule_engine.go:326 修复点）取 lifecycleCtx 时 nil 则退化为 `context.Background()`（或 `Start` 内用互斥保证原子性），勿依赖启动顺序
- [ ] P1：`Stop()`（引擎关闭）仍正确 cancel 所有 run（lifecycleCtx 取消 + handle.cancel 双路径）
- [ ] P2（本批连带核对，不改）：dispatch（executeLoop）路径 ctx 已是引擎 ctx 无误——确认无同类 handler ctx 泄漏（grep `StartSchedule\|launchEventSession\|dispatch(` 的调用方）
- [ ] proto 改动：前后端 gen 同步生成；`strategy_name` 查询失败路径（schedule 已删）不 panic（name 空字符串）
- [ ] UI：tab1 运行时 scheduleId 为空的 run（手动 StartStrategy 启动）按钮 disabled 且 tooltip 说明
- [ ] UI：healthId 联动在 modal 关闭后清理 URL（避免刷新残留）
- [ ] UI：Enable 跳转时保留高亮（scheduleId 参数）不冲突
- [ ] 门禁：`go build ./...` / `go test ./...`（相关包）/ `check-file-lines --strict` / 前端 `tsc` + `npm run build` + vitest
- [ ] 部署前查 `git status backend/migrations/`——有未提交 WIP `.up.sql` 先移走
- [ ] 唯一合法部署：`rtk proxy docker compose build backend && rtk proxy docker compose up -d backend` + 前端 `docker cp` 流程（禁宿主机 go build）
- [ ] 提交核对：registry/handover 变更日志只追加不删（pre-commit 钩子拦删，禁 `--no-verify`）

## 七、回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：DEPLOY-LIVE-8 条目 `🟦open → ✅done` + 追加真实修复记录（commit、集成测试输出、冒烟 run 持续时长、删行红绿）。**若真实根因与审计方不同，如实写明**。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行（含 UI 重设计完成说明）。
3. 完成后本文件移到 `docs/audits/archive/`（审计方验收后处理）。

## 八、沟通

- 完成后报告：每任务修复动作 + 对抗红绿记录 + 冒烟输出（run 持续时长实证）+ 回填位置。**不自行宣告完成**，等审计方核对状态 + 独立删行复测 + 冒烟实测后 ✅ 才权威。
