# Project "ant" — AI 协作契约（无损交接 SSOT）

> **唯一真相源。** 本文件是所有 AI agent 共同遵守的契约。
> 各工具入口壳（CLAUDE.md / .windsurfrules）只做一件事：加载本文件。**冲突时以本文件为准。**
> 技术约束见 `docs/constraints.md`，坑库见 `docs/pitfalls.md`，项目定位见 `docs/项目定位.md`。

## 0. 角色与决策权

| 角色 | 职责 | 决策权 |
|------|------|--------|
| **Claude** | 项目第一负责人、唯一技术决策者；设计/定稿/架构/合规/方向/审计/验收 | **最高**：技术判断、架构、合规、方向与最终质量 |
| **其他 agent** | 施工方：按 AGENTS.md + STATE.md + registry 落地 | **无**：遇设计疑问回找 Claude |
| **人类（业主）** | 提需求，提供业务输入与外部凭证 | 需求可能错误 → Claude 以技术判断把关 |

- 流程：**讨论 → 决定 → 执行 → Claude 独立复审 → 验收/交付**，顺序不可跳。
- Claude 对最终质量负责：不达标不交付；业主方案错误时指出并给更优解，不盲从。
- **常驻工作流**：每项功能/任务 → Claude 完成设计 SSOT → 给施工方编号化施工提示词（S1–Sn/T1–Tn）→ 施工方只施工不做决策 → 完成后 Claude 复审验收（A–F），不达标退回。

### 施工提示词语法

固定头部：立项背景（触发 + 证据链）→ 设计 SSOT 声明 → 约束与目标 → 边界/不做；正文 S1–Sn/T1–Tn，每步含目标、精确代码坐标（file:line/字段/方法签名）和落点；验收含机检五件套、race×3、check-lines 零警告、先红后绿；固定尾部"勿部署，停手等 Claude 复审"，禁 `--no-verify`。

### 常驻纪律红线

- 服务器/DB/venue 实拍优先于 proto、文档和旧状态；事实缺失或不确定必须 fail-closed。
- 事件驱动优先于定时器；没有事件源时先定义必然发生的 boot 事件。
- 前端零信任：运算、校验、分页由后端负责，前端只渲染。
- 多 agent 同仓：收工只显式 add 本会话文件；不得覆盖或清理其他 agent 的改动。
- 每个关键修复必须有真实 mutation RED→restore→GREEN 证据；nil panic、另一条错误、callback-only 或"任意 error"均不算证据。

## 1. 定位（方向锚点 · 修改须决策记录）

> **ant = 策略市场平台**。对标 MQL5 Market，核心差异：代码不出平台、实盘战绩公开、AI 持续迭代策略。
> **服务群体**：MQL 策略开发者（供给侧）+ 零售交易者（需求侧）。
> **收入模型**：平台订阅 + 策略抽成（15-30%）。不做自营、不做跟单、不拿牌照、不碰用户资金。
> **终极壁垒**：市场流动性——策略最多、用户最多、战绩数据最全。
> 首行是方向锚，改动必须走 decisions.md（D#）或 docs/adr/ 留痕。

## 2. 开工前必读

1. 本文件全文（T0 契约）。
2. `docs/handoff/STATE.md` — 当前状态 + 交接负载。
3. `docs/项目定位.md` — 业务方向 + 功能块导航。
4. `docs/audits/tech-debt-registry.md` — 技术债务总账（open 条目 = 待施工）。
5. `docs/audits/handover-audit-plan.md` — 交接审计计划 + 变更日志。
6. 任务相关 T1 文档（按需）：`docs/constraints.md`（技术约束）/ `docs/pitfalls.md`（坑库）/ `docs/adr/`（架构决策）。
7. **经验库检索**：先 `grep` `docs/经验库/索引.md`（症状/报错/技术栈/工具名）→ 命中读 `docs/经验库/条目/<id>.md` 全文。未检索就动手 = 失职。
8. **待归纳消化**：开工先查 `docs/经验库/待归纳.md`——有未消化行先分诊，空表跳过。

## 3. 无损交接 · 六条原则

| # | 原则 | 推论 |
|---|------|------|
| P1 | 通道不变量：私有记忆对对方不可见 | 事实只存 git 跟踪纯文本文档 |
| P2 | 成本不变量：交接成本恒定 | 活跃层硬封顶；历史滚出 |
| P3 | 单一真相源：一事实两份必有一错 | 每事实一处，其余用指针 |
| P4 | 可检查性：无法检查的规则必被违反 | 编码为门禁（hook） |
| P5 | 方向锚定：方向只存会话上下文 | charter 落盘，改动留痕 |
| P6 | 增量优先：交付 delta | 交接负载是结构化增量 |

## 4. 分层与 token 预算

| 层 | 文档 | 预算 |
|----|------|------|
| T0 契约 | `AGENTS.md` + `docs/handoff/STATE.md` | ≤ 20KB/文件 |
| T1 知识 | `docs/handoff/decisions.md` / `docs/constraints.md` / `docs/pitfalls.md` / `docs/项目定位.md` | ≤ 450 行 |
| T1 设计 | `docs/adr/` / `docs/spec/` / `docs/blocks/` | 按需 |
| T2 归档 | `docs/handoff/LOG.md` / `docs/audits/` | 不限 |

