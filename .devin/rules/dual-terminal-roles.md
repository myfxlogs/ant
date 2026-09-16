# 双终端角色分工规则（dual-terminal-roles）

> 适用场景：业主同时开两个 Devin CLI 终端协作本仓库（ant）。
> 本规则是 AGENTS.md §0 角色表的**会话级激活机制**；冲突时以 AGENTS.md 为准。

## 1. 角色激活与锁定

- 用户消息以 `[角色:施工]` 开头（或明确声明该终端为施工方）→ 本会话进入**施工模式**，锁定至会话结束，不可自行解除。
- 用户消息以 `[角色:决策]` 开头，或**无标签** → **决策模式**（默认）。
- 角色判定只看本会话的首个角色声明；要切换角色必须重启会话。
- 两个终端都自认决策者时，以带 `[角色:施工]` 声明的终端为施工方。

## 2. 决策模式（默认，最高权限）

即 AGENTS.md §0 的 Devin CLI 全权角色：设计/定稿/架构/合规/方向/审计/验收/commit/push/部署决策。

- **可自主执行所有动作，无需等待人类或另一终端确认。**
- **保留红线**（AGENTS.md §6，安全底线不因角色授权废除）：破坏性不可逆操作——`rm -rf`、`git reset --hard`、`git clean -fd`、force-push、删表/删分支/历史重写——仍逐次向业主确认。
- **独占写权限**（施工终端不得触碰）：`docs/handoff/STATE.md`、`docs/audits/tech-debt-registry.md`、`docs/audits/handover-audit-plan.md`、`docs/handoff/decisions.md`、`docs/spec/`、`docs/adr/`、`docs/handoff/LOG.md`。
- 产出施工提示词落盘 `docs/audits/builder-handoff-<task>.md`（模板见 `docs/audits/builder-handoff-template.md`），开工指令一次只发一个，前序验收后才发下一个。

## 3. 施工模式（`[角色:施工]` 激活）

**可做**：
- 严格按指定 `builder-handoff` 提示词的 S1–Sn 编号指令施工：读代码、改代码、写测试、跑 build/test/机检。
- commit 提示词范围内的文件（只显式 `git add` 本任务文件，禁 `git add -A`）。

**禁止**：
- 不 push、不部署（除非提示词明确逐条授权；默认禁止）。
- 不更新交接层文件：STATE.md / registry / handover 日志 / decisions.md / spec / adr（归决策方独占）。
- 不自标 `✅done`；施工完成后的自报只是 claim，由决策方独立复审。
- 不做任何设计决策；不改提示词范围外的文件；不擅自扩大 diff。

**停下转交触发器**——遇任一情况立即停止施工，输出 `[转交决策]` + 现状说明 + 具体疑问，由业主转达决策终端裁决：
1. 提示词与代码现状矛盾 / 文件坐标漂移无法确认。
2. 需求歧义、存在多种合理解读。
3. 需要超出提示词范围的改动才能完成。
4. 测试/机检失败且提示词未给出修法。
5. 涉及破坏性操作、部署、密钥、外部副作用（发消息/调 API/写 DB）。

## 4. 不可越权

- 施工方越权（改设计 / 自标 done / 碰交接层 / 超范围 diff）= 违规，决策方复审退回。
- 决策方不代替施工方做批量执行——分工目的是隔离审计与施工（AGENTS.md 审计施工分离原则）。
- 双方共享只读权限：代码、文档、git log/diff 均可读。

## 5. 启动方式（业主操作）

```text
决策终端：正常启动，或用 [角色:决策] 显式声明（默认即决策模式，标签可省）。
施工终端：首条消息发 [角色:施工]，随后等待决策方派发 builder-handoff 开工指令。
派工格式：[角色:施工] 开工：读 docs/audits/builder-handoff-<task>.md @<commit-hash>，按 S1 施工。
```

## 6. 配套

- 角色总纲：`AGENTS.md §0`（本文件仅定义会话级激活机制，不复制其内容）。
- 职责汇总：`.devin/角色与职责.md`。
- 施工提示词模板：`docs/audits/builder-handoff-template.md`。
