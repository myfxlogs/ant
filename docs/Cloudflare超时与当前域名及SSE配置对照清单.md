# Cloudflare 超时与当前域名 / Nginx / SSE 配置对照清单

本文将 **Cloudflare 边缘常见超时与错误** 与 **AntTrader 当前仓库中的实际路径与配置** 逐项对照，便于运维自检。**不**在「本机安装的 Cloudflare WARP 客户端」上调整 `proxy_read_timeout` 类参数；那些属于 **Zone（域名）在 Cloudflare Dashboard / API**、**Enterprise 功能** 或 **源站 / 隧道 / Nginx** 层。

---

## 1. 与本项目强相关的 HTTP 路径（便于对照）

| 用途 | 方法 / 路径（示例） | 后端注册（Go） |
|------|---------------------|----------------|
| 辩论 v2 代码生成 / Advance 进度 | `GET /antrader/sse/debate-v2/advance-jobs/{jobId}/stream` | `backend/internal/server/debate_v2_advance_sse.go` |
| 辩论 v2 对话进度 | `GET /antrader/sse/debate-v2/chat-jobs/{jobId}/stream` | `backend/internal/server/debate_v2_chat_sse.go` |
| Connect / gRPC-Web | `POST` 等到 `*.anttrader...` 或 `/antrader.*` 形态 | 经 `frontend` 或直连 `backend`，见 `nginx/nginx.conf` |

前端 SSE 拼装：`frontend/src/client/debateV2.ts`（`apiBaseUrl` + `/antrader/sse/debate-v2/...`）。

---

## 2. Cloudflare 侧：建议核对项与「在我们栈里」的含义

| Cloudflare 概念 / 文档用语 | 典型现象 | 在本项目中的含义 | 建议动作 |
|----------------------------|----------|------------------|----------|
| **Proxy Read Timeout**（边缘等待源站**完整响应**或持续有字节的时间窗口；免费/Pro 约 **100s** 量级，精确值以官方为准） | **524**（边缘已连源，但在超时内未收到认为「足够」的响应） | 历史上 **同步 Unary** 长推理易 524；现已将 Advance/代码生成改为 **Job + SSE**，浏览器长连接走 **`GET .../stream`**，需保证 **SSE 路径上** 边缘与源站超时 **大于** 模型单次无 token 的最长间隔，并配合 **应用层 ping**（见下） | ① Dashboard：**Speed** / **Rules** / **Load Balancing** 等与计划相关的项按官方文档核对是否可调大（多需 **Enterprise**）。② 无法调大时：**灰云 DNS Only** 子域直连 API，或缩短同步请求、保持异步 SSE。 |
| **100s 空闲**（社区与官方对「长时间无上游字节」的普遍描述） | SSE / 流在静默期被掐 | Job Hub 对辩论任务 **`startKeepalive`** 周期性下发 `{"event":"ping"}`（如 12s），见 `backend/internal/service/debate_v2_advance_job.go` 及 `debate_v2_advance_async.go` / `debate_v2_chat_async.go` | 确认边缘到源整条链 **ping 行** 未被缓冲：`proxy_buffering off`（见下）。 |
| **524 / 522** | 源慢或不可达 | `backend/internal/service/ai_call.go` 中 `isRetryableAIError` 含 **524** 等，便于 LLM 调用退避重试 | 区分是 **CF→源** 还是 **源→LLM**；分别调超时或异步化。 |
| **WARP / cloudflared（客户端）** | 用户误以为在此调「网站超时」 | **隧道入口** 与 **Zone 代理** 是不同层面；隧道有自有 `config.yml`（如 `originRequest` 等），**不是**「装个 WARP 就改网站 Proxy Read Timeout」 | 若使用 **Cloudflare Tunnel**：在 **cloudflared 配置 + 后面 Nginx** 与 Dashboard Zone 设置 **分别** 核对。 |

权威入口：[Error 524](https://developers.cloudflare.com/support/troubleshooting/http-status-codes/cloudflare-5xx-errors/error-524/)（以当前官方文档为准）。

---

## 3. 源站 Nginx：当前仓库中的超时（请与上表一起验收）

| 配置文件 | 片段说明 |
|----------|----------|
| `nginx/nginx.conf` | `location ^~ /antrader/sse/` → `proxy_read_timeout` / `proxy_send_timeout` **3600s**，`proxy_buffering off`；`location ~ ^/antrader\..+` → 后端 Connect 路径 **3600s**。 |
| `frontend/nginx.conf` | `location ~ ^/antrader(\.|/|$)` → `proxy_read_timeout` / `proxy_send_timeout` **3600s**，`proxy_buffering off`。 |

**验收关系**：边缘允许静默时长 ≤ 应用 ping 间隔 < Nginx `proxy_read_timeout` < 业务可接受上限。

---

## 4. 应用与模型侧（与「长请求」同一验收表）

| 项 | 位置 |
|----|------|
| 代码生成 system 提示词（体积优化入口） | `backend/internal/service/debate_v2_prompts.go` → `CodeSystemPromptV2`；指标目录 `strategy_param_catalog.go` |
| 智谱 **关闭思考**（全请求带 `thinking: disabled`） | `backend/internal/service` 经 `internal/ai/zhipu/client.go` 中 `thinkingDisabled()` |
| 流式首包 / 总耗时日志 | `backend/internal/service/ai_call.go` → `streamChatWithRetry`；代码生成 Job 完成日志 → `debate_v2_advance_async.go` → `runCodeGenerationStreamed` |

---

## 5. 与纲领文档的交叉引用

- 架构与流式原则：`docs/接口与数据流架构约定.md`  
- 辩论异步 Job 设计：`docs/辩论与代码生成异步任务设计.md`  
- Connect / 524 / Nginx 背景：`docs/边缘网关与长连接问题处理参考.md`

---

## 6. 自检勾选（发布前）

- [ ] 浏览器 Network 中 SSE 请求 URL 是否为 **`/antrader/sse/debate-v2/.../stream`**，且状态长时间为 **pending** 时仍有 **ping** 或 **chunk** 事件。  
- [ ] `nginx/nginx.conf` 与 `frontend/nginx.conf` 中 **`/antrader`** / **`/antrader/sse/`** 的 `proxy_read_timeout` 是否与文档 §3 一致。  
- [ ] Cloudflare 橙云代理下若仍 524：已确认是 **Unary** 还是 **SSE**；SSE 则查缓冲与 ping，Unary 则查是否仍有一次性超长 RPC。  
- [ ] 日志中可检索 `ai stream success`（含 `time_to_first_chunk`）、`debate_v2_code_gen_stream_done` / `debate_v2_code_gen_stream_failed`。
