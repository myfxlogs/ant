# AGENT.md — ANT

> 工作仓库 `/opt/ant/` | M0 启动阶段 | 2026-05-19
>
> 🤖 **AI Agent 第一次进仓库**：本文件是当前唯一执行入口。
> `docs/tasks/AGENT-RUNBOOK.md`（阅读顺序 / 执行循环 / 卡住升级路径）**待建立**；建立前以本文为唯一约束源。
>
> 📌 **项目定位**：以 ant 为主体，从 AlfQ 选择性迁移优秀功能并优化。
>
> - **过程文档**（plan / ADR / 规范 / 验收脚本）全新独立编写，不复用 AlfQ 旧文档。
> - **功能实现**：可以 AlfQ 代码/设计为蓝本参考。迁移任何一块功能时必须：出 ADR 说明为何迁 + 独立重写或适配本仓架构 + 补测试；不允许原样大块拷贝。
> - AlfQ 原始资料是「输入」不是「约束」：ant 仓库内部本文档 + 本仓 ADR 才是唯一约束源。

## 项目身份

用户驱动的智能量化交易平台（Go + Python + React），基于 AI 策略生成与 Python 沙箱执行。面向普通散户，支持多交易账号绑定、自然语言策略生成、策略市场交易。设计原则：**先抄后改**。

## 三域结构

`backend/`（Go 服务 + proto）| `strategy-service/`（Python 沙箱执行，pip + requirements.txt）| `frontend/`（React SPA，pnpm）

- Go 1.26 / Python 3.14 / TypeScript 5.9 / Node 24 LTS（运行时主版本基线，仅 ADR 可锁）
- Proto 单一源 `proto/ant/v1/` → `buf generate` 出 Go/TS/Python stub
- 5 服务（容器内端口；宿主仅暴露 frontend）：`ant-frontend` 容器 8080（Nginx 静态托管 + `/api/` 反代，宿主端口 `${ANT_FRONTEND_PORT:-8022}`）、`ant-backend` 8080（业务 API + OMS + 风控，仅集群内）、`ant-strategy-service` 8081（Python 沙箱，仅集群内）、`ant-postgres` 5432、`ant-redis` 6379
- 部署形态：单机 docker-compose（独立于 anttrader 生产环境，命名/网络/卷统一 `ant-*` 前缀）

## 硬性规则

**协议**：Connect RPC + SSE（Server Streaming）。禁止 REST 新接口、禁止 WebSocket。内部 gRPC，异步走 Redis。

**数据**：PG 18（主数据）+ Redis 8（缓存/锁）。时序数据（如需）方案待 ADR 决策。

**MT4 vs MT5**：两套完全独立的协议/平台，proto 定义、枚举语义、撮合模型均不可共用。`adapter/mt4/` 与 `adapter/mt5/` 设计待立项。

**安全红线**：用户 Python 代码仅在 `strategy-service` 沙箱中执行，禁止直接访问生产数据库或外部网络。沙箱实现方案（DSL / ONNX / 受限解释器等）待 ADR 决策。

**前后端职责**：所有业务计算在后端完成，前端仅负责展示和渲染。后端对前端零信任——所有输入必须独立验证。数字格式化、货币计算、状态推断、数据转换等逻辑一律在后端执行后返回最终展示值。

**价格**：`NUMERIC(20,8)` / decimal，禁止 float64 直接比较。时间统一 UTC。

**日志**：结构化 JSON，必带 `trace_id` `user_id` `request_id`。

**版本**：默认追随官网最新稳定版。当前运行时基线：Go 1.26 / Python 3.14 / Node 24 LTS / PostgreSQL 18 / Redis 8 / TypeScript 5.9。主版本仅允许通过 ADR 锁版，理由必须在 ADR 中写明；其余工具/库不得保留旧版本（除非有明确兼容性问题并记录）。

**部署形态**：**单机 docker-compose**。不引入 K8s/Helm/ArgoCD/Service Mesh/HPA/多副本。容器隔离细节见上文「三域结构」。

## ADR（不可逆）

本项目独立编写 ADR（AlfQ 可作设计参考，但不直接引用其 ADR 编号/结构）。需要立项的初始决策清单：

- Connect RPC + SSE 通信协议
- 三域 monorepo（backend / strategy-service / frontend）
- PostgreSQL + Redis 主存方案
- 用户 Python 沙箱隔离方案
- 用户中心架构（非多租户）
- sqlc 优先，不引入 ORM
- AI 策略生成的 bounded tools 设计
- 单机 docker-compose 生产部署形态
- 运行时主版本基线锁版（PG 18 / Go 1.26 / Python 3.14 / Node 24 LTS / Redis 8 / TypeScript 5.9）

