# 施工交接：Tier 0 Launch-Blocking 批次（5 项，实盘核心价值断裂）

> **🔍 审计方自我审计状态（2026-08-13，每条修复方向的关键假设已独立读码验证）**：
> - EXEC-1：诊断 + 修复方向**二次纠错**（原 UserID 误诊→哈希链；xmax 方案→DO NOTHING，见下）。**以本文 EXEC-1 段为准**。
> - EXEC-2：修复方向**成立**（`OrderIDByTicket` 反查存在 oms_writer.go:149；UpdateType 语义 mdtick.go:97）。
> - EXEC-3：修复方向**成立**（BrokerTradeEvent 结构存在 trade_broker.go:24）。
> - SUPPLY-1：修复方向**成立**（`CompilePython` 返回 `*VMRunner` 同 CompileMQL，interp_runner.go:79/127；路径已订正为 connect/strategy/）。
> - LIVE-PRICE-4：修复方向**成立**（FetchAllSymbols orders.go:258 可用）。
> - **教训**：审计 agent 诊断/方向会出错，凡派给施工方的关键修复，审计方须独立读码验证假设，不照单转发。

> **审计方定论 2026-08-13，🟦待施工。** 来源 `docs/audits/launch-readiness-2026-08-13.md`。5 项都是 launch 硬阻断，工程量不大（EXEC-1 一行，其余中等），集中在 live 执行 + AI 部署。
>
> **铁律**：每项必带删行 RED 对抗证明 + 回填 `docs/audits/tech-debt-registry.md` 状态（🟦→✅，标日期+commit+对抗结果+部署后实测），不删条目不改审计方事实，**不自行宣告完成**（审计方验收）。部署唯一方式 `docker compose build backend && docker compose up -d backend`。

---

## EXEC-1：⚠️诊断已修正（审计方独立复核 2026-08-13）—— trade_records 哈希链冲突写入失败（非 UserID）

> **原 agent 诊断（UserID=nil 违反 FK）是错的**——审计方读码 + 实测日志纠正：UserID 一直设对了（`trade_record_repository.go:208` 列含 user_id、`:230` 传 record.UserID）。真实错误是**哈希链 entry_hash 在 upsert 冲突时更新被触发器挡**。

- **真实根因**：`insertWithHashChain`（`backend/internal/repository/trade_record_repository.go:206-251`）用 `ON CONFLICT (account_id,ticket,close_time) DO UPDATE SET ...`（:214）做 upsert，然后**无条件** `UPDATE trade_records SET entry_hash = $1 WHERE id = $2`（:248）。哈希链触发器（migration 263/209）只允许 entry_hash **NULL→非NULL**。当同一 `(account_id,ticket,close_time)` 已存在（重连/重同步/realtime+sync 双写都常见）→ 走 ON CONFLICT 分支 → 该记录 entry_hash 已非 NULL → :248 的 UPDATE 被触发器 RAISE P0001 "immutable" 挡掉。
- **证据**：日志 `OnOrderUpdate: write closed trade failed ... first: "trade hash chain: set entry_hash: ERROR: trade_records hash chain fields are immutable (seq/prev_hash/entry_hash)"` + `SyncAccountHistory: batch create failed ... 同错误`——**两条路径（realtime 平仓 + 历史同步）同一个错**。
- **影响**：trade_records 写入在冲突时失败（同 ticket 第二次写就挂）→ trade 笔数/胜率在 live performance 里不准/延迟。equity/PnL/回撤不受影响（来自 OnAccountProfit push）。UserID 全程正确。
- **修法（2026-08-13 二次纠错——xmax 方案有哈希脱敏缺陷，改 DO NOTHING）**：`ON CONFLICT (account_id,ticket,close_time) DO UPDATE SET ...`（:214）改成 **`ON CONFLICT ... DO NOTHING`**（幂等：同 `(account,ticket,close_time)` 重报 = 第一条为准，不更新任何字段——符合 append-only 设计）。然后 :229 的 `QueryRow(...).Scan(&returnedID,&seq)` 改成检测是否真插入了新行（`ON CONFLICT DO NOTHING` 冲突时 RETURNING 无行 → pgx.ErrNoRows）：只有新建行才执行 :248 的 `UPDATE entry_hash`（NULL→非NULL，触发器允许）；冲突行直接 return nil（原记录原哈希保留，无脱敏）。
  - ⚠️ **不要用 `xmax` 方案**：xmax 只跳过 entry_hash 更新，但 DO UPDATE 仍改 profit/close_price 等 → 哈希与数据脱敏 → VerifyChain 失败。DO NOTHING 才是正解（不更新就不脱敏）。
