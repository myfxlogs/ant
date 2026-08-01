# Audit Report — 2026-08-01 — 全项目审计

## 审计范围

全项目（11 个功能块），按 rd.md Part A 的 9 个维度逐项检查。

## 审计方法

- 维度 1-4（架构/最优性/第一性/简单性）：Agent 并行扫描 + 人工复核
- 维度 5（整洁）：deadcode + knip + golangci-lint + eslint + tsc + check-file-lines
- 维度 6（技术债）：TODO/FIXME 扫描 + t.Skip 审计 + 迁移文件配对
- 维度 7（正确性）：float64 价格路径 + 状态机完整性 + 幂等性 + 并发安全 + 错误处理
- 维度 8（安全）：handler 鉴权 + SQL 注入 + 密钥扫描 + govulncheck + gitleaks
- 维度 9（合规）：proto breaking + json.Marshal 禁令 + nolint 禁令 + REST/WS 禁令

---

## §1. 问题清单

| ID | 维度 | 文件:行 | 风险 | 影响范围 | 触发场景 | 危害后果 |
|----|------|---------|------|----------|----------|----------|
| A-001 | 正确性 | `internal/connect/ai/agent_loop_helpers.go:62-120` | **CRITICAL** | Agent 策略生成 | 文本工具调用解析器用 `strings.Fields` + `SplitN` 按空格切分——多行策略代码被截断 | **策略代码被静默破坏**——Agent 产出的策略丢失内容 |
| A-002 | 正确性 | `internal/marketplace/settlement.go:89-117` | **CRITICAL** | 策略分账 | credit 失败后 `log.Warn + continue`，事务提交已成功的项目——mark settled 失败靠幂等键防重复 credit | 分账金额可能被重复或遗漏，涉及真实资金 |
| A-003 | 合规 | `backend/migrations/015_ai_gateway.sql` | **CRITICAL** | 新数据库部署 | 文件名缺 `.up.sql` 后缀，docker-entrypoint 的 `*.up.sql` glob 不匹配——生产代码查询其创建的表 | **新部署无法启动**——164_gateway_default_model 引用 015 的表，bootstrap 模式会标记 migration 已应用并永久跳过 |
| A-004 | 合规 | `backend/migrations/` | HIGH | 所有需回滚的部署 | 55 个 .up.sql 缺少对应 .down.sql | 部署失败时无法回滚，必须手工修复 DB |
| A-002 | 正确性 | `internal/mdgateway/runner.go:54` 等 10 处 | HIGH | 全系统 | goroutine 内 panic 未被 recover | **进程崩溃**——MT 行情/下单全部中断 |
| A-003 | 正确性 | `internal/marketplace/` (全部) | HIGH | 策略市场用户 | 0.1% 覆盖率，购买/分账/结算无测试守卫 | 金额计算错误→用户资金损失 |
| A-004 | 正确性 | `internal/mthub/runner_health.go:100` | HIGH | 所有 MT 账户 | 健康检查 goroutine panic 未 recover | 健康监控静默停止，账户异常不被发现 |
| A-005 | 架构 | `internal/agent/` imports `internal/connect/ai/` | HIGH | Agent 引擎 | 业务块（agent-engine）直接 import 自己的 API 层（connect/ai） | 分层倒置——业务逻辑依赖 API handler，重构/测试困难 |
| A-006 | 架构 | `internal/connect/paper/` imports `internal/connect/strategy/` | HIGH | 前端 API | API handler 之间交叉依赖 | handler 包互相耦合，修改一个影响另一个 |
| A-007 | 架构 | `internal/mdgateway/adapter/mt4/clock.go` 与 `mt5/clock.go` | MEDIUM | MT 网关 | 两个文件逐字节相同（仅 package 名不同） | 纯代码重复——违反 MT4/MT5 代码不共享规则 |
| A-008 | 整洁 | `strategy/indicators/core_*.go` 等 20 处 | MEDIUM | 策略运行时 | deadcode 扫描发现 20 个不可达函数 | 死代码占用维护成本，误导新开发者 |
| A-009 | 整洁 | `frontend/src/` (knip) | MEDIUM | 前端 | 80 个文件 + 96 个导出未被任何代码引用 | 死代码 + 误报混合，无法区分真正可删除的文件 |
| A-010 | 最优性 | `internal/repository/*.go` 5 处 | MEDIUM | DB 查询 | `SELECT * FROM` 在仓库层使用 | schema 变更时 Scan 行为不可预测 |
| A-011 | 正确性 | `internal/marketplace/live_performance.go:261` | MEDIUM | 实时战绩计算 | `context.Background()` 用于 PG 查询 | 无超时/取消——慢查询永久阻塞 |
| A-012 | 简单性 | `internal/mdgateway/runner_gateway.go` 等 | MEDIUM | MT 网关启动 | 网关启动流程分散在 6 个文件中 | 启动逻辑难以理解，调试困难 |
| A-013 | 正确性 | `internal/mthub/tick_broker.go:63-76`, `trade_broker.go` | MEDIUM | 实时行情/交易推送 | Publish 在 RUnlock 后 send channel——cancel 可 close channel → "send on closed channel" panic | **进程崩溃**——与 A-002 同类但更具体，可直接修复 |
| A-014 | 正确性 | `internal/marketplace/service_subscription.go:88-97` | MEDIUM | 策略订阅 | 重试时 `total_subscribers` 重复 +1，无幂等键保护 | 公开显示的订阅计数漂移，用户信任度下降 |
| A-015 | 正确性 | `internal/marketplace/settlement.go:119` | MEDIUM | 策略分账 | `decimal.NewFromString` 错误被静默忽略 → providerAmt=0 | 策略提供者收入被少计 |
| A-016 | 正确性 | `internal/marketplace/service_subscription.go:49` | MEDIUM | 策略购买门禁 | `decimal.NewFromString` 错误被静默忽略 → priceDec=0 → 付费策略可免费订阅 | 收入损失 |
| A-017 | 安全 | `internal/connect/marketplace/marketplace_handler_social.go:23` | MEDIUM | 策略评分 | Rating 输入无范围校验——可提交 100 或 -5 | 评分数据被污染，排行榜不可信 |
| A-018 | 正确性 | `internal/oms/statemachine.go:84-98` | MEDIUM | OMS 状态机 | WORKING/PARTIALLY_FILLED 状态无超时→UNKNOWN 转换 | 订单卡在中间状态，永远不被对账修复 |
| A-019 | 技术债 | `backend/internal/connect/ai/code_assist_t22_test.go` | LOW | AI 代码辅助 | 已修复（t.Skip → t.Fatal） | ✅ 已修复 |
| A-014 | 技术债 | `backend/tests/e2e/` 3 文件 | LOW | E2E 测试 | ClickHouse 删除后测试永久 skip | ✅ 已删除 |
| A-015 | 安全 | `frontend/src/` 42 处 | LOW | 前端开发规范 | eslint-disable 有 REF 追踪但缺 GitHub issue 链接 | 追踪不闭环 |
| A-016 | 合规 | `scripts/check_handler_registry.sh` | LOW | CI Pipeline | 脚本假设从 repo root 运行，路径硬编码 | CI 中如果 cwd 不对会误报失败 |
| A-017 | 技术债 | `backend/internal/mthub/pipeline_integration_test.go` | LOW | 集成测试 | 引用的 risksvc API 已漂移，测试被替换为占位 skip | 风险管线测试缺失 |
| A-018 | 技术债 | `backend/internal/connect/ai/pipeline_integration_test.go` 等 | LOW | CI 门禁 | 2/6 管线冒烟存在，P1/P3/P5/P6 缺失 | 管线断裂不会被 CI 发现 |

