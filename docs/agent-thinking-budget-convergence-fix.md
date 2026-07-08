# 修复方案: 推理模型"想到 token 耗尽也不出代码" —— 向 Claude Code 对齐

> 交付对象: DeepSeek 落地实现
> 症状: 复杂 prompt 下，模型把整段思维链塞满 8192 token 预算，`finish_reason=length` 截断，最终从未产出"策略代码"
> 用户诉求: Claude Code 是怎么处理这种对话的？向它对齐
> 日期: 2026-07-08

---

## §1 精确诊断（已核实到行）

看用户贴的对话：**整段全是英文"Let me think... hmm... maybe... actually..."的自我辩论**，反复纠结 `5美金空间`/`200倍`/`时间周期` 到底怎么解释，最后一句被硬生生截断。

三个已核实的事实：
1. `defaultMaxTokens = 8192`（`@/opt/ant/backend/internal/service/systemai/chat.go:136`）—— 推理 + 代码必须共享这 8192，推理先把它吃光。
2. **没有任何 reasoning 预算控制**：`ChatCompletionRequest`（`@/opt/ant/backend/internal/service/systemai/chat.go:56-64`）只有 `max_tokens`，没有 `reasoning_effort` / thinking budget。
3. `finish_reason=length`：模型在推理阶段就撞到上限，`content` 通道永远是空的 → 没有代码。

**根因不是 bug，是架构错配**：我们把一个"无界思考"的推理模型丢进一个"一次性长文本生成"的流程里，既没给它足够预算，也没有任何机制强制它收敛。它于是陷入 analysis paralysis（分析瘫痪）。

---

## §2 Claude Code 是怎么处理的（对齐目标）

Claude Code 面对同样复杂/模糊的任务不会卡死，靠四条机制：

**A. 思考预算与输出预算分离、且有界。**
Anthropic 的 extended thinking 有独立的 `thinking.budget_tokens`，且 `max_tokens > budget_tokens`。思考**不能**吃掉答案的预算；思考预算耗尽时，模型被强制停止思考、用剩余预算产出答案。我们现在是思考和答案共用一个池子，思考把池子抽干。

**B. 拆解任务 + 用工具产出，而不是一次性长文本。**
Claude Code 把大任务拆成 plan（TodoWrite），逐步执行；代码通过**结构化工具调用**（Write/Edit）产出——交付物是 tool 参数，不是自由文本。工具调用在独立通道，模型无法"永远说下去"，因为流程结构要求它**动手**。

**C. 对模糊性"果断决策"，绝不内部空转。**
Claude Code 遇到歧义要么用专业默认值直接推进并注明假设，要么问**一个**聚焦的问题——绝不会花 4000 token 纠结"5美金到底是不是 5 pip"。这既是训练，也是提示词纪律。

**D. 用对模型。**
Agentic tool-calling 循环用的是为"有界思考 + 工具使用"调校的模型，不是纯推理模型（会 dump 无界 CoT、且不擅长 tool calling）。

---

## §3 落地方案（按优先级，DeepSeek 逐条实现）

### 🔴 Fix 1 — 分离并限制思考预算（直接消除 `finish_reason=length`）

对齐 Claude Code 的机制 A。

1. **给 agent 功能调高 `max_tokens`**：8192 → 至少 `16384`，推荐 `32768`（推理 + 完整策略代码要能都放下）。做成 **feature-aware**：agent/generator 用高值，普通 chat 保持 8192。
   - 实现：`doChatRequest`（`@/opt/ant/backend/internal/service/systemai/chat.go:141`）增加 feature 感知，或在 agent 调用链把 `maxTokens` 传成高值（`chatProvider.maxTokens` 已可透传，见 `chat_failover.go:262`）。
2. **传 reasoning 预算控制**（`ChatCompletionRequest` 增字段，provider 门控）：
   ```go
   ReasoningEffort string `json:"reasoning_effort,omitempty"` // "low"|"medium"|"high"（OpenAI o系 / 部分网关）
   ```
   agent 场景设 `reasoning_effort:"low"` —— 让模型少想、快出。多数 OpenAI 兼容网关会接受或忽略未知字段，风险低；但要**按 provider 门控**，避免个别严格网关 400。
   - 若该网关支持 DeepSeek/Anthropic 风格 `thinking:{type:"enabled",budget_tokens:N}`，优先用它并保证 `max_tokens > budget_tokens`。

> ⚠️ 单独调高 max_tokens 不够 —— 模型可能只是想得更久。必须配合 Fix 2/3 强制收敛。

