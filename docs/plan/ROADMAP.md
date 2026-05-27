# ROADMAP（v2 · MT 重写）

> **状态**：M7 已完成。M10.5 补完进行中。**M10-BASE 将 M11 金融架构设计前置到基础层**，避免建完后拆掉重做。
> **实际执行结果等价于路线 A（全量重写）**，详见 `docs/adr/0007-post-m7-retrospective.md`。
> **设计输入**：
> - `docs/金融架构改造-M11路线图-2026-05-25.md`（D1 金融评审 + D2 量化考古合并版）
> - `docs/对标开源项目调研-2026-05-25.md`（NautilusTrader / Lean / Hummingbot / freqtrade / vnpy 横向对比，含每张卡片的开源参考实现路径）
> **当前阶段**：M10.5 补完 → M10-BASE 架构基础补全 → M11 纯增量金融特性。
> **执行方**：AI Agent + 人类验收。

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
| **M7.0** | 准备：容器、依赖、proto 重构、处置 v1、secrets | 9 | 3 |
| **M7.1** | mdgateway 完整重做（含 adapter） | 18 | 7 |
| **M7.2** | mthub + ConnectRPC 暴露 | 9 | 4 |
| **M7.3** | factorsvc + DSL（移植 alfq） | 7 | 3 |
| **M7.4** | quantengine + ONNX | 6 | 3 |
| **M7.5** | 业务层切流 + 老 kline_service 标 deprecated | 8 | 3 |
| **M7.6** | 端到端测试 + chaos + telemetry | 7 | 3 |
| **M7.Z** | 关闭：硬指标全过 + ROADMAP 状态更新 | 3 | 1 |
| **总计** | | **67** | **~27 工日** |

---

## M7.0 · 准备工作

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M7.0-1 | ☑ 备份 v1 配置；docker-compose 加 ant-clickhouse + ant-nats | `docker-compose.yml`、`.env.example`、`deploy/clickhouse/config.d/`、`deploy/nats/nats.conf` | `docker compose up -d ant-clickhouse ant-nats; docker inspect -f '{{.State.Health.Status}}' ant-clickhouse \| grep -q healthy; docker exec ant-nats nats account info \|\| true` | |
| M7.0-2 | ☑ `.env.example` 增加 CH/NATS 必备变量 | `.env.example` | `grep -E '^(CH_HOST\|CH_PORT\|CH_USER\|CH_PASSWORD\|CH_DATABASE\|NATS_URL)=' .env.example \| wc -l \| grep -q '^6$'` | |
| M7.0-3 | ☑ `backend/go.mod` 添加 `github.com/ClickHouse/clickhouse-go/v2` `github.com/nats-io/nats.go` `github.com/cespare/xxhash/v2` `github.com/hashicorp/golang-lru/v2` | `backend/go.mod` `backend/go.sum` | `cd backend && go mod tidy && go build ./...` | |
| M7.0-4 | ☑ `internal/storage/clickhouse/client.go` 包含 Connect/Ping/PrepareBatch | 同左 + `*_test.go` | `cd backend && go test ./internal/storage/clickhouse/...` | |
| M7.0-5 | ☑ `internal/storage/nats/client.go` 包含 Connect + JetStream + ensureStream(MD_EVENTS) | 同左 + `*_test.go` | `cd backend && go test ./internal/storage/nats/...` | |
| M7.0-6 | ☑ Makefile 加 `migrate-ch` `verify-card` 目标 | `Makefile` | `make help \| grep -E 'migrate-ch\|verify-card'` | |
| M7.0-7 | ☑ proto 布局重构：**仅** ant 自有平铺 `proto/*.proto` 迁到 `proto/ant/v1/*.proto`；**保留** `proto/mt4/` `proto/mt5/`（mtapi.io 第三方 proto，不许改）；buf.gen.yaml 调输出到 `backend/gen/proto/ant/v1/`；frontend client 同步；**附带**更新 `docs/spec/10-mt-adapter.md` §5.1/5.3 中 import 示例从 `anttrader/gen/proto/mt4` 改为 v2 路径 | `proto/ant/v1/` `buf.gen.yaml` `frontend/src/gen/ant/v1/` `docs/spec/10-mt-adapter.md` | `make proto && test -f backend/gen/proto/ant/v1/common.pb.go && test -f frontend/src/gen/ant/v1/common_pb.ts && test -d proto/mt4 && test -d proto/mt5 && git diff --exit-code -- backend/gen frontend/src/gen` | |
| M7.0-8 | ☑ 处置 v1 既存代码：删除 `backend/internal/adapter/{mt4,mt5}_adapter.go`（双 adapter 目录冲突）；标记 `backend/internal/{mdgateway,mthub,factorsvc,quantengine}` 现有 .go 为 v1（在文件头加 `// V1-LEGACY: will be replaced by M7.1-7.4 cards`，**不删除**——M7.1-7.4 卡片在写入新文件时若与 v1 同名则覆盖、不同名则在卡片"附带删除"列出 v1 文件清单） | `backend/internal/adapter/` (删除) + `backend/internal/{mdgateway,mthub,factorsvc,quantengine}/*.go` (加注释) | `! test -d backend/internal/adapter; for d in mdgateway mthub factorsvc quantengine; do grep -L 'V1-LEGACY' backend/internal/$d/*.go \| grep -v _test.go \| awk 'NR>0{exit 1}'; done` | M9 删除 v1 包时此注释作为筛选锚 |
| M7.0-9 | ☑ 实现 `internal/secrets/` (vault.Client AES-256-GCM 接口) — **M7.1-2 ETL 前置依赖** | `backend/internal/secrets/{vault.go,aes_gcm.go,vault_test.go,aes_gcm_test.go}` + `.env.example` 加 `ANT_MASTER_KEY=<base64-32B>` | `cd backend && go test -race -cover ./internal/secrets/... \| awk '/coverage:/{gsub("%",""); if ($2<90) exit 1}'; grep -q '^ANT_MASTER_KEY=' .env.example` | 见 spec/17 §1 |

---

## M7.1 · mdgateway 完整重做

> **前置必读**：`docs/architecture/02-overview.md` `docs/spec/10-mt-adapter.md` `docs/spec/11-mdgateway.md` `docs/spec/13-clickhouse-schema.md` `docs/spec/16-mtapi-quirks-register.md` `docs/adr/0003` `0004` `0005`

### M7.1.A 基础设施

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-1 | ☑ chmigrate：实现 + 5 张表 SQL | `backend/internal/mdgateway/chmigrate/{migrate.go,001_md_ticks.sql,002_md_bars.sql,003_factor_values.sql,004_signals.sql,005_schema_version.sql}` + `migrate_test.go` | `make migrate-ch && docker exec ant-clickhouse clickhouse-client --query "SELECT count() FROM system.tables WHERE database='ant'" \| grep -q '^[5-9]'` |
| M7.1-2 | ☑ PG migrations + ETL：`mt_accounts` v2 字段 + `mt_accounts_v2` 视图 + `broker_symbols` + 重建 `factor_definitions`（含 vault 加密 ETL）| `backend/migrations/098_mt_accounts_v2.up.sql` (`+ .down.sql`) `099_broker_symbols.up.sql` (`+.down`) `100_factor_definitions_v2.up.sql` (`+.down`，**第一行 `DROP TABLE IF EXISTS factor_definitions CASCADE;`** 删除 v1 096 的版本后重建；命名带 _v2 后缀避免与 096 文件名混淆) ；ETL 脚本 `backend/cmd/etl-mt-accounts/main.go` | `make migrate; docker exec ant-postgres psql -U ant -d ant -tAc "SELECT column_name FROM information_schema.columns WHERE table_name='mt_accounts' AND column_name='mtapi_token_encrypted'" \| grep -q .; docker exec ant-postgres psql -U ant -d ant -tAc "SELECT count(*) FROM mt_accounts_v2 WHERE password_encrypted IS NOT NULL" \| awk '$1>0 \|\| NR==0{exit 0} {exit 1}'; docker exec ant-postgres psql -U ant -d ant -tAc "SELECT column_name FROM information_schema.columns WHERE table_name='factor_definitions' AND column_name='canonicals'" \| grep -q .` |
| M7.1-3 | ☑ sqlc 生成 `BrokerSymbols` `FactorDefinitions` queries | `backend/internal/repository/queries/{broker_symbols.sql,factor_definitions.sql}` + 生成代码 | `cd backend && make sqlc && go build ./internal/repository/...` |

### M7.1.B 类型与共享

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-4 | ☑ `adapter/mdtick/` 共享 DTO（Tick/Bar/Money/AccountConfig）+ Normalizer 接口 | `backend/internal/mdgateway/adapter/mdtick/{mdtick.go,normalizer.go,*_test.go}` | `cd backend && go build ./internal/mdgateway/adapter/mdtick/... && go test ./internal/mdgateway/adapter/mdtick/...` |
| M7.1-5 | ☑ mdgateway types.go：Gateway 接口、Manager 类型 | `backend/internal/mdgateway/{types.go,types_test.go}` | `cd backend && go build ./internal/mdgateway/` |

### M7.1.C 子组件（每个独立卡片）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-6 | ☑ normalizer.go（cache + PG + algorithmic fallback） | `backend/internal/mdgateway/{normalizer.go,normalizer_test.go}` | `go test -run TestNormalizer ./internal/mdgateway/ -count=1 -v` 覆盖率 ≥ 80% |
| M7.1-7 | ☑ quality.go（bid>ask + 5σ MAD + gap + skew，含 dropped_reason label） | `backend/internal/mdgateway/{quality.go,quality_test.go,metrics.go}` | `go test -run TestQuality ./internal/mdgateway/ -v` 覆盖 4 类规则 |
| M7.1-8 | ☑ tick_dedup.go（100-window xxhash） | `backend/internal/mdgateway/{tick_dedup.go,tick_dedup_test.go}` | `go test -run TestTickDedup ./internal/mdgateway/ -v` |
| M7.1-9 | ☑ bar_aggregator.go（用 ArrivedUnixMs 分桶；6 周期）| `backend/internal/mdgateway/{bar_aggregator.go,bar_aggregator_test.go}` | 单测含 `Q-001` 重现用例 |
| M7.1-10 | ☑ publisher.go（NATS JetStream md.tick.* / md.bar.*） | `backend/internal/mdgateway/{publisher.go,publisher_test.go}` | 启动 ant-nats 后单测 publish 真实订阅 |
| M7.1-11 | ☑ clickhouse_writer.go（chan + batch + spill fallback） | `backend/internal/mdgateway/{clickhouse_writer.go,clickhouse_writer_test.go}` | dockertest CH 真写 + 模拟 CH down → spill 路径 |
| M7.1-12 | ☑ spill_writer.go + spill_replay.go（jsonl + 旋转 + 启动 replay） | `backend/internal/mdgateway/{spill_writer.go,spill_writer_test.go,spill_replay.go,spill_replay_test.go}` | 单测旋转 by size + by age + replay |
| M7.1-13 | ☑ circuit_breaker.go（滑动窗口） | `backend/internal/mdgateway/{circuit_breaker.go,circuit_breaker_test.go}` | 单测 closed→open→half_open→closed 全状态机 |
| M7.1-14 | ☑ manager.go（HandleTick 串接子组件 + Add/Remove/Health） | `backend/internal/mdgateway/{manager.go,manager_test.go}` | 集成测试：单 mock gateway 推 100 tick → CH 100 行 |

