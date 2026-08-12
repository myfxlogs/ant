# 施工交接批次 3：DEPLOY-LIVE-4/5/6（P2）+ DEPLOY-LIVE-7（P3）

> 审计方（Claude Code）2026-08-12 batch2 验收通过（commit `b240a7ca`）后发出。
> 施工方（Windsurf）任务：**one task = one scope，只做本批次四件事**。**不重新审计、不自由发挥、不扩大范围。**
> 主文档：`docs/audits/tech-debt-registry.md` DEPLOY-LIVE 段（§DEPLOY-LIVE-4~7 条目，本文件是其执行批次外壳 + 增量细节）。

---

## 一、批次范围

| # | 任务 | 级别 | 位置（审计方已核对，直接照做） |
|---|------|------|-------------------------------|
| 1 | **DEPLOY-LIVE-4**：mthub gate fail-open → fail-closed | P2 | `backend/internal/mthub/service_orders.go:130-133` + `service_orders_close.go:103-106` |
| 2 | **DEPLOY-LIVE-5**：KYC GeoIP 空转 → 接真实 IP | P2 | `backend/cmd/server/handlers_strategy.go:116` |
| 3 | **DEPLOY-LIVE-6**：dispatch / launchEventSession ~100 行重复 → 抽公共函数 | P2 | `backend/internal/connect/strategy/schedule_engine.go:259-360` + `schedule_event.go:54-173` |
| 4 | **DEPLOY-LIVE-7**：a) handlers.go 死 gate 删除；b) WatchSchedules SSE 接通 | P3 | a) `backend/cmd/server/handlers.go:209-211`；b) `strategy_schedules.go:243` + 前端 LiveSchedulesTab |

## 二、现状（审计方逐行核对过，不要重新排查）

1. **DEPLOY-LIVE-4**：`evaluatePlaceGate`/`evaluateCloseGate` 开头 `if s.gate == nil || s.accountStateProvider == nil { return nil }`——gate/accountStateProvider 未注入时**静默放行所有单**。live_runner preflight 挡了 PlaceOrder 侧的 nil gate，但 **CloseOrder 无 preflight**（gate 空转仍放行）。**语义：gate 是风控咽喉，未配置时必须拒绝（fail-closed），不是放行。**
2. **DEPLOY-LIVE-5**：`setupRiskGate` 里 `ClientIPFn: func(ctx) string { return "" }` 恒空串 → `risksvc/jurisdiction.go:93` `clientIP != ""` 不满足 → **GeoIP/sanctioned 国家检查永远跳过**（RequireKYC/Disclaimer/Questionnaire 仍按 Store 走，不受影响）。**现成基建已存在，勿新写**：`interceptor.GetClientIP(ctx)`（`backend/internal/interceptor/auth.go:193`）——auth interceptor 已从 X-Real-IP/X-Forwarded-For 提取注入 ctx（**REUSE，勿重复造轮子**）。nginx 已转发 `X-Real-IP $remote_addr`（`nginx/nginx.conf:79` 等 7 处）。GeoIP resolver = MaxMind（`handlers_pipeline.go:40`，非 nil）。
3. **DEPLOY-LIVE-6**：两函数从「entitlement gate → quota gate → bound gate → template 加载 → GetParameters → runCtx/handle/activeRuns → entCheck → cfg 组装 → run record 创建」**逐行相同**。差异仅：① 失败语义（dispatch 只 `UpdateLastRun` 后 return；launch 返回 err）；② launch 尾部多 sessionRegistry 预注册（:158-164）。**LEAKAGE-1 接线漏传就是"同类代码两处、改一处漏一处"的教训——抽公共函数即根治此类。**
4. **DEPLOY-LIVE-7a**：`handlers.go:209-211` `gate := risk.NewDefaultGate()` + 两个 Setter——`gate.SetKillSwitch(...)` 是方法调用，**骗过编译器"declared and not used"检查**，但配置后的 gate **没有任何消费者**（不注入任何服务）= 配置不生效的死代码。正式 gate = `handlers_strategy.go:90-92` `setupRiskGate` → `srv.SetGate(gate)` + `mthubSvc.SetGate(gate)` **双咽喉**。
5. **DEPLOY-LIVE-7b**：`strategy_schedules.go:243` `WatchSchedules` 监听 `schedule_change`，**全 backend 零 `pg_notify('schedule_change')`** 触发方 → SSE 永不推送；前端 LiveSchedulesTab 用 list + mount/effect refresh（轮询式，违反 Push-First 架构）。

## 三、任务 1：DEPLOY-LIVE-4（fail-closed）

