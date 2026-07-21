# 上线前审计与修复指导书（权威依据）

> 本文档由代码级逐路径审计产出，作为上线阻断项的**唯一修复依据**。
> 每条发现均带 `文件:行` 证据。落地 Agent 必须逐项完成并补测试后方可关闭。
> 审计基线：`go build ./...` 通过；153 个 Go 测试文件；CI 带 race+coverage+lint+行数门禁。
> 审计范围：钱包/存款/订阅/marketplace/AI 计费、OMS 下单/幂等/风控 Gate/对账、auth/secrets/JWT、Dockerfile、migrations。

---

## 0. 结论

**当前不可上线。** 阻断项：**1 × P0（安全红线）+ 5 × P1（资金/风控/凭证正确性）**。

> 修订说明（2026-07-17，两轮）：原 P0-1「凭证明文落库」定性错误。经核实：凭证加密**已在生产实现并接线**（account_service.go:149 加密写、:352 解密、main.go:176 实例化），且约束文档无明文要求。真正问题是**加密迁移半成品：读写路径分裂 + 未提交迁移炸弹**（见 P1-7）。

系统性问题：**"机制造好了却没接线"**——多个高质量的正确性组件被完整实现并测试，却未接入生产热路径，热路径跑的是不安全的旧实现。

修复必须遵守 `AGENTS.md` 全部强制约束（proto-only、decimal 计价、无 REST/WS、docker compose 部署、Reuse Preflight、文件行数）。

---

## 1. 发现清单（按优先级）

### P0-1　生产运行时 `go run` 执行策略代码 🔴

- **证据**
  - 装配：`backend/cmd/server/handlers_strategy.go:49` — `SetGoExecutor(strategy.NewGoExecutor(".", log))`。
  - 执行：`backend/internal/connect/strategy/go_executor.go:103` — `exec.CommandContext(ctx, "go"); cmd.Args = {"go","run",...}`。
  - 触发：`backend/strategy/sdk/language.go:48` — `isGoSource` 仅需代码含 `package ` + `alphaforge/strategy/sdk`。
  - 运行时镜像：`backend/Dockerfile:33` — `FROM golang:1.26.2-alpine`（全工具链），与文件顶部 "run in scratch/distroless" 注释矛盾。
- **风险**：无沙箱任意代码执行（RCE）+ 资源耗尽 + 数百 MB 攻击面；违背"MQL/Python 统一进程内 Bytecode VM"第一性设计。
- **根因**：ADR-0023 已用 Bytecode VM 取代 Go 运行时路径，但旧路径未移除。
- **修复方向**
  1. 删除 `GoExecutor` 及其 `Run`/`RunBacktest`/`RunLive`/`CompileCheck`、harness 模板、`executeGoBacktest` 与 `ExecuteLive` 中的 `isGoStrategy` 分支。
  2. 若有存量 Go 策略，先一次性迁移为 MQL/Python。
  3. `Dockerfile` runtime stage 改最小 `alpine`，仅拷贝二进制 + `mql.so`。
- **验收**：`grep -rn "go_executor\|NewGoExecutor\|isGoStrategy" backend` 无生产引用；镜像体积显著下降；`go build ./...` 通过。

---

### P1-7　MT 凭证加密半迁移：读写路径分裂 + 未提交迁移炸弹 �

> **两次定性修正后的最终事实**：凭证加密**已在生产实现并接线**（非策略、非死代码）。写路径 `account_service.go:149` 已加密写 `password_encrypted`；解密路径 `GetDecryptedPassword`（`:352`）已就绪；`main.go:176` `secrets.New` 已实例化；`pipeline.go:93` 已注入 `Secrets`；`BackfillPlaintextCredentials`（`:375`）已备。**约束文档（`.windsurfrules`/`AGENTS.md`/`CLAUDE.md`/ADR）无明文要求**；仅 `mt-accounts` skill 与代码注释（陈旧）说明文。加密迁移已约 80%，但处于**危险半成品态**。

