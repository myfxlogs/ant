# AGENT.md — ant (v2 · MT 重写期)

> 仓库 `/opt/ant/` · 文档版本 v2 · 2026-05-23
> v1 文档已归档至 `docs.old/`、`AGENT.md.v1.bak`
> 本文档是 **AI Agent 唯一约束源**。所有冲突以本文档 + `docs/` 下版本最新文件为准。

---

## 0. 角色与执行模式（重读三遍）

### 0.1 谁来写代码

> **本仓库所有代码由 AI Agent 实现。人类只做：① 评审文档；② 前端验收；③ 决策审批；④ 不写代码。**

含义：
- **文档不能有歧义**：模糊描述 = AI 走偏 = 返工
- **验收必须可机械执行**：所有验收步骤用 shell 命令表达，不允许"人工目测"
- **每张卡片 = 一个独立 PR**：AI 一次执行一张卡片，commit 完成后**自动继续下一张**（不等待人类 ack），除非：
  1. 卡片验收命令失败且无法自排障 → 写明"🅒 阻塞 + 已尝试方案"停下
  2. 文档指引缺失/矛盾 → 停下报告
  3. 跨 milestone 依赖不满足 → 停下报告
- **AI 不许擅自跨范围**：超出当前卡片的代码改动，无论多"顺手"，都禁止合入同一 PR

### 0.2 AI 执行流程（每张卡片必走）

```
1. 阅读卡片 → 列出输入/输出/依赖
2. 阅读 docs/spec/ + docs/adr/ 中卡片引用的所有文件（无遗漏）
3. 运行卡片"前置检查"shell（必须全过）
4. 实施代码改动（仅限卡片范围）
5. 跑卡片"验收命令"（全部退出码 0）
6. 把验收 stdout 追加到 docs/handover/verify-MNNN.log
7. git commit (Conventional Commits + Verify: 引用 log 行号)
8. 把卡片状态从 🅒 改为 ☑（在 ROADMAP.md / BACKLOG.md）
9. STOP，等人类 review
```

**违反任何一步 → 卡片自动降回 🅒，禁止再宣称完成**（详见 §11 防偷懒约束）。

---

## 1. 项目身份

**ant**：用户驱动的智能量化交易平台（MT4/MT5 + AI 策略生成 + 策略市场）。

- **当前阶段**：M7-rewrite（地基重做：MT 接入 / ClickHouse 时序 / 因子 DSL / quantengine / order hub），路径 B（地基重做 + 业务渐进重构）
- **不重做**的范围：AI 助手、策略市场、admin、auth、worker、frontend、user/tenant 业务表
- **重做**的范围：见 `docs/plan/ROADMAP.md` §M7-rewrite

参考蓝本：`/opt/alfq/`（同 owner，代码可自由复用；但**不允许大块原样拷贝，必须改写适配 ant 包名/接口/schema**）。

---

## 2. 三域 + 五容器

```
backend/         Go 1.26 + ConnectRPC + sqlc        → ant-backend (8080, 集群内)
strategy-service/ Python 3.14 + uv (研究模式沙箱)    → ant-strategy-service (8081, 集群内)
frontend/        React 19 + TS 5.9 + pnpm + Tailwind → ant-frontend (Nginx, 宿主 ${ANT_FRONTEND_PORT:-8022})
                                                     → ant-postgres (5432)
                                                     → ant-redis (6379)
                                                     → ant-clickhouse (9000) ← v2 新增
                                                     → ant-nats (4222) ← v2 新增
```

- **proto 单源** `proto/ant/v1/` → `buf generate` 出 Go/TS/Python stub
- **Go module 名**：`anttrader`（历史原因保留，与项目身份 `ant` 不同）。所有 import 路径以 `anttrader/...` 开头，仓库路径 `/opt/ant/`、容器名 `ant-*`。这 3 者不要混淆。
- **不引入** K8s / Helm / Service Mesh / 多副本
- **运行时基线**（仅 ADR 可变）：Go 1.26 / Python 3.14 / Node 24 LTS / TS 5.9 / PG 18 / Redis 8 / ClickHouse 24 / NATS 2.10

---

## 3. 硬性规则（违反 = PR 拒绝）

### 3.1 协议
- 对外：**ConnectRPC + SSE**。禁止新增 REST（除 `/healthz` `/readyz` `/livez` `/metrics`）。禁止 WebSocket。
- 内部：进程内函数调用或 NATS JetStream（异步）。

