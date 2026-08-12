# 施工交接批次 2：CREATE-SCHEDULE-200EMPTY（含 DEPLOY-LIVE-3）+ DEPLOY-LIVE-1-COVERAGE 补强

> 审计方（Claude Code）2026-08-12 DEPLOY-LIVE-1/2 验收通过（commit `1a54ec21`）后发出。
> 施工方（Windsurf）任务：**one task = one scope，只做本批次两件事**。**不重新审计、不自由发挥、不扩大范围。**
> 主文档：`builder-handoff-create-schedule-200empty-2026-08-12.md`（200EMPTY 根因/修复/冒烟/验收全部细节在此，本文件是其执行批次外壳 + 增量）。

---

## 一、批次范围

| # | 任务 | 引用 |
|---|------|------|
| 1 | **CREATE-SCHEDULE-200EMPTY + DEPLOY-LIVE-3**（P1 接线修复）| 主文档 §二~五（修复步骤/验收/对抗）|
| 2 | **DEPLOY-LIVE-1-COVERAGE**（审计方新发现覆盖缺口，补 1 个 live 路径测试）| 本文件 §四 |

## 二、现状纠偏（必读，上一轮你没做到位）

1. **你的 a3416a95 只做了"掩盖"没做"修复"**：`bound_account_svc.go:35-40` 的 `defer recover()` 把 panic 转 500——这是掩盖不是修复（根因=接线，未动）。**本批次必须：删除 recover + 做 `handlers.go:191` 的 `BoundSvc: boundSvc` 一行接线修复**。当前部署的容器二进制（Aug 12 22:25）**仍带 recover 掩盖**，症状 = 500 "ensure bound: panic: ..."，接线未通。
2. **主文档 §二 根因链已定论**（审计方逐环实证），不要重新排查、不要质疑。你上轮"nil 字段未定位"的结论已被审计方补全：nil 来源 = `handlers_strategy_runtime.go:87` `SetBoundSvc(boundSvc)` 收到 nil 的 `boundSvc`（`handlers.go:191` 漏传）→ typed-nil 接口 → `bound_account_svc.go:41` nil 接收者解引用。
3. **对照**：`handlers.go:126` `registerPostAccountDeps` 同源变量 `boundSvc` 有传——直接用同名变量即可。

## 三、任务 1 执行要点（细节以主文档为准）

1. `backend/cmd/server/handlers.go:191` `strategyTradingParams{...}` 加 `BoundSvc: boundSvc,`（一行）。
2. `backend/internal/service/bound_account_svc.go` 删除 `EnsureBoundAccount` 的 `(retErr error)` 命名返回值 + `defer recover()` 块，恢复原签名 `func (s *BoundAccountService) EnsureBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error`。
3. 门禁：`go build ./...` + `go test ./internal/connect/strategy/... ./internal/service/...`。
4. **唯一合法部署**：`rtk proxy docker compose build backend && rtk proxy docker compose up -d backend`（禁宿主机 go build → docker cp；禁容器内 build）。
5. **冒烟（主文档 §三.5 的 curl 流程）**：Login → CreateSchedule（用户模板 `8403ffab-5840-4825-acb3-7b042f41db59` + 账户 `904d14e6-8d67-4541-80f9-f3b7f9587a00`）→ 断言 200 + body 含 `id` + `strategy_schedules` count 增加 → **UpdateSchedule 切账户补测（DEPLOY-LIVE-3）**：换另一账户 UUID → 200 + `account_id` 变更（DB 断言）→ DeleteSchedule 清理。
6. 回填（主文档 §七）。

## 四、任务 2：DEPLOY-LIVE-1-COVERAGE 补强测试（新）

**背景（审计方实测）**：DEPLOY-LIVE-1 的 live 调用点无直接对抗测试——把 `live_dispatch.go:63/74` 的 `barOpenTimeForSignal(bar, cfg)` 还原为 `bar.OpenTime` 后，`go test ./internal/connect/strategy/` **仍全绿**。原因：现有 nil bar 测试走 paper 分支（`live_dispatch.go:42-44` 提前 return）；live 路径 `s.mtHub == nil` 时 :47-50 early return。

**要求（补 1 个测试到 `deploy_live_test.go`）**：

```go
// live 路径：非 paper + 非 nil mtHub → dispatchLiveSignal 遇 nil bar 不 panic，
// 且 PlaceOrder 收到的 barOpenTime 来自 TickSeq（两次连续 tick 单 ClientID 不同）。
func TestDeployLive1_LivePathNilBarNoPanic(t *testing.T) {
    // mock mthub.MtHubService：实现 PlaceOrder 记录 req（含 ClientID）返 ticket 无 err。
    // srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: mockHub}
    // cfg := LiveStrategyConfig{AccountID: "...", UserID: "...", Symbol: "EURUSD",
    //     Mode: "live", RunID: uuid.New(), TickSeq: new(atomic.Int64)}
    // srv.dispatchLiveSignal(ctx, cfg, nil, &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}, nil)
    // 断言：不 panic + mockHub 收到 1 个 PlaceOrder + 两次连续调用 ClientID 不同。
}
```

