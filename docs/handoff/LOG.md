# LOG — 历史归档（T2）

> 会话纪要 + 已滚出的历史内容。不限行数。

## 2026-08-26 治理结构重构

**会话**: 应用 ai-collab-contract 方法论到 ant 项目。

**变更**:
- AGENTS.md 从 42KB（291 行）拆分瘦身到 ~6KB（145 行）契约
- 技术约束迁到 docs/constraints.md（T1）
- 坑库迁到 docs/pitfalls.md（T1）
- 项目定位迁到 docs/项目定位.md（T1）
- 新增 docs/handoff/STATE.md（T0 交接负载）
- 新增 docs/handoff/decisions.md（T1 日常决策 D#）
- 新增 docs/handoff/LOG.md（T2 历史归档）
- 新增 docs/经验库/三件套（索引+条目+待归纳）
- pre-commit hook 扩展（STATE.md 必更新+文档预算）
- CLAUDE.md / .windsurfrules 改为入口壳

**决策记录**: D-001 ~ D-005（见 docs/handoff/decisions.md）

**原 AGENTS.md 内容归属**:

| 原章节 | 新位置 |
|--------|--------|
| §0 角色定位 | AGENTS.md §0（保留+精简） |
| File & Function Size | docs/constraints.md |
| Command Output Discipline | docs/constraints.md |
| Prohibited | docs/constraints.md |
| Reuse Preflight | docs/constraints.md |
| Platform Protocol | docs/constraints.md |
| Push-First Architecture | docs/constraints.md |
| Data Precision | docs/constraints.md |
| Deployment | docs/constraints.md |
| MQL2GO VM Pitfalls | docs/pitfalls.md |
| Strategy Runner Rules | docs/pitfalls.md |
| Strategy Schedule Engine Pitfalls | docs/pitfalls.md |
| Backtest Status Management | docs/pitfalls.md |
| Frontend Auth & Stream Error Pitfalls | docs/pitfalls.md |
| Broker Snapshot & Stream Pitfalls | docs/pitfalls.md |
| PG Connection Pool | docs/pitfalls.md |
| Before Commit | AGENTS.md（保留） |

**原 CLAUDE.md 内容归属**:

| 原章节 | 新位置 |
|--------|--------|
| Business Direction | docs/项目定位.md |
| Collaboration Principle | AGENTS.md §0（合并） |
| Root-Cause-First Rule | AGENTS.md §7.4 |
| AI 协作工作方法 | AGENTS.md §0（合并） |
| Codebase Navigation | docs/项目定位.md |
| Documentation Rules | docs/constraints.md |
| Mandatory Constraints | docs/constraints.md（合并去重） |
| Frontend Zero-Trust | docs/constraints.md |
| Deployment Pitfalls | docs/constraints.md |
| MQL2GO VM Pitfalls | docs/pitfalls.md（合并去重） |
| Strategy Runner Rules | docs/pitfalls.md（合并去重） |
| RTK 兼容规范 | docs/constraints.md |