### 3.2 数据
- **业务库**：PostgreSQL 18（用户、账户绑定、订单、风控、AI、市场、审计）
- **时序库**：ClickHouse 24（tick / bar / factor_value / signal），见 `docs/spec/13-clickhouse-schema.md`
- **缓存/锁**：Redis 8
- **MQ**：NATS JetStream（行情 fan-out + 因子触发）
- **PG ↔ CH 边界**：见 `docs/adr/0002-clickhouse-as-timeseries.md`

### 3.3 MT 接入
- mtapi gRPC 是**唯一**的 MT 接入路径。**禁止新增对 `internal/mt4client` `internal/mt5client` 的 import**（这两个包将在 M7-rewrite 完成后删除）。
- mt4 与 mt5 是两套独立协议：`adapter/mt4/` `adapter/mt5/` 不许共享代码（除 `adapter/mdtick/` 共享 DTO）。

### 3.4 安全红线
- 用户 Python 代码**只在研究模式**沙箱中执行（`strategy-service`）。
- **生产路径**（实盘下单）禁止任何 Python 代码执行。生产策略 = DSL 字符串 + ONNX 模型引用，由 `quantengine` 加载。
- 见 `docs/adr/0003-direct-mtapi-no-wrapping.md` §"沙箱降级"（原误标 ADR-0005）。

### 3.5 数据归属（ADR-0006）
- **平台共享层**（`platform_strategies` `platform_factors` `platform_ai_agents` `broker_symbols` `admins`）：无 `user_id`，所有用户可读
- **用户私有层**（`mt_accounts` `user_strategies` `user_factor_overrides` `user_ai_agents` `orders` `positions` `trades` `user_subscriptions` `copy_trade_links`）：必须 `user_id` 外键 + RLS
- **禁止**：在 user 表中复制官方/平台数据（per-user seed 模式禁止）
- **禁止**：用 `users.role='admin'` 鉴权平台运营；走独立 `admins` 表 + JWT scope `platform:admin`
- 详见 `docs/adr/0006-platform-shared-vs-user-private.md`

### 3.6 前后端职责
- 所有业务计算在后端完成；前端仅展示。
- 后端对前端**零信任**——所有输入独立校验。
- 数字格式化、货币计算、状态推断**一律后端处理**后返回展示值。

### 3.7 类型与精度
- 价格：`NUMERIC(20,8)` (PG) / `Decimal(18,6)` (CH) / `decimal.Decimal` (Go) / `Decimal` (Python)。**禁止 float64 直接比较或参与价格计算**。
- 时间：UTC，毫秒精度（`int64 ts_unix_ms`）。
- 日志：结构化 JSON，必带 `trace_id` `user_id` `request_id` `account_id`（涉及账户时）。

### 3.8 部署
- **单机 docker-compose**，命名前缀 `ant-`。
- 不许破坏 `anttrader`（独立项目，端口/卷/网络隔离）。

---

## 4. 文档权威源

> **冲突时按下表第一行顺序裁定，下行让上行。**

| 优先级 | 主题 | 文件 |
|---|---|---|
| P0 | 项目身份与硬规则 | `AGENT.md`（本文档）|
| P0 | ADR（不可逆决策） | `docs/adr/NNNN-*.md` |
| P1 | 架构总览 | `docs/architecture/*.md` |
| P1 | 模块规范 | `docs/spec/*.md` |
| P2 | 实施计划 | `docs/plan/ROADMAP.md` |
| P2 | 待办与缺陷 | `docs/plan/BACKLOG.md` |
| P3 | 应急手册 | `docs/runbook/*.md` |

冲突处理：选**优先级高**的；同级别选**编号大/时间新**的；不能仲裁时**停下来报告人类**，禁止自行裁决。

---

## 5. B 路线长期硬指标（必须达成才能关闭 milestone）

| 指标 | M7 完成 | M8 完成 | M9 完成 |
|---|---|---|---|
| **MT 接入非测试 LOC**（验收命令 `find ... -not -name "*_test.go"`）| ≤ 1500 | ≤ 1200 | ≤ 1000 |
| **业务代码 grep 直调 mt4client/mt5client** | 0 处 | 0 处 | mt4client/mt5client 包已删除 |
| **service/ 包文件 ≤ 400 行** | 不要求 | 100% 文件达标 | 100% 文件达标 |
| **sqlc 覆盖率** | 不要求 | ≥ 80% | ≥ 95% |
| **CH tick 写入零丢失（连续 7 天）** | ✅ | ✅ | ✅ |
| **生产路径 Python 执行 grep** | 0 处 | 0 处 | 0 处 |