- **缺陷 1 — 读写路径分裂（当前 P0 功能 bug）**
  - 写：`CreateAccountTx` 只写 `password_encrypted`，**不写明文 `password`**。
  - 读：`loadAccountConfigs`（`backend/internal/mdgateway/wiring.go:34`）仍读**明文** `password`（注释 `:33` 陈旧）。
  - 后果：**新建账户明文列为空 → mdgateway 用空密码连经纪商 → 新账户实盘/行情连接失败**（老账户残留明文才正常）。
- **缺陷 2 — 未提交迁移炸弹**：`188_account_credential_encryption_view`（视图 `password` TEXT→BYTEA）与 `189_drop_plaintext_credentials`（删明文列）均**未提交（`??`）**。一旦 `docker compose build` 触发：188 → `loadAccountConfigs` 用 `string` 扫 BYTEA **全表加载失败 → 全部实盘中断**；189 → `GetAccount` sqlc 查询（SELECT password）崩溃。
- **修复方向（最优解 = 完成加密迁移，勿退回明文）**
  1. 改 `loadAccountConfigs`：读 `password_encrypted`/`mtapi_token_encrypted`，用 `deps.Secrets` 解密；删陈旧 "plaintext" 注释。
  2. 启动时执行 `BackfillPlaintextCredentials` 迁移老账户。
  3. 统一所有读路径为"读加密列 + 解密"；同步改 sqlc 查询不再 SELECT 明文列。
  4. backfill 验证通过后，**按序**应用 188 → 189 并提交。
  5. 更正陈旧文档：`mt-accounts` skill、`wiring.go` 注释——统一为"信封加密 at-rest"。
- **验收**：新建账户能成功连接经纪商；`grep -rn "plaintext" backend/internal/mdgateway` 无残留；188/189 已提交且构建通过；无 `string` 扫 BYTEA。
- **仅当存在书面明文硬约束时才走回退**（当前查无此规定）。

---

### P1-3　AI Token 计费 TOCTOU 双花；钱包无非负约束 🟠

- **证据**
  - `backend/internal/connect/gateway/ai_gateway_handler.go:277` — 未加锁 `GetOrCreateWallet` 读余额 → 判断 → **另一事务** `AdjustBalance` 扣款。
  - `backend/internal/repository/wallet_repo.go:102` — `AdjustBalanceTx` 是 `balance = balance + amount`，**无 `WHERE balance>=` 守卫**。
  - `backend/migrations/147_user_wallets.up.sql` — **无 `CHECK(balance>=0)`**。
  - `backend/internal/marketplace/purchase.go:130` — 注释谎称"DB 有 CHECK 约束"（实际不存在）。
- **对照（正确实现，勿动）**：marketplace/subscription 用"`FOR UPDATE` + 条件扣减 `WHERE balance>=`"；deposit 审批用 `WHERE status='PENDING'` + 单事务回滚。
- **风险**：并发 AI 请求把余额扣成负数 = 平台真实成本损失。
- **修复方向**
  1. 新增 migration：`ALTER TABLE user_wallets ADD CONSTRAINT chk_balance_nonneg CHECK (balance >= 0);`（先确认无存量负值）。
  2. AI 计费改为**单事务**：`FOR UPDATE` 锁钱包 → 校验 → 条件扣减 `WHERE balance >= cost` → 插入 usage 记录，全在一个 tx。
  3. `AdjustBalanceTx` 增加非负守卫（或依赖 CHECK 约束，扣款失败即回滚）。
  4. 修正 `purchase.go` 虚假注释。
- **验收**：并发扣款集成测试（N 个 goroutine 抢同一钱包，余额不为负、总扣款正确）。

---

### P1-4　对账只探测不修复，且无脑解锁 🟠

- **证据**
  - `backend/internal/mthub/reconciliation.go:133` — 只统计 ghost/orphan，`r.log.Debug(...)`（生产不可见）。
  - `backend/internal/mthub/reconciliation.go:161` — 无条件 `gate.MarkReconciled(accountID)`，无告警/无修复/无人工钩子。