```go
// 修复前（service_orders.go:130-133 / service_orders_close.go:103-106）：
if s.gate == nil || s.accountStateProvider == nil {
    return nil
}
// 修复后：
if s.gate == nil {
    return fmt.Errorf("gate not configured: order rejected (fail-closed)")
}
if s.accountStateProvider == nil {
    return fmt.Errorf("account state provider not configured: order rejected (fail-closed)")
}
```

**⚠️ 必做预查（先于改码）**：全 `internal/mthub` 单测 + strategy 包测试里，构造 `MtHubService` 不带 gate 且期望下单成功的用例——fail-closed 后它们会红。**这是预期内的**：给这些用例注入最小 gate（`risk.NewDefaultGate()`）或断言 error。逐处核对后如实记录每个改动的测试文件。

**对抗证明（必做）**：
- 新测试：`gate = nil` → `PlaceOrder`/`CloseOrder` 返回 error（fail-closed 断言）
- 删行实验：把 fail-closed return 还原为 `return nil` → 新测试 **RED**（断言级）→ 还原 → GREEN
- 回归：mthub 全包测试绿（含你补的 gate 注入）

## 四、任务 2：DEPLOY-LIVE-5（一行修复 + 行为测试）

```go
// 修复前（handlers_strategy.go:116）：
ClientIPFn: func(ctx context.Context) string { return "" },
// 修复后：
ClientIPFn: interceptor.GetClientIP,
```

**注意**：`interceptor` = `alphaforge/internal/interceptor`，检查 import 冲突（handlers_strategy.go 已 import 的包）。

**对抗证明（必做）**：
- 行为测试（`internal/risk/rules_risksvc_test.go` 或新文件）：构造 `KycJurisdictionGateRule{ClientIPFn: 返回真实 IP}` + `JurisdictionGate{GeoIP: mock resolver 返回 sanctioned 国家, Store: mock 无 override}` → `Evaluate` 必须 **block**（ErrSanctionedCountry）；`ClientIPFn` 返回 "" → **放行**（现状行为，红/绿对照）
- 删行实验：删 `ClientIPFn: interceptor.GetClientIP` 行 → 测试 RED → 还原 → GREEN
- 若 `cfg.GeoIPDBPath` 未配置（MaxMind nil）如实记录——修复仍正确（IP 注入是前提）

## 五、任务 3：DEPLOY-LIVE-6（抽公共函数）

**目标**：四道门 + cfg 组装 + run record 全部进一个函数，两个调用点各剩 ~20 行。**加一道门从此只改一处。**

```go
// schedule_engine.go（新公共函数，行为不变）
// buildLiveRun runs the four pre-launch gates (entitlement/quota/bound/template),
// assembles LiveStrategyConfig, pre-creates the run record, and registers the
// run handle. Denied gates call repo.UpdateLastRun and return an error; callers
// decide whether to propagate (launch) or swallow (dispatch).
// Shared by dispatch and launchEventSession so a new gate can never be added
// to one path and missed on the other (LEAKAGE-1 lesson).
func (e *ScheduleEngine) buildLiveRun(ctx context.Context, schedule *model.StrategySchedule) (*LiveStrategyConfig, *runHandle, context.Context, error)
```

**契约（照做）**：
- 函数体 = 两函数完全相同的部分（entitlement → quota → bound → template → GetParameters → runCtx/handle → activeRuns → entCheck → cfg 组装 → run record）
- 门拒绝时：`repo.UpdateLastRun` + return error（dispatch 现在吞 err 的行为由调用点保持：`if _, _, _, err := e.buildLiveRun(...); err != nil { return }`）
- launch 的 sessionRegistry 预注册（schedule_event.go:158-164）**留在 launchEventSession**（dispatch 没有）
- dispatch/launchEventSession 尾部 goroutine + log 各留原样
- 命名/签名你可以微调，但**四道门+cfg+run record 必须全部在公共函数内**

**对抗证明（必做）**：
- 回归：现有 dispatch/launch 测试全绿（行为不变）
- **删行实验（本任务核心）**：删公共函数里 bound gate 一行（`if err := e.runner.checkBoundAccount(...)`）→ **dispatch 和 launch 两个路径的测试都 RED**——一处删行两路径同红的证据 = 重复已消除。还原 → 全绿
- 结构证据：`grep checkBoundAccount` 全 backend 只在公共函数出现 1 次（修复前 2 次：schedule_engine.go:283 + schedule_event.go:80）

## 六、任务 4：DEPLOY-LIVE-7（死代码 + 断链）

**a) 死 gate（3 行删除）**：删 `handlers.go:209-211`。删除后 `go build ./...` 必须绿（无其他引用，审计方已确认）。对抗 = 这是死代码，**"删了不红"本身就是证明**：删除前后 build 绿 + 行为不变（gate 功能由 handlers_strategy.go:90 双咽喉提供，未触碰）。验证：删后 grep 确认 `risk.NewDefaultGate()` 全 backend 仅剩 handlers_strategy.go:109 一处。