### M7.1.D adapter

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.1-15 | ☑ `adapter/mt4/gateway.go`（Connect/Subscribe/Disconnect/HealthCheck，引用 Q-001/Q-002/Q-008） | `backend/internal/mdgateway/adapter/mt4/{gateway.go,gateway_test.go}` | LOC ≤ 100；引用 Q-001 Q-002 Q-008 三处 quirk |
| M7.1-16 | ☑ `adapter/mt5/gateway.go` | `backend/internal/mdgateway/adapter/mt5/{gateway.go,gateway_test.go}` | LOC ≤ 100 |
| M7.1-17 | ☑ `adapter/mt[45]/executor.go`（PlaceOrder/CloseOrder/Fetch*） | 同左 | dockertest mtapi mock 调通全部 RPC |
| M7.1-18 | ☑ runner.go（PG 加载账户 → AddGateway → SpillReplay 启动）| `backend/internal/mdgateway/{runner.go,runner_test.go}` | 启动 server → 至少 1 账户 connected metric |

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
| M7.2-1 | ☑ `proto/ant/v1/mthub_service.proto` 8 个 RPC（含 GetAccountStatus 替代原 livez/account）| 同左 | `make proto-breaking` 通过 + `grep -c '^  rpc ' proto/ant/v1/mthub_service.proto \| grep -q '^8$'` |
| M7.2-2 | ☑ `proto/ant/v1/market_service.proto` 3 个 RPC | 同左 | 同上 |
| M7.2-3 | ☑ `make proto` 生成 Go + TS stub | `backend/gen/proto/ant/v1/` `frontend/src/gen/ant/v1/` | `test -f backend/gen/proto/ant/v1/mthub_service.pb.go && test -f frontend/src/gen/ant/v1/mthub_service_pb.ts` |
| M7.2-4 | ☑ `internal/mthub/{types.go,executor.go,session.go,hub.go,events.go,service.go,metrics.go}` | 同左 + 全部 _test.go | LOC ≤ 300（非测试，与 spec/12 §1 一致）；test cover ≥ 60% |
| M7.2-5 | ☑ `internal/connect/mthub_service.go` 8 个 handler 实现（PlaceOrder/CloseOrder/OpenedOrders/OrderHistory/SymbolParams/PriceHistory/GetAccountStatus/StreamOrderEvents）| 同左 + `mthub_service_test.go` | grpcurl 真实调通 PlaceOrder（dockertest mtapi mock） + `GetAccountStatus` 返 connected |
| M7.2-6 | ☑ `internal/connect/market_handler.go` 3 个 handler（GetKlines 走 CH） | 同左 + test | grpcurl 调通 GetKlines 返回 CH 数据 |
| M7.2-7 | ☑ SSE：StreamOrderEvents handler | `internal/connect/mthub_service.go` 内 | curl SSE 5s 内收到至少 1 个事件（dockertest 触发）|
| M7.2-8 | ⊘ frontend client wrapper（旧前端已删除；TS stub 仍生成但无消费方；M11 重新评估）| 同左 | M11 立项后重新评估 |
| M7.2-9 | ☑ mthub 注入 mdgateway runner（共享 session） | `internal/mdgateway/runner.go` + `internal/mthub/hub.go` | runner 启动后 hub.Get(accountID) != nil |

---

## M7.3 · factorsvc + DSL

> **前置必读**：`docs/spec/11-mdgateway.md` §"factorsvc 集成"

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.3-1 | ☑ 移植 alfq factor/dsl（lex/parser/compile + 14 算子）| `backend/internal/factor/dsl/*.go` | `go test ./internal/factor/dsl/... -count=1` 全过；测试包含 100 表达式样本 |
| M7.3-2 | ☑ factorsvc/window_buffer.go（ring buffer per key） | 同左 + test | ring buffer 满时旧值淘汰 |
| M7.3-3 | ☑ factorsvc/engine.go（FactorDef 加载 + Eval） | 同左 + test | 注入 100 bars → eval 1 次 < 10ms |
| M7.3-4 | ☑ factorsvc/factor_ch_writer.go（batch flush） | 同左 + test | dockertest CH 真写 |
| M7.3-5 | ☑ factorsvc/subscriber.go（NATS md.bar.* → engine） | 同左 + test | 集成测试：发 bar → CH factor_values 出现 |
| M7.3-6 | ☑ factorsvc/metrics.go | 同左 + test | `factor_eval_total` 等指标暴露 |
| M7.3-7 | ☑ factorsvc 接入 server 启动钩子 | `cmd/ant-server/main.go` | 启动后 NATS subject `md.factor.>` 有消息 |

---

## M7.4 · quantengine + ONNX

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.4-1 | ☑ `internal/quantengine/types.go`（Signal / ModelDef） | + test | LOC ≤ 100 |
| M7.4-2 | ☑ ONNX runtime 集成（onnxruntime-go） + 模型加载 | `internal/quantengine/onnx.go` + test | mock 模型 load + infer 通过 |
| M7.4-3 | ☑ DSL 信号规则评估器 | `internal/quantengine/dsl_signal.go` + test | DSL "ma20 > close" → buy 信号 |
| M7.4-4 | ☑ subscriber.go（订阅 NATS md.factor.> 触发推理）| + test | 端到端：factor → signal → CH `signals` |
| M7.4-5 | ☑ SignalRouter 接 oms（PG.signals 之外补 CH 审计） | `internal/oms/signal_router.go`（重构）+ test | 信号被 risk reject 时 CH `signals.rejected=1` |
| M7.4-6 | ☑ quantengine 接入 server 启动 | `cmd/ant-server/main.go` | 启动后 quant_inference_total 计数 |

---

## M7.5 · 业务层切流 + 标 deprecated

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.5-1 | ☑ `internal/connect/market_service.go` GetKlines 切到 CH（带 PG fallback） | 同左 + test | grpcurl 查近 1h K 线 90% 走 CH（log 计数）|
| M7.5-2 | ☑ `connect/market_regime_service.go` 切流 | 同左 + test | 同上 |
| M7.5-3 | ☑ `connect/backtest_dataset_service.go` 切流 | 同左 + test | 回测拉历史数据走 CH |
| M7.5-4 | ☑ `connect/python_strategy_service.go` 切流 | 同左 + test | 沙箱拉历史走 CH |
| M7.5-5 | ☑ `service/kline_service*.go` 全部加 `// Deprecated: see internal/mdgateway. To be removed in M9.` | 同左 7 个文件 | `grep -c '// Deprecated' backend/internal/service/kline_service*.go \| grep -q '^7$'` |
| M7.5-6 | ☑ grep 验证业务代码 0 处直 import mt4client/mt5client（除 service/kline_service*）| — | `! grep -rE 'anttrader/internal/(mt4\|mt5)client' backend/internal/{ai,marketplace,oms,risk,connect,quantengine,factorsvc,mthub}/ \|\| exit 1` |
| M7.5-7 | ☑ 老 `kline_data` 表设为只读（trigger 阻止 INSERT） | `backend/migrations/101_kline_data_readonly.up.sql` (`+ .down.sql`) | `psql -c "INSERT INTO kline_data ..."` 返回 error |
| M7.5-8 | ⊘ frontend K 线组件切到新 RPC（旧前端已删除，2026-05-24 复盘后失效；M11 重建时一并重做）| `frontend/src/api/kline.ts` 与相关组件 | M11 立项后重新评估 |

---

## M7.6 · E2E 测试 + chaos

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.6-1 | ☑ 端到端 happy path：新建账户 → 订阅 → 5min 内 CH 有 tick/bar | `tests/e2e/mt_foundation_test.go` (build tag e2e) | `go test -tags=e2e ./tests/e2e/... -run TestHappyPath -timeout 10m` |
| M7.6-2 | ☑ chaos：CH 60s 中断 → spill → recovery，零数据丢失 | `tests/chaos/ch_outage_test.go` (build tag chaos) | 对比 spill 期前后 tick 计数差 ≤ 1% |
| M7.6-3 | ☑ chaos：单 broker 失联 → 其他账户照常 | `tests/chaos/broker_outage_test.go` | 故障账户 metric `md_circuit_state=1`，其他 ≠ 1 |
| M7.6-4 | ☑ SSE 重连：客户端断开 30s → 重连后继续收事件 | `tests/e2e/sse_reconnect_test.go` | 重连后 5s 内收到事件 |
| M7.6-5 | ☑ 7 天稳定性运行（人工 + monitoring）：tick rate / drop / circuit / spill 全部健康 | `docs/handover/verify-M7.6-5.log` | Grafana 截图归档 + 指标证据 |
| M7.6-6 | ☑ LOC 终检：mdgateway + adapter + mthub 三者非测试 LOC ≤ 1500 | — | `LOC=$(find backend/internal/mdgateway backend/internal/mthub -name "*.go" -not -name "*_test.go" \| xargs wc -l \| tail -1 \| awk '{print $1}'); test "$LOC" -le 1500` （注：`mdgateway/adapter/` 为 mdgateway 子目录，`find backend/internal/mdgateway` 已含）|
| M7.6-7 | ☑ telemetry 完整性测试：spec/15 + spec/11 §12 + ADR-0005 §5.3 列出的所有 metric 必须能在 `/metrics` 抓到（含至少 1 个 sample）| `tests/e2e/telemetry_test.go` (build tag e2e) | `go test -tags=e2e -run TestTelemetryCompleteness ./tests/e2e/... -timeout 5m`：测试断言 `curl /metrics` 输出包含 spec 列出的全部指标名（白名单文件 `tests/e2e/metrics_required.txt`）|

---

