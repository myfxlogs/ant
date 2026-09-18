# Builder Handoff — EXT-BOUNDARY-WAVE2

> 第二梯队边界缺陷批（P2）：6 项 fail-open/静默失真，2026-08-13 止血扫描登记。
> 2026-09-19 Devin CLI 设计实查：六项全部实证仍在，坐标与修法逐项核实。
> 设计 SSOT：Devin CLI。施工只执行、不决策。完成报证据等独立复审。

## 0. 六项实况（逐项实证）

| # | 项 | 实证坐标 | 现状 |
|---|---|---|---|
| S1 | LLM SSE 流死检测 | `internal/service/systemai/chat_stream.go:143-148,190-264` | `http.Client{Timeout:0}`（有意——流可长），`ResponseHeaderTimeout` 只管首字节；`handleStreamResponse` `scanner.Scan()` **块间无超时**——中途 stall（连接活、无数据）永久挂起。调用方 ctx 无 deadline 兜底（generator_agent/strategy_plan_handler 均透传 RPC ctx，无 WithTimeout）。 |
| S2 | 链上 monitor stall | `internal/chain/monitor.go:117-131` | scanTicker 3s 循环调 `scanBlocks`，持续 error 只 `log.Error` 重试——checkpoint 永不前进时无任何观测（充值不到账静默）。 |
| S3 | SMTP auth 吞 | `internal/notifier/email.go:105-109` | `conn.Auth(auth)` 失败 → `log.Warn("trying without auth")` → **无认证续发**。凭证腐烂不报错——margin call/kill switch 告警邮件静默失败或走未认证路径。 |
| S4 | webhook transport fail-open | `internal/agent/hooks.go:166-177` | `http.DefaultClient.Do` error → warn → `HookResult{}`（allow）；请求构造失败 :167-170 同样 allow。同文件 `>=400→Abort`、`allow:false→Abort`——transport 错单向 fail-open，不对称。 |
| S5 | TronScan 第二源 fail-open | `internal/chain/monitor.go:293-304` | `VerifyTransaction` error → retry 一次 → 仍败 → `verified = true // degrade to single-source`——**真钱充值**第二源不可用时自动确认（fail-open）。MANUAL_REVIEW 路径（:307-324）已存在。 |
| S6 | FinishReason 丢弃 | `internal/service/systemai/chat.go:330-345` | `ChatCompletionResponse.Choices[].FinishReason` 已解析（:74）但 `parseChatResponse` 从不读——`finish_reason="length"` 截断响应原样交付（截断的策略代码当完整代码收下）。流式路径已透传 `ChatStreamChunk.FinishReason`（消费方处置属边界外）。 |

## 施工步骤

### S1 — SSE stall 看门狗（`chat_stream.go handleStreamResponse`）
- stall 预算：`stallTimeout := effectiveTimeout(p.timeoutSeconds, 5*time.Minute)`——复用既有钳位（5s-10min）；默认 5min 给推理模型块间思考留量，`timeout_seconds` 已是 per-provider 调钮（与 `:146` ResponseHeaderTimeout 同源惯例）。
- 机制（不重构 scanner 循环）：循环前启看门狗 goroutine——`stalled atomic.Bool` + `lastActivity atomic.Int64`（unixnano）；每成功 `scanner.Scan()` 一行即刷新 `lastActivity`；看门狗 ticker（`stallTimeout/2` 间隔）发现 `time.Since(lastActivity) > stallTimeout` → 置 `stalled` + `resp.Body.Close()`（scanner 解除阻塞）+ 退出。
- 循环结束后 `stalled` → `return &failoverErr{msg: fmt.Sprintf("[%s] chat stream stalled: no data for %s", p.providerID, stallTimeout), transient: true}`（vendor 中途死=合理 failover 场景）；`defer` 停看门狗。
- 原子量用 `sync/atomic`（仓内惯例）或最小 mutex——勿引入新依赖。

### S2 — monitor checkpoint stall 告警（`monitor.go`）
- `Monitor` 加字段 `lastProgressAt time.Time` + `lastSafeLatest int64`（select 单 goroutine，无锁）。
- `scanBlocks` 内 `*lastBlock` 前进处刷新 `m.lastProgressAt = time.Now()`；每 tick 记 `m.lastSafeLatest = safeLatest`。
- `Run` 主循环加 `stallTicker`（1min）：`time.Since(m.lastProgressAt) > 15*time.Minute && m.lastSafeLatest > lastBlock` → `m.log.Error("chain monitor: checkpoint stalled — deposits not being credited", zap.Time("last_progress",...), zap.Int64("last_block",...), zap.Int64("safe_latest",...))`。
- **链停不误报**：`lastSafeLatest <= lastBlock`（链无可确认块）不告警——只报"有活干但推不动"。复用 `runner_health.go` ticker+阈值+log 惯例（REUSE: `internal/mdgateway/runner_health.go:13-43`）。

