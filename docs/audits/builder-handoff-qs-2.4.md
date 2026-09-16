# builder-handoff-qs-2.4 — VM 管线 data race 审计（只审不改）

> 修正 v1 @2026-09-16（随 QS-2.2 验收后发单）

## 立项背景

VM 管线质量方案 v2 阶段 2 第二单（spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` §4.2，registry QS-2.4）。QS-2.2 已落地 goleak 门禁（`174b8405` ✅done）；本单是**审计型任务**：跑 race×3 三棵包树 + 手工审计两处共享状态，产出审计报告。**发现即立债**——债务登记由决策方在验收时做，施工方不改生产代码、不登记 registry。

## 设计 SSOT

- spec §4.2（本任务唯一范围依据）+ `AGENTS.md` + `.devin/rules/dual-terminal-roles.md`。
- 前序验收口径：QS-2.2 已验证 `-race -count=3 ./internal/connect/strategy/` 绿（301s），本单需将其纳入三树统一证据并补齐另两树。

## 约束与目标

1. **只审不改**：不改任何生产代码、不改测试断言、不修发现的 race——全部记入报告由决策方裁定立债。
2. 交付物：`docs/audits/qs-2.4-race-audit.md`（审计报告，唯一可新建文件）。
3. 串行，勿部署、勿 push、禁 `--no-verify`；commit 用 `ANT_ROLE=builder git commit` 前缀。
4. 完工按 D-012 自审+D-014 末行 `[施工完成:QS-2.4] @<commit-hash>` 自报，停手等复审。

## S1 — race×3 三树证据（机械执行）

逐树跑并**原样留存输出**（含 race report 全文，若有）：

```bash
cd backend
go test -race -count=3 ./tools/mql2go/...
go test -race -count=3 ./internal/connect/strategy/...
go test -race -count=3 ./strategy/...
```

- 坐标已核实：`./strategy/` 顶层无 `_test.go` 但子包有 29 个测试文件（sdk/indicators/runner/backtest）——race 覆盖真实非空转；若某子包实际无测试文件被编译进 run，在报告中标注覆盖空洞。
- 判定：任何 `WARNING: DATA RACE` 原样截取完整栈存入报告，**禁止**自行修复或加 `-run` 跳过。

## S2 — PositionCache mutex 覆盖审计（坐标已核实，逐项填表）

文件：`backend/internal/connect/strategy/position_cache.go`（239 行，4 可变字段全为非导出：`mu`/`snapshots`/`financialsReceivedAt`/`positionsReceivedAt`，`log` 构造后不可变）。

已预核实的事实（复核确认即可，如有出入立即 `[转交决策]`）：

- 写字段路径：`put` :85 Lock、`Unsubscribe` :148 Lock；读字段：`GetSnapshot` :157 RLock、三个 `GetFresh*` :165/:184/:208 RLock。
- 字段仅本文件访问（全仓 grep 无外部触碰）。
- `put` 从不原地改已存快照——总是 `merged := *current`/`*snap` 拷贝后存新指针（:94/:109/:120/:134）。

**审计点（报告逐项给结论）**：

- a) `GetFresh*` 系列在 `RUnlock` 后继续读 `snap.*` 字段（:169-178 等）——论证是否安全： retained 快照对象是否 immutable（put 只换指针不原地改 → 是），并给出证据行号。
- b) **返回指针共享契约**：`GetSnapshot`/三个 `GetFresh*` 返回 `*mthub.PositionSnapshot` 裸指针——grep 全部调用方（`cmd/server`、`internal/connect/strategy` 及其他包），检查是否存在调用方原地写 `snap.Positions`/字段的代码路径；若有 → race 隐患立为发现项。
- c) `Subscribe` goroutine（:54-77）：`c.put` 内部自锁、recover 已存在（:59-65）、unsub defer——确认无遗漏共享访问。

## S3 — TradeBarrier cond/Broadcast 配对审计（坐标已核实，逐项填表）

文件：`backend/internal/connect/strategy/trade_barrier.go`（447 行）。

已预核实坐标：**10 处 `cond.Broadcast()`**（:200/:205/:237/:269/:285/:303/:341/:357/:377/:397）与 **2 处 `cond.Wait()`**（:320 WaitConfirmed、:411 WaitState）。

**审计点**：

- a) 每处 `Broadcast` 是否都在 `b.mu` 持锁内（含 watcher goroutine 内 :302-304 的 `mu.Lock→Broadcast→Unlock`）——逐行列出所在函数与持锁证据行号。
- b) 两处 `cond.Wait()` 是否在持锁 for-loop 内且醒后重检条件（`sync.Cond` 正确范式）。
- c) `NotifyOutcomeUnknown` :281-285 的幂等分支（state 已是 outcomeUnknown 时 early return 不 Broadcast）——确认不 Broadcast 时无等待方会被饿死（WaitConfirmed/WaitState 醒后重检终端态，论证或立发现项）。
- d) 死锁形态检查：是否存在持 `b.mu` 时调用会再取 `b.mu` 的路径（如 Broadcast 前后调用外部回调）。

## S4 — 报告落盘 + commit

`docs/audits/qs-2.4-race-audit.md` 结构：

```
# QS-2.4 VM 管线 data race 审计报告
## 1. race×3 三树证据（原样输出摘录 + 结论）
## 2. PositionCache 审计表（S2a/b/c 逐项：结论 + 证据行号）
## 3. TradeBarrier 审计表（S3a/b/c/d 逐项：结论 + 证据行号）
## 4. 发现项清单（每条：位置/证据/严重度建议/建议债务标题；无发现写"无"）
## 5. 覆盖空洞与局限（无并发测试的共享结构、审计未覆盖面）
```

commit：只显式 `git add docs/audits/qs-2.4-race-audit.md`（报告属审计层但为本任务交付物，比照 QS-1.7-INV 先例施工方落盘），`ANT_ROLE=builder git commit`。

## 验收标准

- [ ] 三树 `-race -count=3` 输出原样入报告；有 race → 完整栈 + 未修复标注。
- [ ] S2/S3 审计表逐项有结论+行号证据，无"看了没问题"式空结论。
- [ ] S2b 调用方 grep 清单完整（每个调用方文件:行 + 是否只读）。
- [ ] 零生产/测试代码改动（`git diff HEAD~1 --stat` 只有报告文件 + go.mod/sum 不应动）。
- [ ] `git status` 干净，无交接层文件改动。

## 对抗证明（审计任务形态）

- race 检测器本身即对抗条件；报告须含 `-race` 实跑证据（非 `-count=1` 普通输出冒充）。
- 若三树全绿：报告须说明"goleak 已绿（QS-2.2）+ race 全绿 + 手工审计无发现"三者关系，区分"已证无泄漏"与"未证无 race 但未发现"。

## D-013 出件自审记录（决策方）

- 坐标回验：`position_cache.go` 全文已读（239 行，4 字段/6 方法坐标如上）；`trade_barrier.go` Broadcast×11/Wait×2 行号经 grep 核实；`./strategy/` 29 测试文件经 find 核实；PositionCache 字段外部零触碰经 grep 核实。
- 路径可达性：S2b 返回指针共享契约为本单新增审计点（出件自审发现：RLock 内读字段安全，但裸指针返回后调用方可能原地写——已列为必查项）。
- 生命周期：Subscribe goroutine 生命周期=ctx；watcher goroutine 生命周期=WaitConfirmed/WaitState 调用期——均在审计点覆盖。
- 冲突检查：不改码原则与 spec"发现即立债"一致（债务登记归决策方）；QS-2.2 豁免惯例沿用。
- 复用核对：无新建设施；报告格式比照 `qs-1.7-inv-findings.md` 先例。
- **署名**：最终决策：Devin CLI（[角色:决策终] 激活）