## M7.Z · 关闭

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M7.Z-1 | ☑ 跑 ADR-0001 §6 全部断言（4 条） | — | 全 0 退出 |
| M7.Z-2 | ☑ 更新 `docs/plan/ROADMAP.md` 状态为 ✅；`AGENT.md` §14 更新当前阶段 | 同左 | grep verify |
| M7.Z-3 | ☑ 写 `docs/handover/M7-closure.md`：里程碑总结 + 后续 M8/M9 输入 | 同左 | 人类 review |

---

## M10-BASE · 架构基础补全（融合 M11 金融架构设计）

> **设计输入**：
> - `docs/金融架构改造-M11路线图-2026-05-25.md`（D1 金融评审 + D2 量化考古合并版）
> - `docs/对标开源项目调研-2026-05-25.md`（NautilusTrader / Lean / Hummingbot / freqtrade 横向对比，含每张卡片的开源参考实现路径）
> **核心原则**：基础架构必须按正确的金融语义建设。PositionSizer 不是"更多风控规则"而是范式转换；15-state OMS 不是"更多状态"而是金融安全性；costsvc 不是"更多字段"而是回测可信度的根基。M11 架构决策提前到 M10-BASE 执行，避免 M10 建完后被 M11 拆掉重做。
> **研读前置**（Phase B 开工前）：通读 NautilusTrader 4 个核心文件 + Lean Algorithm.Framework README（~1 工日）。具体文件清单见对标调研文档 §九 Step 1。
> **前置**：M10.5 全部 12 张卡片 ☑
> **状态**：⚠️ **完成度声明不实**（2026-05-26 双 reviewer 审计）— 详见 `docs/audit/2026-05-26-项目真实状态-合并报告.md`
>   - **2026-05-26 P0-8 回滚**：Phase B (B1-B6) + Phase E (E0) + M11-13~17 共 12 张 ☑→🅒
>     - 原因：OMS/reconcile/idempotency/ledger/cache/backpressure/strategy-lifecycle 虽代码完整但 **MT5 真实 broker 回环未验证**（P0-3 刚实现 MT5 PlaceOrder/CloseOrder/ModifyOrder/FetchSymbolParams，尚未在真实 MT5 账户上回放）
>     - P0-1 已删 controlplane/quantengine 死代码；P0-3 已修 MT5 4 个 `not yet implemented` stub
>   - 46 张 ☑ 卡片**全部缺 handover log**；`mthub 28.5%` / `oms 51.4%` 覆盖率不达 60% 也标 ☑
>   - 修复执行计划：`docs/plan/REMEDIATION-2026-05-26.md`（R0+P0-P3 共 36 卡 / 21-33 工日）
>   - **进度**：R0 ☑ 五件套完成 → P0 8 张卡中 P0-1~P0-7 已 ☑，P0-8 当前

### M10-BASE 总览

| Phase | 内容 | 卡片数 | 预估工日 | 设计来源 | 对标开源参考 |
|---|---|---|---|---|---|---|
| **A** | 数据管道稳定 + 多用户运行时（Per-user limiter / Clock / Determinism） | 5 | 6.5 | 现有 M10 + spec/31 §一·二·三 | — | ☑ |
| **B** | OMS + 幂等 + Event Ledger（B6 含 Backpressure 规约） | 6 | 11 | M11-2, M11-3, M11-11 + spec/31 §五 | **NautilusTrader**（OrderStatus 枚举 + MessageBus + Cache） |
| **C** | 风控引擎重构（PositionSizer + Capability 扩展 user_risk_profiles） | 6 | 10.5 | M11-1, M11-4 + spec/31 §六 | **Lean**（五段框架：Alpha→Portfolio→Risk→Execution） |
| **D** | 回测引擎 + costsvc | 7 | 11 | M11-5, M11-7, M11-9 | **NautilusTrader**（BacktestEngine + FillModel）+ **Hummingbot**（funding_rate）| ☑ |
| **E** | AI 策略多级质量门控（E0 Strategy Lifecycle 前置） | 7 | 12 | M11-8 + spec/31 §四 | **freqtrade FreqAI**（walk-forward + OOS gate 生命周期）+ **NautilusTrader Strategy**| ☑ |
| **F** | 数据质量升级（MarketState + 双时间戳 + SRE） | 7 | 9 | M11-6, M11-10, M11-12 | **Lean**（Consolidator + timezone）+ **NautilusTrader**（MarketStatus）| ☑ |
| **总计** | | **38** | **~60 工日** | | |

### Phase A · 数据管道稳定 + 多用户运行时

> M10.5 关闭后，确保 M10 各子系统的数据通路完整可验证。**身份模型已固化**（user_id 业务 / account_id MT 账户 / broker 数据源；ctx 传播走 `interceptor.UserIDKey`），本 Phase 仅补**主管线 limiter** + **Clock 抽象** + **Determinism 契约**。

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M10-BASE-A1 | ☑ M10.5 全部关闭判据通过 | — | `make verify-cards-strict MILESTONE=M10 && make detect-stubs \| grep -q '0 hits'` | M10.5 关闭即 A1 完成 |
| M10-BASE-A2 | ☑ M10 未完成的 ADR 卡片扫尾（M10.1-1~M10.4-4 中未 ☑ 的卡片逐张验收） | 对应 M10 卡片文件 | 每张卡片独立 go test + metric 验证 | 禁止批量 ☑ |
| M10-BASE-A3 | ☑ md-doctor + slo-report 纳入 CI（每日 cron） | `Makefile` `deploy/cron/` | `make ci-nightly` 跑 md-doctor all --strict --window 24h | |
| M10-BASE-A4 | ☑ **多用户运行时补强**：(a) `internal/usermgr/limiter.go` Per-user signal/order/CH-write limiter (LRU 50000, 30d idle 驱逐)；(b) LoggerWithUser/SpanWithUser 复用 `interceptor.GetUserID(ctx)`；(c) `chmigrate/016_user_metrics_5m.sql` CH 用户级聚合表 (SummingMergeTree) + `user_metrics_flusher.go` 后台 5min flush；(d) Prometheus 无 user_id label | `backend/internal/usermgr/{limiter.go,context.go,*_test.go}` `backend/internal/mdgateway/{user_metrics_flusher.go,user_metrics_flusher_test.go,chmigrate/016_user_metrics_5m.sql}` | `go test -race -cover ./internal/usermgr/...` 89.2% coverage, 16+7 tests PASS | spec/31 §一 |
| M10-BASE-A5 | ☑ **Clock 抽象 + Determinism 契约**：(a) `internal/clock/{clock.go,real.go,simulated.go}` 完整 Clock interface；(b) `.golangci.yml` forbidigo 禁 time.Now/time.Sleep 等；(c) usermgr 已接入 Clock 包级变量；(d) `docs/spec/21-backtest-replay.md` §determinism；(e) `TestSimulatedClock_Determinism` 同输入 2 次 hash 一致 | `backend/internal/clock/{clock.go,real.go,simulated.go,*_test.go}` `backend/.golangci.yml` `docs/spec/21-backtest-replay.md` | `go test -race -cover ./internal/clock/...` 67.4% coverage, 12 tests PASS, determinism verified | spec/31 §二·三 |

### Phase B · OMS + 幂等 + Event Ledger

> **设计输入**：M11-2（幽灵订单恢复）、M11-3（三层幂等）、M11-11（event-sourced ledger）
> **对标开源**：**NautilusTrader** — `execution/engine.py`（OrderStatus 枚举 + 转换图）、`common/src/msgbus.rs`（MessageBus 模式）、`cache/cache.py`（全状态可重建 Cache）。vnpy `event/engine.py` 作为简化版 MessageBus 参考。
> **关键决策**：15-state OMS（10 现有 + StateRequoted / StateSlippageRejected / StateUnknown / StateReconciling / StateMarginCall）；三层幂等（PG advisory lock + Redis 96h TTL + broker magic）；NATS JetStream 升格为 Tier-0 唯一事实源。Event schema 命名对齐 Nautilus `OrderEvent` / `PositionEvent` / `AccountState` 类型层级。

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M10-BASE-B1 | 🅒 OMS 状态机升级：新增 5 个状态（Requoted / SlippageRejected / Unknown / Reconciling / MarginCall）+ timeout transition（SUBMITTED 30s 无响应 → Unknown） | `backend/internal/oms/{statemachine.go,types.go,*_test.go}` | `go test -race -run TestOrderStateMachine ./internal/oms/... -v` → 14 tests PASS (51.4% cov) | 参考 M11-2；状态命名对齐 NautilusTrader `nautilus_core/model/src/orders/` OrderStatus 枚举 |
| M10-BASE-B2 | 🅒 OMS 启动对账门控：reconcile-before-accept——OMS 重启/重连时所有账户进 Reconciling，mthub.Reconciliation 完成（broker 持仓全量拉回+对账）前 PlaceOrder 直接拒绝 | `backend/internal/mthub/{reconcile_gate.go,*_test.go}` | `go test -tags=integration ./internal/mthub/...` → 7 unit + 1 integration ALL PASS | 参考 M11-2；修"重连后双倍仓位"bug |
| M10-BASE-B3 | 🅒 三层幂等实现：Layer 1 PG advisory lock (account_id, client_id)；Layer 2 Redis SETNX 96h TTL（覆盖 G7 假期）；Layer 3 broker magic = hash(client_id) 截 32 位 | `backend/internal/mthub/{idempotency.go,idempotency_test.go}` | `go test -tags=integration ./internal/mthub/...` → 4 unit + 1 integration ALL PASS | 参考 M11-3 |
| M10-BASE-B4 | 🅒 Event-sourced ledger：NATS JetStream 定义 `OMS_EVENTS` stream（append-only）+ CH 投影表 `trade_event_log`（017_trade_event_log.sql）；所有订单状态变更先写 NATS 再更新 PG | `backend/internal/mthub/{trade_event_store.go,*_test.go}` `backend/internal/mdgateway/chmigrate/017_trade_event_log.sql` | `go test ./internal/mthub/...` → 4 tests PASS；CH 表含 23 字段 MergeTree | 参考 M11-11 Tier-0；event schema 对齐 NautilusTrader `nautilus_core/common/src/msgbus.rs` MessageBus 模式 |
| M10-BASE-B5 | 🅒 Live State Cache（Redis + in-memory）：从 Tier-0 重放重建 OMS state / position cache / idempotency；重启时全量重放最近 7 天 events | `backend/internal/mthub/{state_cache.go,state_cache_test.go}` | `go test ./internal/mthub/...` → 8 tests PASS | 参考 M11-11 Tier-1 |
| M10-BASE-B6 | 🅒 **Tier-2 Derived Quantities + 端到端 Backpressure 规约**：(a) PnL（Gross/Net 双轨）+ Greeks / VaR / margin / exposure，5s 重算，全从 Tier-0 + 行情快照算出不存；(b) 文档 `docs/spec/11-mdgateway.md` §13.6 backpressure 段（6 条流水线满载策略 + 4 条核心 metric）；(c) `internal/mdgateway/manager.go` `internal/factor/subscriber.go` `internal/quantengine/subscriber.go` 各 bounded chan 满载 drop + 4 条核心 metric（md_chan_full_total / md_nats_publish_dropped_total / md_consumer_lag / signal_dropped_total） | `backend/internal/mthub/{derived_state.go,derived_state_test.go}` `backend/internal/mdgateway/{manager.go,metrics.go}` `backend/internal/factor/subscriber.go` `backend/internal/quantengine/subscriber.go` `docs/spec/11-mdgateway.md` | `go test -tags=integration ./internal/mthub/...` → 3 tests PASS；spec §13.6 含完整规约 + 4 条 metric 全部实现 | M11-11 Tier-2 + spec/31 §五 |

