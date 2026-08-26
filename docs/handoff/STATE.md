# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: VM-CACHE-INTEGRITY-1/2 ✅done；LIVE-ORDER-REENTRY-1 R4 ✅done；VM-TRADE-CONTEXT-1/2 ✅done；VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done；VM 返工批第四批待派工。
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
| VM 返工批第四批（2 个漂移 ID） | 🟦open | spec + builder-handoff 待落档 |
| DATA-TRUTH-2b MT4 margin 补齐 | ✅ | spec 验证通过，修复+对抗证明存活 |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效（D-VM-LIVE-001 验收前禁止从 main 构建部署 backend）。
- **下一步**: 派工 Devin IDE 施工 VM 返工批第四批（VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1）。
- **清扫上翻**: 无私有记忆需清扫。

## 活跃 registry 条目指针

> 完整明细见 `docs/audits/tech-debt-registry.md`。这里只列当前活跃（🟦open / ⚠️待独立复审）条目。

- **D-REVERT-CLEANUP-001** ✅done — revert 遗留拆分文件 build 断裂修复（2026-08-26）
- **D-REVERT-SCOPE-DRIFT-001** 🟦open — revert 实际范围远超 commit message，8 个 VM ID 状态漂移需重新施工
- **VM-CACHE-INTEGRITY-1/2** ✅done — SourceHash 绑定（返工后 Devin CLI 验收通过 2026-08-26）
- **VM-TRADE-CONTEXT-1/2** ✅done — 交易上下文失真（Devin CLI 验收通过 2026-08-26）
- **VM-COMPILER-SEMANTICS-1** ✅done — MQL→IR/Bytecode 语义丢失（Devin CLI 验收通过 2026-08-26）
- **BT-FUNC-ENTRYPC-FWD** ✅done — 前向引用 stale marker PC（Devin CLI 验收通过 2026-08-26）
- **VM-TIMESERIES-SEMANTICS-1** 🟦open — timeseries 语义（被 revert，需重新施工）
- **VM-RUNTIME-FAILCLOSED-1** 🟦open — 增强修复被 revert（基本机制幸存）
- **LIVE-ORDER-REENTRY-1** ✅done（R4-REVIEW） — P0 实盘重复开仓（R4 复审阻断返工后 Devin CLI 验收通过 2026-08-26）
- **DATA-TRUTH-2b** ✅done — MT4 margin 从 AccountSummary 补齐（修复+对抗证明 revert 后存活，2026-08-26 验收）
- **VM 返工批 round 4-5** ⚠️待独立复审 — VM-CACHE-INTEGRITY-5/VM-TRADE-CONTEXT-6/VM-API-TRUTH-3 等
- **SCHEDULE-HOTLOOP-1** ⚠️待生产部署验收

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-08-26 VM-TRADE-CONTEXT-1/2 ✅done：Devin CLI 验收通过。invalidateOrderCaches + CTrade setter 透传 + OppositeTicket + AccountNumber 从 context + IsTesting=!signalMode + brokerImpl lastError fail-closed。对抗证明 8 项 RED→restore→GREEN，门禁全绿。
- 2026-08-26 LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done：返工后 Devin CLI 验收通过。S1a/S1b 的 2 处 time.Sleep→WaitState，S2 FullBrokerPath 防御性同步注释+对抗证明引用。对抗证明 2 项 RED→restore→GREEN，门禁全绿。
- 2026-08-26 VM-CACHE-INTEGRITY-1/2 ✅done：返工后 Devin CLI 验收通过。marshalHook 注入对抗证明 RED→restore→GREEN（MQL+Python 双路径），binary 引用删除，门禁全绿。
- 2026-08-26 DATA-TRUTH-2b ✅done：revert 后验证修复+对抗证明完整存活，无需重新施工。
- 2026-08-26 三个 spec 文档落档：`docs/spec/vm-revert-redo-spec.md`、`docs/spec/live-order-reentry-r4-spec.md`、`docs/spec/data-truth-2b-mt4-margin-spec.md`。
- 2026-08-26 D-REVERT-SCOPE-DRIFT-001：对账发现 revert `830b2c79` 实际范围远超 commit message，8 个 VM ID 状态漂移。降级回 `🟦open（待施工）`。
- 2026-08-26 D-REVERT-CLEANUP-001：修复 revert `830b2c79` 遗留 122 个拆分文件导致的 build 断裂（14 包 redeclaration）。纯死代码清理，无行为变更。
- 2026-08-26 治理结构重构：AGENTS.md 拆分瘦身，引入 ai-collab-contract 方法论（STATE.md/decisions.md/经验库/pre-commit 扩展）。
- 2026-08-26 D-006：项目第一负责人/技术决策者/独立复审方由 Claude 整体移交给 Devin CLI（业主授权）；Claude 不再担任任何固定角色。AGENTS.md §0/§5 + STATE.md 标记同步。
- 2026-08-26 D-007：业主全权授权 Devin CLI 自主执行常规 commit/push/deploy，无需逐次授权；破坏性操作仍需确认。AGENTS.md §6 同步。

- 2026-08-26 commit VM-CACHE-INTEGRITY-1/2 已验收代码：SourceHash 绑定 + 序列化完整性 + marshalHook 测试注入（含返工）。

- 2026-08-26 commit LIVE-ORDER-REENTRY-1 R4 已验收代码：open mutation fail-closed + adapter label pipeline + WaitState 确定性同步（含返工）。

- 2026-08-26 落档第三批施工提示词：VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD（编译器正确性）。

- 2026-08-26 VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done：Devin CLI 验收通过。compileDeclaration 多变量 + binaryOp error + switch fallthrough + single-statement body + initGlobals ValClass + ClassTypes 序列化 + compileCall relocation + patchUserCalls + sort.Strings。对抗证明 9 项 RED→restore→GREEN，门禁全绿。