- **对抗证明**：① 第一次写某 ticket → INSERT 成功 + entry_hash 设上（GREEN）；② 第二次写同 ticket（冲突）→ 旧代码 "immutable" 失败（RED）；新代码 DO NOTHING 无行返回 → 跳过 entry_hash、return nil 成功（GREEN）；③ `VerifyChain` 该账户通过（哈希与数据一致）。

## EXEC-2：OMS orders 永远卡 SUBMITTED —— broker 成交事件不转状态

- **根因**：`buildOnOrderUpdate` 回调（`pipeline_callbacks.go:27-34`）只写 position snapshot + close trade_record，**从不调用 `omsTransition`/`omsWriter.Transition`**。broker fill/reject/cancel 事件不触发 OMS 状态转换 → 16 态中 9 个终态（FILLED/CANCELLED/REJECTED/EXPIRED…）在开仓路径不可达，`orders` 表状态不可信。
- **修法**：在 `buildOnOrderUpdate` 回调里解析 `o.UpdateType`（开仓/平仓/成交/拒绝），通过 `idem.GetTicket`（或现有 orderID↔ticket 映射）反查 orderID，调 `omsWriter.Transition(ctx, orderID, newState)`。
- **对抗证明**：mock broker fill 事件 → 旧代码 order 仍 SUBMITTED **RED**；新代码转到 FILLED **GREEN**。

## EXEC-3：OnTrade 执行模型生产死亡 —— PublishTradeEvent 零调用方

- **根因**：`mthub.MtHubService.PublishTradeEvent`（`backend/internal/mthub/service.go:221`）**全 backend 零调用方**（grep 实证）。subscribe 端连通（`live_runner.go:360` `subscribeTradeEvents`→`tradeBroker.Subscribe`），但 publish 端死 → `tradeCh` 永不收事件 → 策略 `OnTrade` 回调生产永不触发。依赖 OnTrade 的策略（移动止损、martingale）静默失效。
- **修法**：在 `buildOnOrderUpdate` 回调（或 `publishPositionSnapshot` 旁）检测 trade 事件（成交/开平仓），构建 `*BrokerTradeEvent` 调 `mtHub.PublishTradeEvent`。镜像 OnAccountProfit/OnBar 的 publish 模式。
- **对抗证明**：注入 mock TradeBroker + 喂成交事件 → 旧代码 tradeBroker 收不到 **RED**；新代码收到 BrokerTradeEvent **GREEN**。（注：需真实报价流，配合 LIVE-PRICE-4 后才能端到端验。）

## SUPPLY-1：AI/Python 策略能上架但买家无法部署实盘、无法经 worker 重测

- **根因**：运行时重入（live 部署 + 异步回测）**只有 MQL 分支**，Python 缺 dispatch（审计方已复核路径订正）。关键文件：`backend/internal/connect/strategy/vm_live_session.go:33,43`（`CompileMQL`/`CompileMQLCached` 无 Python 分支）、`backend/internal/connect/strategy/backtest_worker.go:186`（`isMQLStrategy` 分支，else 走 Python 但）、`backend/internal/connect/strategy/backtest_worker_vm.go:61`（注释引用的 `executePythonVMBacktest` **从未实现**）。编译器 `CompilePython`（`backend/strategy/sdk/language.go`）和 VMRunner 本身**语言无关**，缺的只是 dispatch 层 Python 分支 / 加载已缓存 bytecode。
- **影响**：打在"AI 持续迭代策略"核心差异化——AI 生成的 Python 策略"能生成能上架但不能给买家跑"。MQL 手写作者不受影响。
- **修法**（二选一）：① 4 个 dispatch 点加 Python 分支（调 `CompilePython`/加载 bytecode，走同一 VMRunner）；② 让 live/worker 路径**加载已持久化的 bytecode**（发布快照里有），绕过重编译。推荐 ①（与 AI 迭代 in-process 回测路径一致）。
- **对抗证明**：用 Python 策略部署 live + 经 worker 回测 → 旧代码"go strategy retired"/无法部署 **RED**；新代码 Python 策略真跑出 bar/信号/回测结果 **GREEN**。

## LIVE-PRICE-4：硬编码 symbol 原子订阅失败 → 实盘零报价

- **详见** `docs/audits/builder-handoff-live-price-2026-08-13.md` 任务 5（已有完整说明）。核心：删 `defaultQuoteSymbols()`，`postConnectSetup`（`mdgateway/runner_gateway.go:125`）订阅改用 `FetchAllSymbols`（broker 真实清单）过滤 + **逐 symbol `Subscribe`**（非原子，不存在的跳过），MT4+MT5。对抗证明：mock FetchAllSymbols 含 XAUUSDm 不含 XAUJPYm → 旧整批失败 RED / 新逐 symbol 订成功 GREEN。