### Phase C · 风控引擎重构（PositionSizer 范式）

> **设计输入**：M11-1（PositionSizer + HardLimit 拆分）、M11-4（跨账户净敞口）
> **对标开源**：**QuantConnect Lean** `Algorithm.Framework/` — 五段架构（Alpha → Portfolio → Risk → Execution）是本 Phase 的精确模板。`Risk/MaximumDrawdownPercentPortfolio.cs`（drawdown-based scaling）、`Portfolio/MeanVarianceOptimization*.cs`（Markowitz + Black-Litterman）。NautilusTrader `risk/engine.py`（RiskEngine 是流水线节点，非二值 deny）。
> **关键决策**：把"是否下单"问题转成"下多少手"问题。HardLimit 只挡 4-5 条不可妥协的硬规则（KYC/jurisdiction/margin floor/Kill Switch）；SoftLimit 通过 Sizer 渐进逼近 lot（vol-target / Kelly-fraction / risk-parity），不直接 deny。

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M10-BASE-C1 | 🅒 **HardLimit 接口 + 4 硬规则 + Capability 扩展现有 user_risk_profiles**：(a) HardLimit 接口 + 4 硬规则（KYC 地缘 / 保证金下限 / Kill Switch / 合约到期日）；(b) PG migration `116_user_risk_profiles_capabilities.up.sql` 给现有表加 6 字段（capability_tier / order_types_allowed / lot_per_order_max / daily_order_max / leverage_max / always_require_confirmation）；(c) `internal/risksvc/capability.go` 加载 + Tier 0-3 枚举 + PreCheck 强制检查；复用现有 symbol_whitelist / killswitch_enabled 字段 | `backend/migrations/116_user_risk_profiles_capabilities.up.sql` `backend/internal/risksvc/{hardlimit.go,capability.go,*_test.go}` | 13 tests PASS (8 hardlimit + 5 capability)；Tier 0-3 全部枚举 | M11-1 §二·A + spec/31 §六 |
| M10-BASE-C2 | 🅒 PositionSizer 接口 + VolTargetSizer 实现（`lot = target_risk / (ATR × contract_size × √holding_period)`） | `backend/internal/risksvc/{sizer.go,vol_target_sizer.go,*_test.go}` | 8 tests PASS；BTCUSD vs EURUSD ratio=5.71× ✅ | 参考 M11-1；对标 Lean `Portfolio/MeanVarianceOptimization*.cs` |
| M10-BASE-C3 | 🅒 KellyFractionSizer 实现（`f* = (p·b - q) / b`，half-Kelly 默认保守 + max 上限 25%） | `backend/internal/risksvc/{kelly_sizer.go,kelly_sizer_test.go}` | 7 tests PASS；positive/negative/zero edge + max cap + invalid inputs | 参考 M11-1 |
| M10-BASE-C4 | 🅒 跨账户净敞口聚合器（5s 刷新全平台视图：NetExposureBySymbol + TotalMarginUsed + BrokerLimitUsage） | `backend/internal/risksvc/{platform_aggregator.go,platform_aggregator_test.go}` | 4 tests PASS；EURUSD long+short net=0, multi-symbol, broker limits | 参考 M11-4 |
| M10-BASE-C5 | 🅒 平台总量限额 + Block Trade 公平分配（pro-rata / FIFO / VWAP） | `backend/internal/risksvc/{platform_limits.go,block_allocator.go,*_test.go}` | 13 tests PASS (7 limits + 6 allocator)；3 种分配策略覆盖 | 参考 M11-4 |
| M10-BASE-C6 | 🅒 Signal → HardLimit → Sizer → Submit 管线集成：替换旧 PreCheck deny 路径 | `backend/internal/risksvc/{pipeline.go,pipeline_test.go}` | 12 tests PASS；覆盖 FullPass/Capability/KillSwitch/HardLimit/PlatformLimit/RiskEngine/NoSizer/ZeroLots/BlockAllocation/Kelly | 参考 M11-1；对标 Lean 五段框架 + NautilusTrader `risk/engine.py` |

### Phase D · 回测引擎 + costsvc

> **设计输入**：M11-5（costsvc + Gross/Net P&L）、M11-7（回测悲观化）、M11-9（VaR / 压力测试）
> **对标开源**：**NautilusTrader** `backtest/models.py`（FillModel + LatencyModel，不允许 0 延迟 0 滑点）+ `backtest/engine.py`（回测引擎故意比实盘悲观）。**Hummingbot** `binance_perpetual/`（funding rate 完整实现，对应 funding_rate.go）。freqtrade backtest report HTML（Gross/Net P&L 双轨输出参考）。
> **关键决策**：回测引擎故意比实盘更悲观——commission/swap/spread 强制非零、fill probability < 1.0、bar 消费严格 timing gate。所有 P&L 必须双轨输出（Gross / Net）。同一 Source interface + 不同注入参数（LiveSource 无注入，BacktestSource 强制注入滑点/延迟/成本），**不再声称"完全相同代码路径"**。

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M10-BASE-D1 | 🅒 `costsvc/` 独立服务：spread_model（tick 流实时计算）+ commission（broker×symbol 费率表）+ swap_calculator（周三 triple）+ funding_rate（永续每 8h）+ estimator（Estimate(order)→CostBreakdown） | `backend/internal/costsvc/{spread_model.go,commission.go,swap_calculator.go,funding_rate.go,estimator.go,*_test.go}` | `go test -race -cover ./internal/costsvc/... \| awk '/coverage:/{gsub("%",""); if ($2<80) exit 1}'` | 97.6% coverage |
| M10-BASE-D2 | 🅒 costsvc 注入所有下单路径：Signal → CostEstimate → RiskCheck → Sizer → Submit；cost_breakdown 写入 order（PG JSONB） | `backend/internal/mthub/service.go` `backend/internal/oms/*.go` | `go test -tags=integration -run TestCostInjectedOrderPath ./internal/mthub/... -v` | integration deferred |
| M10-BASE-D3 | 🅒 CostModelSnapshot 数据流：回测启动时 `json.Marshal(costsvc.SnapshotConfig(broker, symbols))` → 存入 backtest_run.cost_model_snapshot；审计可追溯 | `backend/internal/costsvc/snapshot_config.go` | `go test -tags=integration -run TestCostModelSnapshot ./internal/... -v` | integration deferred |
| M10-BASE-D4 | 🅒 FillModel 拆分：GrossPrice + SpreadCost + SlippageBps + MarketImpactBps + Commission + Swap → NetFillPrice；回测路径强制非零默认 | `backend/internal/oms/{fill_model.go,fill_model_test.go}` | `go test -race -run TestFillModel ./internal/oms/... -v` | 5 tests PASS |
| M10-BASE-D5 | 🅒 Gross P&L / Net P&L 双轨输出：回测结果必须同时报告两个值；`Net P&L = Gross - SpreadCost - Commission - Swap - Slippage` | `backend/internal/oms/{pnl_calculator.go,pnl_calculator_test.go}` | `go test -race -run TestDualTrackPnL ./internal/oms/... -v` | 6 tests PASS |
| M10-BASE-D6 | 🅒 Bar finality 时序 gate：策略侧 bar 消费严格 `now ≥ bar.close_ts + finality_delay` | `backend/internal/factor/subscriber.go` `backend/internal/factorsvc/engine.go` | `go test -tags=integration -run TestBarFinalityGate ./internal/quantengine/... -v` | 4 bar finality tests PASS |
| M10-BASE-D7 | 🅒 VaR / CVaR 异步计算（历史模拟法 90 天窗口）+ 预设压力测试场景（2008 暴跌 / 2015 SNB / 2020 Covid / FOMC 秒杀） | `backend/internal/risksvc/{var_calculator.go,stress_test.go,*_test.go}` | `go test -race -run 'Test(VaR\|StressTest)' ./internal/risksvc/... -v` | 7 tests PASS |

### Phase E · AI 策略多级质量门控

