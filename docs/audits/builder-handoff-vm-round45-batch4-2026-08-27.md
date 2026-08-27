# Builder Handoff: QUOTE-RECONNECT-LOOP + BROKER-SEARCH-1（Batch 4）

> **设计/验收方**：Devin CLI
> **施工方**：Devin IDE / Windsurf
> **基线 HEAD**：Batch 3 验收后开工（可与 Batch 1-3 并行，无代码耦合）
> **边界**：只施工这 2 个 ID 的修复，禁改写历史审计事实，禁扩 scope，禁 commit/push/deploy。
> **施工后状态**：`🟦open（施工完成，待独立复审）`，不得自标 ✅done。

---

## 立项背景

### QUOTE-RECONNECT-LOOP（P1 报价稳定性）

**根因**：`connection.go:186-229` `Disconnect` 调 `time.Sleep(200ms)` 后杀全 session——`cancelSub`/`cancelProfitSub`/`cancelOrderUpdateSub`/`cancelHubOrderSub` 全部 cancel + `g.conn.Close()`。后果：
1. quote stream silence timeout 时 `recvQuoteFrame`（`quotes.go:187-215`）只 cancel subCtx 重试 stream，**不调 Disconnect**——这部分已正确。
2. 但 `ensureConnected`（`:231-262`）在 `Connect` 失败时返回 error，`recvLoop`（`:124`）收到 error **直接 return**——整个 quote loop 退出，不再重试。profit/order loop 同理。
3. `Disconnect` 杀全 session 影响其他 stream——如果任何路径调 Disconnect 重建 quote，profit/order stream 也被杀。

**核心问题**：`recvLoop`/`profitRecvLoop`/`orderEventLoop` 三个独立 loop 共享一个 `g.conn`，任一 loop 的 `ensureConnected` 失败让 loop 退出后不再自愈。`Disconnect` 杀全 session 是核弹级。

### BROKER-SEARCH-1（P1 报价配置）

**根因**：`brokersearch/search.go:53-61` `New(mt4Gateway, mt5Gateway)` 空字符串 fallback 到硬编码 `mt4grpc3.mtapi.io:443`。3 处生产调用点（`cmd/server/pipeline.go:71`、`cmd/server/handlers.go:67`、`internal/mdgateway/runner.go` 间接）全部传空字符串。配置未接线——环境变量/config 文件无 mtapi host 字段。

---

## 🔴 绝对边界

1. **只改** `internal/mdgateway/adapter/mt4/connection.go` + `internal/mdgateway/adapter/mt5/connection.go` + `internal/mdgateway/adapter/mt4/quotes.go` + `internal/mdgateway/adapter/mt5/quotes.go`（如存在）+ `internal/mdgateway/adapter/brokersearch/search.go` + `cmd/server/pipeline.go` + `cmd/server/handlers.go` + 新建测试文件 + 文档。
2. **禁止删/改** `recvQuoteFrame` 的 silence timeout 逻辑（已正确，不调 Disconnect）。
3. **禁止改** profit/order stream 的业务逻辑（只改 reconnect 行为）。
4. 禁止改 proto / DB schema / 部署。
5. 禁止 commit / push / deploy。禁 `--no-verify`。

---

## 施工步骤

### QUOTE-RECONNECT-LOOP

- **S1** `connection.go` `ensureConnected`：`Connect` 失败时**不返回 error 让 loop 退出**，改为 log.Warn + sleep(backoff) + `return nil`（loop 继续，下次迭代重试）。只有 `ctx.Done()` 才让 loop 退出。MT4 和 MT5 同步修改。

- **S2** `connection.go` `Disconnect`：删除 `time.Sleep(200 * time.Millisecond)`（drain grace period 改用 context cancellation 等待，不阻塞）。或改为 `select { case <-time.After(200ms): case <-ctx.Done(): }` 可取消。

- **S3** `quotes.go` `recvLoop`：`ensureConnected` 返回 nil 后继续循环（不退出）。`ensureConnected` 返回 error 只在 `ctx.Err() != nil` 时（context cancelled）。MT4 和 MT5 同步。

- **S4** `profit.go` `profitRecvLoop` + `order_stream.go` `orderUpdateRecvLoop`：同样改为 `ensureConnected` 失败不退出 loop（除非 ctx cancelled）。