任一指标不达标 → milestone 不许标 ☑。

---

## 6. 复杂度硬上限（CI 强制）

| 维度 | Go | Python | TypeScript |
|---|---|---|---|
| 单文件行数 | ≤ 300 | ≤ 400 | ≤ 250 |
| 单函数行数 | ≤ 50 | ≤ 50 | ≤ 50 |
| 圈复杂度 | ≤ 10 | ≤ 10 | ≤ 10 |
| 函数参数数 | ≤ 5 | ≤ 5 | ≤ 5 |
| 嵌套深度 | ≤ 4 | ≤ 4 | ≤ 4 |

- 严禁 `//nolint` `# noqa` `// @ts-ignore`（特殊场景见 `tools/lint/baseline.json`）
- 单 PR ≤ 800 行业务代码（生成代码 / YAML / Dockerfile 不计入）
- baseline 列表中已有违规不阻塞；新增违规一律 CI red

### 6.5 Proto 源文件 / 生成代码规则

| 维度 | 规则 |
|---|---|
| 单个 `.proto` 文件 | ≤ 300 行 |
| 单个 `service` 的 RPC 数 | ≤ 15（超过即拆分 service） |
| 单个 `message` 字段数 | ≤ 30（超过即聚合根承担多职责，拆 message） |
| 生成代码（`backend/gen/proto/` `frontend/src/gen/`） | **不限大小，禁止手改** |
| PR 中生成代码行数 | 不计入 800 行预算 |

**生成代码必须由 `make proto` 完全重新生成**。CI 校验：

```bash
make proto
git diff --exit-code -- backend/gen/proto frontend/src/gen \
  || { echo "FAIL: gen/ has hand-edits"; exit 1; }
```

新增/改动 `.proto` 必须跑 `make proto-breaking` 通过（不允许 breaking change，除非 ADR 明示）。

---

## 7. 工程纪律（10 条）

1. **单一职责**：handler 只编排；业务在 service；数据访问在 repo
2. **接口驱动**：跨包边界先定 interface，后实现
3. **代码生成优先**：RPC = buf；SQL = sqlc；TS 类型 = buf 自动生成
4. **三处下沉**：同一逻辑出现 3 次 → 抽到 `internal/common/`
5. **错误集中**：用 `internal/errs/`，**禁止裸字符串错误**
6. **状态机外置**：订单/连接等显式状态机，不许散落 if/else
7. **零循环依赖**：CI 强制（go list / madge）
8. **canonical 入口规范化**：所有 (broker, symbol_raw) 在 adapter 出口转 canonical；下游禁止再做 symbol 转换
9. **价格不丢精度**：PG `NUMERIC(20,8)` ↔ Go `decimal.Decimal` ↔ CH `Decimal(18,6)`，转换在 ORM 层完成
10. **可观测性默认开**：每个 service 必须暴露 Prometheus metrics + structured log + healthz

---

## 8. 编码规范

### 8.1 Go
- gofumpt + golangci-lint（v1.62+）+ `go test -race`
- zap 日志；`ctx context.Context` 永远是首参；禁 panic（除 main 启动失败）
- 包名 snake_case；导出符号 PascalCase；私有 camelCase
- 文件头注释：`// Package <name> ...`（一行说明）

### 8.2 Python（仅 strategy-service / research）
- ruff strict + mypy strict（`disallow_untyped_defs = true`）
- loguru；强制类型注解
- 模块名 snake_case

### 8.3 TypeScript（frontend）
- strict mode；禁 any（必要时 `unknown` + 类型守卫）
- TanStack Query + Zustand；Tailwind；shadcn/ui
- 文件名 kebab-case

### 8.4 通用
- 依赖白名单见 `docs/spec/dep-allowlist.md`（待建）；新增依赖必须 ADR
- AGPL 代码禁入仓
- secrets 永不入仓；`.env` 在 gitignore

---

## 9. 提交与分支

### 9.1 Conventional Commits
```
type(scope): subject

[optional body]

Verify: docs/handover/verify-MNNN.log:<起始行>-<结束行>
[optional footer]
```

`type ∈ {feat, fix, refactor, docs, test, chore, perf, build, ci}`

### 9.2 分支
- `main` 受保护，禁止直接 push
- 工作分支：`<type>/<scope>/<short-desc>`（例：`feat/mdgateway/runner`）
- 一张卡片 = 一个分支 = 一个 PR

### 9.3 PR
- 必填：关联卡片 ID、验收日志路径、风险评估
- AI 自检 7 条（§11）全过才允许提交
- 人类 review 通过后 squash merge

