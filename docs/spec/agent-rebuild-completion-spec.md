# Agent 重构收尾 Spec（agent-first-principles-rebuild 剩余项）

> **定位**：`agent-first-principles-rebuild.md`（2026-07-08 地基设计）的大部分**已落地**。本 spec 只覆盖**审计方 2026-08-10 实测后确认仍未做**的剩余项——是收尾，不是大重构。
> **关联**：`docs/audits/agent-first-principles-rebuild.md`（上位设计）、registry §8 POST-5/6、agent-engine skill。
> **状态**：🏗 待施工。**日期**：2026-08-10

---

## 1. 实测现状（大部分已 ✅，别重做）

审计方对着代码逐项核验（2026-08-10）：

| rebuild 项 | 状态 | 证据 |
|---|---|---|
| 阶段1 地基（write_strategy/I1/ExtractCode 删/reasoning 回退删）| ✅ | `agent_tools_write.go:60`、ExtractCode 全仓零引用、`chat.go` reasoning 回退删 |
| 阶段2 逐块流式 | ✅ | `generator_agent.go` streamChunk/reasoningStream/toolStream + `agent_loop.go` onChunk 转发 |
| 阶段2 收敛安全网（finish_reason=length 重试）| ✅ | `agent_loop.go:177` `handleConvergenceRetry`（lengthConvergences<2）|
| 阶段3 SDK 多时间框架（旗舰 4H+1H+5M blocker）| ✅ | `BarsForSymbol(symbol,timeframe)`(`runner/context.go:113`)+`BarsTF`+VM builtin `vm_builtin_market.go:163` 真接通 |
| 阶段3 语义压缩 | ✅ | `agent_loop.go:85` `compressContext(messages)` |
| Apply 入口 | ✅ | `GeneratedCodeCard.onApplyCode` 全链路接通 |

**剩余未做（本 spec 范围）**：
| 项 | 状态 | 说明 |
|---|---|---|
| **阶段2 plan 驱动升级** | ❌ 剩余 | `update_plan` 工具在（`agent_tools_plan.go`），但是**"跟踪/展示"**，非"**驱动分步执行**"（§3.4）|
| **阶段2 语义追问** | ❌ 剩余 | 系统提示词**无 clarify/歧义规则**（§3.5）——钱域语义歧义（200倍单位/1h vs 4H/仓位基准）静默默认 = 风险 |
| 阶段2 Apply 自动切 code tab | ⚠️ minor | Apply 已接通；"点击后自动切到 code tab"这个小 UX 待确认/补 |

---

## 2. 目标 / 非目标

**目标**：① plan 驱动——复杂策略先分解（入场/仓位/加仓/出场），逐步 `write_strategy`+验证，而非一个大 turn 想爆 token；② 语义追问——系统提示加规则：**语义性歧义（改变盈亏）→ 必须追问一个聚焦问题**，装饰性（period/阈值）→ 专业默认+注释；③（minor）Apply 后自动切 code tab。

**非目标**：❌ 重做已 ✅ 的项（流式/收敛/SDK 多周期/语义压缩/write_strategy）；❌ 改 I1-I4 不变量（已兑现）；❌ 碰钱模型/计费（FEAT-5 范畴）。

## 3. 实现

### Part 1 — plan 驱动升级（§3.4）
| # | 任务 | 锚点 |
|----|------|------|
| 1 | 系统提示：复杂策略（多条件/多周期/入场+仓位+出场）→ 先 `update_plan` 产 JSON 分步计划；简单策略可跳过直接 write_strategy | agent 系统提示词 |
| 2 | agent loop：plan 存在时，按 step 逐步推进——每步 `write_strategy`（或 edit_code）+ 验证（回测/编译）→ 标 step 完成 → 下一步。**单步任务边界限界推理**（防爆 token）| `agent_loop.go` + `update_plan` 语义（`agent_tools_plan.go`，从"展示"升为"驱动状态机"）|
| 3 | plan 状态持久化进 generateState（已有 plan 字段则复用），前端渲染进度 | `agent_tools_plan.go` + 前端 plan 卡片 |

### Part 2 — 语义追问（§3.5）
| # | 任务 | 锚点 |
|----|------|------|
| 4 | 系统提示加歧义分类规则表：**语义性**（仓位计算基准 / 杠杆倍数单位 / 时间周期 1h vs 4H/5M / 方向定义）→ **必须问一个聚焦问题，禁止乱猜**；**装饰性**（period=14 / 阈值）→ 专业默认 + 一行注释 | agent 系统提示词 |
| 5 | 追问走自由文本通道（不调 write_strategy，§3.1b 前提1）——确保追问不会被当产码 | 提示词 + loop 已支持的文本通道 |

### Part 3 — Apply 自动切 tab（minor）
| # | 任务 | 锚点 |
|----|------|------|
| 6 | 核验 `onApply` 后是否自动切 code tab；若否，补（点击 Apply → setActiveTab('code')）| `AgentGenChat.tsx` onApply / 父组件 tab 状态 |

> **复用核对**（动工前 `cap.sh`）：`update_plan` 工具（已有，升级语义）、`write_strategy`/`edit_code`（已有）、agent loop（已有）、系统提示词（改提示，非新建机制）、自由文本追问通道（已有）。NEW：plan 驱动状态机逻辑 + 追问规则提示词。**禁止重做已 ✅ 项**。

## 4. 验收 + 对抗证明（以 rebuild §6 不变量为准）

- **plan 驱动**：复杂多条件策略 prompt → agent 先出 plan（分步）→ 逐步 write_strategy+验证 → 交付；**对抗证明**：删 plan 驱动 → 复杂策略一个 turn 撞 finish_reason=length（复现"想爆 token"）。
- **语义追问**：给"200 倍杠杆 + 4H"这种语义歧义 prompt → agent **必须追问**（不静默默认）；**对抗证明**：删追问规则 → agent 静默猜（复现风险）。
- **I1-I4 不回归**：交付仍唯一终稿（I1）、仍真回测（I2）、思考不冒充成品（I3）、回测输入透明（I4）——用 rebuild §6 逐条过。

## 5. 更新 registry §8

完工后：**POST-6（阶段3 SDK多周期+语义压缩）→ ✅done**（实测已落地，本 spec 仅确认）；**POST-5（阶段2）→ 收窄为 plan驱动+语义追问**，完工后 ✅。

## 6. 完工回填

registry §8 POST-5/6 状态更新 + handover 变更日志 + 对抗证明。不自行宣告 ✅，等审计方实测。`go build`+`go test`+`check-file-lines` 0🔴 + 前端 `npm run build` 为底线。