> **设计输入**：M11-8（AI 策略多级 gate）+ spec/31 §四（Strategy Lifecycle）
> **对标开源**：**freqtrade FreqAI** — `freqai/freqai_interface.py`（训练/推理/验证生命周期，完整的 ML 策略 gate 流程，是本 Phase 的现成参考代码）、`freqai/data_drawer.py`（walk-forward 数据管理）、`optimize/hyperopt.py`（Optuna 集成 + CPCV-like split）。López de Prado 《Advances in Financial ML》（Deflated SR + CPCV 理论源头）。LookAhead Gate 无强开源参考，需自研 AST scanner。
> **关键决策**：CPCV（Combinatorial Purged Cross-Validation）walk-forward 替代简单 train/test split；Deflated Sharpe Ratio（López de Prado 2014）替代裸 Sharpe > 0 做 multiple testing correction。6 级 gate：Compliance → LookAhead → Walk-Forward+CPCV → DSR → Paper(14d) → Correlation。**E0 Strategy Lifecycle 是 E1-E6 全部卡片的隐藏前置**——没有 lifecycle 抽象，gate 验证的是"策略黑盒回测"，无法检验 stateful 策略的正确性。

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M10-BASE-E0 | 🅒 **Strategy Lifecycle 抽象**：(a) `internal/strategy/lifecycle.go` Strategy interface（OnStart/OnBar/OnTick/OnOrderEvent/OnEndOfDay/OnStop/Snapshot/Restore 8 hooks）；(b) PG `104_strategy_state.up.sql` 新建 `strategy_state` 表（含 strategy_version 二进制兼容性约束）；(c) snapshot/restore 走 protobuf；(d) hot-reload 路径：旧 OnStop()→PG→新 OnStart(snapshot)；(e) 不兼容版本拒绝 hot-reload 强制 rollover | `backend/internal/strategy/{lifecycle.go,snapshot.go,hot_reload.go,*_test.go}` `backend/migrations/104_strategy_state.up.sql` (`+.down.sql`) `backend/proto/ant/v1/strategy_state.proto` | `cd backend && go test -race -cover ./internal/strategy/... && go test -tags=integration -run 'TestStrategySnapshotRestore\|TestStrategyHotReload\|TestStrategyVersionMismatch' ./internal/strategy/... -v` | spec/31 §四；92.3% coverage, 8 tests PASS |
| M10-BASE-E1 | 🅒 LookAhead Gate：DSL AST 扫描，检测 `t+δ (δ>0)` 的未来引用（`close[t+1]` / `ref(close, -1)` / `high[t+1]` 等模式） | `backend/internal/ai/lookahead_scanner.go` | `go test -race -run TestLookAhead ./internal/ai/... -v` | 6 tests PASS, detects explicit future index, negative ref, future OHLC |
| M10-BASE-E2 | 🅒 Walk-Forward + CPCV：5-fold purged walk-forward，train_sharpe - test_sharpe > 1.0 → reject（过拟合）；所有 fold max_DD > 30% → reject；trade_count < 30 → reject（统计不显著） | `backend/internal/ai/walkforward.go` | `go test -race -run 'Test(WalkForward\|CPCV)' ./internal/ai/... -v` | 5 tests PASS, overfitting detected, maxDD fractional, CPCV median |
| M10-BASE-E3 | 🅒 Deflated Sharpe Ratio：DSR = SR × √[(1 - γ·log(N)) / (1 - skew·SR + (kurt-1)/4 · SR²)]，按用户对该策略族的尝试次数 N 衰减；DSR < 0.95(95% confidence) → reject | `backend/internal/ai/deflated_sharpe.go` | `go test -race -run TestDeflatedSharpe ./internal/ai/... -v` | 7 tests PASS, N=100 deflates, Edgeworth fallback, N=1 unchanged |
| M10-BASE-E4 | 🅒 Paper Gate 强化：14 天强制仿真 + paper_return < 0.5 × backtest_net_return → reject（regime fail）+ Net P&L > 0 | `backend/internal/ai/paper_gate.go` `backend/internal/connect/ai_handler.go` | `go test -race -run TestPaperGate ./internal/ai/... -v` | 5 tests PASS, insufficient days/negative PnL/regime fail/too few trades |
| M10-BASE-E5 | 🅒 Correlation Gate：新策略信号方向与用户已有实盘策略相关性 < 0.7；计算策略间信号方向一致率 | `backend/internal/ai/correlation_gate.go` | `go test -race -run TestCorrelationGate ./internal/ai/... -v` | 4 tests PASS, low/high correlation, Pearson ±1.0, NaN safe |
| M10-BASE-E6 | 🅒 6 级 Gate 管线串联 + PromoteToLive 条件更新（DSR >= 0.95, Paper >= 14d Net P&L > 0, Correlation < 0.7） | `backend/internal/ai/gate_pipeline.go` `backend/internal/connect/ai_handler.go` | `go test -race -run 'Test(AIGatePipeline\|PromoteToLive\|GateResults)' ./internal/ai/... -v` | 8 tests PASS, all-6-gates pipeline, early exit, promote-to-live conditions |

### Phase F · 数据质量升级（MarketState + 双时间戳 + SRE）

> **设计输入**：M11-6（MarketState 抽象）、M11-10（SessionClock + 双时间戳）、M11-12（SRE 控制面）
> **对标开源**：**NautilusTrader** `MarketStatus`（综合可交易性判断）。**Lean** `Engine/DataFeeds/Enumerators/RealTimeProviderConsolidator.cs`（bar 边界处理 + timezone + session phase，直接对应 SessionClock）。SRE 控制面（Kill Switch / Strategy Breaker / Canary）无强开源参考，需按机构内部经验自研。
> **关键决策**：Quality 从"挡脏数据"升级为"声明可交易性"。策略消费 MarketState 而非 raw tick。双时间戳保留：BrokerTsUnixMs 用于 bar 归属（仅当 ClockSkew < 阈值），ArrivedTsUnixMs 用于审计/反作弊回退。

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M10-BASE-F1 | ☑ MarketState 结构体 + Quality 升级：LastQuote / QuoteAgeMs / IsTradeable / SpreadZscore / GapMarker / SwapWindow / SessionPhase / HolidayMarker / TickRateZscore / TriangulationDelta | `backend/internal/mdgateway/{market_state.go,quality.go}` | `go test -race -run 'TestMarketState' ./internal/mdgateway/... -v` | 3 tests PASS |
| M10-BASE-F2 | ☑ SessionClock 抽象：SessionOpen / BarBoundary / SwapWindow(17:00 EST ±5min) / HolidayMarker / BrokerClockOffsetMs | `backend/internal/mdgateway/session_clock.go` | `go test -race -run TestSessionClock ./internal/mdgateway/... -v` | 8 tests PASS |
| M10-BASE-F3 | ☑ 双时间戳 Tick + NTP 偏离丢弃：BrokerTsUnixMs + ArrivedTsUnixMs + ClockSkewMs；abs > 5000 → drop + metric | `backend/internal/mdgateway/{quality.go,metrics.go}` | `go test -race -run TestQuality_NTP ./internal/mdgateway/... -v` | 2 tests PASS, clock_skew drop with metric |
| M10-BASE-F4 | ☑ Quote stuffing 检测 + 自动暂停：tick rate Z-score > 4σ → 暂停该 symbol 30s | `backend/internal/mdgateway/{quote_stuffing.go,metrics.go,manager.go}` | `go test -race -run TestStuffing ./internal/mdgateway/... -v` | 3 tests PASS |
| M10-BASE-F5 | ☑ 点差扩大检测 + 告警：spread Z-score > 3σ → metric md_spread_anomaly | `backend/internal/mdgateway/{quality.go,metrics.go,manager.go}` | `go test -race -run 'TestQuality_Spread' ./internal/mdgateway/... -v` | 3 tests PASS, zero-variance safe |
| M10-BASE-F6 | ☑ MarketState 注入策略通路：factor/signal/order 管线消费 MarketState.IsTradeable，不可交易时 skip | `backend/internal/mdgateway/manager.go` `backend/internal/factor/subscriber.go` | `go test -race ./internal/factor/...` | MarketState injection in HandleTick + factor subscriber gate |
| M10-BASE-F7 | ☑ SRE 控制面：Kill Switch + Strategy Breaker（30min 亏 X% 自动熔断）+ Canary（新版策略小账户跑 1 周再全量） | `backend/internal/controlplane/{controlplane.go,controlplane_test.go}` | `go test -race -cover ./internal/controlplane/...` | 16 tests PASS, 94.7% coverage |

### M10-BASE 关闭判据

```bash
# 1. Phase A-F 全部 38 张卡片 ☑（含 A4/A5/E0 SaaS 地基）
grep -E '^\| M10-BASE-' docs/plan/ROADMAP.md | grep -c '🅒' | awk '$1>0{print "PENDING:"$1; exit 1}'

# 1.5 SaaS 地基断言（spec/31）
go test -race ./internal/{tenant,clock,strategy}/... && \
  go test -tags=integration -run 'TestBacktestDeterminism|TestStrategySnapshotRestore' ./internal/... -v

# 2. 回测双轨 P&L 断言
go test -tags=integration -run TestDualTrackPnL ./internal/oms/... -v

# 3. AI 策略 6 级 Gate 全链路
go test -tags=integration -run TestAIGatePipeline ./internal/ai/... -v

# 4. MarketState 策略通路
go test -tags=integration -run TestMarketStateBlocksSignal ./internal/quantengine/... -v

# 5. 风控管线 HardLimit → Sizer → Submit
go test -tags=integration -run TestRiskPipeline ./internal/risksvc/... -v

# 6. Event-sourced ledger 重建验证
go test -tags=integration -run TestStateCacheRebuild ./internal/mthub/... -v

# 全过 → 写 docs/handover/M10-BASE-closure.md
```

---

## M8（已合并到 M10-BASE）

> M8 原有的"业务层渐进重构"各卡片已按优先级重新分配到 M10-BASE Phase B/C/D/E/F。
> 详见 `docs/plan/BACKLOG.md` §M8/M9 重建优先级。

## M9（已合并到 M10-BASE）

> M9 原有的"老包删除"已完成（mt4client/mt5client 已迁 legacy/，kline_service 已删除）。
> 剩余 PG 旧表清理纳入 M10-BASE-A2 扫尾。

---

## M10.5 · 补完段（A+ 真兑现）

> ⚠️ **触发原因**：Cascade 独立审计（`docs/handover/M10-REVIEW-by-Cascade.md`）+ DeepSeek design review（`docs/plan/M10-DESIGN-REVIEW.md`）联合发现 **3 P0 + 5 P1 + 11 P2 + 9 设计缺陷**。`scripts/verify-cards-strict.sh M10` 实测 17/18 卡片未达 §0.3 硬条件，机械回退至 🅒。
>
> **本段目标**：补完真实施 + 修复设计缺陷，让 M10 从"骨架完成"升级到"实质 A+"。
>
> **新规则（AGENT.md §0.3-§0.5 起立即生效）**：
> - 卡片 ☑ 必须通过 `make verify-cards-strict MILESTONE=M10` C1/C2/C3 三条硬条件
> - 反 stub 红线见 §0.4
> - 卡片粒度模板见 §0.5
>
> **预算**：~12 工日

### M10.5 总览

| Sub | 内容 | 卡片数 | 工日 |
|---|---|---|---|
| 修复 P0 runtime bug | M10.5-3 | 1 | 1.5 |
| 补 mdgateway 单元测试 | M10.5-4 | 1 | 1.5 |
| md-doctor 真实施 | M10.5-5 | 1 | 1.5 |
| slo-report 真实施 | M10.5-6 | 1 | 1 |
| normalizer_invalidator 接 pgx | M10.5-7 | 1 | 0.5 |
| backfiller 修限速 + 接 mtapi + PG NOTIFY | M10.5-8 | 1 | 1.5 |
| spill_replay NATS 去重 + finalizedBars fatal | M10.5-9 | 1 | 0.5 |
| DLQ 异步化 + Buffer 开关 + S-2 文档 | M10.5-10 | 1 | 1 |
| 端到端 smoke 真跑 | M10.5-11 | 1 | 1 |
| 100 账户负载测真跑 | M10.5-12 | 1 | 1 |
| spec/13 文档补：FINAL + 容量 + 迁移前置 | M10.5-13 | 1 | 0.5 |
| 红队独立验收（Cascade）| M10.5-14 | 1 | 0.5 |
| **总计** | | **12** | **~12 工日** |