---

## 10. 禁止清单

- ❌ main 直接 push / force push 共享分支 / `--no-verify`
- ❌ REST 新接口（除 healthz/readyz/livez/metrics）
- ❌ WebSocket
- ❌ 用户 Python 代码访问生产 DB / 网络
- ❌ 生产路径调用 Python 解释器
- ❌ 业务代码直接 import `internal/mt4client` `internal/mt5client`（v2 起）
- ❌ 业务代码直接读写 `kline_data` `tick_data` 等老 PG 行情表
- ❌ float64 参与价格运算
- ❌ proto 改动不跑 buf breaking
- ❌ 硬编码秘钥 / .env 入仓 / >50MB 文件入仓
- ❌ AGPL 代码复制
- ❌ 跨里程碑卡片实施（一次只做一张）
- ❌ 凭常识决定安全/合规（必须 ADR 或 spec 引用）
- ❌ 破坏 anttrader 生产环境（端口/卷/容器名冲突）

---

## 11. 防偷懒约束（8 条强制）

落地任何卡片必须遵守以下 8 条；违反任一条 = "假完成"，卡片自动降回 🅒。

### 11.1 物证留痕

每张卡片在 `docs/handover/verify-<card_id>.log` 留 ≥ 20 行连续真实 stdout，含 UUID / 时间戳 / 行数。文档改字 + 不带验收日志 = 失败。

```bash
<verify_cmd> 2>&1 | tee -a docs/handover/verify-<card_id>.log
```

### 11.2 验收命令禁改

ROADMAP / BACKLOG / spec 中写的 `psql -c` `clickhouse-client --query` `grpcurl -d` 等命令是**契约**。AI 改命令使其更"好通过"= 整轮失败。

### 11.3 24h 重跑可复现

卡片 commit 后 24h 内人类可挑任意 3 张已 ☑ 的卡片原样重跑：
- 非时间窗 SQL：结果非 0 且与原贴数量级相符（±10%）
- 时间窗 SQL：仍非 0
- UUID/ticket 具体值：原样 SELECT 仍能查到

任一不满足 → 整轮作废、所有 ☑ 降回 🅒。

### 11.4 禁 mock / stub 顶替真实依赖

凡涉及 broker 真实下单、CH 真写表、MT 真返回 ticket，必须用真实容器 + 真实账户。

```bash
BASE=$(git merge-base origin/main HEAD)
git diff --name-only "$BASE"...HEAD \
  | grep -E '^(backend/(cmd|internal)|strategy-service/(src|app)|frontend/src)/' \
  | grep -vE '_test\.go$|test_.*\.py$|/testutil/|fixtures/|broker_sim\.py' \
  | xargs -r grep -lE '\b(mock|stub|[Ff]ake[A-Z])' 2>/dev/null \
  && echo "FAIL: production code references mock/stub/Fake" && exit 1 || true
```

例外白名单：`strategy-service/src/broker_sim.py` / `internal/testutil/` / proto 生成代码 / Connect interceptor noop。

### 11.5 禁删 / 弱化测试

```bash
BASE=$(git merge-base origin/main HEAD)
git diff --numstat "$BASE"...HEAD -- '*_test.go' 'test_*.py' \
  | awk '{add+=$1; del+=$2} END{ if (del>add) exit 1 }'
```

新增功能必须配回归测试。`t.Skip` / `pytest.skip` / `xfail` 不得净增。

### 11.6 卡片状态机闭环

每个 ☑ 必须满足：
- 有对应 commit short-sha（在卡片表"备注"列）
- commit 实际改动 ≥ 卡片预估的 30%（`git show --stat` 统计，proto 生成代码除外）
- commit body 含 `Verify: docs/handover/verify-<id>.log:<行>-<行>`

### 11.7 自检脚本（合入前必跑）

```bash
set -euo pipefail

# (a) ROADMAP/BACKLOG 当前里程碑无 🅒 / TODO / FIXME
PLAN=docs/plan/ROADMAP.md
grep -cE '^🅒|TODO|FIXME|XXX-hack' "$PLAN" | awk '$1>0{exit 1}'

# (b) 当前卡片验收日志存在且行数 ≥ 20
test -f "docs/handover/verify-${CARD_ID}.log"
wc -l "docs/handover/verify-${CARD_ID}.log" | awk '$1<20{exit 1}'

# (c) build & test
( cd backend && go build ./... && go test -race ./internal/... )
( cd strategy-service && uv run pytest -q )

# (d) 复杂度
make lint

# (e) 关键运行时（容器在线时）
if docker inspect ant-postgres >/dev/null 2>&1; then
  docker exec ant-postgres psql -U ant -d ant -tAc "SELECT 1" | grep -q 1
fi
if docker inspect ant-clickhouse >/dev/null 2>&1; then
  docker exec ant-clickhouse clickhouse-client --query "SELECT 1" | grep -q 1
fi
```