新增决策 → `docs/adr/NNNN-<slug>.md`，编号从 `0001` 单调递增。

## 文档唯一源

不同文档冲突时，以下为权威：

| 主题 | 唯一源 |
|---|---|
| 项目实施计划（里程碑） | **待建立** `docs/plan/ROADMAP.md` |
| AlfQ 功能迁移清单（输入蓝本，非约束） | `docs/AlfQ功能迁移计划.md`（只读参考；实际里程碑以 ROADMAP 为准） |
| 订单状态机 | **待建立** `docs/domain/订单状态机.md` |
| 全量错误码 | **待建立** `docs/错误码与异常处理规范.md` |
| 数据库设计与表索引 | **待建立** `docs/数据库设计.md` |
| 权限角色 | **待建立** `docs/权限设计.md` |
| NFR（NFR ≥ SLO ≥ SLA） | **待建立** `docs/总体架构与技术决策.md` |
| 依赖白名单 | **待建立**（建立前：新增依赖必须 ADR 立项，禁 AGPL） |
| 复杂度上限 | 本文档 §复杂度硬上限（CI 强制） |
| Proto 包结构 | **待建立** `docs/API与接口规范.md` §2 |
| ADR 索引 | **待建立** `docs/adr/README.md` |

冲突处理：选编号大的（更新的）+ PR 中指出。

## 复杂度硬上限（CI 强制）

| 维度 | Go | Python | TS |
|---|---|---|---|
| 单文件行数 | ≤300 | ≤400 | ≤250 |
| 单函数行数 | ≤50 | ≤50 | ≤50 |
| 圈复杂度 | ≤10 | ≤10 | ≤10 |
| 函数参数 | ≤5 | ≤5 | ≤5 |
| 嵌套深度 | ≤4 | ≤4 | ≤4 |

严禁 `// nolint`。PR ≤ 800 行业务代码（生成/YAML/CI/Dockerfile 不计入）。

### 存量违规分层处理（避免反 ROI 的全量重构）

仓库存量 ~254 处违规（Go 文件 52 / TS 文件 37 / Go 函数 165）按以下分层处理：

- **Tier 1 — 必拆**：M1-M5 热路径文件 + 文件 >500 行 或 函数 >100 行。**进入对应 milestone 时作为前置工作量拆到上限以内**，不单独立项；清单见 `docs/plan/ROADMAP.md` §M0.3 Tier 1 热路径清单。
- **Tier 2 — Boy Scout**：路径上的 300–500 行文件 / 50–100 行函数。任何卡片触碰时**顺手拆，合并到该卡片提交**。
- **Tier 3 — 永久豁免**：① i18n 资源文件（`**/i18n/**`）；② proto 生成代码（`**/gen/**`）；③ 数据/常量/默认值文件（`**/*defaults.ts` `**/*const*.go`）；④ 单一职责的大型 model 集中定义（`backend/internal/model/models.go` 等，独立 ADR 评估后可上调到 ≤500）。

### CI 强制方式

- 维护 `tools/lint/baseline.json` 冻结**初版** 254 处违规作为 waiver
- 任何 PR 触碰文件后若该文件仍超线 → CI red（即 baseline 列表中的项**不允许新增**，已存在的不阻塞）
- 新文件 / 新增函数零容忍：超线一律 CI red
- baseline 项目须每月 review 一次：减少了多少、增加了多少（违规净增 → 工程债增长信号）

## 工程纪律

1. 单一职责 — Handler 只编排，业务在 service
2. 接口驱动 — 跨边界先 interface
3. 代码生成优先 — RPC: buf / SQL: sqlc / 前端类型: buf
4. 三处下沉 — 重复 3 次 → `internal/common/`
5. 错误集中 — `errs` 包，禁裸字符串
6. 状态机外置 — 订单/连接等显式状态机
7. 零循环依赖 — CI 检测

## 编码要点

- **Go**：gofumpt+golangci-lint, zap 日志, `ctx` 首参, 禁 panic, `go test -race`
- **Python**：ruff+mypy strict, loguru, 类型注解强制（strategy-service）
- **TS**：strict mode, 禁 any, TanStack Query + Zustand, Tailwind
- **通用**：Go snake_case / Py snake_case / TS kebab-case · 依赖白名单待建立，新增依赖一律走 ADR · 禁 AGPL 入仓

## 提交与 PR

Conventional Commits: `type(scope): subject`。分支: `feat|fix|chore|docs|refactor|test/<scope>`。main 保护，PR + 2 reviewer。PR 必带：关联文档、测试结果、风险评估。

