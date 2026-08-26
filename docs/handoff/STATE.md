# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: 治理结构重构完成——AGENTS.md 拆分瘦身（42KB→≤20KB），技术约束/pitfalls 迁到 T1 文档，引入 STATE.md/decisions.md/经验库三件套。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| AGENTS.md 拆分瘦身 | ✅ | 新 AGENTS.md ≤20KB |
| docs/constraints.md | ✅ | T1 技术约束 |
| docs/pitfalls.md | ✅ | T1 坑库 |
| docs/项目定位.md | ✅ | T1 业务方向+导航 |
| STATE.md / decisions.md / LOG.md | ✅ | docs/handoff/ |
| 经验库三件套 | ✅ | docs/经验库/ |
| pre-commit hook 扩展 | ✅ | STATE.md 必更新+文档预算 |
| CLAUDE.md / .windsurfrules 入口壳 | ✅ | → @AGENTS.md |
| 提交+推送 | ⚠️待Claude复审 | 待业主授权外向操作 |

- **阻塞/待决策**: 无。等待业主授权 commit+push。
- **下一步**: 业主授权后 `git add -A && git commit && git push`。
- **清扫上翻**: 无私有记忆需清扫。

## 活跃 registry 条目指针

> 完整明细见 `docs/audits/tech-debt-registry.md`。这里只列当前活跃（🟦open / ⚠️待Claude复审）条目。

- **BT-FUNC-ENTRYPC-FWD** 🟦open — user→user 前向引用 stale marker PC
- **VM 返工批第三阶段** ⚠️待Claude复审 — 6 条 🟦open→✅done（2026-08-25）
- **VM 返工批第二阶段** ⚠️待Claude复审 — 10 条 🟦open→✅done（2026-08-25）
- **SCHEDULE-HOTLOOP-1** ⚠️待生产部署验收

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-08-26 治理结构重构：AGENTS.md 拆分瘦身，引入 ai-collab-contract 方法论（STATE.md/decisions.md/经验库/pre-commit 扩展）。
