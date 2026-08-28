# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: VM-AUDIT-2026-08-27 全 3 批 ✅done（-1~-8）+ round 4-5 全 5 batch ✅done。**P1 业务管线**：2 个 live 执行 bug 修复完成（login lookup 类型不匹配 + proto3 nil/empty slice 误拒）+ 1 个架构缺陷修复完成（FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP，Session interface 改传结构体指针消除进程内 proto round-trip）。**FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1+S2+S3 ✅done**（修复 B+A+C，Devin CLI 验收通过 2026-08-27）。**FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP ✅done**（schedule_health_repo 2 处改查 trade_records，Devin CLI 验收通过 2026-08-28）。spec 见 `docs/spec/fix-2026-08-27-schedule-health-order-history-gap.md`。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| D-006 角色移交 Claude→Devin CLI | ✅ | AGENTS.md §0 |
| D-007 业主全权授权常规操作 | ✅ | AGENTS.md §6 |
| D-REVERT-CLEANUP-001 build 断裂修复 | ✅ | registry D-REVERT-CLEANUP-001 |
| D-REVERT-SCOPE-DRIFT-001 状态漂移对账 | ✅ | registry D-REVERT-SCOPE-DRIFT-001 |
| VM-CACHE-INTEGRITY-1/2（第一批） | ✅done | 返工后 Devin CLI 验收通过 2026-08-26 |
| LIVE-ORDER-REENTRY-1 R4 复审阻断 | ✅done | 返工后 Devin CLI 验收通过 2026-08-26 |
| VM-TRADE-CONTEXT-1/2（第二批） | ✅done | Devin CLI 验收通过 2026-08-26 |
| VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD（第三批） | ✅done | Devin CLI 验收通过 2026-08-26 |
| VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1（第四批） | ✅done | Devin CLI 验收通过 2026-08-26，8 项对抗证明 |
| DATA-TRUTH-2b MT4 margin 补齐 | ✅ | spec 验证通过，修复+对抗证明存活 |
| VM-AUDIT-2026-08-27 批次 1（-1 Python live SourceHash + -2 fatalError 重置） | ✅done | Devin CLI 验收通过 2026-08-27，2 项对抗证明独立验证 |
| VM-AUDIT-2026-08-27 批次 2（-3 stack depth + -4 popN + -5 dispatch default） | ✅done | Devin CLI 验收通过 2026-08-27，3 项对抗证明独立验证 |
| VM-AUDIT-2026-08-27 批次 3（-6 compileForLive + -7 recovery ctx + -8 PositionCache panic） | ✅done | Devin CLI 验收通过 2026-08-27，3 项对抗证明独立验证 |
| VM round 4-5 遗留 5 ID 复审（VM-TRADE-CONTEXT-6/API-TRUTH-3/CACHE-INTEGRITY-5/COMPILER-SEMANTICS-4/TEST-EVIDENCE-4） | ✅done | Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27 |
| P1 管线审计（13 条目） | 🟦open | 3 still-open（TRON-SECURITY-1/DATA-TRUTH-1/TRUST-1）+ QUOTE-RECONNECT-LOOP + BROKER-SEARCH-1 ✅done |
| VM round 4-5 + 报价管线派工（5 batch） | ✅done | Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27 |
| P1 live 执行 bug 修复（login lookup + nil/empty slice） | ✅done | 已部署验证 2026-08-27 |
| FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP | 🟦open | 施工完成 2026-08-27，待 Devin CLI 独立复审（S10 对抗证明 RED→restore→GREEN 已执行） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1（修复 B） | ✅done | Devin CLI 验收通过 2026-08-27（4 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S2（修复 A） | ✅done | Devin CLI 验收通过 2026-08-27（6 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S3（修复 C） | ✅done | Devin CLI 验收通过 2026-08-27（5 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP S1 | ✅done | Devin CLI 验收通过 2026-08-28（4 项对抗证明独立重跑 RED→restore→GREEN） |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效。DATA-TRUTH-1 需架构决策。TRUST-1 需业务决策。TRON-SECURITY-1 业主暂缓。
- **下一步**: P1 业务管线核心 bug 已修复。剩余 P1 管线 3 still-open（TRON-SECURITY-1/DATA-TRUTH-1/TRUST-1）待决策/施工。
- **清扫上翻**: 无私有记忆需清扫。