- **风险**：重连后经纪商侧有 OMS 不知道的持仓（ghost），系统照常恢复交易，幽灵仓位无人管理。
- **修复方向**
  1. 发现分歧时提升日志级别（≥Warn）并通过通知渠道告警。
  2. 存在 ghost/orphan 时**保持 gate 锁定**或进入人工复核状态，而非无条件解锁。
  3. 提供修复动作接口（拉取并登记 ghost / 关闭 orphan）。
- **验收**：注入分歧场景的单测，断言 gate 未被解锁 + 告警被触发。

---

### P1-5　健壮三层幂等被弃用，生产跑单层 Redis 🟠

- **证据**
  - `backend/cmd/server/main.go:208` — 装配 `mthub.NewIdempotencyGuard(rdb.Client())`（单层 Redis，已标注 Deprecated）。
  - `ThreeLayerGuard`（PG advisory xact lock + Redis，`backend/internal/mthub/idempotency.go:35`，含并发集成测试）**生产未接线**。
- **风险**：下单幂等仅靠单层 Redis SETNX；Redis 抖动/跨实例时防重弱于已实现方案。
- **修复方向**：将 `ThreeLayerGuard` 接入 `MtHubService.PlaceOrder`（`backend/internal/mthub/service_orders.go:49`），替换 deprecated 实现；移除或废弃单层实现。
- **验收**：`PlaceOrder` 使用 `ThreeLayerGuard`；并发同 clientID 下单只成交一次的集成测试在 CI 实跑。

---

### P1-6　实盘下单 fire-and-forget，信号可静默丢失 🟠

- **证据**：`backend/internal/connect/strategy/live_dispatch.go:256` — `go func(){ PlaceOrder(...) }()` + `context.WithoutCancel`，错误仅 `s.log.Error` 打日志，无重试/死信/意图持久化。
- **风险**：瞬时经纪商错误 = 交易信号永久丢失，用户无感知。
- **修复方向**：订单意图落库（outbox 模式）→ 提交失败进重试/死信队列 → 失败通知用户。遵守 push-first：用 NATS JetStream 承载重试，禁止轮询。
- **验收**：模拟 broker 失败，断言意图被持久化并重试/告警。

---

## 2. P2 — 设计缺陷与死代码（上线后尽快）

| 项 | 证据 | 处理 |
|----|------|------|
| `frozen_balance` 幻影功能（永远为 0，无写入逻辑） | `backend/internal/model/wallet.go:14` 仅展示 | 实现冻结逻辑 或 从 API/UI/schema 移除 |
| 默认风控 Gate 为空（回撤/保证金/日亏损全选配） | `backend/internal/risk/gate.go:206` | 确认风控策略；考虑默认注入基础保护 |
| Cookie 未设 Secure | `backend/cmd/server/handlers.go:113` | 生产强制 `Secure`（按环境变量区分） |
| SSE 用 query 传 access_token（日志泄漏） | `backend/internal/interceptor/auth.go:97` | 改用短时一次性 SSE ticket |
| 已废弃 RPC 仍在 proto/装配 | paper Start/Stop、UpdateTradingPassword、VerifyAccount | 从 proto 删除 |
| 越硬红线大文件 | `interp/analyze.go` 813 行等 | 按语义域拆分 |

---

## 3. P3 — 可扩展性与测试盲区

- **单实例假设**：内存态 `Guard.dedup`（`backend/internal/risk/guard.go:72`）、`ReconcileGate`、rate limiter 非跨实例共享，水平扩容会重复下单 → 需外置到 Redis/PG。
- **每条 SSE 流各占一个 PG LISTEN 连接**（历史池耗尽根因）→ 改共享监听器扇出。
- **CI 用 `go test -short`**（`.github/workflows/ci.yml:80`）：钱包/幂等等集成测试可能被跳过 → 关键资金/交易路径需在 CI 实跑（去掉 `-short` 跳过 或 增设集成测试 job）。

---

## 4. 已验证的优点（保留，勿破坏）

