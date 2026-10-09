# FILE-SPLIT 超限文件折分计划（多批次派工单，Devin CLI 设计 SSOT 2026-10-09）

## 立项背景

`check-file-lines --strict` 实拍（commit `017a66d7`）：🟡 警告 57 文件（源文件 >360 行×40 + 测试文件 >599 行×17），🔴 0。业主令「超限的文件要折分」。目标：57 文件全部回到各自 limit 内（src ≤300 / test ≤450），新增文件自身不得逼近 limit。

## 通用拆纪律（每批适用，违则退回）

1. **纯 verbatim 搬运**：同包拆兄弟文件，代码零语义改动——不改名、不改签名、不顺手重构、不改注释语义；文件级头注按拆分职责改写或保留。
2. **命名**：`<stem>_<责任>.go`（src）/`<stem>_<域>_test.go`（test）；共享测试 fixture/helper → 就近既有 helper 文件或 `<stem>_helpers_test.go`。
3. **拆分前**先列每文件切割方案（哪些 func/Test* 去哪个新文件）写入回报；切割轴=职责/域（如 publish.go 拆 pricing/validation/lifecycle），禁止机械腰斩。
4. **测试守恒**：拆分前后 `go test -list '.*' ./<pkg>` 输出的 Test* 清单须完全一致（一测不丢不增）；helpers 未被任何测试引用不得搬。
5. **import 最小化**：新文件只带自己需要的 import；拆完 `gofmt -l` 零输出。
6. **cap.sh 复用核对**：每个新文件 `bash scripts/cap.sh <核心词>`，回报标 REUSE:/NEW:。
7. **顺序约束**：同一 package 内多文件分批时按批序串行，禁止并行拆同包文件。

## 批次与文件清单

### FILE-SPLIT-T1 —— 测试巨件（5 文件，~7.0k 行）

| 文件 | 行/limit | 建议切割轴 |
|---|---|---|
| `internal/mthub/service_coverage_test.go` | 2052/450 | 按被测 service 域分 3-4 文件 |
| `internal/connect/strategy/mutation_coordinator_test.go` | 1651/450 | 按 mutation 种类（open/close/modify/cancel/barrier） |
| `internal/mdgateway/adapter/mt4/mt4_test.go` | 1285/450 | 按 adapter 功能区 |
| `internal/mdgateway/pure_test.go` | 1058/450 | 按纯函数域 |
| `internal/mdgateway/adapter/mt5/mt5_test.go` | 1013/450 | 按 adapter 功能区 |

### FILE-SPLIT-T2 —— 测试中件（12 文件，~8.9k 行）

| 文件 | 行/limit |
|---|---|
| `internal/sweep/sweep_test.go` | 884/450 |
| `internal/connect/strategy/schedule_hotloop_test.go` | 815/450 |
| `internal/connect/strategy/live_diag_truth_test.go` | 814/450 |
| `internal/connect/marketplace/marketplace_test.go` | 792/450 |
| `internal/mthub/service_orders_unit_test.go` | 766/450 |
| `internal/connect/system/mthub_service_integration_test.go` | 745/450 |
| `internal/marketplace/money_flow_integration_test.go` | 743/450 |
| `internal/connect/strategy/schedule_event_test.go` | 690/450 |
| `internal/execalgo/algo_test.go` | 656/450 |
| `internal/repository/trade_record_global_verify_integration_test.go` | 638/450 |
| `internal/risk/gate_test.go` | 637/450 |
| `internal/connect/strategy/trade_fields_invariant_test.go` | 606/450 |

### FILE-SPLIT-S1 —— 源文件第一梯队 ≥430（11 文件）

| 文件 | 行/limit |
|---|---|
| `internal/marketplace/publish.go` | 450/300 |
| `internal/connect/strategy/strategy_schedules.go` | 450/300 |
| `cmd/server/pipeline.go` | 450/300 |
| `internal/sweep/worker.go` | 448/300 |
| `internal/mthub/service_orders.go` | 448/300 |
| `internal/connect/strategy/trade_barrier.go` | 448/300 |
| `internal/connect/strategy/vm_live_handlers.go` | 445/300 |
| `cmd/coldsign-gui/main.go` | 445/300 |
| `internal/connect/strategy/backtest_worker_vm.go` | 441/300 |
| `internal/marketplace/live_performance.go` | 431/300 |
| `internal/service/systemai/chat.go` | 430/300 |

### FILE-SPLIT-S2 —— 源文件第二梯队 401-426（12 文件）

`strategy_execution_handler.go` 426 · `marketplace/strategy_optimizer.go` 421 · `connect/strategy/session_registry.go` 419 · `connect/strategy/live_runner.go` 416 · `connect/strategy/live_context.go` 413 · `chain/tron_grid.go` 413 · `mthub/service.go` 412 · `repository/wallet_repo.go` 410 · `marketplace/quality.go` 409 · `repository/ai_gateway_repository.go` 407 · `connect/strategy/strategy_experiment_worker.go` 405 · `connect/gateway/ai_gateway_handler.go` 401

### FILE-SPLIT-S3 —— 源文件第三梯队 362-398（17 文件）

`marketplace/service_subscription.go` 398 · `service/subscription_service.go` 387 · `risk/rules.go` 385 · `mthub/broker_types.go` 378 · `chain/monitor.go` 376 · `connect/ai/code_assist_handler.go` 375 · `mdgateway/adapter/mt4/profit.go` 373 · `connect/strategy/mutation_coordinator.go` 373 · `mdgateway/adapter/mt5/connection.go` 372 · `marketplace/decay_detector.go` 372 · `mdgateway/adapter/mt4/orders.go` 369 · `knowledgebase/service.go` 367 · `connect/strategy/schedule_execute.go` 366 · `mdgateway/adapter/mt4/connection.go` 365 · `sweep/builder.go` 364 · `connect/user/share_service.go` 363 · `service/template_svc.go` 362

## 每批门禁（统一）

- `go build ./...`、`go vet ./...` 0 错
- **integration tag 可编译守恒**（T2 复审补入）：`go build -tags integration ./...` + `go vet -tags integration ./<触及包>` 0 错——`go test -list` 默认不带 tag，不编译 `//go:build integration` 文件，拆分层叠加/残余文件重复声明会逃逸默认守恒门禁（connect/system 实证）
- `go test -count=1` 每触及包：拆分前后 Test* 清单与 PASS 数一致（输出贴回报）
- `go test -race` 触及并发包（connect/strategy、mthub、mdgateway、sweep、runner 相关）
- `check-file-lines --strict`：本批触及路径全部移出 🟡 清单，且不产生新 🟡/🔴
- `gofmt -l` 触碰文件零、`git diff --check` 零
- mutation 等价抽查：任选 1 个被搬函数删 1 行实质逻辑→其测试须 RED→恢复 GREEN（证明搬过去的测试仍链接着被搬代码，非死代码）

## 坐标漂移条款

各批施工会使行数漂移——本清单为 `017a66d7` 基线快照；每批发单前复审方复核当批文件仍超限，若某文件已被前批收敛则剔除该批。

## 派工节奏

顺序：TEST-DSN-ENV-1 → FILE-SPLIT-T1 → T2 → S1 → S2 → S3。每批验收 ✅ 后发下一批开工指令。勿部署。
