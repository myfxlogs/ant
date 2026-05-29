# Reasonix 规则：ant 项目（M10 数据基础 A+ 硬化期）

> 整合自 `AGENT.md` + `AGENTS.md` + 项目 pinned memory
> 适用仓库：`/opt/ant/`

---

## 0. 项目身份

ant = 用户驱动的智能量化交易平台（MT4/MT5 + AI 策略生成 + 策略市场）。

- **当前里程碑**：M10 数据基础 A+ 硬化（后端为主，前端待 M11 重建）
- **工期规则**：一张卡片 = 一个独立分支/PR，一次只做一张，完成后自动继续下一张。除非：
  1. 验收失败且无法自排障 → 报告阻塞
  2. 文档指引缺失/矛盾 → 报告
  3. 跨 milestone 依赖不满足 → 报告
- **不许擅自跨范围**：超出当前卡片的改动，无论多顺手，都禁止合入同一 PR

---

## 1. 文档权威分层

| 优先级 | 主题 | 文件 |
|:------:|------|------|
| P0 | 项目身份与硬规则 | 本文档 |
| P0 | ADR（不可逆决策） | `docs/adr/NNNN-*.md` |
| P1 | 架构总览 | `docs/architecture/*.md` |
| P1 | 模块规范 | `docs/spec/*.md` |
| P2 | 实施计划 | `docs/plan/ROADMAP.md` |
| P2 | 待办与缺陷 | `docs/plan/BACKLOG.md` |
| P3 | 应急手册 | `docs/runbook/*.md` |

**冲突处理**：选优先级高的；同级别选编号大/时间新的；不能仲裁时**停下报告人类**，禁止自行裁决。

---

## 2. 技术栈

| 域 | 语言/框架 | 容器 | 端口 |
|----|-----------|------|:----:|
| 后端 | Go 1.26 + ConnectRPC + sqlc | ant-backend | 8080 |
| 策略引擎 | Python 3.14 + FastAPI | ant-strategy-service | 8081 |
| 前端 | React 19 + TS 5.9 + pnpm + Tailwind | ant-frontend (Nginx) | 8022(宿主) |
| DB | PostgreSQL 18 | ant-postgres | 5432 |
| 缓存 | Redis 8 | ant-redis | 6379 |
| 时序 | ClickHouse 24 | ant-clickhouse | 9000 |
| MQ | NATS JetStream 2.10 | ant-nats | 4222 |

- **Go module 名**：`anttrader`（import 路径以 `anttrader/...` 开头）
- **仓库路径**：`/opt/ant/`
- **不引入**：K8s / Helm / Service Mesh / 多副本
- **Proto 单源**：`proto/ant/v1/` → `buf generate` 出 Go/TS/Python stub

---

## 3. 硬性规则（违反 = PR 拒绝）

### 3.1 协议
- 对外 **ConnectRPC + SSE**，禁止新增 REST（除 `/healthz` `/readyz` `/livez` `/metrics`），禁止 WebSocket
- 内部进程内函数调用或 NATS JetStream（异步）

### 3.2 MT 接入
- mtapi gRPC 是**唯一** MT 接入路径；**禁止新增对 `internal/mt4client` `internal/mt5client` 的 import**
- mt4 与 mt5 两套独立协议，`adapter/mt4/` `adapter/mt5/` 不许共享代码（除 `adapter/mdtick/` 共享 DTO）

### 3.3 安全红线
- 用户 Python 代码**只在研究模式沙箱**执行（`strategy-service`）
- **生产路径（实盘下单）禁止任何 Python 代码执行** —— 生产策略 = DSL 字符串 + ONNX 模型引用

### 3.4 数据归属（ADR-0006）
- **平台共享**（无 `user_id`）：`platform_strategies` `platform_factors` `platform_ai_agents` `broker_symbols` `admins`
- **用户私有**（必须 `user_id` 外键 + RLS）：`mt_accounts` `user_strategies` `orders` `positions` `trades` 等
- **禁止** per-user seed 复制平台数据；**禁止** `users.role='admin'` — 走独立 `admins` 表 + JWT scope `platform:admin`

### 3.5 类型与精度
- 价格：PG `NUMERIC(20,8)` ↔ Go `decimal.Decimal` ↔ CH `Decimal(18,6)`，**禁 float64 直接参与价格计算**
- 时间：UTC，毫秒精度（`int64 ts_unix_ms`）
- 日志：结构化 JSON，必带 `trace_id` `user_id` `request_id` `account_id`（涉及账户时）

### 3.6 部署
- 单机 docker-compose，命名前缀 `ant-`
- 不许破坏 `anttrader`（独立项目，端口/卷/网络隔离）

---

## 4. 工程纪律