- 订单管线纵深防御结构正确：kill switch → 3-rule Guard → 账户属主校验(防 IDOR) → 幂等 → reconcile gate → 限流 → OMS 状态机 → 风控 Gate → broker（`backend/internal/mthub/service_orders.go:19`）。
- 风控 Gate nil-state **fail-closed** 正确（`backend/internal/risk/gate.go:143`）。
- 账户查询强制 `WHERE id=$1 AND user_id=$2`（`backend/internal/repository/accounts.sql.go:15`）+ 有 IDOR 集成测试。
- JWT 有 `ExpiresAt` 且校验 HMAC 签名方法防 alg 混淆（`backend/internal/interceptor/auth.go:172`）。
- marketplace/subscription/deposit 资金路径事务与锁正确。
- 信封加密实现质量高且**已接线**（写/解密/实例化均在），仅差「完成迁移收尾」（P1-7）；三层幂等已实现但未接线（P1-5）。

---

## 5. 修复顺序（严格按此执行）

1. **P0-1** 移除 GoExecutor（`go run`）+ 最小镜像
2. **P1-3** 钱包非负约束 + AI 计费单事务条件扣减
3. **P1-4** 对账修复动作 + 告警 + 不无脑解锁
4. **P1-5** 接入 ThreeLayerGuard
5. **P1-6** 下单 outbox/重试
6. **P1-7** MT 凭证加密迁移收尾（修读写分裂 + 按序提交 188/189）— 建议尽早，未提交迁移一旦 build 即触发
7. **P2 / P3** 收尾

> 每项修复后必须补**跨进程/并发集成测试**并在 CI 实跑，杜绝"造好不接线"复现。

---

## 6. 落地 Agent 提示词（逐项，可直接粘贴）

> 通用前置（每个任务都适用）：
> - 遵守 `AGENTS.md`：proto-only（禁 JSON 持久化/交换）、价格用 `decimal.Decimal`、禁 REST/WS、禁 `//nolint` 等抑制注释、一任务一 scope。
> - **Reuse Preflight**：动工前 `bash scripts/cap.sh <动词>`，在结果里写 `REUSE:` 或 `NEW:`。
> - 部署仅：`docker compose build backend && docker compose up -d backend`；禁宿主 `go build` → `docker cp`。
> - 提交前：`go build ./...` + `cd backend && go run ./tools/check-file-lines --strict` + `bash scripts/gen_capability_map.sh`。
> - 完成标准：附带新增/修改的测试，并说明如何验证。

### 提示词 · P0-1（移除 go run 路径 + 最小镜像）
```
任务：移除运行时 `go run` 执行策略代码的 GoExecutor 路径，运行时镜像切最小 alpine。依据 P0-1。
Reuse Preflight：确认 MQL/Python 已统一走进程内 Bytecode VM（backend/internal/connect/strategy/backtest_worker.go:162 的 executeVMBacktest/executePythonVMBacktest）。
要做：
1. 删除 backend/internal/connect/strategy/go_executor.go 及 harness 模板、SetGoExecutor 装配（handlers_strategy.go:49）、executeGoBacktest 与 ExecuteLive 里的 isGoStrategy 分支、sdk.IsGo 相关生产引用。
2. 若存在 Go 类型策略，先写一次性迁移转 MQL/Python 后再删；若确认无存量，直接删除并说明。
3. backend/Dockerfile runtime stage（:33）由 golang:1.26.2-alpine 改为最小 alpine，仅拷贝 /app/alphaforge + tools/mql2go/mql.so + configs + migrations + entrypoint。
验收：grep 无 GoExecutor 生产引用；go build ./... 通过；镜像可正常启动（docker compose build backend && up -d backend）。
```

### 提示词 · P1-3（钱包非负 + AI 计费原子化）
```
任务：修复 AI Token 计费的 TOCTOU 双花，并为钱包加非负约束。依据 P1-3。
要做：
1. 新增 migration：确认无存量负余额后，ALTER TABLE user_wallets ADD CONSTRAINT chk_balance_nonneg CHECK (balance >= 0)。
2. 重写 backend/internal/connect/gateway/ai_gateway_handler.go:277 RecordTokenUsage 的付费分支为单事务：BEGIN → SELECT ... FOR UPDATE 锁钱包 → 校验 → UPDATE ... WHERE balance >= cost RETURNING（0 行则 InsufficientBalance）→ 插入 wallet_transactions + ai_token_usage → COMMIT。
3. 为 AdjustBalanceTx（wallet_repo.go:102）增加非负守卫（WHERE balance + amount >= 0 或依赖 CHECK）。
4. 修正 backend/internal/marketplace/purchase.go:130 关于 CHECK 约束的错误注释。
约束：金额一律 decimal.Decimal / NUMERIC(20,8)，禁 float64。
验收：并发集成测试（多 goroutine 抢同一钱包）断言余额不为负、扣款总额正确；在 CI 实跑（不被 -short 跳过）。
```