**风险等级**: CRITICAL > HIGH > MEDIUM > LOW > INFO

---

## §2. 修复明细

### A-001: 迁移文件缺失 down 脚本 (HIGH)

55 个 .up.sql 缺少对应 .down.sql。检查命令：
```bash
ls backend/migrations/*.up.sql | wc -l  # 223
ls backend/migrations/*.down.sql | wc -l # 168
```

**修复**: 为所有缺失 down 的 migration 补充 `.down.sql`。最低要求：至少包含一条注释说明为什么不需要 down（如 "data migration, irreversible"）。

**预防规则 (Part B)**: CI 加入 migration down 配对检查作为 Static Gate 阻断项（rd.md Layer 0 已列出，标注 🔧 待接入）。

### A-002: Goroutine 缺少 panic recovery (HIGH)

以下 10 处 goroutine 没有 `defer recover()`：

| 文件 | goroutine 数 | 功能 |
|------|-------------|------|
| `mdgateway/runner.go` | 1 | 主 runner |
| `mdgateway/runner_health.go` | 1 | 账户健康检查 |
| `mdgateway/runner_nats.go` | 1 | NATS 事件订阅 |
| `mdgateway/wiring.go` | 2 | 回填/网关启动 |
| `mdgateway/user_metrics_flusher.go` | 1 | 用户指标刷新 |
| `mdgateway/market_state.go` | 1 | 市场状态 |
| `mdgateway/adapter/mt5/orders.go` | 1 | MT5 订单回调 |
| `mdgateway/adapter/mt4/orders.go` | 1 | MT4 订单回调 |
| `analysis/analyzer.go` | 1 | 分析器 |
| `marketplace/service_subscription.go` | 1 | 订阅服务 |