1. **单一职责**：handler 只编排；业务在 service；数据访问在 repo
2. **接口驱动**：跨包边界先定 interface，后实现
3. **代码生成优先**：RPC = buf；SQL = sqlc；TS 类型 = buf 自动生成
4. **三处下沉**：同一逻辑出现 3 次 → 抽到 `internal/`
5. **错误集中**：用 `internal/pkg/errors/`，禁止裸字符串错误
6. **状态机外置**：订单/连接等显式状态机，不许散落 if/else
7. **零循环依赖**：CI 强制
8. **canonical 入口规范化**：所有 (broker, symbol_raw) 在 adapter 出口转 canonical
9. **价格不丢精度**：类型链严格转换
10. **可观测性默认开**：每个 service 暴露 Prometheus metrics + structured log + healthz

---

## 5. 编码规范

### Go
- `gofumpt` + `golangci-lint` + `go test -race`
- zap 日志；`ctx context.Context` 永远是首参；禁 panic（除 main 启动失败）
- 包名 snake_case；导出符号 PascalCase；私有 camelCase
- 单文件 ≤ 300 行，单函数 ≤ 50 行，圈复杂度 ≤ 10

### TypeScript（frontend）
- strict mode；禁 any（必要时 `unknown` + 类型守卫）
- TanStack Query + Zustand；Tailwind
- 文件名 kebab-case；单文件 ≤ 250 行

### Python（仅 strategy-service / research）
- ruff strict + mypy strict；强制类型注解
- 模块名 snake_case；单文件 ≤ 400 行

---

## 6. 提交规范

### Conventional Commits
```
type(scope): subject

[optional body]
Verify: docs/handover/verify-MNNN.log:<起始行>-<结束行>
```

`type ∈ {feat, fix, refactor, docs, test, chore, perf, build, ci}`

### 分支
- `main` 受保护，禁止直接 push
- 工作分支：`<type>/<scope>/<short-desc>`（例：`feat/mdgateway/runner`）
- 一张卡片 = 一个分支 = 一个 PR

### PR 自检
1. ROADMAP/BACKLOG 当前里程碑无 🅒 / TODO / FIXME
2. 当前卡片验收日志存在且行数 ≥ 20
3. `go build ./...` + `go test -race ./internal/...` 通过
4. make lint 通过
5. 关键运行时（PG/CH 容器）在线可连

---

## 7. 反 stub 红线

生产代码路径中**绝对禁止**：
- `Printf("stub")` / `Println("TODO")` / `Errorf("not yet implemented")`
- 注释含 "not wired" / "not connected" / "not implemented" / "placeholder"
- `t.Skip` 不附卡片引用；`t.Skip` 引用自身卡片 = 循环引用
- verify log 含 `[no test files]` 视为 FAIL
- 测试函数只有 `t.Log("...requires...")` 一行 = 桩测试 = FAIL
- verify log 是其他卡片 log 的逐字复制（md5 相同 = 模板伪造）
- verify log 出现裸 `PASS` 但无 `ok <pkg> <N.Ns>` 或 `--- PASS: TestXxx`

---

## 8. 开工前必读清单

按以下顺序阅读，然后回答自检 5 问：

1. 本文档
2. `docs/architecture/01-vision.md`
3. `docs/architecture/02-overview.md`（§8 不变量的 11/12/13 必读）
4. `docs/architecture/03-data-flow.md`
5. `docs/adr/0006-platform-shared-vs-user-private.md`
6. `docs/spec/*.md`（卡片所在 milestone 的全部）
7. `docs/plan/ROADMAP.md` §当前 milestone
8. `docs/spec/16-mtapi-quirks-register.md`（涉及 MT 时）

**自检 5 问**（在 PR 描述中写明）：
1. 本卡片要改动哪些文件？（精确路径列表）
2. 本卡片的输入是什么？
3. 本卡片的输出是什么？
4. 本卡片的验收命令是什么？（精确 shell）
5. 本卡片可能踩哪些坑？

任一答不上来 → 暂停，回去读文档。

---

## 9. 卡片完成的唯一定义

> 卡片 ☑ 需要同时满足：
> - 有对应 commit
> - 有 ≥ 20 行真实 verify log（含 go test OK 输出、无 `[no test files]`）
> - 代码不含 stub/TODO/Placeholder/not wired/not connected/not implemented
> - 卡片声明的测试函数实际存在且含真断言
> - 验收命令可重跑复现

**做不通时，坦诚降级写明"🅒 + 阻塞原因 + 已尝试方案"并停下汇报。禁止用文档改字、mock 替换、放宽验收命令绕过。**

---

## 10. Makefile 标准入口

```
make proto          buf lint + breaking + generate
make build          go build ./...
make test           go test -race ./internal/... + uv run pytest
make lint           gofumpt + golangci-lint + ruff + mypy + tsc
make migrate-pg     PostgreSQL 迁移
make migrate-ch     ClickHouse 迁移
make docker-up      docker-compose up -d
make verify-card    自检脚本（CARD_ID=M7.X-Y）
```
