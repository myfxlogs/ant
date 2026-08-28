# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: VM-AUDIT-2026-08-27 全 3 批 ✅done（-1~-8）+ round 4-5 全 5 batch ✅done。**P1 业务管线**：2 个 live 执行 bug 修复完成（login lookup 类型不匹配 + proto3 nil/empty slice 误拒）+ 1 个架构缺陷修复完成（FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP ✅done，Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1+S2+S3 ✅done**（Devin CLI 验收通过 2026-08-27）。**FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP ✅done**（Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE S1-S4 ✅done**（Devin CLI 验收通过 2026-08-28：reconciliation 收敛 24h 下界 + ghost 自动补写 + orphan 全非终态修复）。
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
| FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP | ✅done | Devin CLI 验收通过 2026-08-28（S10 对抗证明两步 mutation 独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1（修复 B） | ✅done | Devin CLI 验收通过 2026-08-27（4 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S2（修复 A） | ✅done | Devin CLI 验收通过 2026-08-27（6 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S3（修复 C） | ✅done | Devin CLI 验收通过 2026-08-27（5 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP S1 | ✅done | Devin CLI 验收通过 2026-08-28（4 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE | ✅done | Devin CLI 验收通过 2026-08-28（4 项对抗证明独立重跑 RED→restore→GREEN + 机检五件套全绿） |
| FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION | 🔄施工中 | 施工提示词已发 `docs/audits/builder-handoff-fix-2026-08-28-trust-1-demo-real-account-distinction.md` |
| FIX-2026-08-28-MAGIC-ENRICHMENT（magic 列 `-` 三条断裂） | ✅done | 断裂 1: buildClosedTradeRecord 从 orders 表回查 magic + 断裂 2: proto OrderUpdateEvent 加 magic_number + 前端映射 + 断裂 3: DB 回填 252 条 trades。对抗证明 RED→restore→GREEN（断裂 1+2 各 2 测试）。门禁全过。审计补加断裂 2 对抗测试。已部署 2026-08-28（container healthy）。 |
| FIX-2026-08-28-ORDER-LOG-COLUMNS-TYPE-MISMATCH | ✅done | Devin CLI 直接施工+验收 2026-08-28。scheduleLogColumns.tsx 4 列 render `typeof v === 'number'`→`v ? String(v) : '-'`（proto string/bigint vs number 类型不匹配）。tsc+build 全绿。已部署。 |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效。TRON-SECURITY-1 业主暂缓（不做）。DATA-TRUTH-1 + TRUST-1 施工提示词已发，待施工方落地。
- **下一步**: 施工方按两份施工提示词落地（DATA-TRUTH-1 S1-S4 + TRUST-1 S1-S8），完成后 Devin CLI 独立复审。
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
- **FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP** ✅done — Session interface 改传结构体指针消除进程内 proto round-trip（Devin CLI 验收通过 2026-08-28，S10 对抗证明两步 mutation 独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S1** ✅done — `writeClosedTradeRecord` 补齐 Magic + ScheduleID（Devin CLI 验收通过 2026-08-27，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S2** ✅done — `GetOrderHistory` 改查 `trade_records` + proto 加 `magic_number` + 前端加 Magic 列（Devin CLI 验收通过 2026-08-27，6 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S3** ✅done — 删除 5 个死代码方法（WriteClosedTrade/ClosedTradeParams/LogOrder/UpdateOrderHistoryClose×2/CreateOrderHistory）（Devin CLI 验收通过 2026-08-27，5 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP** ✅done — schedule_health_repo.go:136,172 2 处 `FROM order_history`→`FROM trade_records`（Devin CLI 验收通过 2026-08-28，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE** ✅done — reconciliation 收敛：S1 ant 查询加 24h 下界 + S2 ghost 自动补写 ImportBrokerOrder + S3 新增 ImportBrokerOrder 方法（ON CONFLICT (mt_account_id, ticket) + trade_records hash chain）+ S4 orphan 修复扩展到所有非终态（isNonTerminalOMSState）（Devin CLI 验收通过 2026-08-28，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-28-ORDER-LOG-COLUMNS-TYPE-MISMATCH** ✅done — scheduleLogColumns.tsx 4 列 render 类型守卫修复（Devin CLI 直接施工+验收 2026-08-28）

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-08-27 **FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP ✅done**：Devin CLI 验收通过。Session interface 从 `[]byte`（proto-marshaled）改为 `*antv1.ExecuteLiveRequest`/`*antv1.ExecuteLiveResponse` 指针——消除进程内 proto marshal/unmarshal round-trip。11 文件 +75/-185。门禁全绿。
- 2026-08-27 **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1+S2+S3 ✅done**：Devin CLI 验收通过。S1 修复 B（`writeClosedTradeRecord` 补齐 Magic + ScheduleID）+ S2 修复 A（`GetOrderHistory` 改查 `trade_records` + proto 加 `magic_number` + 前端加 Magic 列）+ S3 修复 C（删除 5 个死代码方法）。15 项对抗证明独立重跑 RED→restore→GREEN。门禁全绿。
- 2026-08-28 **FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP ✅done**：Devin CLI 验收通过。`schedule_health_repo.go:136,172` 2 处 `FROM order_history`→`FROM trade_records`。4 项对抗证明独立重跑 RED→restore→GREEN。门禁全绿。
- 2026-08-28 **FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE S1-S4 ✅done**（Devin CLI 验收通过 2026-08-28）：reconciliation 只检测不收敛（3 层根因：A ghost 仅 log.Warn / B orphan 仅修 SUBMITTED / C ant 全量 vs broker 24h 不对称 → 129 条假 orphan/账户/轮）。修复：S1 ant 查询加 24h 下界；S2 ghost 自动补写 `ImportBrokerOrder`；S3 新增 `MtHubService.ImportBrokerOrder`（`ON CONFLICT (mt_account_id, ticket) DO NOTHING` + 已平仓写入 `trade_records` 含 hash chain）+ `OmsWriter.Pool()` getter + `tradeRecordRepo` 字段 + `handlers_pipeline.go` 装配；S4 orphan 修复扩展到所有非终态（`isNonTerminalOMSState`）。对抗证明 5 测试 RED→restore→GREEN。门禁全绿。文件拆分 `service_orders_import.go`（保持 `service_orders.go` <300 行）。风险/gap：部署后需实测 warn 数 <5；S5 一次性回填脚本待编写。Devin CLI 验收通过 2026-08-28（A-F 全绿 + 4 项对抗证明独立重跑 + 机检五件套全绿）。
- 2026-08-28 **FIX-2026-08-28-ORDER-LOG-COLUMNS-TYPE-MISMATCH ✅done**（Devin CLI 直接施工+验收 2026-08-28）：策略调度日志页 Order Logs tab 4 列（手数/开仓价/平仓价/订单号）全部显示 `-`。根因：`scheduleLogColumns.tsx` `buildOrderColumns` 4 列 render 用 `typeof v === 'number'` 守卫，但 proto TS 类型 `lots/openPrice/closePrice: string` + `ticket: bigint` → ConnectRPC JSON 传 string → 守卫永远 false。修复：4 列改为 `v ? String(v) : '-'`。tsc+build 全绿。已部署。另：调查策略运行 898035e2 无信号——GetActiveStrategy RPC 诊断确认 evalCount=4060/tickCount=4054/barCount=6，策略正常 hold（MACD 无交叉），非系统 bug。
- 2026-08-27 **VM round 4-5 + 报价管线 5 batch ✅done**：Devin CLI 验收通过。Batch 1-5 全部闭环（VM-COMPILER-SEMANTICS-4 / VM-CACHE-INTEGRITY-5 / VM-TRADE-CONTEXT-6 / VM-API-TRUTH-3 / QUOTE-RECONNECT-LOOP / BROKER-SEARCH-1 / VM-TEST-EVIDENCE-4）。详见 `docs/audits/handover-audit-plan.md`。
- 2026-08-27 VM-AUDIT-2026-08-27 全 3 批 ✅done：Devin CLI 验收通过。8 个 ID（-1~-8）全部闭环。

> 2026-08-26 及更早的变更日志（VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