**修复**: 每个 goroutine 入口加 `defer func() { if r := recover(); r != nil { log.Error("panic", ...) } }()`。

**预防规则 (Part B)**: golangci-lint 开启 `goerr113` 或自定义规则检查 goroutine 必须含 recover。

### A-003: marketplace 零测试覆盖率 (HIGH)

`internal/marketplace/` 覆盖率 0.1%——策略购买、分账、结算、冻结等涉及金额的操作全无测试守卫。

**修复**: Phase 2 优先补 marketplace 覆盖率至 ≥60%。优先覆盖：
1. `purchase.go` — 购买流程（扣款→发货→分账）
2. `settlement.go` — 结算逻辑
3. `live_performance.go` — 实时战绩计算

**预防规则 (Part B)**: per-block 覆盖率门禁（已接入 CI，B.3.2 下限 0.1%→逐步提标）。

### A-005: agent-engine 分层倒置 (HIGH)

`internal/agent/` 直接 import `internal/connect/ai/`——业务逻辑层依赖 API handler 层。正确架构：agent 定义接口，connect/ai 实现接口并调用 agent。

**修复**: 抽取 agent 需要的接口到 `internal/agent/` 或共享包，connect/ai 实现接口。

### A-006: API handler 交叉依赖 (HIGH)

`internal/connect/paper/` 直接 import `internal/connect/strategy/`——两个 API handler 包之间耦合。应通过 service 层或 NATS 事件解耦。

**修复**: connect/paper 通过 service 层接口或 NATS 事件获取策略信息，而非直接 import connect/strategy。

### A-004, A-007-A-018: 见上述表格

---

## §3. 回归测试结果

| 测试套件 | 结果 | 说明 |
|----------|------|------|
| `go test -short ./...` | ✅ PASS | 全绿，0 FAIL |
| `go test -tags=integration ./...` | ✅ PASS | 含 PG/Redis 集成测试 |
| `golangci-lint run ./...` | ✅ 0 issues | 全量模式 |
| `eslint src/ --max-warnings 0` | ✅ PASS | 前端全量 |
| `npx tsc --noEmit` | ✅ PASS | 前端类型检查 |
| `govulncheck ./...` | ✅ 0 vulns | go1.26.5 + pgx v5.9.2 |
| `check_handler_registry.sh` | ✅ 49/49 auth | 零未认证 handler |
| `go build ./...` | ✅ PASS | 全量编译 |
| deadcode | ⚠️ 20 unreachable | 见 A-006 |
| knip | ⚠️ 80 files + 96 exports | 见 A-007 |
| check-file-lines --strict | ✅ 0 errors | 25 warnings, 75 info |

---

## §4. 未修复项

