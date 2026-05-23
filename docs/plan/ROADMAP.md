# ROADMAP（v2 · MT 重写）

> **状态**：M7-rewrite 进行中
> **路径**：路线 B（地基重做 + 业务渐进重构）
> **执行方**：AI Agent（人类不写代码，仅评审 + 前端验收）
>
> **AI 必读**：每张卡片必须严格按 §0 流程执行；任何步骤违反 → 卡片自动作废。

---

## 0. 卡片执行协议（强制）

### 0.1 状态符号

| 符号 | 含义 |
|---|---|
| 🅒 | 待开工 / 当前进行中 / 阻塞 |
| ☑ | 已完成（commit + 验收 log + 自检全过）|
| ⊘ | 已废弃（在备注列写明替代卡片）|

### 0.2 一张卡片 = 一个 PR

每张卡片必有：
1. **唯一 ID**：`M7.<minor>-<seq>`
2. **完整路径**：在卡片表"文件"列列出**所有**改动绝对路径
3. **验收命令**：在"验收"列给可机械执行的 shell（多行用 ```bash 块）
4. **commit short-sha**：完成后填入"备注"
5. **handover log**：`docs/handover/verify-M7.<minor>-<seq>.log` ≥ 20 行真实 stdout

### 0.3 执行顺序

**严格按本文件从上到下顺序**。前置卡片未 ☑，后续禁止开工。
跨 milestone 不允许。

---

## M7-rewrite 总览

| Sub-milestone | 内容 | 卡片数 | 预估工日 |
|---|---|---|---|
| **M7.0** | 准备：归档 v1、容器、依赖 | 6 | 2 |
| **M7.1** | mdgateway 完整重做（含 adapter） | 18 | 7 |
| **M7.2** | mthub + ConnectRPC 暴露 | 9 | 4 |
| **M7.3** | factorsvc + DSL（移植 alfq） | 7 | 3 |
| **M7.4** | quantengine + ONNX | 6 | 3 |
| **M7.5** | 业务层切流 + 老 kline_service 标 deprecated | 8 | 3 |
| **M7.6** | 端到端测试 + chaos | 6 | 3 |
| **M7.Z** | 关闭：硬指标全过 + ROADMAP 状态更新 | 3 | 1 |
| **总计** | | **63** | **~26 工日** |

---

## M7.0 · 准备工作

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M7.0-1 | 🅒 备份 v1 配置；docker-compose 加 ant-clickhouse + ant-nats | `docker-compose.yml`、`.env.example`、`deploy/clickhouse/config.d/`、`deploy/nats/nats.conf` | `docker compose up -d ant-clickhouse ant-nats; docker inspect -f '{{.State.Health.Status}}' ant-clickhouse \| grep -q healthy; docker exec ant-nats nats account info \|\| true` | |
| M7.0-2 | 🅒 `.env.example` 增加 CH/NATS 必备变量 | `.env.example` | `grep -E '^(CH_HOST\|CH_PORT\|CH_USER\|CH_PASSWORD\|CH_DATABASE\|NATS_URL)=' .env.example \| wc -l \| grep -q '^6$'` | |
| M7.0-3 | 🅒 `backend/go.mod` 添加 `github.com/ClickHouse/clickhouse-go/v2` `github.com/nats-io/nats.go` `github.com/cespare/xxhash/v2` `github.com/hashicorp/golang-lru/v2` | `backend/go.mod` `backend/go.sum` | `cd backend && go mod tidy && go build ./...` | |
| M7.0-4 | 🅒 `internal/storage/clickhouse/client.go` 包含 Connect/Ping/PrepareBatch | 同左 + `*_test.go` | `cd backend && go test ./internal/storage/clickhouse/...` | |
| M7.0-5 | 🅒 `internal/storage/nats/client.go` 包含 Connect + JetStream + ensureStream(MD_EVENTS) | 同左 + `*_test.go` | `cd backend && go test ./internal/storage/nats/...` | |
| M7.0-6 | 🅒 Makefile 加 `migrate-ch` `verify-card` 目标 | `Makefile` | `make help \| grep -E 'migrate-ch\|verify-card'` | |

---

## M7.1 · mdgateway 完整重做

> **前置必读**：`docs/architecture/02-overview.md` `docs/spec/10-mt-adapter.md` `docs/spec/11-mdgateway.md` `docs/spec/13-clickhouse-schema.md` `docs/spec/16-mtapi-quirks-register.md` `docs/adr/0003` `0004` `0005`

### M7.1.A 基础设施

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-1 | 🅒 chmigrate：实现 + 5 张表 SQL | `backend/internal/mdgateway/chmigrate/{migrate.go,001_md_ticks.sql,002_md_bars.sql,003_factor_values.sql,004_signals.sql,005_schema_version.sql}` + `migrate_test.go` | `make migrate-ch && docker exec ant-clickhouse clickhouse-client --query "SELECT count() FROM system.tables WHERE database='ant'" \| grep -q '^[5-9]'` |
| M7.1-2 | 🅒 PG migrations：`mt_accounts` v2 字段 + `broker_symbols` + `factor_definitions` | `backend/migrations/0XX_mt_accounts_v2.up.sql` `0XX_broker_symbols.up.sql` `0XX_factor_definitions.up.sql` 及对应 .down.sql | `make migrate; docker exec ant-postgres psql -U ant -d ant -tAc "SELECT column_name FROM information_schema.columns WHERE table_name='mt_accounts' AND column_name='mtapi_token_encrypted'" \| grep -q .` |
| M7.1-3 | 🅒 sqlc 生成 `BrokerSymbols` `FactorDefinitions` queries | `backend/internal/repository/queries/{broker_symbols.sql,factor_definitions.sql}` + 生成代码 | `cd backend && make sqlc && go build ./internal/repository/...` |

### M7.1.B 类型与共享

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-4 | 🅒 `adapter/mdtick/` 共享 DTO（Tick/Bar/Money/AccountConfig）+ Normalizer 接口 | `backend/internal/mdgateway/adapter/mdtick/{mdtick.go,normalizer.go,*_test.go}` | `cd backend && go build ./internal/mdgateway/adapter/mdtick/... && go test ./internal/mdgateway/adapter/mdtick/...` |
| M7.1-5 | 🅒 mdgateway types.go：Gateway 接口、Manager 类型 | `backend/internal/mdgateway/{types.go,types_test.go}` | `cd backend && go build ./internal/mdgateway/` |

### M7.1.C 子组件（每个独立卡片）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-6 | 🅒 normalizer.go（cache + PG + algorithmic fallback） | `backend/internal/mdgateway/{normalizer.go,normalizer_test.go}` | `go test -run TestNormalizer ./internal/mdgateway/ -count=1 -v` 覆盖率 ≥ 80% |
| M7.1-7 | 🅒 quality.go（bid>ask + 5σ MAD + gap + skew，含 dropped_reason label） | `backend/internal/mdgateway/{quality.go,quality_test.go,metrics.go}` | `go test -run TestQuality ./internal/mdgateway/ -v` 覆盖 4 类规则 |
| M7.1-8 | 🅒 tick_dedup.go（100-window xxhash） | `backend/internal/mdgateway/{tick_dedup.go,tick_dedup_test.go}` | `go test -run TestTickDedup ./internal/mdgateway/ -v` |
| M7.1-9 | 🅒 bar_aggregator.go（用 ArrivedUnixMs 分桶；6 周期）| `backend/internal/mdgateway/{bar_aggregator.go,bar_aggregator_test.go}` | 单测含 `Q-001` 重现用例 |
| M7.1-10 | 🅒 publisher.go（NATS JetStream md.tick.* / md.bar.*） | `backend/internal/mdgateway/{publisher.go,publisher_test.go}` | 启动 ant-nats 后单测 publish 真实订阅 |
| M7.1-11 | 🅒 clickhouse_writer.go（chan + batch + spill fallback） | `backend/internal/mdgateway/{clickhouse_writer.go,clickhouse_writer_test.go}` | dockertest CH 真写 + 模拟 CH down → spill 路径 |
| M7.1-12 | 🅒 spill_writer.go + spill_replay.go（jsonl + 旋转 + 启动 replay） | `backend/internal/mdgateway/{spill_writer.go,spill_writer_test.go,spill_replay.go,spill_replay_test.go}` | 单测旋转 by size + by age + replay |
| M7.1-13 | 🅒 circuit_breaker.go（滑动窗口） | `backend/internal/mdgateway/{circuit_breaker.go,circuit_breaker_test.go}` | 单测 closed→open→half_open→closed 全状态机 |
| M7.1-14 | 🅒 manager.go（HandleTick 串接子组件 + Add/Remove/Health） | `backend/internal/mdgateway/{manager.go,manager_test.go}` | 集成测试：单 mock gateway 推 100 tick → CH 100 行 |

### M7.1.D adapter

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-15 | 🅒 `adapter/mt4/gateway.go`（Connect/Subscribe/Disconnect/HealthCheck，引用 Q-001/Q-002/Q-008） | `backend/internal/mdgateway/adapter/mt4/{gateway.go,gateway_test.go}` | LOC ≤ 100；引用 Q-001 Q-002 Q-008 三处 quirk |
| M7.1-16 | 🅒 `adapter/mt5/gateway.go` | `backend/internal/mdgateway/adapter/mt5/{gateway.go,gateway_test.go}` | LOC ≤ 100 |
| M7.1-17 | 🅒 `adapter/mt[45]/executor.go`（PlaceOrder/CloseOrder/Fetch*） | 同左 | dockertest mtapi mock 调通全部 RPC |
| M7.1-18 | 🅒 runner.go（PG 加载账户 → AddGateway → SpillReplay 启动）| `backend/internal/mdgateway/{runner.go,runner_test.go}` | 启动 server → 至少 1 账户 connected metric |

### M7.1.Z 关闭检查

```bash
# (1) 全部文件齐全
for f in manager.go runner.go normalizer.go quality.go tick_dedup.go \
         bar_aggregator.go publisher.go clickhouse_writer.go \
         spill_writer.go spill_replay.go circuit_breaker.go \
         metrics.go types.go; do
  test -f "backend/internal/mdgateway/$f"
