# Builder Handoff — OMS-EXIT-FIX（平仓/订单状态/判重/误标记修复批）

> 批次名：OMS-EXIT-FIX ｜ 日期：2026-08-15 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> 涉及功能块：`risk-gate`（OMS/风控）+ `strategy-runtime`（run 生命周期）
> 根因/证据全量见 `docs/audits/tech-debt-registry.md` 段「实盘"无法开仓"调查」（2026-08-14 审计方容器/DB/日志实测，铁证均在段内）。
> **与 LIVE-HARNESS-PARITY 返工（W1-W3）零文件冲突**（本批：mthub/risk/repository；彼批：connect/strategy）。可并行或串行，串行更稳（`handlers_strategy.go` 两批各有小 hunk）。

---

## 0. 背景（4 个缺陷，同一批实测发现）

| # | 缺陷 | 实测铁证 |
|---|---|---|
| CLOSE-ORDER-UUID（P1） | 平仓单 OMS ID 是合成字符串非 UUID → insert/transition 全 22P02 失败 → **平仓动作 OMS 零记录**（broker 平仓本身成功） | 每次平仓日志 `oms insert order: invalid input syntax for type uuid: "close-904d14e6-...-344012976"` |
| SUBMIT-STUCK-RACE（P1） | OMS 先写负数占位 ticket，真实 ticket 等 mtapi RPC 返回后回填；OnOrderUpdate 流事件**先于回填到达** → 按真实 ticket 查无 → 事件丢弃无重投 → **订单永久卡 SUBMITTED** | 17:04:01.404 `order not found by ticket` 比 17:04:01.407 ticket 回填**早 3ms**；该两单至今 SUBMITTED 而 broker 侧仓位真实存在 |
| DEDUP-5S-THROTTLE（P2） | 风控判重 key 缺 AccountID/Magic(ticket)——close 意图 symbol/side/price 全空、ticket 在 Magic → **任何账户任何 ticket 的两次平仓 5s 内互拒（还跨账户误伤）** | 17:02:01 平 344010713 被拒 `duplicate order detected within 5s`（3.4s 前平的是**另一个** ticket） |
| CLEANUP-MISFIRE（P2） | `CleanupStaleRuns` 启动时无差别把所有 running 行标 stopped "server restarted"——16:52:21 被一个**短暂第二后端实例**执行（当前容器 16:42 启动，不可能执行它）→ DB/UI 显示已停而 goroutine 实际存活交易 | 三个活 run 行 16:52:21 被标 stopped；54091a67 此后仍交易到 17:49 |

---

## 1. 任务分解

### Task 1 — CLOSE-ORDER-UUID：平仓单 OMS 记录修复（P1）

**文件**：`backend/internal/mthub/service_orders_close.go`

1. `closeOrderID := fmt.Sprintf("close-%s-%d", accountID, ticket)`（:22）**保留**——它只作 idem key（`idem.CheckAndSet/DeleteKey`，字符串 key 合法）。
2. 新增 OMS 行 ID（**REUSE `IdempotencyKey` 的 MD5-UUID 模式**，`oms_writer.go:205` 同款）：
   ```go
   omsOrderID := uuid.NewMD5(uuid.NameSpaceOID, []byte(closeOrderID)).String()
   ```
   确定性 → 同一 ticket 重复平仓幂等映射同一行（`ON CONFLICT (id) DO NOTHING` 已支持）。
3. :62 `InsertOrder(ctx, omsOrderID, ...)` 与 :68-70/:135/:144 所有 `omsTransition(ctx, omsOrderID, ...)` 全部换用新 ID。
4. 对抗测试：集成（参照 `oms_writer_integration_test.go` 模式）——CloseOrder 成功路径 → 断言 orders 表出现 `state=FILLED` 且 id 为合法 UUID 的平仓行（GREEN）；还原合成字符串 ID → insert 报错无行（RED）。

### Task 2 — SUBMIT-STUCK-RACE：流事件竞态丢弃修复（P1）