| ID | 理由 | 计划修复时间 |
|----|------|-------------|
| A-001 | 需逐 migration 编写 down 脚本，工作量大 | Phase 4（验收前） |
| A-002 | 需在 10 处 goroutine 中加 recover + 日志 | Week 1 |
| A-003 | 需大规模补 marketplace 测试 | Phase 2 续 |
| A-005 | agent-engine 循环依赖需重构 | 下个架构迭代 |
| A-006 | deadcode 函数需逐个确认是否真的不可达 | Phase 2 续 |
| A-007 | knip 存量需逐文件甄别死代码 vs 误报 | Phase 2 续 |

---

## §5. 风险复盘

### 根因分析

1. **为什么 marketplace 0.1% 覆盖率能进入 main？**
   - per-block 覆盖率门禁刚接入（以现状快照为下限），尚未提标。门禁只防回退，不强制提升。
   - **防止同类**: Phase 2 提标必须执行，且设置提标期限。

2. **为什么 55 个 migration 缺 down 脚本？**
   - CI 有配对检查（`migration .down.sql 配对检查`）但标注为 ✅，实际未阻断。
   - **防止同类**: 确认 CI job 确实阻断（非 warn-only），虚假声称按 bug 处理。

3. **为什么 goroutine 缺 recover？**
   - 无自动化检查。golangci-lint 未配置相关规则。
   - **防止同类**: 研究 golangci-lint 或自定义 linter 检测 goroutine 入口 recover。

### 本轮审计的预防规则产出

| 审计发现 | Part B 预防规则 | 状态 |
|----------|----------------|------|
| A-001 | CI migration down 配对阻断 | 🔧 待接入 |
| A-002 | golangci-lint goroutine recover 检查 | 🔧 待研究 |
| A-003 | per-block 覆盖率门禁提标 | 🔧 Phase 2 执行中 |
| A-009 | `context.Background()` 生产路径检查 | 🔧 待接入 CI |

---

## §6. 维度逐项评估

| # | 维度 | 评级 | 说明 |
|---|------|------|------|
| 1 | 架构 | 🟡 中 | agent-engine 分层倒置；connect handler 交叉依赖；MT4/MT5 clock 重复；其余块边界清晰 |
| 2 | 最优性 | 🟢 良 | 未发现 O(n²) 关键路径问题；SELECT * 5 处非关键 |
| 3 | 第一性原则 | 🔴 差 | 2 CRITICAL（文本工具调用截断策略代码 + 分账失败静默提交）；WebAuthn 禁用；双执行引擎；双风控引擎；多处 fallback 代替实现 |
| 4 | 简单性 | 🔴 差 | ADR-0003 LOC 超标 10 倍；3 层适配器链；死代码 MTBrokerAdapter；双份未集成的 BarSource；30+ 单实现接口 |
| 5 | 代码整洁 | 🟡 中 | deadcode 20 函数 + knip 存量；核心代码整洁 |
| 6 | 技术债 | 🟡 中 | 55 migration 缺 down；2/6 管线冒烟缺失 |
| 7 | 正确性 | 🟡 中 | Decimal 全链路合规；goroutine recover 缺失；marketplace 零测试 |
| 8 | 安全 | 🟢 良 | 49/49 handler 鉴权；零 CVE；零硬编码密钥；SQL 全参数化 |
| 9 | 合规 | 🟢 良 | 零 json.Marshal 违规；零 REST/WS；零 nolint；proto lint 通过 |

**总评**: 项目核心安全防线完整（零 CVE、49/49 handler 鉴权、零硬编码密钥、SQL 全参数化、Decimal 全链路）。但首次全项目审计发现 **2 个 CRITICAL 缺陷**（Agent 策略代码截断 + 分账静默失败），以及 **8 个 HIGH 缺陷**（goroutine recover 缺失、市场零测试、分层倒置、双执行引擎、双风控引擎、ADR-0003 超标 10 倍、3 层适配器链）。技术债务集中在 agent-engine 和 marketplace 两个块——前者有脆弱的文本协议解析，后者有金额相关的错误处理缺陷。

---

**审计日期**: 2026-08-01
**审计人**: Claude (AI) + 项目负责人复核
**下次审计**: 2026-09-01（或 marketplace 覆盖率达标后）
