# ROADMAP（v2 · MT 重写）

> **状态**：M7-M9 已完成。**实际执行结果等价于路线 A（全量重写）**，详见 `docs/adr/0007-post-m7-retrospective.md`。
> **当前阶段**：前后端打通 → 行情数据可看 → 最小可登录 → 业务功能逐步恢复。
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

## M8（预留）· 业务层渐进重构

> 留给 M7 完成后立项；目标：业务代码风格统一、sqlc 覆盖 ≥ 80%、所有 service/* ≤ 400 行
> 规划见 `docs/plan/BACKLOG.md` §M8

## M9（预留）· 老包删除

> 目标：删除 `internal/mt4client` `internal/mt5client`、删除 `kline_data` 表
> 触发条件：M7 + M8 完成且业务代码 30 天 0 直调

---

## 历史

- v2.0 (2026-05-23)：完全重写 ROADMAP；v1 归档至 `docs.old/plan/ROADMAP.md`

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
| M10.4-1 | 🅒 chmigrate `009_md_buffer_tables.sql`；`clickhouse_writer.go` INSERT 改 `md_ticks_buffer`/`md_bars_buffer`；默认配置 QueueSize=50000 / MaxBatch=10000 / Flush=500ms（env 可覆盖） | `backend/internal/mdgateway/chmigrate/009_md_buffer_tables.sql` `clickhouse_writer.go` + 测试 | `docker exec ant-clickhouse clickhouse-client --query "SELECT engine FROM system.tables WHERE database='ant' AND name='md_ticks_buffer'" \| grep -q '^Buffer$' && grep -E 'QueueSize.*50000\|MaxBatchSize.*10000\|FlushInterval.*500' backend/internal/mdgateway/clickhouse_writer.go` |
| M10.4-2 | 🅒 100 账户负载测试（mock broker 25k tick/s × 5min）：`md_spill_writes_total == 0` 且 `md_e2e_latency_seconds` P99 < 500ms | `tests/loadtest/{mock_broker.go,load_100_accounts_test.go}` (build tag loadtest) | `go test -tags=loadtest -timeout 15m -run Test100AccountsNoSpill ./tests/loadtest/...`；测试函数内部必须包含 ADR-0011 §6 列出的两条断言：(1) `md_spill_writes_total == 0`；(2) `histogram_quantile(0.99, md_e2e_latency_seconds) < 0.5`。断言失败 → `t.Fatalf` 退出非 0 |
| M10.4-3 | ☑ `internal/secrets/` 重构 envelope encryption（version+dek_kid+nonce+wrapped_dek+ciphertext+tag）；保留旧格式自动迁移；`cmd/ant-vault rotate` CLI；`secrets.MasterProvider` 接口预留 KMS | `backend/internal/secrets/{vault.go,envelope.go,vault_legacy.go,master_provider.go,*_test.go}` `backend/cmd/ant-vault/main.go` `docs/spec/17-secrets-and-errors.md` | `go test -race -cover ./internal/secrets/... \| awk '/coverage:/{gsub("%",""); if ($2<90) exit 1}' && go run ./cmd/ant-vault rotate --dry-run \| grep -q rows_to_rewrite` |
| M10.4-4 | ☑ PG migration `102_broker_symbols_notify.up.sql`（trigger pg_notify）；`internal/mdgateway/normalizer_invalidator.go` LISTEN + 30s ticker fallback；`normalizer.go` 注入 invalidator | `backend/migrations/102_broker_symbols_notify.up.sql` (`+.down.sql`) `backend/internal/mdgateway/{normalizer_invalidator.go,normalizer.go}` + 测试 | `make migrate && go test -tags=integration -run "TestNormalizer(Invalidation\|ListenerFallback)" ./internal/mdgateway/... -v` |

### M10.5 · md-doctor + slo-report CLI

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M10.5-1 | ☑ `cmd/md-doctor/`：reconcile/bar-continuity/canonical-liveness/dlq-tail/all 五个子命令（spec/19）；text+json 输出；--strict | `backend/cmd/md-doctor/{main.go,reconcile.go,bar_continuity.go,canonical_liveness.go,dlq_tail.go,*_test.go}` | `cd backend && go build -o /tmp/md-doctor ./cmd/md-doctor/ && /tmp/md-doctor --help \| grep -E 'reconcile\|bar-continuity\|canonical-liveness\|dlq-tail\|all' \| wc -l \| grep -q '^5$' && /tmp/md-doctor all --window 10m --output json \| jq -e '.reconcile != null'` |
| M10.5-2 | ☑ `cmd/slo-report/`：从 Prometheus 拉指标计算 4 条 SLO（spec/20 §1）+ budget 消耗，markdown 输出；--strict；Prometheus recording rules `deploy/prometheus/rules.yml`（`md:up:1m` `md:availability:30d`） | `backend/cmd/slo-report/{main.go,*_test.go}` `deploy/prometheus/rules.yml` | `cd backend && go build -o /tmp/slo-report ./cmd/slo-report/ && /tmp/slo-report --window 1h --output text \| grep -E 'SLO-MD-[1-4]' \| wc -l \| grep -q '^4$' && promtool check rules deploy/prometheus/rules.yml` |

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