## 活跃 registry 条目指针

> 完整明细见 `docs/audits/tech-debt-registry.md`。这里只列当前活跃（🟦open / ⚠️待独立复审）条目。

- **D-REVERT-CLEANUP-001** ✅done — revert 遗留拆分文件 build 断裂修复（2026-08-26）
- **D-REVERT-SCOPE-DRIFT-001** ✅done — revert 实际范围远超 commit message，8 个 VM ID 状态漂移全部返工完成（2026-08-26）
- **VM-CACHE-INTEGRITY-1/2** ✅done — SourceHash 绑定（返工后 Devin CLI 验收通过 2026-08-26）
- **VM-TRADE-CONTEXT-1/2** ✅done — 交易上下文失真（Devin CLI 验收通过 2026-08-26）
- **VM-COMPILER-SEMANTICS-1** ✅done — MQL→IR/Bytecode 语义丢失（Devin CLI 验收通过 2026-08-26）
- **BT-FUNC-ENTRYPC-FWD** ✅done — 前向引用 stale marker PC（Devin CLI 验收通过 2026-08-26）
- **VM-TIMESERIES-SEMANTICS-1** ✅done — timeseries 语义（Devin CLI 验收通过 2026-08-26）
- **VM-RUNTIME-FAILCLOSED-1** ✅done — fail-closed 错误传播（Devin CLI 验收通过 2026-08-26）
- **LIVE-ORDER-REENTRY-1** ✅done（R4-REVIEW） — P0 实盘重复开仓（R4 复审阻断返工后 Devin CLI 验收通过 2026-08-26）
- **DATA-TRUTH-2b** ✅done — MT4 margin 从 AccountSummary 补齐（修复+对抗证明 revert 后存活，2026-08-26 验收）
- **VM 返工批 round 4-5** ✅done — Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27
- **VM-COMPILER-SEMANTICS-4** ✅done — 从零重做 round 6（2026-08-27 Devin CLI 验收通过）：comma_expression ExprSeq + checkReservedKeywordUsage before switch + hasMissingInitializer
- **VM-CACHE-INTEGRITY-5** ✅done — 从零重做 round 6（2026-08-27 Devin CLI 验收通过）：coverage restore + Version check + payload limit + no Language field
- **TRON-SECURITY-1** 🟦open — 提现冷签 MITM，`tron_client.go:34` 仍 `insecure.NewCredentials()`（P0 资金）
- **DATA-TRUTH-1** 🟦open — orders 表 reconciliation 只检测不收敛，ghost 仅 log.Warn（P0 数据，需架构决策）
- **QUOTE-RECONNECT-LOOP** ✅done — 报价流自持重连循环修复（2026-08-27 Devin CLI 验收通过）
- **BROKER-SEARCH-1** ✅done — mtapi host 配置接线（2026-08-27 Devin CLI 验收通过）
- **TRUST-1** 🟦open — Demo/真实账户战绩混展无标注（P2 业务，需业务决策）
- **SCHEDULE-HOTLOOP-1** ⚠️待生产部署验收
- **VM-AUDIT-2026-08-27-1** ✅done — Python live 路径 SourceHash 验证（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-2** ✅done — runEvent fatalError 重置（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-3** ✅done — executeCallUser MaxStackDepth 检查（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-4** ✅done — popN 栈下溢后 callBuiltin early return（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-5** ✅done — dispatch default 未知请求类型 error（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-6** ✅done — compileForLive helper 统一 4 live 路径缓存逻辑（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-7** ✅done — recoverFromOutcomeUnknown select+ctx 可取消（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-8** ✅done — PositionCache.Subscribe panic recovery（Devin CLI 验收通过 2026-08-27）
- **FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP** ✅done — Session interface 改传结构体指针消除进程内 proto round-trip（Devin CLI 验收通过 2026-08-27，S10 对抗证明 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S1** ✅done — `writeClosedTradeRecord` 补齐 Magic + ScheduleID（Devin CLI 验收通过 2026-08-27，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S2** ✅done — `GetOrderHistory` 改查 `trade_records` + proto 加 `magic_number` + 前端加 Magic 列（Devin CLI 验收通过 2026-08-27，6 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S3** ✅done — 删除 5 个死代码方法（WriteClosedTrade/ClosedTradeParams/LogOrder/UpdateOrderHistoryClose×2/CreateOrderHistory）（Devin CLI 验收通过 2026-08-27，5 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP** ✅done — schedule_health_repo.go:136,172 2 处 `FROM order_history`→`FROM trade_records`（Devin CLI 验收通过 2026-08-28，4 项对抗证明独立重跑 RED→restore→GREEN）

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-08-27 **FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP ✅done**：Devin CLI 验收通过。Session interface 从 `[]byte`（proto-marshaled）改为 `*antv1.ExecuteLiveRequest`/`*antv1.ExecuteLiveResponse` 指针——消除进程内 proto marshal/unmarshal round-trip（proto3 把空 repeated slice 折叠为 nil，导致"无持仓"与"数据缺失"不可区分）。S1-S7 代码坐标全匹配，S8 proto import 清理 3 文件，S9 测试适配 5 文件（3 文档列出 + 2 文档遗漏但施工方正确发现），S10 对抗证明 `TestVMLiveSession_NilPositionsSurviveRoundTrip` RED→restore→GREEN 独立验证。11 文件 +75/-185（净 -110 行）。门禁全绿（build/vet/test 98.3s/race×3 294.8s/check-lines 0 errors）。
- 2026-08-27 **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1 施工完成（🟦open）**：修复 B——`writeClosedTradeRecord` 构造 `model.TradeRecord` 时遗漏 `MagicNumber` + `ScheduleID`（`SyncOrderHistory` 路径已正确设置，实时路径遗漏对称接线）→ `trade_records.magic_number=0` / `schedule_id=NULL` → 前端 Magic 列显示 `-`。修复：`mdGatewayPipelineDeps` 加 `scheduleResolver` + `main.go` 注入 + `buildOnOrderUpdate` 加 `resolver` 参数 + 提取 `buildClosedTradeRecord` 纯函数 + `rec` 补齐 `MagicNumber: int(o.UpdateMagic)` + `ScheduleID: mthub.ResolveScheduleID(...)`。对抗证明 4 测试 RED→restore→GREEN。门禁全绿。停手等 Devin CLI 复审。勿部署。
- 2026-08-27 **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S2 施工完成（🟦open）**：修复 A——`GetOrderHistory` 查死表 `order_history`（0 行），实际数据在 `trade_records`（331 行）→ 前端"订单日志"tab 永远空。`OrderHistoryRecord` proto 无 `magic_number` → 前端无法渲染 Magic 列。修复：① proto 加 `int64 magic_number = 13;`；② `GetOrderHistory` 改查 `trade_records`（显式 SELECT 14 字段，scan 适配 schema 差异：close_time NOT NULL → `*time.Time`，schedule_id nullable → `uuid.NullUUID`）；③ `orderHistoryToProto` 补 `MagicNumber`；④ 前端加 Magic 列 + i18n `orders_table_magic` key（5 语言）。i18n-build 副作用（base.ts 丢 53 行手动 diag keys）已 `git checkout` 恢复。对抗证明 6 测试 RED→restore→GREEN。门禁全绿（build/vet/test/race×3/check-file-lines 0 errors/tsc --noEmit/diff --check clean）。停手等 Devin CLI 复审。勿部署。
- 2026-08-27 **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S3 施工完成（🟦open）**：修复 C——S1+S2 修复后 `order_history` 表写入路径完全无调用方，但 5 个死代码方法仍残留。修复：删除 `WriteClosedTrade`+`ClosedTradeParams`/`LogService.LogOrder`/`LogService.UpdateOrderHistoryClose`/`LogRepository.CreateOrderHistory`/`LogRepository.UpdateOrderHistoryClose`，清理 unused imports。对抗证明 5 测试 RED→restore→GREEN。门禁全绿。风险/gap：`schedule_health_repo.go:136,172` 仍直接查 `order_history` 表——超出 S3 scope，需后续单独修复。停手等 Devin CLI 复审。勿部署。
- 2026-08-28 **FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP S1 施工完成（🟦open）**：`schedule_health_repo.go:136`（`GetLatestOrderProfit`）+ `:172`（`ListOrders`）2 处 `FROM order_history`→`FROM trade_records`，scan 不变。对抗证明 4 测试 RED→restore→GREEN（T1 全局守卫 + T2/T3 精确断言 + T4 回归守卫 schedule_run_logs）。门禁全绿。停手等 Devin CLI 复审。勿部署。
- 2026-08-27 **Batch 5（VM-TEST-EVIDENCE-4）施工完成（🟦open，待独立复审）**：从零重写 `docs/audits/vm-adversarial-proofs.md`（旧版标记 SUPERSEDED）。15 项对抗证明，每条含 mutation target（精确 file:line + 改什么）、预期 RED（测试名 + 断言失败消息）、restore 指令、测试文件位置。Proof 1-6 Batch 1（VM-COMPILER-SEMANTICS-4 + VM-CACHE-INTEGRITY-5）、Proof 7-12 Batch 2（VM-TRADE-CONTEXT-6）、Proof 13-15 Batch 3（VM-API-TRUTH-3）。所有引用测试文件和函数 `grep` 验证存在。文档 165 行（T1 预算 450 行内）。纯文档任务，无代码改动。
- 2026-08-27 **Batch 3（VM-API-TRUTH-3）施工完成（🟦open，待独立复审）**：从零重做 round 6。S1-S3 `vm_builtin_checkup.go` 的 `builtinIsConnected`/`builtinIsDemo`/`builtinIsTradeAllowed` 改为从 `vm.ctx.Account()` 读取（不再硬编码 true），`vm.ctx == nil` 时保留 true（backtest 默认）；S4 `sdk.AccountInfo` 新增 `IsDemo`/`IsConnected`/`IsTradeAllowed` 字段 + `Runner.SetAccountStatus` + `context.go` 加 3 个字段 + `brokerImpl.Account()` 返回 + `SimBroker.Account()` 默认全 true；S5 `vmHandleBar`/`Start()`/`dispatchVMLive` 在 Init 前调用 `SetAccountStatus`。12 个行为测试（`vm_api_truth3_batch3_test.go`）：T1-T3 builtin readback false/true 双向、T4 nil ctx defaults true、T5-T7 e2e VM readback。golangci-lint 0 issues。门禁全绿（build/vet/mql2go test 7.9s/race×3 1.2s/check-lines 0 errors/connect/strategy 96.3s/diff-check clean）。
- 2026-08-27 **Batch 2（VM-TRADE-CONTEXT-6）施工完成（🟦open，待独立复审）**：从零重做 round 6。S1 `parseDecimalStrict`/`parseInt64Strict`（`backtest_worker_helpers.go`，返回 error 不转零）；S2 `validateOHLCVLengths` 在 `vmHandleBar` 校验 OHLCV 数组长度（含多 symbol）；S3 所有 live handler strict parse（bar/tick/trade 的 OHLCV/financial/trade 字段）；S4 nil repeated message 拒绝（live mode positions/pending_orders nil = data missing）；S5 `validateFirstBarContext` 在 `VMLiveSession.Start()` 和 `dispatchVMLive` 的 `Init()` 前执行；S6 `Runner.SetLogin` + `brokerImpl.Account()` 返回 `liveLogin` + `injectAccountTruth` 注入 Login/Company/IsDemo/IsConnected/IsTradeAllowed（investor 账户 IsTradeAllowed=false）；S7 `cmd/server/handlers_strategy.go` 接入 5 个 mt_accounts lookup。13 个行为测试（`vm_trade_context6_batch2_test.go`）。golangci-lint 0 issues。门禁全绿（build/vet/test/race×3/check-lines 0 errors）。
- 2026-08-27 **fix(ci): proto codegen drift 修复**：commit `830b2c79`（revert D-CODE-HYGIENE-001）回滚了生成的 proto 文件但未回滚 `.proto` 源文件，导致 CI proto-drift job 失败。`make proto` 重新生成 3 个文件（`strategy_runtime.pb.go`/`strategy_signal_messages.pb.go`/`strategy_runtime_pb.ts`），补齐 `.proto` 源中已有但 gen 文件缺失的字段（StrategySignal: Magic/Deviation/OppositeTicket；ExecuteLiveRequest: AccountId；LiveStrategyContext: Login/Company/IsDemo/IsConnected/IsTradeAllowed）。build + tsc --noEmit 通过。
- 2026-08-27 **fix(lint): golangci-lint CI failures**：Batch 1 改动引入 5 个 lint 错误。goconst（3）：`"type_identifier"` 字面量跨 4 文件 8 次出现达阈值→提取为 `nodeTypeIdentifier` 常量（`constants.go`）。gocyclo（1）：`compileExpr` 圈复杂度 31>30→提取 `comma_expression` case 为 `compileCommaExpression` helper。funlen（1）：`CompileAST` 125 行>120→提取 OnTrade/OnTimer/OnDeinit/OnTradeTransaction/OnBookEvent 编译为 `compileOptionalEvents` helper。golangci-lint 0 issues，build/vet/test 全绿。
- 2026-08-27 **Batch 1 + Batch 4 施工完成（🟦open，待独立复审）**：Batch 1（VM-COMPILER-SEMANTICS-4 + VM-CACHE-INTEGRITY-5）从零重做：comma_expression ExprSeq + checkReservedKeywordUsage before switch + hasMissingInitializer（精确区分 missing initializer vs missing `;`）+ coverage restore + Version check + payload limit。15 项测试。修复回归（hasMissingNode 对 Python source 过于激进→TestCompileForLive_PythonBranch RED）+ data race（coverageRestoreHook 并发读写→移除 t.Parallel）。Batch 4（QUOTE-RECONNECT-LOOP + BROKER-SEARCH-1）施工完成：ensureConnected 返回 nil + Disconnect ctx-cancellable + recvLoop/profitLoop/orderLoop 不退出 + NewFromConfig + env var wiring。8 项测试 + 3 项对抗证明。门禁全绿。
- 2026-08-27 **派工 5 batch**：VM round 4-5 遗留 5 ID + 报价管线 2 ID 从零重做施工提示词落档。Batch 1（VM-COMPILER-SEMANTICS-4 + VM-CACHE-INTEGRITY-5，编译器+缓存）+ Batch 4（QUOTE-RECONNECT-LOOP + BROKER-SEARCH-1，报价管线）可并行；Batch 2（VM-TRADE-CONTEXT-6，live context）依赖 B1；Batch 3（VM-API-TRUTH-3，builtin truth）依赖 B2；Batch 5（VM-TEST-EVIDENCE-4，对抗证明文档）依赖 B1-3。业主指示：TRON-SECURITY-1 暂缓（VM 管线跑通优先），DATA-TRUTH-1/TRUST-1 待决策。施工提示词：`docs/audits/builder-handoff-vm-round45-batch{1-5}-2026-08-27.md`。
- 2026-08-27 **第二轮审计完成**：Devin CLI 独立审计 VM round 4-5 遗留 5 ID + P1 资金/数据/报价管线 13 条目。VM round 4-5：5 ID 全部 FAIL（代码被 D-REVERT-SCOPE-DRIFT-001 回滚删除）。P1 管线：5 still-open + 8 fixed-acceptable。spec 落档 `docs/spec/audit-2026-08-27-vm-round45-p1-pipeline-spec.md`。
- 2026-08-27 VM-AUDIT-2026-08-27 全 3 批 ✅done：Devin CLI 验收通过。8 个 ID（-1~-8）全部闭环。对抗证明 8 项独立验证 RED→restore→GREEN。门禁全绿。
- 2026-08-27 VM-AUDIT-2026-08-27 全面审计完成：Devin CLI 独立审计 VM 管线 10 个组件 ~5500 行，发现 5 BUG + 3 架构问题。8 个 registry 条目落档（-1~-8），spec 落档 `docs/spec/vm-audit-2026-08-27-spec.md`。

> 2026-08-26 及更早的变更日志（VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
