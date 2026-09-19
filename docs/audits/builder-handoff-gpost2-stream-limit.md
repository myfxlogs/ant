# 施工派工单 — G-POST2-1/2 ConnectRPC 流上限+metric

**日期**: 2026-09-19 · **设计方**: Devin CLI · **施工**: 施工方（ANT_ROLE=builder）

## 立项背景（设计实查实证）

POST-2 探针批实测：每流 ~2 server goroutine+~45-50KB，前端全量走 ConnectRPC binary。双层缺陷：

1. **形态错位**：`SSEStreamLimitMiddleware(5)`（main.go:276）的 `isSSERequest`（sse_limiter.go:65-72）只匹配 `text/event-stream`；前端零 EventSource、纯 `application/connect+proto` → limiter 在主形态从不触发。
2. **键失效**（本实查新发现）：`sseOwnerKey` 取 `GetUserID`/`GetClientIP` 皆由 auth Connect interceptor 注入（auth.go:184/:193），跑在 HTTP middleware **之后**→执行时恒空→key 恒 `"anon"`→即便真 SSE 请求也是全进程共享 5 条流（非 per-user 5）。

## S1 新建 `backend/internal/interceptor/stream_limit.go`

`StreamLimitInterceptor` 实现 `connect.Interceptor`（参照 auth.go:41-87 三方法形态）：

- `WrapUnary`/`WrapStreamingClient`：直通。
- `WrapStreamingHandler`：handler 调用前 acquire、返回后 defer release（handler 返回=流结束）。
- **键**（顺序）：`GetUserID(ctx)`→cap `userCap`；否则 `GetClientIP(ctx)`→cap `ipCap`；否则 `"anon"`→cap `ipCap`。
- **拒绝**：`connect.NewError(connect.CodeResourceExhausted, errors.New("too many concurrent streams"))`（RPC 层错误，前端可正常消费，非裸 429）。
- **限额**：`NewStreamLimitInterceptor(userCap=10, ipCap=30)` 构造参数。依据：实测每 tab ~3-6 并发流（subscribeEvents+userSummary+watchActive+paper+analytics+chat），user 10≈2 tab 余量；IP 30 覆盖 NAT 多用户未认证场景。
- **内部**：mutex+`map[string]int`，acquire/release（参照 sse_limiter.go:44-63 形态）。

## S2 metric（同文件，关 G-POST2-2）

参照 `internal/connect/strategy/metrics.go` promauto 形态：

- `ant_stream_active_streams{key_type="user|ip|anon"}` GaugeVec——acquire Inc/release Dec。
- `ant_stream_limit_rejections_total{key_type}` Counter——拒绝时 Inc。

## S3 接线——withSency 收口（禁逐调用点改）

`handlers.go:43` `withSency` 是全 55 个 handler 注册的唯一收口。改造：

```go
// streamLimit 为包级 var（cmd/server），registerHandlers 启动时赋值为
// interceptor.NewStreamLimitInterceptor(10, 30)；withSency 链尾追加（nil 安全跳过），
// 保证跑在 auth/admin/ratelimit 之后——GetUserID/GetClientIP 已填充。
func withSency(interceptors ...connectrpc.Interceptor) connectrpc.Option {
    chain := append([]connectrpc.Interceptor{alphasentry.NewErrorInterceptor()}, interceptors...)
    if streamLimit != nil { chain = append(chain, streamLimit) }
    return connectrpc.WithInterceptors(chain...)
}
```

- `main.go:99` 附近构造 interceptor，经 handlerDeps/interceptorSet 传入 registerHandlers 后赋包级 var（或在 registerHandlers 内直接构造——取改动最小者）。
- 55 个调用点**零改动**（全形态覆盖：含无 auth 的 public handler→键落 IP，正确）。

## S4 既有 middleware 键修复（小改）

`sse_limiter.go` `sseOwnerKey`：context 值在 middleware 层永不填充→改为直接读 header——X-Real-IP 优先、XFF 兜底（`ratelimit.go:115-127` 已有同形态先例+理由注释：XFF 可被客户端伪造）。修后 text/event-stream 形态恢复"per-IP=5"设计本意。

## T 测试（`stream_limit_test.go`）

- **T1 容量门**：cap=2，用阻塞 fake `StreamingHandlerFunc`（等 channel）持槽→第 3 并发调用确定性拒绝 `CodeResourceExhausted`；首个释放后新调用成功。
- **T2 键类**：ctx 置 `UserIDKey`→user 桶计数；无→IP 桶（ctx 置 `ClientIPKey`）；皆无→anon。
- **T3 mutation**（审计方将独立复验）：删 acquire 检查→T1 RED。
- 阻塞用 channel 精确控制时序，**禁 sleep 概率测试**（SCHEDULE-HOTLOOP-1a 先例）。

## 边界

- 不动 55 个调用点；不动 unary；不引入新中间件层。
- 不部署不 push；`--no-verify` 禁。

## 门禁

```bash
cd backend && go build ./... && go vet ./... && go test ./internal/interceptor/... -count=1 && go test -race ./internal/interceptor/... -count=1 && go run ./tools/check-file-lines --strict && git diff --check
```

串行，完成报证据等复审。