### M10.5 卡片详表

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.5-3 | ☑ | ☑ **修 3 P0 runtime bug + S-3 finality 逻辑**：(a) `chmigrate/011_buffer_account_id.sql` 给 `md_ticks_buffer`/`md_bars_buffer` 加 `account_id` 列；(b) `quality.Check(ctx, t)` 加 ctx 参数所有调用方传递；(c) `manager.HandleTick` 真实加 6 段 OTel span（normalize/quality/dedup/aggregate/publish/enqueue）；(d) `bar_aggregator.IngestExternalBar` 改为按 `(broker,canonical,period,close_ts)` 精确去重，不再用 `<= MAX` | `backend/internal/mdgateway/chmigrate/011_buffer_account_id.sql` `backend/internal/mdgateway/{quality.go,manager.go,bar_aggregator.go}` | <pre>cd backend && go build ./... && go test -race ./internal/mdgateway/... 2>&1 \| tee /tmp/v.log && grep -qE '^ok ' /tmp/v.log && ! grep -q '\[no test files\]' /tmp/v.log</pre> |
| M10.5-4 | ☑ | ☑ **补 6 个声明过但不存在的 mdgateway 测试**：`TestPublishReplayHeader` `TestSpillReplayDualWrite` `TestBarFinality` `TestDLQParseError` `TestDLQSampling` `TestNormalizerListenerFallback`；每个含 ≥3 行真断言 | `backend/internal/mdgateway/{publisher_test.go,spill_replay_test.go,bar_aggregator_test.go,dlq_writer_test.go,normalizer_invalidator_test.go}` | <pre>cd backend && go test -race -run 'TestPublishReplayHeader\|TestSpillReplayDualWrite\|TestBarFinality\|TestDLQParseError\|TestDLQSampling\|TestNormalizerListenerFallback' ./internal/mdgateway/ -v 2>&1 \| tee /tmp/v.log && test $(grep -cE '^--- PASS' /tmp/v.log) -ge 6</pre> |
| M10.5-5 | ☑ | ☑ **md-doctor 真实施**：5 子命令各 ≥80 LOC；用 FINAL 查询 CH（修 S-1）；text/json 输出；--strict 模式（任一异常 exit 1）；reconcile 用 `clickhouse_writer.go` 内部 metric snapshot；连真实 CH/NATS | `backend/cmd/md-doctor/{main.go,reconcile.go,bar_continuity.go,canonical_liveness.go,dlq_tail.go,*_test.go}`（每个 ≥80 LOC，禁止 stub） | <pre>cd backend && go build -o /tmp/md-doctor ./cmd/md-doctor/ && /tmp/md-doctor --help 2>&1 \| grep -cE 'reconcile\|bar-continuity\|canonical-liveness\|dlq-tail\|all' \| awk '$1==5{ok=1}END{exit !ok}' && CH_DSN=clickhouse://default@localhost:9000/ant /tmp/md-doctor reconcile --window 10m --output json 2>&1 \| jq -e '.window_ms and .ch_count >= 0' && ! grep -rE '(stub\|TODO\|not (wired\|connected))' backend/cmd/md-doctor/</pre> |
| M10.5-6 | ☑ | ☑ **slo-report 真实施**：用 `github.com/prometheus/client_golang/api/prometheus/v1` 拉指标；4 SLO 各算 availability% + budget remaining；markdown 输出；Prometheus recording rules `deploy/prometheus/rules.yml` 真上线 | `backend/cmd/slo-report/{main.go,prom_client.go,*_test.go}` `deploy/prometheus/rules.yml` | <pre>cd backend && go build -o /tmp/slo-report ./cmd/slo-report/ && PROM_URL=http://localhost:9090 /tmp/slo-report --window 1h --output md 2>&1 \| tee /tmp/v.log && grep -cE 'SLO-MD-[1-4]' /tmp/v.log \| awk '$1==4{ok=1}END{exit !ok}' && promtool check rules deploy/prometheus/rules.yml && ! grep -rE 'stub\|"— stub"' backend/cmd/slo-report/</pre> |
| M10.5-7 | ☑ | ☑ **normalizer_invalidator 接 pgx LISTEN**：runner.go 创建独立 `pgx.Conn`；`listenLoop` 真调 `WaitForNotification`；解析 JSON payload (broker, symbol_raw) 回调 `normalizer.cache.Remove`；连接断 → 切 30s ticker fallback + metric `md_normalizer_listener_state` | `backend/internal/mdgateway/{normalizer_invalidator.go,runner.go}` + 测试 | <pre>cd backend && go test -tags=integration -run 'TestNormalizerListener(PgListen\|Fallback)' ./internal/mdgateway/ -v 2>&1 \| tee /tmp/v.log && test $(grep -cE '^--- PASS' /tmp/v.log) -ge 2</pre> |
| M10.5-8 | ☑ | ☑ **backfiller 修 M-1/M-2 限速 + 接 mtapi + PG NOTIFY**：(a) `accountLimiters map[string]*rate.Limiter`（per-account 6 req/min）+ `globalLimiter`（60 req/s 总顶）；(b) `source_mtapi.go` 真调 `adapter.GetPriceHistory`；(c) 订阅新增触发 PG NOTIFY → backfiller 立即 BackfillAccount（不等 6h cron） | `backend/internal/mdgateway/backfiller/{backfiller.go,source_mtapi.go,trigger_pg.go}` + 测试 | <pre>cd backend && go test -race -run 'TestBackfillerPerAccountRate\|TestBackfillerPgTrigger' ./internal/mdgateway/backfiller/ -v 2>&1 \| tee /tmp/v.log && test $(grep -cE '^--- PASS' /tmp/v.log) -ge 2 && wc -l backend/internal/mdgateway/backfiller/source_mtapi.go \| awk '$1>=60{ok=1}END{exit !ok}'</pre> |
| M10.5-9 | ☑ | ☑ **spill_replay NATS 去重（M-2）+ finalizedBars fatal-on-error（M-3）**：(a) `publisher.PublishMsg` 加 `Nats-Msg-Id: <broker>:<canonical>:<ts>:<xxhash>` header；JetStream stream 加 `Duplicates: 2m`；(b) `runner.loadFinalizedBars` CH 不可达 → 返回 error 阻塞启动 | `backend/internal/mdgateway/{publisher.go,spill_replay.go,runner.go}` + 测试 | <pre>cd backend && go test -race -run 'TestPublisherDedupHeader\|TestRunnerFatalOnChDown' ./internal/mdgateway/ -v 2>&1 \| tee /tmp/v.log && test $(grep -cE '^--- PASS' /tmp/v.log) -ge 2</pre> |
| M10.5-10 | ☑ | ☑ **DLQ 异步化（L-1）+ Buffer 开关（S-2）+ 文档**：(a) `DLQWriter` 加 `dlqQ chan` + 后台 goroutine flush；(b) `clickhouse_writer.go` 加 env `ANT_CH_BUFFER_ENABLED`（默认 true；false → INSERT 直写 md_ticks）+ metric `md_ch_buffer_age_seconds`；(c) `docs/spec/13-clickhouse-schema.md` §2.7 加 Buffer engine OOM 风险声明 | `backend/internal/mdgateway/{dlq_writer.go,clickhouse_writer.go,metrics.go}` `docs/spec/13-clickhouse-schema.md` + 测试 | <pre>cd backend && go test -race -run 'TestDLQAsync\|TestCHBufferEnvSwitch' ./internal/mdgateway/ -v 2>&1 \| tee /tmp/v.log && test $(grep -cE '^--- PASS' /tmp/v.log) -ge 2 && grep -q 'ANT_CH_BUFFER_ENABLED' docs/spec/13-clickhouse-schema.md</pre> |
| M10.5-11 | ☑ | ☑ **端到端 smoke 真跑**：tests/e2e/smoke_test.go 启 runner.Run → 注入 100 真 tick（含 1% 重复 + 1% bid>ask）→ 等 5s → 三方对账：metric.md_tick_total ≥ 99，CH.md_ticks count ≥ 95（FINAL），NATS sub.md.tick 收 ≥ 95；DLQ.md_ticks_dlq 中 bid_gt_ask 行 ≥ 1 | `backend/tests/e2e/smoke_test.go` (build tag e2e) | <pre>cd backend && E2E_CH_HOST=localhost E2E_NATS_URL=nats://localhost:4222 go test -tags=e2e -run TestE2ESmoke ./tests/e2e/... -v -timeout 2m 2>&1 \| tee /tmp/v.log && grep -qE '^--- PASS: TestE2ESmoke' /tmp/v.log</pre> |
| M10.5-12 | ☑ | ☑ **100 账户负载测真跑**：mock broker 100 goroutine × 250 tick/s × 5min；真实跑通；断言 spill=0 且 P99<500ms | `backend/tests/loadtest/{mock_broker.go,load_100_accounts_test.go}` (build tag loadtest，去 t.Skip) | <pre>cd backend && go test -tags=loadtest -timeout 15m -run Test100AccountsNoSpill ./tests/loadtest/... -v 2>&1 \| tee /tmp/v.log && grep -qE '^--- PASS: Test100AccountsNoSpill' /tmp/v.log</pre> |
| M10.5-13 | ☑ | ☑ **spec/13 文档补**：FINAL 查询规范（§9 新增）+ EXCHANGE TABLES 迁移前置条件（§2.8 注释）+ md_bars 长期容量与归档（§8 新增） | `docs/spec/13-clickhouse-schema.md` `backend/internal/mdgateway/chmigrate/spec_test.go` | <pre>cd backend && go test -count=1 -run TestSpec13Keywords ./internal/mdgateway/chmigrate/ -v</pre> |
| M10.5-14 | ☑ | ☑ **红队独立验收（Cascade 跑，非 builder agent 自验）**：`make verify-cards-strict MILESTONE=M10` 必须 PASS=18 FAIL=0；再跑 `make detect-stubs` 全仓库 stub 关键词 0 命中；最后 Cascade 写 `docs/handover/M10-REVIEW-by-Cascade-v2.md` 二次审计 | `Makefile`（加 detect-stubs target）`docs/handover/M10-REVIEW-by-Cascade-v2.md` | <pre>make verify-cards-strict MILESTONE=M10 && make detect-stubs \| grep -qE 'OK.*0 hits' && test -s docs/handover/M10-REVIEW-by-Cascade-v2.md && grep -q '等级\s*[:：]\s*A-\?\|A\+' docs/handover/M10-REVIEW-by-Cascade-v2.md</pre> |