### 🔴 Fix 2 — 收敛守卫（对齐机制 B："必须动手"）

在 `AgentLoop.run`（`@/opt/ant/backend/internal/connect/ai/agent_loop.go`）里，每轮结束后检测"只思考、没产出"：

- 触发条件：本轮 `finishReason=="length"` **或**（`roundText`/`content` 为空 **且** 没有 tool call），但 reasoning 很长。
  - 需要在 onChunk 里捕获本轮最后的 `chunk.FinishReason`（`ChatStreamChunk.FinishReason` 已有，见 `chat.go:91`）。
- 处理（最多重试 2 次，每次给全新预算）：注入一条纠偏消息后重跑本轮：
  ```
  role: user (或 system)
  content: "你已经分析够了，停止思考。现在立刻输出完整的 Python 策略代码
  （一个 markdown 代码块，class MyStrategy，含 on_bar），然后调用 [TOOL: compile_python]。
  任何不确定的参数一律用专业默认值并在一行注释里注明，不要再权衡多种解释。"
  ```
- 这就是 Claude Code"思考预算耗尽 → 强制产出"的等价实现。

### 🟡 Fix 3 — 提示词从"禁止分析"改为"限制分析 + 强制果断"（对齐机制 C）

**现在的提示词是反效果的**：`@/opt/ant/backend/internal/ai/locale_agent_en.go` 说"只输出代码、不要分析、不要 [THINK]"。对推理模型这**适得其反**——它照样在 reasoning_content 里狂想（提示词压不住），而"不许分析"又让它在 content 通道永远不敢下结论。用户你已经在改这些文件，请按下面方向定稿（EN/ZH/ZHTW/JA/VI 全改）：

- 允许**简短**思考：`Think briefly (2-3 sentences max), then output the code.`
- **强制果断**（直接打击观察到的空转）：
  ```
  For ANY ambiguous requirement, immediately pick a professional default and note it
  in ONE inline comment. NEVER deliberate between interpretations — decide and move on.
  If requirements conflict (e.g. timeframe), choose the most sensible one and state it once.
  Do not enumerate alternatives. Do not re-analyze the same point twice.
  ```
- 保留"缺方向/入出场逻辑就问一个问题"，但**扩大**到"实质影响行为的量化歧义"（如仓位计算基准、盈利目标单位）：宁可问一个聚焦问题，也不要内部空转。

### 🟡 Fix 4 — 用对模型（对齐机制 D）

agent/generator 这种 tool-calling 循环，**默认选非纯推理的 instruct/chat 模型**（如 `deepseek-chat`/v3），而不是把无界 CoT 塞进 reasoning_content 的纯推理网关模型。

- 短期：在 AI Settings 给用户提示"策略生成推荐使用 chat 类模型"。
- 或：agent 功能对模型做软性优选/警告（若检测到 reasoning_content 占比畸高）。

---

## §4 落地顺序

```
1. Fix 1  调高 agent max_tokens + reasoning_effort=low     ← 立刻缓解截断
2. Fix 3  提示词改果断（用户已在改，定稿即可）              ← 从源头减少空转
3. Fix 2  收敛守卫（length/空产出 → 强制出代码重试）        ← 兜底保证一定有代码
4. Fix 4  模型选择建议                                      ← 长期
```

Fix 1+3 大概率就解决问题；Fix 2 是"无论如何都要拿到代码"的安全网。

---

## §5 验收标准

1. 用户那条复杂 prompt → 不再出现"整段思考被截断、无代码"。
2. `docker logs ant-backend | grep "loop done"` → `raw_len > 0` 且提取到 `class MyStrategy`。
3. 若模型仍想太久 → 收敛守卫触发，日志可见强制重试，最终产出代码。
4. 对话里模型对 `5美金空间`/`200倍` 用默认值 + 注释说明，而非反复权衡。

---

## §6 硬约束提醒

- ✅ `reasoning_effort` 等字段必须 **provider 门控**，避免严格网关 400（沿用现有 failover 逻辑）。
- ✅ 收敛守卫的重试次数要有上限（≤2），避免无限循环烧钱。
- ❌ 不新增 REST/WebSocket/JSON 持久化；价格/盈亏 `decimal.Decimal`。
- ✅ 提交前 `go build ./...` + `cd backend && go run ./tools/check-file-lines --strict`。
- ✅ 部署 `docker compose build backend && docker compose up -d backend`。
- ✅ 新 file/function 标 `REUSE:`/`NEW:`。