## 当前阶段：M0（启动阶段）

ant 是主体项目；**过程文档**全新独立，**功能**从 AlfQ 选择性迁移。`backend/` `frontend/` 中从 anttrader 带过来的代码作为初始脚手架，迁移/调整任何一块都走 ADR 立项 → 重写/适配 → 补测试 三步，不允许原样大块复制 AlfQ 代码。

仓库现状：

- `backend/`：Go 后端服务（脚手架，待按 ADR 逐块迁移/重构）
- `frontend/`：React SPA（脚手架，待按 ADR 逐块迁移/重构）
- `strategy-service/`：Python 沙箱策略执行服务（脚手架）
- `docker-compose.yml`：已隔离 anttrader 命名冲突，统一 `ant-*` 前缀
- `.env`：已配置 `JWT_SECRET`、`DB_PASSWORD` 等敏感配置（不入仓）

M0 待办：

- 起草 `docs/plan/ROADMAP.md`（参考 `docs/AlfQ功能迁移计划.md` 里的功能选项，但里程碑、验收口径、优先级由 ant 独立定义）
- 起草 `docs/adr/0001-*.md` 起的初始 ADR 序列（见上文 ADR 章节清单）
- 建立 `docs/tasks/AGENT-RUNBOOK.md` 执行入口

## Makefile

```
make proto          # buf lint + breaking + generate
make build / test / lint
make go-lint / go-test / go-build
make py-lint / py-test
make web-lint / web-build
make docker-up / docker-down / docker-logs
make migrate        # 数据库迁移
```

## 禁止

- main 直接 push · force push 共享分支 · `--no-verify`
- REST 新接口（除 healthz/metrics）· WebSocket
- 用户 Python 代码直接访问生产数据库或外部网络
- proto 不跑 buf breaking
- 硬编码秘钥 · .env 入仓 · >100MB 入仓
- AGPL 代码复制 · 跨里程碑实施 · 凭常识决定安全/合规
- 破坏 anttrader 生产环境（命名冲突、端口冲突、数据冲突）

---

## 防偷懒约束（强制）

落地 `docs/tasks/*.md` / `docs/audit/*.md` 中任何卡片必须遵守以下 7 条；违反任一条即视为"假完成"，相关卡片自动降回 🅒，禁止再宣称完成。

### 1. 物证强制留痕

每条卡片必须在 `docs/handover/RS-final-verify.log` 留下 ≥20 行连续真实 stdout，含可复现的 UUID / 时间戳 / ticket 号 / 行数计数。文档仅改字、不带验收日志 = 失败。

所有验收命令统一以下列形式留痕：
```bash
<verify_cmd> 2>&1 | tee -a docs/handover/RS-final-verify.log
```

### 2. 验收命令禁止改动

plan / audit 里写的 `psql -c "..."` / `grpcurl -d ...` / `docker exec ...` 等验收命令是契约。Agent 若改命令使其更"好通过"（放宽 WHERE 条件、降低 COUNT 阈值、换更宽容的 SQL）= 整轮失败。

### 3. 重跑可复现（24h 窗口）

卡片完成提交后 24h 内，人类审查者随时挑任意 3 条已标 ☑ 的卡片，原样重跑其验收命令（命令中 `git diff` 类范围以 PR 的 `merge-base origin/main` 为基线，单 commit PR 可退化为 `HEAD~1`）。要求：

- 涉及非时间窗 SQL（`COUNT(*) FROM risk_events` / `broker_ticket IS NOT NULL` 等）：结果必须仍非 0、且与 Agent 贴的数量级相符（±10%）；
- 涉及时间窗 SQL（`WHERE created_at > now()-interval N`）：必须仍为非 0（系统在线持续产出），数量不要求精确一致；
- 涉及 UUID/ticket 等具体值：原样 SELECT 必须仍能查到对应行。

任一条不满足 → 整轮作废、所有 ☑ 卡片全部降回 🅒。

### 4. 禁止 mock / stub 顶替真实依赖

凡涉及"broker 真实下单 / risk_events 真写表 / strategy 真执行 / vault 真读 secret / MT 网关真返回 ticket"，必须用真实容器、真实账户、真实 PG/Redis。

```bash
# 仅检查生产路径，排除测试 / fixtures / broker_sim 等合法 stub
BASE=$(git merge-base origin/main HEAD)
git diff --name-only "$BASE"...HEAD \
  | grep -E '^(backend/(cmd|internal)|strategy-service/(src|app)|frontend/src)/' \
  | grep -vE '_test\.go$|test_.*\.py$|/testutil/|fixtures/|broker_sim\.py' \
  | xargs -r grep -lE '\b(mock|stub|[Ff]ake[A-Z])' 2>/dev/null \
  && echo "FAIL: production code references mock/stub/Fake" && exit 1 || true
```

