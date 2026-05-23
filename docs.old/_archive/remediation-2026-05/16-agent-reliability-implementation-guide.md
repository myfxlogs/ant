---
title: Agent 异步、重试与 Provider Fallback 实施指南
date: 2026-05-02
status: draft
---

# 目标

提升 AntTrader AI Agent 在跨区域、慢模型、限流、短暂网络异常下的可靠性。

本指南遵守项目治理约束：

- 浏览器到后端业务通信统一使用 ConnectRPC。
- 业务调度、重试、fallback、状态机均在后端完成。
- 前端只展示状态、触发用户动作和轮询/订阅结果。
- 单个代码/proto/配置生成文件不得超过 800 行。
- 先完成文档设计，再分批实现。

# 当前问题

当前 `/ai/debate` 专家讨论链路中，关键步骤仍同步调用模型：

- `DebateV2Service.invokeStep` 直接执行 `provider.Chat`。
- `DebateV2Service.runCodeGenerationWithFeedback` 直接执行 `provider.Chat`。
- 一次 ConnectRPC 请求会一直等待模型返回。

这会导致：

- 国内大模型跨境访问慢时，前端长时间等待。
- provider 短暂 502/503/504/timeout 会直接失败。
- 单个 provider 限流或故障时没有自动降级。
- 页面刷新后用户对“任务是否仍在执行”感知弱。

# 分阶段范围

## P1：最小可靠版本

本阶段只做低侵入增强：

1. Provider 调用重试
2. Provider fallback
3. Debate V2 步骤异步执行

不做：

- 多区域部署调度
- 前端本地重试
- 新增 REST 接口
- 复杂队列系统
- 多 worker 分布式锁

## P2：增强版本

后续可继续增加：

- 可配置 fallback 顺序
- 任务取消
- 任务超时回收
- SSE/streaming 状态推送
- provider 级 SLA 统计

# 后端设计

## 1. Provider 调用重试

新增后端内部 helper，统一包装 AI provider 调用。

建议入口：

- `backend/internal/service/ai_call.go`

职责：

- 执行 `provider.Chat(ctx, messages)`。
- 对可重试错误最多重试 2 次。
- 使用指数退避和 jitter。
- 保持总超时由调用方 context 和 provider `timeout_seconds` 控制。

可重试错误：

- `context deadline exceeded`
- `timeout`
- `connection reset`
- `EOF`
- `temporary`
- `status 429`
- `status 502`
- `status 503`
- `status 504`

不可重试错误：

- API Key 错误
- 401 / 403
- 400 参数错误
- 模型不存在
- 上下文超长
- 业务校验失败

## 2. Provider Fallback

fallback 在后端 provider 选择层完成。

本阶段规则：

1. Agent 绑定 provider 优先。
2. 用户 primary provider 第二优先。
3. 其它 enabled + has_secret + default_model 非空 provider 作为 fallback。
4. 每个 provider 内部先重试，再切下一个 provider。
5. 成功后在 turn meta 中记录最终 provider/model。

初版不增加 UI 配置，避免过度设计。

建议新增方法：

- `AIConfigService.GetFallbackProviders(ctx, userID, preferred)`
- 或在 `DebateV2Service` 内部按现有 `system_ai_configs` 组装候选。

候选去重规则：

- providerID + model 组合唯一。
- agent provider 和 primary provider 不重复加入。

## 3. Debate V2 异步执行

当前 Debate V2 的用户动作包括：

- start
- chat
- approve intent
- run next agent
- approve consensus
- reject code
- approve code

本阶段优先将会触发模型调用的动作异步化：

- chat 后触发 assistant reply
- approve intent 后触发下一个 agent reply
- run next agent
- approve consensus 后触发 code generation
- reject code 后触发 code regeneration

建议状态策略：

- 请求进来只更新 session/turn 状态，并启动后台 goroutine。
- ConnectRPC 立即返回当前 session DTO。
- 前端继续使用已有 Get/Fetch 接口轮询 session。
- 后台完成后写入 assistant/code turn，并更新 session status。

最小状态表示：

- session status 保持现有 step 状态。
- 新增 turn status 或 session metadata 可后续做。
- 初版可通过“已存在用户 turn 但暂无 assistant turn”让前端显示生成中。

后台执行要求：

- 使用 `context.Background()` 派生超时 context，不继承 HTTP 请求取消。
- 超时时间取 provider `timeout_seconds`，上限 10 分钟。
- 出错时写入 error turn 或更新 session error metadata。
- goroutine 内必须 recover，避免 panic 影响进程。

# 前端设计

前端保持轻量：

- 发起动作后立即显示“生成中”。
- 轮询当前 debate session。
- 检测到新 assistant/code turn 后停止轮询。
- 显示 provider/model/错误信息。

不在前端实现：

- provider fallback
- retry
- AI 调度
- 业务计算

# 数据库设计

P1 尽量复用现有表：

- `debate_sessions`
- `debate_turns`
- `ai_workflow_runs`
- `ai_workflow_steps`

如需记录异步任务，可新增轻量迁移：

- `ai_agent_jobs`

字段建议：

- `id`
- `user_id`
- `session_id`
- `job_type`
- `status`
- `attempts`
- `error`
- `created_at`
- `started_at`
- `finished_at`

P1 可不建新表，先以 goroutine + debate turn 持久化结果完成最小闭环。

# 实施顺序

## Step 1：可靠调用 helper

- 新增 provider call helper。
- Debate V2 两处 `provider.Chat` 改为 helper。
- 单测覆盖可重试/不可重试判断。

## Step 2：fallback 候选

- 获取 agent provider、primary provider、enabled system providers。
- 按顺序尝试。
- 成功后记录最终 provider/model。

## Step 3：异步 Debate V2

- 对触发 LLM 的 ConnectRPC handler 改成启动后台任务后立即返回。
- 前端增加轮询或复用已有轮询。
- 错误写入可展示状态。

## Step 4：质量检查和部署

- `go test ./...`
- `npm --prefix frontend run lint`
- `npm --prefix frontend run build`
- `python3 scripts/check-file-lines.py`
- `git diff --check`
- `docker compose up -d --build backend frontend`
- 健康检查 `/health` 和 `/`

# 风险控制

- 异步化要保持幂等，避免用户重复点击生成多个 reply。
- fallback 不应吞掉鉴权类错误导致误判，应记录最后错误。
- retry 总耗时不能无限增长。
- 初版不改变前端数据结构，降低回归风险。
- 所有新增代码文件必须低于 800 行。
