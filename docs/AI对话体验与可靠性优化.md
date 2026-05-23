# AI 对话体验与可靠性优化

> **本文档是实施的依据与约束。**  
> 与 **`docs/接口与数据流架构约定.md`** 一致：通用 AI 助手（`AIService.Chat`）的能力扩展须走 **Connect / gRPC / SSE**；**禁止**为本文所述能力新增 REST 资源面或 WebSocket。

---

## 1. 背景与目标

| 问题 | 目标 |
|------|------|
| Unary `Chat` 在供应商慢或网络抖动时，用户长时间无反馈 | **Connect server-stream**：边生成边下显（TTFT 体感）； Unary 仍保留兼容与简单客户端。 |
| 主路径未使用已有 `chatWithRetry`，可靠性不足 | Unary 与流式路径均调用 **`service.ChatWithRetry` / `service.StreamChatWithRetry`**（与辩论异步共用同一套退避与可重试错误分类）。 |
| 上下文过长导致首包前耗时与失败率上升 | 对载入会话历史做 **条数上限**（仅保留尾部若干条）；记忆检索失败或超时则 **跳过记忆**（已有超时，明确为降级策略）。 |
| 长连接无上限导致资源占用 | 为单次 LLM 调用设置 **合理 `context` deadline**（与网关超时协调，见 `docs/边缘网关与长连接问题处理参考.md`）。 |
| 辩论 V2「聊天步」仍同步挂起，与 Advance 异步体验不一致 | **二期**：`ChatDebateV2`（含 **`intent` 澄清意图** 与 **`agent:*` 专家对话**）采用与 `Advance` 相同的 **Job + SSE**（事件与路由约定见 **`docs/辩论与代码生成异步任务设计.md`**）。澄清意图起即会话阶段，与二期一并交付，不单拆「仅 intent 流式」小改。 |

---

## 2. 一期范围（本文档首期实施）

| 项 | 说明 |
|----|------|
| **Unary `Chat`** | 构建消息后与 LLM 通信改为 **`ChatWithRetry`**；可选附加 **`context.WithTimeout`** 包裹 provider 调用（上限在实现常量中定义）。 |
| **`ChatStream`（Connect server-stream）** | 新增 RPC：`ChatStream(ChatRequest) returns (stream ChatStreamChunk)`；载荷含增量 `delta`、结束标记 `done`、可选 token 统计；**成功结束后**与 Unary 相同逻辑 **持久化 user + assistant**（失败不落库 assistant）。 |
| **历史条数上限** | 从会话库读取的消息列表，在拼入 `aiMessages` 前 **截断为尾部最多 N 条**（N 在代码常量中定义，建议 40，即约 20 轮）。 |
| **前端 AI 助手** | 主发送路径优先 **`ChatStream`**，在 UI 上 **增量追加** assistant 内容；保留 Unary `chat` 作为降级（如流式不可用时可切换，首期可直接以流式为主）。 |

**不在一期**：辩论 `ChatDebateV2`（含澄清意图步）的 Job + SSE（见 §4 二期）。

---

## 3. 接口约定

### 3.1 Unary `Chat`（保留）

- 语义不变：请求体仍为 `ChatRequest`，响应 `ChatResponse`。  
- 实现变更：内部改为 **`ChatWithRetry`**；与流式共享 **同一套消息构建**（system、记忆、历史、`HistoryB64`、`SystemPromptB64`、用户句）。

### 3.2 `ChatStream`（Connect server-stream）

- **请求**：与 `ChatRequest` 相同。  
- **响应消息** `ChatStreamChunk`（`ai_chat_requests.proto`）：  
  - `delta`：本轮新增的可见文本片段（UTF-8）；可为空（心跳类可不发）。  
  - `done`：为 `true` 时表示流结束；最后一帧可携带完整统计或仅 `done`。  
  - `prompt_tokens` / `completion_tokens`：可选，最后一帧填充（若 provider 提供）。  
  - `error_message`：若流以错误结束，**最后一帧**可填人类可读摘要；同时 handler 可返回 `connect.Error`（以实际实现为准）。  
- **持久化**：仅在 **整段生成成功** 后写入 `user` 与 `assistant` 两条（与 Unary 一致）；流中途断开 **不写** assistant。  
- **网关**：流式 RPC 须 **HTTP/2** 端到端；代理须禁用对响应的缓冲（参见边缘网关文档）。

---

## 4. 二期范围（辩论聊天异步化）

- **范围**：凡服务端允许 `ChatDebateV2` 的步键（**`intent`**、**`agent:*`**）均属会话内聊天；**澄清意图**起用户与模型的多轮回复与专家步无差别，**统一**走本节的 Job + SSE，不在一期另接 Unary 补丁式流式。  
- 主路径：`Connect` **`PrepareDebateV2ChatJob`** 返回 `job_id`（不落 LLM）→ 浏览器 **`EventSource`** 订阅 **`GET /antrader/sse/debate-v2/chat-jobs/{job_id}/stream`** → **`RunDebateV2ChatJob`** 再起流式推理；Unary **`ChatDebateV2`** 保留为降级/简单客户端。  
- Worker 内复用 `invokeStep(..., emitChunk)` 已有流式能力，向 Job 总线 **`chunk`** 推送增量，完成后再落库 assistant，与今日 Unary 成功语义一致。  
- 事件类型复用最小集：`queued` / `running` / `chunk`（可选）/ `completed` / `failed`。  
- 完成后前端 **`GetDebateV2Session`** 刷新会话。  
- 详细路径、RPC 名、与 §12（SSE 重连 / 终态对齐）见 **`docs/辩论与代码生成异步任务设计.md`**；**拒绝重写代码**为 **`StartDebateV2RejectCodeJob`**，与 advance **共用** `advance-jobs` SSE 与 `GetDebateV2AdvanceJob`。  
- 手工验收清单：**`docs/辩论与代码生成二期验收说明.md`**。

---

## 5. 可观测性（建议）

- 为 `Chat` / `ChatStream` 记录：**provider、model、消息条数、耗时、是否重试**（与 `ai_call.go` 现有日志对齐）。  
- 指标可选接入现有监控栈（不在一期强制）。

---

## 6. 修订记录

| 日期 | 变更 |
|------|------|
| 2026-05-02 | 初稿：一期 Unary 重试 + `ChatStream` + 历史上限 + 二期辩论 Chat Job 化。 |
| 2026-05-02 | 一期已落地：`AIService.Chat` 使用 `ChatWithRetry`；新增 `ChatStream`（Connect server-stream）+ `StreamChatWithRetry`；会话历史尾部截断；前端 `aiStore` 优先流式并 Unary 降级。 |
| 2026-05-02 | §4 二期：明确 **`intent` 澄清意图** 与 **`agent:*`** 一并纳入 `ChatDebateV2` Job + SSE；与 `invokeStep` 流式回调对齐。 |
| 2026-05-02 | §4：`StartDebateV2RejectCodeJob` 与 advance 共用 SSE；指向 **`docs/辩论与代码生成二期验收说明.md`**。 |