- **S5** 新增 `gatewayHealth` 状态跟踪——`g.health` 字段记录 last quote/profit/order recv 时间，`reportStatus` 在 loop 退出时记 "disconnected" 而非 "connected"。可选：health check endpoint 暴露。

### BROKER-SEARCH-1

- **S6** `brokersearch/search.go` `New`：保留空字符串 fallback（向后兼容），但新增 `NewFromConfig(mt4Gateway, mt5Gateway string) *Searcher` 显式从配置读取。

- **S7** `cmd/server/pipeline.go` + `cmd/server/handlers.go`：从环境变量或 config 读取 `MTAPI_MT4_HOST` / `MTAPI_MT5_HOST`，传入 `brokersearch.New`。空值时仍 fallback 到硬编码（不破坏现有部署）。

- **S8** 配置文档：在 `docs/constraints.md` 或 `docs/runbook/` 记录 `MTAPI_MT4_HOST` / `MTAPI_MT5_HOST` 环境变量。

---

## 测试与对抗证明

### QUOTE-RECONNECT-LOOP 测试

- **T1** `TestRecvLoop_RetriesAfterConnectFailure`：mock Connect 返回 error → recvLoop 不退出（继续迭代）→ 第二次 Connect 成功 → quote 恢复。断言 loop goroutine 在 ctx cancel 前不退出。
- **T2** `TestEnsureConnected_ReturnsNilOnConnectError`：mock Connect error → ensureConnected 返回 nil（非 error）→ caller 继续。
- **T3** `TestDisconnect_DoesNotBlockOnSleep`：Disconnect 在 cancelled ctx 上 <50ms 返回（不 sleep 200ms）。
- **T4** `TestProfitLoop_RetriesAfterConnectFailure`：同 T1 但测 profitRecvLoop。
- **T5** `TestOrderLoop_RetriesAfterConnectFailure`：同 T1 但测 orderUpdateRecvLoop。

### BROKER-SEARCH-1 测试

- **T6** `TestBrokerSearch_NewFromConfig_UsesProvidedHosts`：`NewFromConfig("custom.mtapi.io:443", "custom2.mtapi.io:443")` → Searcher.mt4Gateway == "custom.mtapi.io:443"。
- **T7** `TestBrokerSearch_New_EmptyFallbackToDefault`：`New("", "")` → mt4Gateway == "mt4grpc3.mtapi.io:443"（向后兼容）。
- **T8** `TestPipeline_ReadsMtapiHostFromEnv`：set `MTAPI_MT4_HOST=env.mtapi.io:443` → pipeline 创建的 Searcher 用 env host。

### 对抗证明

- **P1**：revert `ensureConnected` 为返回 error → T1 RED（loop 退出）→ 恢复 → GREEN。
- **P2**：revert `Disconnect` 为 `time.Sleep(200ms)` → T3 RED（>50ms）→ 恢复 → GREEN。
- **P3**：revert `NewFromConfig` 为忽略参数 → T6 RED（host 不变）→ 恢复 → GREEN。

---

## 红队自审

1. `ensureConnected` 不返回 error 后，是否有路径依赖 error 来停止 loop？（只有 ctx.Done 应停止。）
2. `Disconnect` 删除 sleep 后，in-flight Recv() 是否会被 race？（context cancellation 应让 Recv 立即返回。）
3. 三个 loop 独立重试是否会导致三个并发 Connect？（`beginConnect` single-flight 已防止。）
4. 环境变量 fallback 到硬编码是否安全？（是——不破坏现有部署，新部署可配置。）
5. `NewFromConfig` vs `New` 命名是否清晰？（`New` 保留向后兼容，`NewFromConfig` 显式。）

---

## 验收门禁

```
gofmt -l <改动文件>
go build ./...
go vet ./internal/mdgateway/...
go test ./internal/mdgateway/... -count=1
go test -race ./internal/mdgateway/... -count=1  # 连跑 3 次
go test ./internal/connect/... -count=1  # 确认无回归
go run ./tools/check-file-lines --strict
git diff --check
```

---

## 回填与收尾

registry 本条回填 + `handover-audit-plan.md` 追加一行。**状态填 `🟦open（施工完成，待独立复审）`。**

> **勿部署、勿 push、停手等 Devin CLI 复审。禁止 `--no-verify`。**