任一行非 0 退出 → 自动降回 🅒，禁止 commit。

### 11.8 迁移完整性

从 alfq 移植任何模块必须迁移**全部设计文件**，禁止只搬部分：
- 设计文档列出 N 个文件 → 实际写入必须 = N
- 禁止用临时 hack 顶替正式实现
- 依赖链不许断裂

```bash
MODULE_DIR=backend/internal/<module>
for f in $(grep -oE '^\| `[^`]+\.go`' docs/spec/<module>.md | tr -d '|`' | xargs); do
  test -f "$MODULE_DIR/$f" || { echo "MISSING: $f"; exit 1; }
done
go build "./$MODULE_DIR/..."
```

### 兜底原则

> **"代码已写" ≠ "卡片完成"。卡片完成 = 代码 + 测试 + 真实运行时 stdout 三者齐全且可复现。**
>
> 卡片做不通时，坦诚降级写明"🅒 + 阻塞原因 + 已尝试方案"并停下汇报。**禁止用文档改字、mock 替换、放宽验收命令绕过。**

---

## 12. AI 文档阅读清单（每次开工必读）

按顺序读完才能开始执行任何卡片：

1. `AGENT.md`（本文件）
2. `docs/architecture/01-vision.md`
3. `docs/architecture/02-overview.md`（**§8 不变量 11/12/13 是数据归属宪法，必读**）
4. `docs/architecture/03-data-flow.md`
5. **`docs/adr/0006-platform-shared-vs-user-private.md`**（C2C 数据归属决策；M8 起所有业务卡必读）
6. **`docs/adr/0008-storage-dedup-and-time-axis.md`**（M10 起所有数据基础卡必读）
7. **`docs/adr/0009-replay-dual-write-and-bar-finality.md`**（同上）
8. **`docs/adr/0010-slo-alert-dlq-trace.md`**（同上）
9. **`docs/adr/0011-capacity-vault-cache-hardening.md`**（同上）
10. M10 新增 spec：`docs/spec/18-backfiller.md` `docs/spec/19-md-doctor.md` `docs/spec/20-slo.md`
11. 卡片所在 milestone 的全部 `docs/spec/*.md`
12. 卡片引用的所有 `docs/adr/*.md`
13. `docs/plan/ROADMAP.md` §当前 milestone
14. `docs/spec/16-mtapi-quirks-register.md`（涉及 MT 时）
15. `docs/spec/17-secrets-and-errors.md`（涉及加密/错误处理时）

读完后**回答自检 5 问**（在 PR 描述中写明）：
1. 本卡片要改动哪些文件？（精确路径列表）
2. 本卡片的输入是什么？（上游卡片 / 配置 / 数据）
3. 本卡片的输出是什么？（接口 / 表 / 文件）
4. 本卡片的验收命令是什么？（精确 shell）
5. 本卡片可能踩哪些坑？（参考 quirks register）

5 问任一答不上来 → 暂停，回去读文档。

---

## 13. Makefile（标准入口）

```
make proto          buf lint + breaking + generate
make build          go build ./...
make test           go test -race ./internal/... + uv run pytest
make lint           gofumpt + golangci-lint + ruff + mypy + tsc
make migrate-pg     PostgreSQL 迁移
make migrate-ch     ClickHouse 迁移（v2 新增）
make docker-up      docker-compose up -d
make docker-down    docker-compose down
make docker-logs    docker-compose logs -f
make verify-card    跑 §11.7 自检脚本（CARD_ID=M7.X-Y）
```

---

## 14. 当前阶段速览

- **里程碑**：M7-rewrite（MT 地基重做）
- **路径**：B（地基重做 + 业务渐进重构）
- **入口**：`docs/plan/ROADMAP.md` §M7
- **测试覆盖目标**：M7 完成 ≥ 30%，M8 完成 ≥ 50%
- **当前未关闭的 ADR**：见 `docs/adr/README.md`
- **当前活跃 quirks**：见 `docs/spec/16-mtapi-quirks-register.md`

---

> 最后一行原则：**当 AI 在文档中找不到明确指引时，默认行为 = 停下报告人类，而不是自行决策。**