**文件**：`backend/internal/mthub/service_orders.go`（+接线 `cmd/server/pipeline.go`）

1. `TransitionOrderByTicket`（:325）查无（`OrderIDByTicket` 返回 err）时不再直接 return：
   - **第一次查无 → 进程内延迟重试一次**（`go func` + `time.AfterFunc(2s)`，detached ctx 5s 超时）——覆盖 ms 级回填竞态（实测窗口 3ms）；
   - **重试仍查无 → 触发对账修复**（**REUSE `ReconciliationLoop.TriggerReconcile`**，`reconciliation.go:62`——STREAM-KEEPALIVE 批已把它升级为修复型：SUBMITTED + broker 有 ticket → 按 broker 真相转 FILLED/CANCELLED/FAILED）。
2. MtHubService 需能触达 ReconciliationLoop：加字段 `reconcileTrigger func(accountID string)` + `SetReconcileTrigger(f)`；`cmd/server/pipeline.go`（reconLoop 装配处）接线 `mthubSvc.SetReconcileTrigger(reconLoop.TriggerReconcile)`。nil 时不触发（防御）。
3. 对抗测试：mock omsWriter——① ticket 已存在 → 正常 transition（回归）；② 首查无 + 2s 后有（模拟回填完成）→ 重试成功 transition（GREEN）；删重试 → 永不 transition（RED）；③ 重试仍无 → 断言 `reconcileTrigger` 被调用（GREEN）；删触发调用 → RED。测试注入短延迟（如 50ms）避免慢测。

### Task 3 — DEDUP-5S-THROTTLE：判重 key 修复（P2）

**文件**：`backend/internal/risk/rules.go`（`DuplicateProtection.Check` :282）

1. key 从 `symbol|side|volume|type|price` 扩为 **`account|symbol|side|volume|type|price|magic`**：
   ```go
   key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d",
       intent.GetAccountId(), intent.GetSymbol(), intent.GetSide(), intent.GetVolume(),
       intent.GetType(), intent.GetPrice(), intent.GetMagic())
   ```
   - 修 close 误伤：close 意图 ticket 在 `Magic`（`service_orders_close.go:114` `Magic: ticket`）→ 不同 ticket 不再互拒；
   - 修跨账户误伤：AccountID 入 key；
   - 保留真去重语义：同账户同策略（同 magic）同参数 5s 内重复单仍拒（place 路径 magic=策略 magic 不变）；同 ticket 重复 close 仍拒（idem 层也拦）。
2. 对抗测试：① 同账户**不同 ticket** 两个 close 意图 5s 内 → 第二个 **Allowed**（GREEN）；还原旧 key → 拒绝（RED）；② 不同账户同参数两个意图 → 第二个 Allowed（RED 验证用旧 key）；③ 同 key 5s 内重复 → 仍拒（防修过头）。

### Task 4 — CLEANUP-MISFIRE：启动清理不再误杀活 run（P2）

**文件**：`backend/internal/repository/strategy_run_repo.go`（:179）+ `backend/cmd/server/handlers_strategy.go`（:87 调用点）

1. `CleanupStaleRuns(ctx)` 加参数 `excludeIDs []uuid.UUID`，SQL 加 `AND id != ALL($1::uuid[])`（空数组用 `AND TRUE` 兼容或传非空才加）。
2. 调用点（handlers_strategy.go:87）：从 **REUSE `SessionRegistry.ListAll()`**（session_registry.go:280）取本进程活 run ID 传入。**调用时机问题**：当前在 `configureStrategyExecution` 内（registry 尚未注册 run，excludeIDs 恒空）——但语义正确：进程刚启动时本进程确实无活 run，该清的就是**别的实例留下的孤儿**。16:52 事故的根因是**第二实例**清了**第一实例的活 run**——单进程内无法知道别的实例还活着，故本修复防的是"本进程重启后误标自己即将接管的行"场景 + 语义显式化。**真正的第二实例防护**：`WHERE stopped_at IS NULL AND started_at > now() - interval '1 hour'` 之外的行不动？——**不做**（过度设计）；改为 registry 注册 run 时若 DB 行已 stopped 则 UPDATE 回 running（`strategy_run_repo` 加 `MarkRunning(id)`，Register 路径调用）——活 goroutine 是权威，DB 行跟随。
3. 对抗测试：① 插 3 行 running（2 个在 excludeIDs）→ Cleanup 后 exclude 的 2 行仍 running、第 3 行 stopped（GREEN）；删 SQL 排除子句 → 全 stopped（RED）。② Register 时 DB 行 stopped → MarkRunning 后回 running（GREEN；删调用 RED）。
4. **调查项（只记录不阻塞）**：16:52:21 第二实例来源无法完全重建（无 exited 容器/宿主 ps 已逝）——最可能是施工方宿主违规跑二进制（违部署铁律）。修复批内**不追责不深挖**，规则已在 CLAUDE.md。