**b) WatchSchedules SSE 接通（主修方向 = 接通，勿删；Push-First 架构要求）**：
- 后端：四写路径成功后 `pg_notify('schedule_change')`——CreateSchedule / UpdateSchedule / DeleteSchedule / ToggleSchedule（`strategy_schedules.go` 对应 handler）。参考现有 pg_notify 模式（搜 `pg_notify` 在 strategy 包或 `pglisten` 用法，勿新造轮子）
- 前端：LiveSchedulesTab 消费 SSE（替换/补充 mount/effect refresh）。查现有 watch 基建（其他 tab 已有 SSE 消费模式，复用）
- **对抗证明**：集成测试或冒烟——ToggleSchedule 成功后 watch 流收到 schedule_change 消息；删 NOTIFY 行 → 收不到 → RED

## 七、对抗证明汇总（本批次必做，删了不红 = 未完成）

| # | 场景 | 红（删行/修复前） | 绿（修复后） |
|---|------|------------------|-------------|
| 1 | gate=nil → PlaceOrder/CloseOrder | 放行（`return nil`）| error（fail-closed 断言级）|
| 2 | ClientIPFn 恒 ""（删注入行）| sanctioned 国家放行 | 真实 IP → block |
| 3 | 删 buildLiveRun 里 bound gate 一行 | dispatch + launch **两路径**测试 RED | 全绿（删行两红 = 重复消除证据）|
| 4 | 删 handlers.go:209-211 死 gate | 删除前后 build 均绿（死代码证明）+ 行为不变 | 同左 |
| 5 | 删 NOTIFY 行 | SSE watch 收不到 schedule_change | 收到 |
| 6 | 回归：全量 go test + 前端 build | — | 全绿 |

## 八、红队自审（任务级 edge cases，逐条给出结论）

- [ ] LIVE-4：mthub 现有单测里不带 gate 的用例**逐个**核对（多少个、每个期望什么），fail-closed 后是补 gate 还是改断言——如实记录，勿"批量加 gate 了事"
- [ ] LIVE-4：CloseOrder 在 mtHub 里被哪些调用方触发（策略平仓/手动平仓/close_all）——fail-closed 后这些路径在 gate 缺失时行为是否可接受（生产 gate 恒注入，此检查是防御纵深）
- [ ] LIVE-5：`interceptor.GetClientIP` 依赖 auth interceptor 先注入 ClientIPKey——核对策略/下单请求都过 auth interceptor（ConnectRPC handler 层，应都过）；确认 `internal/interceptor` import 不循环
- [ ] LIVE-6：buildLiveRun 抽取后 dispatch 的失败语义不变（UpdateLastRun 仍调用）——launch 的 `return err` 仍传播给调用方；sessionRegistry 预注册只在 launch
- [ ] LIVE-6：抽取后 `runCtx`/`handle` 的 cancel 语义不变（goroutine 使用同一个 runCtx）
- [ ] LIVE-7a：删 3 行后 `risk` import 是否还在用（`risk.NewDefaultGate` 没了，但 handlers.go 是否还有 risk 其他引用——有就留 import，无就删）
- [ ] LIVE-7b：`pg_notify` 用 `pglisten` 现有通道（WatchSchedules 已 Listen，同 channel 直接复用）；前端 SSE 消费注意 cleanup（组件卸载断流，防泄漏）
- [ ] 门禁：`go build ./...` / `go test ./...`（相关包）/ `cd backend && go run ./tools/check-file-lines --strict` / 前端 `tsc` + `npm run build`
- [ ] 部署前查 `git status backend/migrations/`——有未提交 WIP `.up.sql` 先移走
- [ ] 唯一合法部署：`rtk proxy docker compose build backend && rtk proxy docker compose up -d backend`（禁宿主机 go build → docker cp；禁容器内 build）
- [ ] 提交核对：registry/handover 变更日志只追加不删（pre-commit 钩子拦删，被拦 = 改好文档再提交，禁 `--no-verify`）
- [ ] 四个任务各一个 commit（one task = one scope），或至少 commit message 明确对应任务号

## 九、回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：DEPLOY-LIVE-4/5/6/7 四条目 `🟦open → ✅done`（标日期）+ 追加真实修复记录（commit、测试输出、对抗红绿、mthub 测试逐个核对结果）。若真实根因/修复与审计方假设不同，**如实写明**（高价值纠偏）。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. 完成后本文件移到 `docs/audits/archive/`（审计方验收后处理）。

## 十、沟通

- 完成后报告：每任务修复动作 + 对抗红绿记录 + 回填位置 + mthub 测试核对清单。**不自行宣告完成**，等审计方核对状态 + 独立删行复测 + 冒烟实测后 ✅ 才权威。