done
test -d backend/internal/mdgateway/adapter/{mdtick,mt4,mt5}

# (2) LOC
LOC=$(find backend/internal/mdgateway -name "*.go" -not -name "*_test.go" | xargs wc -l | tail -1 | awk '{print $1}')
test "$LOC" -le 800

# (3) 编译 + 测试
cd backend && go build ./internal/mdgateway/... && go test -race -cover ./internal/mdgateway/...

# (4) E2E：5 分钟内 ≥ 1000 tick 入 CH
sleep 300
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT count() FROM ant.md_ticks WHERE arrived_unix_ms > now64()*1000 - 300000" \
  | awk '$1<1000{exit 1}'
```

---

## M7.2 · mthub + ConnectRPC

> **前置必读**：`docs/spec/12-mthub.md` `docs/spec/14-rpc-contracts.md`

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.2-1 | 🅒 `proto/ant/v1/mthub_service.proto` 9 个 RPC | 同左 | `make proto-breaking` 通过 |
| M7.2-2 | 🅒 `proto/ant/v1/market_service.proto` 3 个 RPC | 同左 | 同上 |
| M7.2-3 | 🅒 `make proto` 生成 Go + TS stub | `backend/gen/proto/ant/v1/` `frontend/src/gen/ant/v1/` | `test -f backend/gen/proto/ant/v1/mthub_service.pb.go && test -f frontend/src/gen/ant/v1/mthub_service_pb.ts` |
| M7.2-4 | 🅒 `internal/mthub/{types.go,executor.go,session.go,hub.go,events.go,service.go,metrics.go}` | 同左 + 全部 _test.go | LOC ≤ 600；test cover ≥ 60% |
| M7.2-5 | 🅒 `internal/connect/mthub_handler.go` 9 个 handler 实现 | 同左 + `mthub_handler_test.go` | grpcurl 真实调通 PlaceOrder（dockertest mtapi mock） |
| M7.2-6 | 🅒 `internal/connect/market_handler.go` 3 个 handler（GetKlines 走 CH） | 同左 + test | grpcurl 调通 GetKlines 返回 CH 数据 |
| M7.2-7 | 🅒 SSE：StreamOrderEvents handler | `internal/connect/mthub_handler.go` 内 | curl SSE 5s 内收到至少 1 个事件（dockertest 触发）|
| M7.2-8 | 🅒 frontend client wrapper：`frontend/src/api/mthub.ts` `frontend/src/api/market.ts` | 同左 | `cd frontend && pnpm tsc --noEmit` |
| M7.2-9 | 🅒 mthub 注入 mdgateway runner（共享 session） | `internal/mdgateway/runner.go` + `internal/mthub/hub.go` | runner 启动后 hub.Get(accountID) != nil |

---

## M7.3 · factorsvc + DSL

> **前置必读**：`docs/spec/11-mdgateway.md` §"factorsvc 集成"

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.3-1 | 🅒 移植 alfq factor/dsl（lex/parser/compile + 14 算子）| `backend/internal/factor/dsl/*.go` | `go test ./internal/factor/dsl/... -count=1` 全过；测试包含 100 表达式样本 |
| M7.3-2 | 🅒 factorsvc/window_buffer.go（ring buffer per key） | 同左 + test | ring buffer 满时旧值淘汰 |
| M7.3-3 | 🅒 factorsvc/engine.go（FactorDef 加载 + Eval） | 同左 + test | 注入 100 bars → eval 1 次 < 10ms |
| M7.3-4 | 🅒 factorsvc/factor_ch_writer.go（batch flush） | 同左 + test | dockertest CH 真写 |
| M7.3-5 | 🅒 factorsvc/subscriber.go（NATS md.bar.* → engine） | 同左 + test | 集成测试：发 bar → CH factor_values 出现 |
| M7.3-6 | 🅒 factorsvc/metrics.go | 同左 + test | `factor_eval_total` 等指标暴露 |
| M7.3-7 | 🅒 factorsvc 接入 server 启动钩子 | `cmd/ant-server/main.go` | 启动后 NATS subject `md.factor.>` 有消息 |

---

## M7.4 · quantengine + ONNX

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.4-1 | 🅒 `internal/quantengine/types.go`（Signal / ModelDef） | + test | LOC ≤ 100 |
| M7.4-2 | 🅒 ONNX runtime 集成（onnxruntime-go） + 模型加载 | `internal/quantengine/onnx.go` + test | mock 模型 load + infer 通过 |
| M7.4-3 | 🅒 DSL 信号规则评估器 | `internal/quantengine/dsl_signal.go` + test | DSL "ma20 > close" → buy 信号 |
| M7.4-4 | 🅒 subscriber.go（订阅 NATS md.factor.> 触发推理）| + test | 端到端：factor → signal → CH `signals` |
| M7.4-5 | 🅒 SignalRouter 接 oms（PG.signals 之外补 CH 审计） | `internal/oms/signal_router.go`（重构）+ test | 信号被 risk reject 时 CH `signals.rejected=1` |
| M7.4-6 | 🅒 quantengine 接入 server 启动 | `cmd/ant-server/main.go` | 启动后 quant_inference_total 计数 |

---

## M7.5 · 业务层切流 + 标 deprecated

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.5-1 | 🅒 `internal/connect/market_service.go` GetKlines 切到 CH（带 PG fallback） | 同左 + test | grpcurl 查近 1h K 线 90% 走 CH（log 计数）|
| M7.5-2 | 🅒 `connect/market_regime_service.go` 切流 | 同左 + test | 同上 |
| M7.5-3 | 🅒 `connect/backtest_dataset_service.go` 切流 | 同左 + test | 回测拉历史数据走 CH |
| M7.5-4 | 🅒 `connect/python_strategy_service.go` 切流 | 同左 + test | 沙箱拉历史走 CH |
| M7.5-5 | 🅒 `service/kline_service*.go` 全部加 `// Deprecated: see internal/mdgateway. To be removed in M9.` | 同左 7 个文件 | `grep -c '// Deprecated' backend/internal/service/kline_service*.go \| grep -q '^7$'` |
| M7.5-6 | 🅒 grep 验证业务代码 0 处直 import mt4client/mt5client（除 service/kline_service*）| — | `! grep -rE 'anttrader/internal/(mt4\|mt5)client' backend/internal/{ai,marketplace,oms,risk,connect,quantengine,factorsvc,mthub}/ \|\| exit 1` |
| M7.5-7 | 🅒 老 `kline_data` 表设为只读（trigger 阻止 INSERT） | `backend/migrations/0XX_kline_data_readonly.up.sql` | `psql -c "INSERT INTO kline_data ..."` 返回 error |
| M7.5-8 | 🅒 frontend K 线组件切到新 RPC `MarketService.GetKlines` | `frontend/src/api/kline.ts` 与相关组件 | 浏览器手动验收：K 线展示正常 |

---

## M7.6 · E2E 测试 + chaos

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.6-1 | 🅒 端到端 happy path：新建账户 → 订阅 → 5min 内 CH 有 tick/bar | `tests/e2e/mt_foundation_test.go` (build tag e2e) | `go test -tags=e2e ./tests/e2e/... -run TestHappyPath -timeout 10m` |
| M7.6-2 | 🅒 chaos：CH 60s 中断 → spill → recovery，零数据丢失 | `tests/chaos/ch_outage_test.go` (build tag chaos) | 对比 spill 期前后 tick 计数差 ≤ 1% |
| M7.6-3 | 🅒 chaos：单 broker 失联 → 其他账户照常 | `tests/chaos/broker_outage_test.go` | 故障账户 metric `md_circuit_state=1`，其他 ≠ 1 |
| M7.6-4 | 🅒 SSE 重连：客户端断开 30s → 重连后继续收事件 | `tests/e2e/sse_reconnect_test.go` | 重连后 5s 内收到事件 |
| M7.6-5 | 🅒 7 天稳定性运行（人工 + monitoring）：tick rate / drop / circuit / spill 全部健康 | `docs/handover/verify-M7.6-5.log` | Grafana 截图归档 + 指标证据 |
| M7.6-6 | 🅒 LOC 终检：mdgateway + adapter + mthub LOC | — | `LOC=$(find backend/internal/{mdgateway,mthub} -name "*.go" -not -name "*_test.go" \| xargs wc -l \| tail -1 \| awk '{print $1}'); test "$LOC" -le 1500` |

---

## M7.Z · 关闭

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.Z-1 | 🅒 跑 ADR-0001 §6 全部断言（4 条） | — | 全 0 退出 |
| M7.Z-2 | 🅒 更新 `docs/plan/ROADMAP.md` 状态为 ✅；`AGENT.md` §14 更新当前阶段 | 同左 | grep verify |
| M7.Z-3 | 🅒 写 `docs/handover/M7-closure.md`：里程碑总结 + 后续 M8/M9 输入 | 同左 | 人类 review |

---

## M8（预留）· 业务层渐进重构

> 留给 M7 完成后立项；目标：业务代码风格统一、sqlc 覆盖 ≥ 80%、所有 service/* ≤ 400 行
> 规划见 `docs/plan/BACKLOG.md` §M8

## M9（预留）· 老包删除

> 目标：删除 `internal/mt4client` `internal/mt5client`、删除 `kline_data` 表
> 触发条件：M7 + M8 完成且业务代码 30 天 0 直调

---

## 历史

- v2.0 (2026-05-23)：完全重写 ROADMAP；v1 归档至 `docs.old/plan/ROADMAP.md`