---

## 2. 红队自审（交付前逐项打勾）

- [ ] Task 1：`uuid` import 正确；close 失败路径（postCloseFailure :135）也用 omsOrderID；**不改变 idem 行为**（closeOrderID 字符串 key 原样）
- [ ] Task 2：重试 goroutine 用 detached ctx + 超时（不挂请求 ctx——handler 返回即取消）；`time.AfterFunc` 泄漏面（一次性，无 ticker）；trigger nil 守卫；并发同 ticket 多事件同时 miss → 多次重试无害（transition 幂等按 state）
- [ ] Task 3：**place 路径回归**——同策略连发两单（真实防重场景，如 bar 重投）仍被拒（测试③别删）；`%d` 打印 int64 Magic
- [ ] Task 4：`ALL($1::uuid[])` 空数组语义（pgx 传 nil slice → NULL → `!= ALL(NULL)` 恒 NULL → 0 行更新！必须空时走无子句分支或传 `'{}'`）；MarkRunning 不覆盖 stopped_at 已有值的语义（设 NULL）
- [ ] 门禁 + 对抗自检（verify-adversarial.sh）+ 不碰 connect/strategy 文件（PARITY 返工属地）

## 3. 门禁

```bash
cd backend && go build ./... && go test ./internal/mthub/... ./internal/risk/... ./internal/repository/... ./internal/connect/strategy/...
go run ./tools/check-file-lines --strict    # 0 新增 ERROR
bash scripts/verify-adversarial.sh <test> <pkg> <file> <sed-mutation>   # 每任务自检
```
部署：`docker compose build backend && docker compose up -d backend`（唯一合法方式）。**本批部署可与 PARITY 返工合并一次 build**（若彼批未部署）。

## 4. 验收（审计方）

独立删行复测各任务对抗测试 + 部署后实测：① UI 手动平仓 → orders 表出现合法 UUID 的 FILLED 平仓行 + 日志无 22P02；② 新单成交 → 1-2s 内离开 SUBMITTED（不再卡死）；③ 连续平两个不同 ticket（间隔 <5s）→ 第二个成功（无 duplicate rejected）；④ 重启后端 → 活 run 行不再被标 "server restarted"。

## 5. 回填纪律

registry 对应 4 条目 🟦open → ✅done + 真实根因/对抗结果；handover changelog 追加一行；append-only；不自宣告 ✅，等审计方独立复测 + 生产实测。

## 6. 复用核对

| 项 | 结论 |
|---|---|
| 平仓 UUID | **REUSE: `IdempotencyKey` MD5-UUID 模式** @ `oms_writer.go:205` |
| 竞态修复对账 | **REUSE: `ReconciliationLoop.TriggerReconcile`** @ `reconciliation.go:62`（STREAM-KEEPALIVE 已升级为修复型） |
| 活 run 枚举 | **REUSE: `SessionRegistry.ListAll()`** @ `session_registry.go:280` |
| 新能力 | **NEW**：`SetReconcileTrigger` 注入 + `CleanupStaleRuns(excludeIDs)` + `MarkRunning`——已搜无现成（`grep ReconcileTrigger/MarkRunning` 零命中） |