### 提示词 · P1-4（对账修复与告警）
```
任务：让重连对账在发现分歧时告警并阻止盲目恢复交易。依据 P1-4。
要做：
1. backend/internal/mthub/reconciliation.go:133 发现 ghost/orphan 时日志升级为 Warn 并通过现有通知渠道告警（先 cap.sh 查通知能力，复用现成）。
2. 存在分歧时不无条件 MarkReconciled（:161）：保持 ReconcileGate 锁定或置人工复核状态。
3. 预留修复动作：登记 ghost、关闭 orphan（可先实现登记 + 告警，平仓作为后续）。
验收：注入分歧的单测断言 gate 未解锁且告警触发。
```

### 提示词 · P1-5（接入三层幂等）
```
任务：将生产下单幂等从 deprecated 单层 IdempotencyGuard 换为 ThreeLayerGuard。依据 P1-5。
要做：
1. backend/cmd/server/main.go:208 改为 mthub.NewThreeLayerGuard(pool, rdb.Client())，注入 MtHubService。
2. 核对 service_orders.go:49 的 CheckAndSet/SetTicket 接口签名与 ThreeLayerGuard 一致，必要时适配。
3. 废弃或删除单层 IdempotencyGuard。
验收：PlaceOrder 使用 ThreeLayerGuard；并发同 clientID 只成交一次的集成测试在 CI 实跑。
```

### 提示词 · P1-6（下单 outbox/重试）
```
任务：消除实盘下单 fire-and-forget 导致的信号静默丢失。依据 P1-6。
要做：
1. backend/internal/connect/strategy/live_dispatch.go:256：下单前将订单意图持久化（outbox 表）。
2. 提交失败进入重试（NATS JetStream，push-first，禁轮询）与死信；最终失败通知用户。
约束：禁 setInterval/ticker 轮询；用现有 NATS/JetStream 能力（先 cap.sh 查）。
验收：模拟 broker 失败，断言意图被持久化并重试/告警，无静默丢失。
```

### 提示词 · P1-7（MT 凭证加密迁移收尾）
```
任务：完成 MT 凭证 at-rest 加密迁移，修复读写路径分裂并安全落地 188/189。依据 P1-7。
背景：加密已接线（account_service.go:149 加密写、:352 解密、main.go:176 实例化）；但 loadAccountConfigs 仍读明文，且 188/189 未提交。约束文档无明文要求，不退回明文。
要做：
1. 改 backend/internal/mdgateway/wiring.go:34 loadAccountConfigs：读 password_encrypted/mtapi_token_encrypted，用 deps.Secrets（RunnerDeps.Secrets，runner.go:31）解密；删除 :33/:59 陈旧 "plaintext" 注释。
2. 启动时执行 AccountService.BackfillPlaintextCredentials（account_service.go:375）迁移老账户明文→加密。
3. 统一所有读路径为"读加密列+解密"；改 sqlc 查询（accounts.sql / GetAccount）不再 SELECT 明文 password/mt_token。
4. backfill 验证 password_encrypted IS NOT NULL 后，按序提交并应用 188 → 189。
5. 更正陈旧文档：.claude/skills/mt-accounts、wiring.go 注释 → "信封加密 at-rest"。
约束：禁打印明文凭证；禁用 string 扫描 BYTEA 列。
验收：新建账户能成功连接经纪商（集成测试）；grep -rn "plaintext" backend/internal/mdgateway 无残留；188/189 已提交且 go build ./... 通过。
```