### M10.5 关闭判据（重新定义 M10 整体关闭）

```bash
# 1. M10.5 全部 12 张卡片 ☑
PENDING=$(grep -cE '^\| M10\.5-' docs/plan/ROADMAP.md | grep -c '🅒'); test "$PENDING" -eq 0

# 2. M10 整体（含 M10.1-M10.5）verify-cards-strict 全 PASS
make verify-cards-strict MILESTONE=M10

# 3. 反 stub 探测 0 命中
make detect-stubs | grep -q '0 hits'

# 4. 端到端 smoke + 100 账户负载真实跑过
test -s docs/handover/verify-M10.5-11.log
test -s docs/handover/verify-M10.5-12.log

# 5. Cascade 二次审计等级 ≥ A-
grep -qE '等级\s*[:：]\s*A-?|A\+' docs/handover/M10-REVIEW-by-Cascade-v2.md

# 全过 → 写 docs/handover/M10-closure-v2.md 替代 M10-closure.md
```

## M10 · 数据基础 A+ 硬化

> ⚠️ **前端现状重要前提**（2026-05-24 修正）：旧 frontend 已基本全部删除，**与路线 A（全量重写）无差别**，需在 M11 全面重建。
> - M10 是**纯后端**里程碑，验收手段限定为：CLI 输出 / `curl /metrics` / `docker exec clickhouse-client` / `go test`
> - **禁止**任何 "打开浏览器看 UI" 类验收
> - M7.5-8（"frontend K 线组件切到新 RPC"）的 ☑ 状态在前端被删除后已**事实失效**，由 M11 重做（不在 M10 关闭判据内）
> - ROADMAP 中所有 `frontend/src/...` 路径假定为 M11 待建，M10 卡片不应触及
>
> **状态**：开工
> **目标**：数据基础从 B+ 推到 A+。修复 H-1（CH dedup 与应用层冲突）/ H-2（replay 不补 NATS）正确性 bug；补 M-1（历史回填）/ M-2（TTL 时间轴）/ M-3（容量调优）能力；引入 SLO/DLQ/Trace 工程基础设施。
>
> **前置必读**（卡片开工前 100% 阅读，不许跳）：
> - **ADR**：`docs/adr/0008-storage-dedup-and-time-axis.md` `0009-replay-dual-write-and-bar-finality.md` `0010-slo-alert-dlq-trace.md` `0011-capacity-vault-cache-hardening.md`
> - **Spec（更新版）**：`docs/spec/11-mdgateway.md`（M10 强化叠加段）`docs/spec/13-clickhouse-schema.md` §2.6/§2.7/§2.8 `docs/spec/15-observability.md` §6.x §6.y
> - **Spec（新增）**：`docs/spec/18-backfiller.md` `docs/spec/19-md-doctor.md` `docs/spec/20-slo.md`
>
> **执行规则**：严格按 AGENT.md §0；自动连续执行；仅 3 种情况停下（见 AGENT.md §0.1）。

### M10 总览

| Sub-milestone | 内容 | 卡片数 | 预估工日 |
|---|---|---|---|
| **M10.1** | ADR-0008 存储层去重对齐 + 时间轴纪律 | 3 | 2 |
| **M10.2** | ADR-0009 replay 双写 + bar finality + backfiller | 4 | 3 |
| **M10.3** | ADR-0010 DLQ + e2e latency + OTel + 6 alert | 4 | 3 |
| **M10.4** | ADR-0011 容量调优 + envelope encryption + cache 失效 | 4 | 3 |
| **M10.5** | md-doctor + slo-report CLI | 2 | 1 |
| **M10.Z** | 关闭：md-doctor 全 PASS + 7d SLO 全绿 | 1 | 1 |
| **总计** | | **18** | **~13 工日** |

### M10.1 · 存储层去重对齐（ADR-0008）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.1-1 | ☑ chmigrate 新增 `006_md_ticks_v2.sql` `007_md_bars_v2.sql`：新表 + INSERT SELECT + EXCHANGE TABLES（ORDER BY 含 bid/ask/bid_volume/ask_volume；TTL 用 arrived_unix_ms） | `backend/internal/mdgateway/chmigrate/{006_md_ticks_v2.sql,007_md_bars_v2.sql,migrate_test.go}` | `make migrate-ch && docker exec ant-clickhouse clickhouse-client --query "SELECT sorting_key FROM system.tables WHERE database='ant' AND name='md_ticks'" \| grep -E 'ts_unix_ms.*bid.*ask.*bid_volume.*ask_volume' && docker exec ant-clickhouse clickhouse-client --query "SELECT engine_full FROM system.tables WHERE database='ant' AND name='md_ticks'" \| grep -q 'arrived_unix_ms.*INTERVAL 90 DAY'` |
| M10.1-2 | ☑ `clickhouse_writer.go` `bar_aggregator.go` 用 `ts_unix_ms` 做边界判定的处全改 `arrived_unix_ms`（注释 `// ADR-0008 §2.2`） | `backend/internal/mdgateway/{clickhouse_writer.go,bar_aggregator.go}` + 测试 | `! grep -nE '\bts_unix_ms\b.*\b(bucket\|partition\|cutoff\|TTL\|window)' backend/internal/mdgateway/{clickhouse_writer,bar_aggregator}.go && go test -race ./internal/mdgateway/...` |
| M10.1-3 | ☑ 端到端对账测试：100k tick 注入 → metric / NATS / CH 三方对账误差 < 0.01% | `tests/e2e/dedup_alignment_test.go` (build tag e2e) | `go test -tags=e2e -run TestDedupAlignment ./tests/e2e/... -timeout 5m` |

### M10.2 · Replay 双写 + Bar finality + Backfiller（ADR-0009）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.2-1 | ☑ `mdtick.Tick`/`mdtick.Bar` 加 `IsReplay bool`；`publisher.go` 写 NATS header `X-Ant-Replay` | `backend/internal/mdgateway/adapter/mdtick/mdtick.go` `backend/internal/mdgateway/publisher.go` + 测试 | `grep -q 'IsReplay\s*bool' backend/internal/mdgateway/adapter/mdtick/mdtick.go && go test -tags=integration -run TestPublishReplayHeader ./internal/mdgateway/...` |
| M10.2-2 | ☑ `spill_replay.go` 改双写：先 `publisher.PublishTick/Bar` 再 `chWriter.Enqueue`；自动设 `IsReplay=true` | `backend/internal/mdgateway/spill_replay.go` + 测试 | `go test -tags=integration -run TestSpillReplayDualWrite ./internal/mdgateway/... -v` |
| M10.2-3 | ☑ `bar_aggregator.go` 启动加载 `finalizedBars`（CH MAX close_ts），所有写 bar 路径前置 finality 检查；metric `md_bar_skipped_finalized_total` | `backend/internal/mdgateway/{bar_aggregator.go,metrics.go}` + 测试 | `go test -tags=integration -run TestBarFinality ./internal/mdgateway/... -v` |
| M10.2-4 | ☑ 实现 `internal/mdgateway/backfiller/`（spec/18 全部文件）；runner.go 启动调用 + PG NOTIFY 触发新订阅回填 | `backend/internal/mdgateway/backfiller/{backfiller.go,source_mtapi.go,target.go,metrics.go,*_test.go}` `runner.go` | `( cd backend && go build ./internal/mdgateway/backfiller/... && go test -race -cover ./internal/mdgateway/backfiller/... ) && go test -tags=integration -run TestBackfillGap ./internal/mdgateway/backfiller/... && LOC=$(find backend/internal/mdgateway/backfiller -name "*.go" -not -name "*_test.go" \| xargs wc -l \| tail -1 \| awk '{print $1}'); test "$LOC" -le 350` |

### M10.3 · DLQ + Latency + OTel（ADR-0010）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.3-1 | ☑ chmigrate `008_md_ticks_dlq.sql` + `dlq_writer.go`（采样：parse_error 100%；bid_gt_ask/non_positive 1%）；`quality.go` 注入 `DLQWriter` 接口 | `backend/internal/mdgateway/chmigrate/008_md_ticks_dlq.sql` `dlq_writer.go` `quality.go` + 测试 | `docker exec ant-clickhouse clickhouse-client --query "DESCRIBE ant.md_ticks_dlq" \| grep -q reason && go test -tags=integration -run "TestDLQ(ParseError\|Sampling)" ./internal/mdgateway/...` |
| M10.3-2 | ☑ `metrics.go` 新增 `md_e2e_latency_seconds` Histogram + `md_spill_pending_files` Gauge + `md_dlq_sampled_total` Counter；`clickhouse_writer.go` flush 成功 Observe；`spill_replay.go` 30s 扫目录更新 gauge | `backend/internal/mdgateway/{metrics.go,clickhouse_writer.go,spill_replay.go}` + 测试 | `curl -s localhost:8080/metrics \| grep -E '^md_(e2e_latency_seconds_bucket\|spill_pending_files\|dlq_sampled_total)' \| wc -l \| awk '$1>=3{exit 0} {exit 1}'` |
| M10.3-3 | ☑ OTel SDK：`internal/trace/otel.go`（`OTEL_EXPORTER_OTLP_ENDPOINT` env 开关）；`manager.HandleTick` 加 span 链（normalize/quality/dedup/aggregate/publish/chwrite） | `backend/internal/trace/{otel.go,otel_test.go}` `backend/internal/mdgateway/manager.go` + go.mod 加 OTel 依赖 | `OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 go test -tags=integration -run TestTraceExport ./internal/mdgateway/... -timeout 2m` |
| M10.3-4 | ☑ `deploy/prometheus/alerts.yml` 加 6 条 M10 alert（spec/15 §6.x 列表）；`promtool check rules` 通过 | `deploy/prometheus/alerts.yml` | `for a in BrokerClockSkewHigh TickLatencyP99High SpillBacklog SpillUnwritable DLQSpike NormalizerFallbackHigh; do grep -q "alert: $a" deploy/prometheus/alerts.yml \|\| { echo "MISSING $a"; exit 1; }; done && promtool check rules deploy/prometheus/alerts.yml` |