### S3 — SMTP auth fail-closed（`email.go:99-109`）
- `conn.Extension("STARTTLS")` 后：`conn.Extension("AUTH")` 判定——**未通告 AUTH**（合法无认证 relay）→跳过 auth 静默续发；**通告 AUTH** → `conn.Auth(auth)`，失败 → `return fmt.Errorf("smtp auth: %w", err)`（凭证错误=显式失败，margin call/kill switch 告警不得静默丢）。
- PlainAuth 未加密拒发（`StartTLS` 失败的服务器）同走 error 路径——凭证/告警内容不经明文管道。

### S4 — webhook transport fail-closed（`hooks.go execWebhook`）
- `http.DefaultClient.Do` error → `HookResult{Abort: true, Reason: fmt.Sprintf("webhook unreachable: %v", err)}`（与 :182 `>=400→Abort` 对称——门控不可达=拒绝）。
- `http.NewRequestWithContext` error（:167）同改 Abort（URL 配置坏=门控无法运行）。
- **边界如实记**：`execCommand` 非 exit-2 失败仍 allow（:149-154）——同族缺口但属"exit-2=abort"契约内，另立观察不入本债。

### S5 — TronScan 不可用 → MANUAL_REVIEW（`monitor.go:293-324`）
- 二次重试仍败 → **删 `verified = true` 降级**，走既有 MANUAL_REVIEW 分支：抽 `createManualReviewDeposit(ctx, info, evt)` helper 供 `!verified` 与 verify-error 两路共用（消除 :310-323 与 :330-343 的重复 Create 块）。
- 语义：第二源不可核实≠已确认——真钱 fail-closed 入人工审，TronScan 恢复后人工放行至 ConfirmDeposit。日志区分 `verification inconclusive`（API 不可用）vs `did not confirm`（查无此交易）。

### S6 — finish_reason=length fail-closed（`chat.go parseChatResponse`）
- `msg := cr.Choices[0].Message` 后：`fr := cr.Choices[0].FinishReason`；`fr == "length"` → `return "", nil, nil, &failoverErr{msg: fmt.Sprintf("[%s|%s] output truncated at max_tokens (finish_reason=length)", p.providerID, p.model), transient: false}`——**non-transient**：同参重试必再截断，勿烧 failover 预算；截断策略代码不得当完整交付。
- 其余值（stop/tool_calls/content_filter/""）不动——registry 只点名 length 截断，content_filter 边界如实记。

## 测试规格
- **S1**：httptest server 挂起不出数据（`time.Sleep > stallTimeout`）→ `tryChatCompletionStream` 返 failoverErr 含 `stalled`（测试用 1s stall 预算——注入小超时参数化或改包级变量，勿真 sleep 5min）。
- **S2**：scanBlocks 连续 error 或链新块而 lastBlock 不动 → stallTicker 到点 log.Error（注入 Clock/缩阈值参数化，勿真等 15min）。
- **S3**：fake SMTP server 通告 AUTH+拒凭证 → Send 返 error 含 `smtp auth`；不通告 AUTH → 无认证发送成功。
- **S4**：webhook URL 指向不可达地址（`httptest` server 立即 Close）→ `Fire` 返 `Abort=true` 含 `unreachable`。
- **S5**：`VerifyTransaction` 双败 → depositRepo 收 MANUAL_REVIEW（非 auto-confirm）——mock scan client。
- **S6**：response body `finish_reason:"length"` → `tryChatCompletion` 返 error 含 `truncated`+`finish_reason`，`transient=false`（不 failover）。

## 对抗证明规格
- M1 删 S1 看门狗 → S1 测试挂起/RED；M2 删 S4 Abort → S4 RED；M3 恢复 `verified=true` → S5 RED（auto-confirm 复活）；M4 删 S6 length 检查 → S6 RED；M5 恢复 S3 warn-continue → S3 RED。
- restore → 全 GREEN，工作区逐字节一致。

## 边界（不做）
- `execCommand` 非-2 退出 fail-open（同族，另债）；流式 FinishReason 消费方处置；`content_filter` 语义；数组参数等一切出范围项。

## 门禁
`go build ./...` / `go vet ./...` / 目标包 `go test` + `-race` / `check-file-lines --strict` 0 errors / `git diff --check` / gofmt 触碰文件净。**勿部署，停手等 Devin CLI 复审。**