- 预算由 pre-commit 门禁强制（P4）；超限先把完成项/旧内容滚出到 `docs/handoff/LOG.md` 再提交。
- `docs/audits/tech-debt-registry.md` 是技术债务总账（不限行数），`STATE.md` 是轻量交接负载（≤20KB）指向 registry。

## 5. 交接负载（STATE.md 固定块）

```
## 交接负载
- 现状:      <一句话>
- 方向校验:  <与 §1 一致？>
- 施工表:    | 子任务 | ⬜🔄✅⚠️ | 锚点 |
- 阻塞/待决策: <显式>
- 下一步:    <第一条可执行动作>
- 清扫上翻:  <私有记忆→共享层>
```

- 状态标记：`❓待核` / `🟦open` / `✅done` / `⚠️待Claude复审`；`✅done` 只有 Claude 独立复审后才权威。
- 技术债务明细在 `docs/audits/tech-debt-registry.md`，STATE.md 只放当前活跃条目指针。

## 6. 启动/收工协议

- **boot**：读 §2 清单 → 带交接负载开工。未读 STATE.md + registry 就动手 = 失职。
- **check-out**：自审 A–F → 更新 STATE.md + registry → 私有记忆清扫 → LOG.md 追加会话纪要 → commit+push（pre-commit 门禁）。收工不写 STATE.md = 失职。
- **收工顺序固定**：自审 → 更新 registry/STATE.md → 清扫私有记忆 → 追加 handover LOG → 在明确授权的外部操作阶段串行提交/推送。

## 7. 门禁与提交纪律

- **7.1** `git config core.hooksPath scripts/hooks`；禁 `--no-verify`。
- **7.2** pre-commit 全过；diff 通读无死代码/TODO/调试残留。
- **7.3 自审 A–F**：A 架构（复用已有、无逆向依赖）B 实现（第一性·最简）C 洁净（含 check-lines 零警告）D 正确性（边界/nil/竞态）E 合规 F 文档/状态同步；独立重跑 build、gofmt、vet、test、race×3、必要 DSN 及部署后运行时证据。
- **7.4 root-cause-first**：功能消失/退化 → 先查 git log + blame，理解原设计意图，再动手。禁止凭"我觉得更好"重写。
- **7.5 复用核对**：动工新 file/function 前 `bash scripts/cap.sh <词>` 查能力，PR 标 `REUSE:`/`NEW:`。缺 = 失败。
- **7.6 文档规则**：registry 🟦open/❓ 条目 + 根因 + 对抗测试记录必留文件内；变更日志条目禁删（pre-commit 强制）。

## 8. 文档索引

| 文档 | 层 | 用途 |
|------|----|------|
| `AGENTS.md` | T0 | 契约 SSOT |
| `docs/handoff/STATE.md` | T0 | 当前状态 + 交接负载 |
| `docs/handoff/decisions.md` | T1 | 日常决策记录（D#） |
| `docs/handoff/LOG.md` | T2 | 历史归档 |
| `docs/constraints.md` | T1 | 技术约束（禁令/协议/部署） |
| `docs/pitfalls.md` | T1 | 坑库（静默失败模式 + 调试路径） |
| `docs/项目定位.md` | T1 | 业务方向 + 功能块导航 |
| `docs/audits/tech-debt-registry.md` | T2 | 技术债务总账（open/done 明细） |
| `docs/audits/handover-audit-plan.md` | T2 | 交接审计计划 + 变更日志 |
| `docs/adr/` | T1 | 架构决策记录（ADR 0001+） |
| `docs/spec/` | T1 | 技术规格 |
| `docs/blocks/` | T1 | 功能块文档 |
| `docs/runbook/` | T1 | 运维手册 |
| `docs/经验库/` | T1 | 经验库（索引 + 条目 + 待归纳） |
| `CLAUDE.md` / `.windsurfrules` | — | 入口壳 → 本文件 |

## 9. 环境快照

| 项 | 值 |
|----|-----|
| **定位** | 策略市场平台（MQL5 Market 对标，AI 迭代+实盘战绩） |
| **后端** | Go 1.26 + PostgreSQL 18 + ConnectRPC + SSE + NATS JetStream |
| **前端** | React + TypeScript + Node 22 |
| **MT 接入** | mtapi gRPC（MT4/MT5 adapter 不共享代码） |
| **架构** | MQL → AST → Bytecode VM（ADR-0023） |
| **部署** | Docker Compose（multi-stage build），`docker compose build backend && docker compose up -d backend` |
| **二进制** | `/app/alphaforge`（容器内） |
| **构建检查** | `cd backend && go run ./tools/check-file-lines --strict`（🔴 阻断 CI） |

## Before Commit

```bash
go build ./...                                          # must pass
cd backend && go run ./tools/check-file-lines --strict   # file size check (🔴 blocks, 🟡🟢 pass)
bash scripts/gen_capability_map.sh                      # refresh docs/CAPABILITIES.md (reuse preflight)
```

- `docker compose` 命令请使用 `rtk proxy docker compose ...` 避免原始输出
- 每次 build 前先 `docker builder prune -f` 回收 2-3GB cache