### M10.4 · 容量 + Vault + Cache 失效（ADR-0011）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.4-1 | ☑ chmigrate `009_md_tick_buffer.sql` `010_md_bar_buffer.sql`；`clickhouse_writer.go` INSERT 改 `md_ticks_buffer`/`md_bars_buffer`；默认配置 QueueSize=50000 / MaxBatch=10000 / Flush=500ms（env 可覆盖） | `backend/internal/mdgateway/chmigrate/{009_md_tick_buffer.sql,010_md_bar_buffer.sql}` `clickhouse_writer.go` + 测试 | `docker exec ant-clickhouse clickhouse-client --query "SELECT engine FROM system.tables WHERE database='ant' AND name='md_ticks_buffer'" \| grep -q '^Buffer$' && grep -E 'QueueSize.*50000\|MaxBatchSize.*10000\|FlushInterval.*500' backend/internal/mdgateway/clickhouse_writer.go` |
| M10.4-2 | ☑ 100 账户负载测试（mock broker 25k tick/s × 5min）：`md_spill_writes_total == 0` 且 `md_e2e_latency_seconds` P99 < 500ms | `tests/loadtest/{mock_broker.go,load_100_accounts_test.go}` (build tag loadtest) | `go test -tags=loadtest -timeout 15m -run Test100AccountsNoSpill ./tests/loadtest/...`；测试函数内部必须包含 ADR-0011 §6 列出的两条断言：(1) `md_spill_writes_total == 0`；(2) `histogram_quantile(0.99, md_e2e_latency_seconds) < 0.5`。断言失败 → `t.Fatalf` 退出非 0 |
| M10.4-3 | ☑ `internal/secrets/` 重构 envelope encryption（version+dek_kid+nonce+wrapped_dek+ciphertext+tag）；保留旧格式自动迁移；`cmd/ant-vault rotate` CLI；`secrets.MasterProvider` 接口预留 KMS | `backend/internal/secrets/{vault.go,envelope.go,vault_legacy.go,master_provider.go,*_test.go}` `backend/cmd/ant-vault/main.go` `docs/spec/17-secrets-and-errors.md` | `go test -race -cover ./internal/secrets/... \| awk '/coverage:/{gsub("%",""); if ($2<90) exit 1}' && go run ./cmd/ant-vault rotate --dry-run \| grep -q rows_to_rewrite` |
| M10.4-4 | ☑ PG migration `111_broker_symbols_notify.up.sql`（trigger pg_notify）；`internal/mdgateway/normalizer_invalidator.go` LISTEN + 30s ticker fallback；`normalizer.go` 注入 invalidator | `backend/migrations/111_broker_symbols_notify.up.sql` (`+.down.sql`) `backend/internal/mdgateway/{normalizer_invalidator.go,normalizer.go}` + 测试 | `make migrate && go test -tags=integration -run "TestNormalizer(Invalidation\|ListenerFallback)" ./internal/mdgateway/... -v` |

### M10.5 · md-doctor + slo-report CLI

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.5-1 | ⊘ | ☑ `cmd/md-doctor/`：reconcile/bar-continuity/canonical-liveness/dlq-tail/all 五个子命令（spec/19）；text+json 输出；--strict | `backend/cmd/md-doctor/{main.go,reconcile.go,bar_continuity.go,canonical_liveness.go,dlq_tail.go,*_test.go}` | `cd backend && go build -o /tmp/md-doctor ./cmd/md-doctor/ && /tmp/md-doctor --help \| grep -E 'reconcile\|bar-continuity\|canonical-liveness\|dlq-tail\|all' \| wc -l \| grep -q '^5$' && /tmp/md-doctor all --window 10m --output json \| jq -e '.reconcile != null'` |
| M10.5-2 | ⊘ | ☑ `cmd/slo-report/`：从 Prometheus 拉指标计算 4 条 SLO（spec/20 §1）+ budget 消耗，markdown 输出；--strict；Prometheus recording rules `deploy/prometheus/rules.yml`（`md:up:1m` `md:availability:30d`） | `backend/cmd/slo-report/{main.go,*_test.go}` `deploy/prometheus/rules.yml` | `cd backend && go build -o /tmp/slo-report ./cmd/slo-report/ && /tmp/slo-report --window 1h --output text \| grep -E 'SLO-MD-[1-4]' \| wc -l \| grep -q '^4$' && promtool check rules deploy/prometheus/rules.yml` |

### M10.Z · 关闭

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.Z-1 | 🅒 7 天稳定性 + md-doctor 全 PASS + slo-report 全绿 + ROADMAP 状态更新 + handover 报告 | `docs/handover/M10-closure.md` `docs/handover/md-doctor-{date}.json` `docs/handover/slo-report-{date}.md` | 见下方关闭清单 |

### M10.Z 关闭清单

```bash
# (1) 全部 ADR 引用文件就位
for adr in 0008 0009 0010 0011; do
  ls docs/adr/${adr}-*.md > /dev/null || { echo "MISSING ADR-${adr}"; exit 1; }
done

# (2) 全部新 spec 就位
for s in 18-backfiller 19-md-doctor 20-slo; do
  test -f docs/spec/${s}.md || { echo "MISSING spec/${s}.md"; exit 1; }
done

# (3) M10 全部 18 张卡片 ☑
PENDING=$(grep -E '^\| M10\.' docs/plan/ROADMAP.md | grep -c '🅒' || true)
test "$PENDING" -eq 0 || { echo "未完成 M10 卡片：$PENDING 张"; exit 1; }

# (4) 端到端对账 24h PASS
md-doctor all --window 24h --strict --output json > docs/handover/md-doctor-$(date +%Y%m%d).json

# (5) 7 天 SLO 全绿
slo-report --window 7d --strict > docs/handover/slo-report-$(date +%Y%m%d).md

# (6) ADR-0001 §6 4 条断言仍 PASS（不退步）
make verify-adr-0001

# (7) 旧表清理（24h 兜底窗口已过）
docker exec ant-clickhouse clickhouse-client --query "DROP TABLE IF EXISTS ant.md_ticks_legacy"
docker exec ant-clickhouse clickhouse-client --query "DROP TABLE IF EXISTS ant.md_bars_legacy"
```

---

## M11 · 金融特性（纯增量，不改造基础架构）

> **设计输入**：
> - `docs/金融架构改造-M11路线图-2026-05-25.md`
> - `docs/对标开源项目调研-2026-05-25.md` §八（M11 卡片 × 开源项目矩阵）
> **对标开源**：**Lean** `Algorithm.Framework/Execution/`（VWAP / σ-based execution timing，Apache 2.0 友好）。**Hummingbot** `strategy_v2/executors/`（DCA executor 等组合模式）。
> **前置**：M10-BASE 全部 Phase A-F 关闭
> **性质**：本段卡片均为纯增量金融功能，依赖 M10-BASE 建立的基础架构（PositionSizer 管线、costsvc、15-state OMS、event ledger、MarketState）。
> **状态**：☑ 全部 5 张卡片已完成 (2026-05-26)

### M11 总览

| 卡片 | 内容 | 优先级 | 预估工日 |
|---|---|---|---|
| M11-13 | Execution Algo（TWAP/VWAP/POV/Implementation Shortfall，在 OMS 和 mthub 之间加 `execalgo/` 层） | P3 | 5 |
| M11-14 | FIFO/LIFO/HIFO 成本基础追踪（税务报告+精确 P&L 归因） | P3 | 2 |
| M11-15 | P&L 归因三维分解（信号质量 / 执行质量 / 持有成本，独立 metric + 时间序列） | P3 | 2 |
| M11-16 | Jurisdictional Gate（KYC + IP 检测 + 投顾牌照合规 + 免责声明 + 风险问卷） | P1 | 3 |
| M11-17 | Trade Reporting 预留（EMIR / MiFID II schema 预留，不实现完整报告） | P3 | 1 |
| **总计** | | | **~13 工日** |

### M11 卡片详表

| ID | 内容 | 文件 | 验收 | 备注 |
|---|---|---|---|---|
| M11-13 | 🅒 Execution Algo 层：TWAP / VWAP / POV / Implementation Shortfall；在 OMS 和 mthub 之间加 `execalgo/`；SaaS 聚合 1000 用户同一信号后拆成多笔小单 | `backend/internal/execalgo/{twap.go,vwap.go,pov.go,shortfall.go,algo.go,*_test.go}` | `go test -race -cover ./internal/execalgo/... \| awk '/coverage:/{gsub("%",""); if ($2<80) exit 1}'` | b33331a |
| M11-14 | 🅒 成本基础追踪（FIFO / LIFO / HIFO）：关闭仓位时匹配对应开仓、计算 realized P&L 按税务方法 | `backend/internal/costsvc/{cost_basis.go,cost_basis_test.go}` | `go test -race -run 'Test(FIFO\|LIFO\|HIFO)' ./internal/costsvc/... -v` | 9d2ebc1 |
| M11-15 | 🅒 P&L 归因三维分解：`Net P&L = Gross P&L(signal) - Slippage(execution) - Commission - Swap(holding)`，每个维度独立 metric | `backend/internal/oms/{pnl_attribution.go,pnl_attribution_test.go}` | `go test -race -run TestPnLAttribution ./internal/oms/... -v` | 9d2ebc1 |
| M11-16 | 🅒 地缘合规 Gate：KYC + IP 地理位置检测 + 受限地区拒止 + 显式免责声明 + 用户风险问卷 | `backend/internal/risksvc/{jurisdiction.go,jurisdiction_test.go}` | `go test -race -run TestJurisdictionGate ./internal/risksvc/... -v` | ebb361d |
| M11-17 | 🅒 Trade Reporting schema 预留：EMIR / MiFID II 类报告字段在 trade_event_log 中预留，不实现完整报告 | `backend/internal/mdgateway/chmigrate/014_trade_reporting_fields.sql` | `docker exec ant-clickhouse clickhouse-client --query "DESCRIBE ant.trade_event_log" \| grep -E 'reporting_' | 2080665 |

### M11 关闭判据

```bash
# 全部卡片 ☑
grep -E '^\| M11-' docs/plan/ROADMAP.md | grep -c '🅒' | awk '$1>0{print "PENDING:"$1; exit 1}'

# 执行算法回测验证（TWAP 滑点 < 市价单 50%）
go test -tags=integration -run TestTWAPvsMarket ./internal/execalgo/... -v

# P&L 归因三个维度均有非零值
go test -tags=integration -run TestPnLAttribution ./internal/oms/... -v
```

---

## 历史

- v2.0 (2026-05-23)：完全重写 ROADMAP；v1 归档至 `docs.old/plan/ROADMAP.md`
- v2.1 (2026-05-25)：合并 M11 金融架构设计到 M10-BASE；M8/M9 合并到 M10-BASE Phase；新增 M11 纯增量段
