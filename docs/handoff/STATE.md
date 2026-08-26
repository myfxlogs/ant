# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: D-006/D-007 治理移交完成并推送；D-REVERT-CLEANUP-001 修复 revert `830b2c79` 遗留 122 个拆分文件导致的 build 断裂（14 包 redeclaration → 全线 `go build` 通过）。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| D-006 角色移交 Claude→Devin CLI | ✅ | AGENTS.md §0 |
| D-007 业主全权授权常规操作 | ✅ | AGENTS.md §6 |
| D-REVERT-CLEANUP-001 build 断裂修复 | ✅ | registry D-REVERT-CLEANUP-001 |
| 盘查 registry open 条目 | 🔄 | 制定优先级计划中 |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效（D-VM-LIVE-001 验收前禁止从 main 构建部署 backend）。
- **下一步**: 完成 registry open 条目盘查，制定 P0/P1 优先级工作计划。
- **清扫上翻**: 无私有记忆需清扫。

## 活跃 registry 条目指针

> 完整明细见 `docs/audits/tech-debt-registry.md`。这里只列当前活跃（🟦open / ⚠️待独立复审）条目。

- **D-REVERT-CLEANUP-001** ✅done — revert 遗留拆分文件 build 断裂修复（2026-08-26）
- **BT-FUNC-ENTRYPC-FWD** 🟦open — user→user 前向引用 stale marker PC
- **VM 返工批第三阶段** ⚠️待独立复审 — 6 条 🟦open→✅done（2026-08-25）
- **VM 返工批第二阶段** ⚠️待独立复审 — 10 条 🟦open→✅done（2026-08-25）
- **SCHEDULE-HOTLOOP-1** ⚠️待生产部署验收

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-08-26 D-REVERT-CLEANUP-001：修复 revert `830b2c79` 遗留 122 个拆分文件导致的 build 断裂（14 包 redeclaration）。纯死代码清理，无行为变更。
- 2026-08-26 治理结构重构：AGENTS.md 拆分瘦身，引入 ai-collab-contract 方法论（STATE.md/decisions.md/经验库/pre-commit 扩展）。
- 2026-08-26 D-006：项目第一负责人/技术决策者/独立复审方由 Claude 整体移交给 Devin CLI（业主授权）；Claude 不再担任任何固定角色。AGENTS.md §0/§5 + STATE.md 标记同步。
- 2026-08-26 D-007：业主全权授权 Devin CLI 自主执行常规 commit/push/deploy，无需逐次授权；破坏性操作仍需确认。AGENTS.md §6 同步。