**实现提示**：`mthub.MtHubService` 接口字段很大——用现有 mock（搜 `internal/mthub` 或 strategy 包内 mock，如 `service_orders_unit_test.go` 的 mock）或最小手写 mock 只实现 `PlaceOrder`（方法签名见 `submitOrder` 调用：`s.mtHub.PlaceOrder(ctx, req) (record, err)`）。注意 `dispatchLiveSignal` 会先调 `persistSignal`（`runRepo == nil` 时安全返回，mock 里可留 nil）+ 熔断检查（`activeSess == nil` 跳过）+ `submitOrder` 内部 go routine（测试需等 channel/原子标志确认 PlaceOrder 被调，或直接用 barrier/close(ch) 同步）。

**对抗证明（必做）**：临时把 `live_dispatch.go:63` 改回 `s.dispatchMarketOrder(ctx, cfg, bar.OpenTime, ...)` → 本测试 **必须 RED（panic）** → 还原 → GREEN。实测记录红/绿各一行。

## 五、对抗证明汇总（本批次必做，删了不红 = 未完成）

| # | 场景 | 红（修复前/删行） | 绿（修复后） |
|---|------|------------------|-------------|
| 1 | 合法 CreateSchedule（当前容器 = recover 掩盖版）| 500 `"ensure bound: panic: ..."` | 200 + JSON 含 id + DB 记录 |
| 2 | 删 `handlers.go:191` `BoundSvc: boundSvc` 一行 + 重建 | panic（200 空 或 500 带 panic）| 200 + JSON 含 id |
| 3 | UpdateSchedule 切账户（DEPLOY-LIVE-3）| 500/200 空 | 200 + account_id 变更 |
| 4 | 新测试：live 调用点还原 `bar.OpenTime` | 测试 RED（panic）| 测试 GREEN |
| 5 | 回归：DEPLOY-LIVE-1/2 现有测试（deploy_live_test.go 6 个 + TestMt4Op）| — | 全绿 |

## 六、红队自审（任务级 edge cases，逐条给出结论）

- [ ] `handlers.go` 里 `boundSvc` 变量在 :191 处作用域可见（:90 定义，:126 用过）——核对后直接用
- [ ] `bound_account_svc.go` 删 recover 后：`EnsureBoundAccount` 签名恢复原样，调用方（strategy_schedules.go:75/applyAccountSwitch）无编译影响
- [ ] 冒烟模板 `8403ffab-...` 是**用户模板**（'E2E 复刻'），系统模板会 403
- [ ] 账户 `904d14e6-...` 状态（reconnecting）——EnsureBoundAccount 只查归属+绑定，不查连接状态；若失败如实记录
- [ ] UpdateSchedule 切账户用第二个账户 UUID（可从 ListAccounts 拿），切完断言 DB `account_id` 变更
- [ ] 冒烟后清理（DeleteSchedule），不留脏数据
- [ ] 部署前查 `git status backend/migrations/`——有未提交 WIP `.up.sql` 先移走
- [ ] 提交核对：registry/handover 变更日志只追加不删（pre-commit 钩子拦删，被拦 = 改好文档再提交，禁 `--no-verify`）
- [ ] 新测试的 mock 不碰真实网络/DB；`submitOrder` 的 go routine 用同步机制等待，禁 sleep
- [ ] 你上轮 a3416a95 里的 `DeployScheduleModal.tsx` SymbolPicker 改动（Form.Item 管理 value）属 UI 顺手改动——保留可，但**不要**再顺手改其他

## 七、回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：CREATE-SCHEDULE-200EMPTY 条目 `🟦open → ✅done`（标日期）+ 追加真实修复记录（commit、冒烟输出、对抗红绿）；DEPLOY-LIVE-3 条目 `🟦open → ✅done`（补切账户实测记录）；DEPLOY-LIVE-1-COVERAGE 条目 `🟦open → ✅done`（补强测试 + 删行红绿）。若真实根因与审计方不同，如实写明。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. 完成后本文件 + 主文档移到 `docs/audits/archive/`（审计方验收后处理）。

## 八、沟通

- 完成后报告：修复动作 + 冒烟输出 + 对抗红绿记录 + 回填位置。**不自行宣告完成**，等审计方核对状态 + 实测后 ✅ 才权威。