---

## 红队自审（动手前自查 edge cases）
- [ ] EXEC-1：UserID 来源要正确（buildOnOrderUpdate 的 userID 是 cfg.UserID，不是 accountID）；hash chain 不可变字段（seq/prev_hash/entry_hash）不要碰，只补 user_id。
- [ ] EXEC-2：orderID↔ticket 反查要用现有幂等映射（idem.GetTicket），不要新建并行映射；UpdateType 语义对照 mt4/mt5 proto 确认（开仓/平仓/成交/拒绝/部分成交）。
- [ ] EXEC-3：BrokerTradeEvent 字段要从 OnOrderUpdate 的 OrderUpdate 正确映射（symbol/volume/price/type）；不要和 position snapshot 重复。
- [ ] SUPPLY-1：Python 策略的 bytecode 已在发布快照持久化吗？确认加载路径；live 部署 Python 策略的 Magic Number 归属（ARCH-4）要和 MQL 一致。
- [ ] LIVE-PRICE-4：逐 symbol Subscribe 的频率（37 个 RPC）可接受；reSubscribeSymbols 也要改逐 symbol；MT5 同改。
- [ ] 全部 fail-closed：错误默认拒绝/报错，绝不静默降级。

## 门禁（Before Commit）
```bash
go build ./...
go test ./internal/mthub/... ./internal/connect/strategy/... ./internal/backtest/... ./strategy/...
cd backend && go run ./tools/check-file-lines --strict
bash scripts/gen_capability_map.sh
```
**部署后审计方实测验收**（不只单测）：EXEC-1 平仓后 trade_records 真写入 / EXEC-2 下单后 order 状态真转 FILLED / EXEC-3 OnTrade 策略真触发 / SUPPLY-1 Python 策略真能部署+跑 / LIVE-PRICE-4 Active Runs 价格列刷新+`md_e2e_latency_count>0`。**LIVE-PRICE-4 是其余几项端到端验的前提**（没报价流，EXEC-2/3 的成交事件也来不了）——建议 LIVE-PRICE-4 先行或并行。

---

## 🆕 下一轮补强项（本批勿插，挂账待下轮提示词带上）

> 审计方独立删行复验发现（2026-08-13），本批 Windsurf 已在干 Tier 0 无法插入，挂此待下一轮提示词一并下发。

1. **LIVE-PRICE-3 对抗测试无效，需补强**：`TestSubscribeMany_ResponseErrorDetected`（`adapter/mt4/live_price_test.go:21`）只断言 `err==nil`，禁用 `resp.GetError()` 检查后**仍 GREEN**（审计方实测）。补强：用 **zaptest observer** 捕获日志，断言响应错误时**有 Error 级日志含 code**（删 GetError 检查 → 无 Error 日志 → RED）。MT5 侧同测同补。**生产修复本身是对的（运行中打 code257），只是测试守不住。**
2. **⚠️ 提交未进 VCS 的修复**：LIVE-PRICE-1/2/3 修复**未 commit**（HEAD 仍 `1b90f581`，修复在工作树+容器二进制）。下轮务必 `git commit`（否则工作树丢失即消失）。本批 Tier 0 修复交付时也**必须 commit**，不要只部署不提交。
3. **⚠️ 对抗测试质量警告（复发模式）**：跨多批次审计方屡次发现施工方对抗证明无效（测拷贝/测字面量/测错路径，如本次 LIVE-PRICE-3）。本批 EXEC-1/2/3 + SUPPLY-1 的对抗测试**审计方会逐个独立删行复验**，要求：测试必须**断言真实可观测行为**（DB 行写入 / 状态转换 / 日志 Error / 真实组件渲染 / 真函数调用），**不能只断言 `err==nil` 或字符串字面量**。删修复行 → 必须 RED，否则判未完成。
   - **🆕 对抗证明自验工具（施工方交付前必跑）**：`scripts/verify-adversarial.sh <test> <pkg> <file> <sed-mutation>` —— 对生产代码施加突变（删行/改条件），跑测试，FAIL=✅有效 / PASS=❌无效，自动还原（不依赖 git，对工作树安全）。**施工方交付对抗测试前用此脚本自验每一个"删行必红"，无效的自觉补强，别等审计方打回。** 用法示例见脚本头注释（已用 LIVE-PRICE-1 有效 / LIVE-PRICE-3 无效双向验证通过）。