例外白名单（合法 stub，不计入违规）：`strategy-service/src/broker_sim.py`（如需）、`internal/testutil/`、proto 生成代码、Connect RPC interceptor 链中的 noop。

### 5. 禁止删 / 弱化测试

```bash
# 测试文件净删除行数 > 净新增行数 → 失败（基线 = PR merge-base）
BASE=$(git merge-base origin/main HEAD)
git diff --numstat "$BASE"...HEAD -- '*_test.go' 'test_*.py' \
  | awk '{add+=$1; del+=$2} END{ if (del>add) exit 1 }'
```

新增功能必须配回归测试，否则该卡片不许标 ☑。`t.Skip()` / `pytest.skip` / `xfail` 数量相对 `merge-base origin/main` 不得净增。

### 6. 卡片状态机闭环检查

每个 `☑` 卡片必须满足：

- 有对应 commit hash（在卡片表格"备注"列写出短 sha）；
- commit 实际改动行数 ≥ 该卡片预估工作量的 30%（防"改一行注释就标完成"，行数按 `git show --stat` 统计，proto 生成代码不计入）；
- commit message 遵守上文 §提交与 PR 的 Conventional Commits 规则，并在 **commit body**（非 subject）中追加一行：
  ```
  Verify: docs/handover/RS-final-verify.log:<起始行>-<结束行>
  ```

对 audit 文档（`docs/audit/DESIGN-REVIEW-*.md`）的卡片，落地完成时把 heading 后追加 `[☑]` 标记，自检脚本据此统计完成度（见下条）。

### 7. 自检脚本卡口（落地前必跑全过）

以 `当前活跃的 plan/audit 文件` 为输入，下列每条非 0 退出即失败：

```bash
set -euo pipefail
PLAN=$(ls -t docs/tasks/REMEDIATION-PLAN-*.md docs/tasks/ROADMAP-*.md 2>/dev/null | head -1 || true)
AUDIT=$(ls -t docs/audit/DESIGN-REVIEW-*.md 2>/dev/null | head -1 || true)

# 无活跃 plan/audit 时本节空转（仍受其他 6 条约束）；存在则必须全过
if [ -n "${PLAN:-}" ]; then
  # (a) plan 中无未完成标记 / 待办占位
  grep -cE '^🅒|TODO|FIXME|XXX-hack' "$PLAN" | awk '$1>0{exit 1}'
fi

if [ -n "${AUDIT:-}" ]; then
  # (b) audit 文档卡片必须 100% 标完（heading 后追加 [☑]）
  total=$(grep -cE '^### (CR|MR|MN)-[0-9]+' "$AUDIT")
  done_n=$(grep -cE '^### (CR|MR|MN)-[0-9]+.*\[☑\]' "$AUDIT")
  test "$total" = "$done_n" || { echo "AUDIT incomplete $done_n/$total"; exit 1; }

  # (c) 验收日志行数下限（每卡片 ≥20 行 × 卡片总数）
  test -f docs/handover/RS-final-verify.log
  wc -l docs/handover/RS-final-verify.log | awk -v t="$total" '$1 < t*20 {exit 1}'
fi

# (d) 关键运行时断言（非 0 行）— 仅当容器在线时强制；CI 离线环境跳过
if docker inspect ant-postgres >/dev/null 2>&1; then
  PSQL=(docker exec ant-postgres psql -U "${DB_USER:-ant}" -d "${DB_NAME:-ant}" -tAc)
  "${PSQL[@]}" "SELECT COUNT(*) FROM strategies"             | awk '$1==0{exit 1}'
  "${PSQL[@]}" "SELECT COUNT(*) FROM accounts"               | awk '$1==0{exit 1}'
  "${PSQL[@]}" "SELECT COUNT(*) FROM orders WHERE state='filled'" | awk '$1==0{exit 1}'
fi
```

任一行非 0 退出 → 自动撤回所有 ☑、降回 🅒，禁止 git tag、禁止提交 handover。

### 兜底原则

> **"代码已写"≠"卡片完成"。卡片完成 = 代码 + 测试 + 真实运行时 stdout 三者齐全且可复现。**

若卡片确实做不通（依赖缺失、broker 限制、设计错误等），坦诚降级写明"🅒 + 阻塞原因 + 已尝试方案"并停下汇报；**禁止用文档改字、mock 替换、放宽验收命令绕过**。
