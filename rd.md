# Project Quality Standard: Audit & Testing (v2 — 验收基线版)

`rd.md` 定义两套互补的流程。不是二选一，两个都要。
**审计 (Part A)** 是侦探：发现已有问题。**测试 (Part B)** 是门卫：阻断新问题。

> **v2 修订说明 (2026-08-01, 验收阶段启动)**
> - 新增 **Part 0 质量宪法**：把"最优解 / 第一性 / 简单 / 干净 / 无BUG / 安全"从口号变成可判定、可验证的标准。
> - 新增 **B.11 验收出口标准**：定义"验收通过"的判定条件（v1 缺失）。
> - 新增 **B.3.1 块→包映射** 与 **B.3.2 覆盖率基线快照**，P0 要求从声称变为可度量。
> - **B.4 管线契约内联正文**，删除两个不存在的 Appendix 死引用。
> - 所有 CI 声称项标注状态（✅已接入 / 🔧待接入 / ❌不做+理由）。**声称与实际脱节 = 文档缺陷，按 bug 处理。**
> - B.9 Roadmap 从建设期排期改为验收导向排期（先止损 → 再门禁 → 后密度 → 终验）。

---

# Part 0: Quality Constitution（质量宪法 — 不可妥协）

以下 6 条是项目最高质量准则，优先级高于本文档其他所有条款。
每条含**判定标准**——不可判定的要求只是口号。

## 0.1 最优解原则

所有功能块、架构、前后端、每个方法的实现，都采用当前约束下的**最优解**。

- **判定**: 存在已知更优方案（复杂度更低 / 更简单 / 更正确）而未采用，且无书面理由（ADR 或代码注释）= 违规。
- **禁止**: 因困难而退而求其次。遇到阻碍必须回到根因，哪怕推翻旧架构、完全重构。
- **验证**: 审计维度 2 + PR review 必答"为什么这是最优解"。

## 0.2 第一性原则

从根因解决问题，不打补丁。

- **判定**: 代码中出现 `workaround` / `hack` / `legacy` / `temporary` 注释 = 违规（唯一豁免：注释附带 issue 链接 + 移除条件 + 期限）。
- **快捷方式视为违规**: 回退代替重新生成、标记 legacy 代替移除、沉默代替修复。

## 0.3 简单性原则

逻辑要简单且正确。**最简单的正确解 > 精巧的通用解**（YAGNI）。

- **判定**: 只用一次的接口/抽象层 = 过度设计；未被调用的参数化 = 冗余；函数嵌套深度 >4 或圈复杂度超标（gocyclo）= 违规。
- **验证**: 审计维度 4 + golangci-lint (gocyclo) + review。

## 0.4 干净原则

零死代码、零冗余、零被注释掉的代码块、lint **全量**全绿、文件行数合规、命名一致。

- **判定**: `deadcode` + `knip` + `golangci-lint`（全量，非增量）+ `eslint`（全量）全部通过。
- **注意**: 当前 CI 的 golangci-lint / eslint 为增量模式，验收前必须切换全量并清零存量（B.9 Phase 1）。

## 0.5 正确性原则（"无 BUG"的可验证定义）

"无 BUG"不可直接证明，定义为以下 4 条同时成立：

1. **关键不变量有测试守卫**: 价格 `decimal.Decimal` 全链路无 float64；时间 UTC 毫秒；状态机迁移完整（无未处理状态）；所有写路径幂等。
2. **零已知缺陷**: 无未修复的已确认 bug；main 分支测试全绿（0 FAIL、0 无理由 Skip）。
3. **回归免疫**: 每个历史 bug 都有对应回归测试（fix commit 必须带测试）。
4. **契约完整**: proto 无 breaking change；handler 注册完整；序列化往还有测试。

## 0.6 安全原则

严谨无漏洞。

- **判定**: handler 鉴权 100% 归类（`authenticated` 或带理由的 `public`）；SQL 全参数化；零硬编码密钥；外部输入全量边界校验；`govulncheck` / `trivy` / `gosec` 零 HIGH/CRITICAL。
- **验证**: `scripts/check_handler_registry.sh`（CI 阻断）+ B.5 边界参数化测试 + 安全扫描三件套。

### 0.6.1 密码策略（项目级决策，审计基准）

| 类别 | 策略 | 依据 |
| ---- | ---- | ---- |
| MT 交易密码（`mt_accounts.password`） | **vault 加密存储（`password_encrypted`）+ 前端明文显示（按需解密）——现状即达标** | 上游 mtapi gRPC 需原始密码（解密后提交）；产品要求前端可见 |
| 平台用户登录密码 | 必须哈希（bcrypt/argon2），禁止明文/可逆存储 | 常规安全基线 |
| token / API 密钥 / 钱包私钥 | 按 `docs/spec/17-secrets-and-errors.md` secrets 规范 | spec 17 |

✅ **密码项豁免审计（2026-08-01，项目所有者裁定）**: 密码存储方案以现状（vault 加密 + 前端按需解密显示明文）为达标态，**不纳入本次验收审计范围**，不回退、不改造。spec 17 / ADR-0011 / migration 189 与现状一致，无冲突，无需修订。
- **审计判定**: 把 MT 密码加密存储或前端明文显示报为"安全漏洞" = 审计误报。
- **仍生效的常设规则**: 任何密码禁止入日志 / metric label / trace span / 错误消息（spec 17 泄漏禁令）。

---

# Part A: Audit（审计方案）

**目的**: 一次性/周期性全面审查，发现已有问题，产出审计报告。**侦探。**

**触发时机**: 大版本前 / 每月一次 / 事故复盘后 / 新人接手模块前。

## A.1 审计流程（6 步）

```
Step 1: 全面扫描
  └── 扫描所有代码，罗列逻辑漏洞、业务异常、安全风险、兼容问题

Step 2: 风险定级
  └── 标注每处问题的风险等级、影响范围、触发场景、危害后果

Step 3: 逐项修复
  └── 遵循原架构逻辑，仅处理问题，不改动无关代码和正常功能

Step 4: 差异逐行比对
  └── 逐行比对修改前后代码，输出版本差异明细

Step 5: 全链路回归
  └── 以 main 全绿为前提：go test -short 全量 + -tags=integration + 核心 E2E 通过，
      核验功能正常无新增 Bug，输出审计报告

Step 6: 交付闭环
  └── 输出合规代码文件，附带修改日志和风险复盘文档
```

## A.2 审计维度（9 维逐项检查）

| # | 维度 | 检查内容 | 判定标准 |
|---|------|----------|----------|
| 1 | **架构** | 功能边界清晰、无跨块直连 DB、接口隔离、无循环依赖 | 每个块只通过公开接口交互 |
| 2 | **最优性** | 算法/数据结构最优、无 O(n²) 当 O(n) 可行、无不必要全表扫描/全量拷贝 | 关键路径复杂度达标；更优方案未采用必有书面理由 |
| 3 | **第一性原则** | 是否从根因解决，而非打补丁 | 无 workaround/hack/legacy/temporary 标记 |
| 4 | **简单性** | 无投机抽象、无过度设计、嵌套 ≤4、圈复杂度达标 | 最简单的正确解；YAGNI 零违反 |
| 5 | **代码整洁** | 死代码、冗余、被注释代码块、lint（全量）、文件行数、命名一致性 | 零死代码，lint 全量全绿 |
| 6 | **技术债** | TODO/FIXME/HACK 注释、跳过的测试、硬编码 | 无未跟踪的技术债（每条必有 issue + 期限） |
| 7 | **正确性** | Decimal 全链路、UTC ms、状态机完整、幂等、边界条件 | Part 0.5 四条全部成立 |
| 8 | **安全** | 鉴权归类、注入、密钥、输入校验、依赖漏洞（密码项豁免，见 Part 0.6.1） | 零 HIGH/CRITICAL；鉴权归类 100% |
| 9 | **合规** | proto 兼容、无禁止项（REST/WS/JSON/float64/nolint）、实现符合规则/规划文档（CLAUDE.md、AGENTS.md、docs/spec/、块级 plans/） | 零规则违反；实现与文档不一致 = 违规（代码错修代码，文档错修文档，裁定结论必须写入审计报告） |

**裁定权**：审计/实施中发现实现与规则文档不符时，执行者不得自行裁定——记录差异并上报项目所有者；裁定结论（修代码或修文档）由项目所有者给出后执行，并写入审计报告。

## A.3 审计报告模板

```markdown
# Audit Report — [日期] — [范围: 全项目 / 某功能块]

## 1. 问题清单

| ID | 维度 | 文件:行 | 风险 | 影响范围 | 触发场景 | 危害后果 |
|----|------|---------|------|----------|----------|----------|
| A-001 | 安全 | handler.go:42 | HIGH | 所有未认证请求 | 不带 token 访问 | 越权操作 |
| A-002 | 整洁 | utils.go:80 | LOW | 仅该文件 | 编译时 | 死代码 |

风险等级: CRITICAL > HIGH > MEDIUM > LOW > INFO

## 2. 修复明细

| ID | 修改文件 | 修改前行 | 修改后行 | 说明 |
|----|----------|----------|----------|------|
| A-001 | handler.go:42 | `mux.Handle(svc)` | `mux.Handle(svc, auth)` | 补充 auth interceptor |

## 3. 回归测试结果

| 测试套件 | 结果 | 新增失败 |
|----------|------|----------|
| go test ./... | PASS | 0 |
| playwright | PASS | 0 |

## 4. 未修复项（附带理由和计划修复时间）

## 5. 风险复盘

- 根因: ...
- 为什么之前没发现: ...
- 防止同类问题: ...
```

**报告落点与跟踪（强制）**：
- 报告存放：`docs/audits/YYYY-MM-DD-<scope>.md`，与历史审计并列，禁止散落在聊天记录/临时文件。
- 发现项 ID 全局递增（A-NNN），在报告 §4 跟踪至关闭；未关闭项必须带理由 + 计划修复时间。
- 每个已修复发现 → 必须产出一条 Part B 预防规则（CI 检查 / 测试 / lint），写入报告 §5 并落实到 PR。**只修不防 = 审计未闭环。**

## A.4 审计频率

```
全项目审计:   每季度 1 次（或大版本发布前）
按块审计:     每月 1 次（轮换，P0 块优先）
事件驱动审计: 生产事故后 48 小时内（针对事故涉及的功能块）
验收期特别条款: 验收签字前必须完成 1 次全项目审计（B.11 Gate 6），
              CRITICAL/HIGH 发现清零方可进入终验
```

## A.5 原 rd.md 要求覆盖确认

| 原要求 | 新文档位置 | 状态 |
|--------|-----------|------|
| 1. 全面扫描代码，罗列问题 | A.1 Step 1-2 + A.2 | ✅ |
| 2. 标注风险等级/影响/触发/后果 | A.3 报告模板 | ✅ |
| 3. 遵循原架构，仅处理问题 | A.1 Step 3 | ✅ |
| 4. 逐行比对修改前后代码差异 | A.1 Step 4 + A.3 修复明细 | ✅ |
| 5. 全链路回归测试，审计报告 | A.1 Step 5-6 + A.3 | ✅ |
| 6. 合规代码 + 修改日志 + 复盘文档 | A.1 Step 6 + A.3 §4-5 | ✅ |
| 7. 逐项审计（架构/最优性/第一性/整洁/技术债） | A.2（已扩展为 9 维） | ✅ |
| 8. 存在的 lint 需要修复 | A.2 维度 5 + Part B Layer 0 | ✅ |
| 9. 最优解/第一性/简单/干净/无BUG/安全（v2 新增） | Part 0 质量宪法 | ✅ |

---

# Part B: Testing Strategy（持续测试方案）

**目的**: 持续预防，每次变更自动验证，阻断问题进入 main。**门卫。**

**触发时机**: 每次 push / PR / merge。

---

## B.1 Philosophy — First Principles

**B.1.1 测试不能证明正确，只能证明错误。** 不追求"覆盖一切"。追求在故障最可能发生的地方，以最低成本找到最大价值的缺陷。

**B.1.2 缺陷发现成本随层级指数上升。**
```
类型检查 < lint < 单元 < 集成 < E2E < 生产事故
```
能左移就左移。每一层只测它最适合发现的那类问题，不重复。

**B.1.3 管线测试优于组件测试。** 系统定义了 6 条管线。组件隔离通过不代表管线通。每条管线至少 1 个 happy-path 集成冒烟。

**B.1.4 测试代码与生产代码同等待遇。** 测试文件适用相同的 lint 规则、文件行数限制、死代码检测。测试中有冗余 = 违规。`t.Skip()` 必须有注释说明原因和恢复条件。

**B.1.5 按风险分配测试投入，不平摊。** 风险 = 影响面 × 故障概率 × 检测难度。P0 块测试密度 > P3 块。

**B.1.6 不测什么和测什么同样重要。**
- ❌ 不测第三方库内部行为（假设其正确）
- ❌ 不测 proto 生成代码的正确性（假设 buf/protoc 正确）
- ❌ 不测 LLM 输出的具体内容（非确定性），只测输出 schema 和工具调用契约
- ❌ 不测 MT4/MT5 broker 的真实行为，只测 adapter 的请求/响应转换

**B.1.7 禁止为通过门禁而降标。** 降低覆盖率门槛、删除/弱化断言、注释掉失败测试、扩大 Skip 范围以使管线变绿 = 违规，视同伪造验收数据。门槛只能随现状快照上调（ratchet），不能下调。

---

## B.2 Layered Testing

### Layer 0: Static Gates（CI 阻断，目标 < 2min）

```yaml
触发: 每次 push
阻断: 任一失败 = ❌
状态: ✅已接入 CI / 🔧待接入（B.9 Phase 1 完成） / ❌不做（需书面理由）

检查项:
  Go:
    - golangci-lint run ./...                  # 🔧 现为 --new-from-rev 增量，须改全量并清零存量
    - go build ./...                           # ✅
    - go vet ./...                             # ✅
    - deadcode ./...                           # 🔧 函数级未接入；包级死代码已由 R0 detect-deadcode 覆盖 ✅
    - govulncheck ./...                        # 🔧 未接入
    - go run ./tools/check-file-lines --strict # 🔧 已接入但缺 --strict（ci.yml 未带该参数）

  TypeScript:
    - npx eslint .                             # 🔧 现为增量 lint（只查改动文件），须改全量
    - npx tsc --noEmit                         # 🔧 npm run build 仅 vite build 不含 tsc；CI 须单独接入
    - npx knip --no-progress                   # 🔧 依赖已装，CI 未接入

  Proto:
    - buf lint                                 # ✅
    - buf breaking --against main              # ✅
    - proto codegen drift 检查                 # ✅（proto-drift job）

  General:
    - 密钥扫描（trivy secret 或 gitleaks）      # 🔧 trivy 已接入，确认 secret 扫描开启或补 gitleaks
    - bash scripts/check_handler_registry.sh   # 🔧 脚本不存在，Phase 1 实现（规格见 B.5）
    - migration .down.sql 配对检查              # ✅
    - gosec（安全静态扫描）                     # ✅（security-scan.yml）
```

### Layer 1: Unit Tests（CI，< 30s）

```yaml
覆盖率目标（per-block 门禁，非全局平均）:
  P0 (risk-gate, mt-gateway):                 ≥ 80%
  P1 (backtest, market-data, strategy-runtime): ≥ 70%
  P2 (marketplace, agent-engine, mql-compiler): ≥ 60%
  P3 (account-mgmt, frontend, api-gateway):   ≥ 50%

门禁机制:
  - 🔧 per-block 覆盖率门禁待接入 CI（当前仅全局 12% 防呆门槛，与上表严重脱节）
  - 接入方式：以 2026-08-01 现状快照为下限先行阻断回退（见 B.3.2），
    再按 B.9 Phase 2 逐块提标至上表目标。门槛只升不降（B.1.7）

范围:
  ✅ 纯逻辑函数、状态机转换、序列化往返、权限逻辑、Decimal 计算、边界条件
  ❌ DB 查询 (→Layer 2)、网络 I/O (→Layer 2)、goroutine 交互 (→Layer 2)

Go:  testing + testify, 表驱动优先, 命名 Test<Type>_<Method>_<Scenario>
TS:  vitest（现状 6 个测试文件，🔧 按 P3 目标扩充）
```

### Layer 2: Integration Tests（PR，< 2min）

```yaml
约定（统一为 ci-nightly 现行方式，v1 三种并存约定作废）:
  - 集成测试统一使用 build tag: go test -tags=integration
  - ❌ 不再使用 -run 'TestIntegration' 命名约定（现存 0 个，名不副实）
  - ❌ 不引入 testcontainers（go.mod 无此依赖；CI services 模式已工作，
       减少移动部件 — 简单性原则 0.3）
  - 依赖：CI 用 GitHub Actions services（PG），本地用 docker compose 起依赖
  - 测试间 DB 隔离；失败保留容器/服务日志

范围:
  ✅ DB 查询（PG via services）
  ✅ handler → service → repository 全链路
  ✅ NATS JetStream pub/sub 契约
  ✅ ConnectRPC 请求/响应序列化
  ✅ Auth interceptor 行为
  ✅ SSE 流生命周期
```

### Layer 3: Pipeline Smoke Tests（PR，< 5min）

```yaml
每条管线: 1 happy-path + 1-2 关键失败场景（契约断言点见 B.4.1）
状态: 🔧 2/6 已存在（P2 risksvc/pipeline_test.go、P4 ai/gate_pipeline_integration_test.go），
      P1/P3/P5/P6 待补（B.9 Phase 2）

P1 行情引入: tick→去重→Bar→PG→NATS                # 🔧 缺
P2 策略执行: Bar→Runner→信号→OMS→风控→MT          # 🔧 部分
P3 订单对账: MT更新→mthub→OMS→NATS                # 🔧 缺
P4 Agent循环: 输入→SSE→generate→compile→backtest→结果  # 🔧 部分
P5 回测:     参数→VMRunner→SimBroker→净值→SSE      # 🔧 缺
P6 实盘调度: LiveRunner→实时bar→信号→OMS→MT       # 🔧 缺
```

### Layer 4: E2E Tests（PR/pre-merge，< 15min）

```yaml
工具: Playwright (tests/e2e/)
浏览器: chromium headless

现状: 19 个 spec files:
  回测冒烟、市场、登录绑定、AI对话、管理页面、i18n、全页扫描、
  交互扫描、回归流、钱包、日期过滤、分享页、导入工作区

门禁分层:
  pre-merge: 🔧 核心 spec 子集（回测冒烟 + 登录绑定）接入 PR 管线
  nightly:   全量 19 spec

待扩展（B.9 Phase 4）:
  - 策略购买完整流
  - Agent 策略生成完整流
  - SSE 断连恢复
  - 并发操作
```

### Layer 5: Security Audit（定期/按需）

```yaml
自动 (每次 push):
  - buf breaking                       # ✅
  - gosec + trivy                      # ✅（security-scan.yml）
  - govulncheck / npm audit / gitleaks # 🔧 未接入 CI（B.9 Phase 1，与 Layer 0 同步清零）

手动 (每月/大版本前):
  - handler 鉴权覆盖率审计
  - SQL 注入扫描
  - Proto 输入校验完整性
  - 依赖许可证合规
```

---

## B.3 Block Risk Matrix & Requirements

| 优先级 | 块 | 致命场景 | 测试重心 | 覆盖率 |
|--------|-----|----------|----------|--------|
| P0 | `risk-gate` | 信号断裂→错误下单→真金损失 | OMS 状态机全覆盖 + 熔断链路 | ≥80% |
| P0 | `mt-gateway` | 下单丢失/重复、持仓不同步 | 幂等门 + 对账门 + 重连恢复 | ≥80% |
| P1 | `backtest-engine` | 撮合偏差→用户误判策略 | SimBroker 撮合 + 滑点/手续费 | ≥70% |
| P1 | `market-data` | 错误价格→全链路污染 | Bar 去重 + 质量门 + 时间轴 | ≥70% |
| P1 | `strategy-runtime` | Bar 重放顺序错→信号错 | Runner 生命周期 + Bar 事件序 | ≥70% |
| P2 | `strategy-marketplace` | 扣款未发货、分账错误 | 状态机 + 金额精度 + 并发 | ≥60% |
| P2 | `agent-engine` | LLM 输出不可用、假成功 | 工具契约 + 回退分支（不测质量） | ≥60% |
| P2 | `mql-compiler` | 编译产物行为不一致、VM 越界 | 基准回归 + IR 语义 + VM 边界 | ≥60% |
| P3 | `account-mgmt` | CRUD 错误、连接校验绕过 | 状态机 + 权限 | ≥50% |
| P3 | `frontend` | UI 错显、SSE 断连无提示 | 组件快照 + 三态覆盖 | ≥50% |
| P3 | `api-gateway` | handler 未注册、序列化错误 | 路由完整性 + proto 兼容 | ≥50% |

### B.3.1 块 → 代码包映射（覆盖率按此度量）

| 逻辑块 | 代码包 |
|--------|--------|
| risk-gate | `internal/risk`, `internal/risksvc` |
| mt-gateway | `internal/mthub`, `internal/mdgateway/adapter/mt4`, `internal/mdgateway/adapter/mt5` |
| backtest-engine | `internal/backtest`, `strategy/backtest` |
| market-data | `internal/mdgateway`（含 backfiller/indicator） |
| strategy-runtime | `strategy/runner`, `strategy/sdk` |
| strategy-marketplace | `internal/marketplace` |
| agent-engine | `internal/agent`, `internal/ai` |
| mql-compiler | `tools/mql2go`（含 interp） |
| account-mgmt | `internal/usermgr` 及账户相关 service |
| frontend | `frontend/src` |
| api-gateway | `internal/server`, `internal/connect`, `internal/interceptor` |

### B.3.2 覆盖率基线快照（2026-08-01 实测，门禁下限）

| 包 | 现状 | 目标 | 差距 |
|----|------|------|------|
| `internal/risk` | 83.7% | ≥80% (P0) | ✅ 达标 |
| `internal/risksvc` | 70.8% | ≥80% (P0) | 🔴 -9.2 |
| `internal/mthub` | 21.1% | ≥80% (P0) | 🔴 -58.9 |
| `internal/oms` | 58.2% | ≥80% (P0, 随 risk-gate) | 🔴 -21.8 |
| `internal/mdgateway` | 45.2% | ≥70% (P1) | 🔴 -24.8 |
| `internal/marketplace` | 0.1% | ≥60% (P2) | 🔴 -59.9 |
| `internal/agent` | 5.7% | ≥60% (P2) | 🔴 -54.3 |

per-block 门禁接入前，上表现状值为回退下限；接入后按 B.9 Phase 2 提标。**只升不降（B.1.7）。**

---

## B.4 Pipeline Testing Contracts

| # | 管线 | 路径 | 致命失败模式 |
|---|------|------|-------------|
| P1 | 行情引入 | mt-gateway→mdgateway(去重/质量)→NATS+PG | 去重误杀→数据缺口 |
| P2 | 策略执行 | market-data→runner→risk-gate→oms→mt-gateway | 信号被风控无声拦截 |
| P3 | 订单对账 | mt-gateway→mthub(幂等/对账)→oms→NATS | 幂等门重复放行→持仓翻倍 |
| P4 | Agent循环 | frontend→api(SSE)→agent-engine→mql-compiler→backtest | 回测假成功→基于假数据决策 |
| P5 | 回测 | frontend→api→backtest-engine→PG→SSE | Decimal累积漂移→曲线错误 |
| P6 | 实盘调度 | frontend→api→runner(LiveRunner)→market-data→mt-gateway | Runner panic→静默停止 |

### B.4.1 管线测试契约（每条冒烟必须断言的点）

- **P1 行情引入**: 重复 tick 只入库一次（去重不误杀相邻合法 tick）；Bar 时间轴连续、无重叠、无缺口；PG 与 NATS 双写一致。
- **P2 策略执行**: 信号被风控拦截时**必须产生可见事件**（拒绝原因入日志 + 可查询），禁止无声拦截；通过的信号到达 OMS 且数量一致。
- **P3 订单对账**: 同一 MT 更新重放 N 次，OMS 持仓不变（幂等门）；重复放行 = CRITICAL 缺陷。
- **P4 Agent循环**: compile 失败的策略禁止进入回测；回测结果与 SSE 推送一致；任何阶段失败向前端返回真实错误而非假成功。
- **P5 回测**: 净值曲线全程 `decimal.Decimal` 计算，断言无 float64 漂移（定点数比对）；SSE 推送点数与引擎产出一致。
- **P6 实盘调度**: Runner panic 必须 recover + 告警 + 状态可查，禁止静默停止；实时 bar 乱序/迟到有定义行为。

---

## B.5 Security Testing

### Handler 鉴权覆盖率

每个 handler 必须归类：
- `authenticated` — 经过 auth interceptor（默认）
- `public` — 明确声明公开（login, refreshtoken, healthz），需注释说明理由

CI 脚本 `scripts/check_handler_registry.sh`（🔧 **当前不存在，B.9 Phase 1 必须实现**）：
1. 扫描所有 `NewXxxServiceHandler` 调用
2. 检查第三个参数是否包含 `authInterceptor`
3. 列出未认证的 handler → ❌ 阻断（或人工标注 public）
4. 公开端点白名单与后端 interceptor 豁免清单双向核对，漂移即阻断

⚠️ **验收前置**：`internal/interceptor` 当前存在失败测试
（`TestExtractClientIPFromHeader/x-forwarded-for_takes_priority`，期望 `1.2.3.4` 实得 `5.6.7.8`）。
客户端 IP 提取优先级涉及限流与审计正确性，必须按根因修复并确认语义，
main 恢复全绿后方可谈验收（B.11 Gate 1）。

### 输入边界参数化测试

所有接收外部输入的 handler 覆盖：
```
空值    "" / 0 / nil
超长    max+1 / max*10
负数    -1 (对正数类型)
特殊    \x00 / <script> / DROP TABLE
边界    max-1 / max / max+1 / min-1 / min / min+1
```

### Proto 向后兼容

`buf breaking` 检查：
- ❌ 删除字段 / 修改字段类型 / 修改字段名 / 修改 enum 已有值
- ✅ 新增字段 / 新增 enum 值 / 新增 RPC 方法

---

## B.6 CI Pipeline

```
Stage 1: Static Gates (并行, 每次 push)
  ✅ go-vet | go-build | proto-lint | proto-breaking | proto-drift
     migration-down | gosec | trivy | R0 detect-all | line-limits(🔧补 --strict)
  🔧 golangci-lint(全量化) | eslint(全量化) | knip | deadcode
     govulncheck | secrets | handler-audit

Stage 2: Unit Tests (并行, 每次 push)
  ✅ go test -short -race -count=1 ./...（全局 12% 防呆门槛）
  🔧 per-block 覆盖率门禁（B.3.1 映射 + B.3.2 基线） | npx vitest run

Stage 3: Integration Tests (PR + nightly)
  🔧 go test -tags=integration -count=1 ./...（现仅 nightly 运行）

Stage 4: Pipeline Smoke (PR)
  🔧 2/6 已存在，P1/P3/P5/P6 待补（契约断言 B.4.1）

Stage 5: E2E
  🔧 pre-merge: 核心 spec（回测冒烟 + 登录绑定）
  🔧 nightly: 全量 19 spec

Stage 6: Deploy (main only, 手动触发)
  ✅ docker compose build backend && docker compose up -d backend
```

---

## B.7 Quality Dimensions (每 PR 门禁)

| 维度 | 准则 | 检查 |
|------|------|------|
| 架构 | 功能边界清晰、无循环依赖、接口优先 | review + CI |
| 整洁 | 零死代码、零 lint、文件行数合规、无 nolint | CI 阻断 |
| 正确 | Decimal 全链路、UTC ms、状态机完整、幂等 | test + review |
| 安全 | 鉴权覆盖、无硬编码密钥、SQL 参数化、输入校验 | CI + audit |
| 可观测 | 错误 wrap、trace_id 贯通、panic recover | review + test |

---

## B.8 Test Environment

```yaml
测试用户:
  Email:    admin@1.com
  Password: 12345678
  Role:     admin

本地环境（以 docker-compose.yml 实际映射为准）:
  前端:  http://localhost:5173 (vite dev) 或 http://localhost:8022 (docker, ANT_FRONTEND_PORT)
  后端:  宿主机直跑 http://localhost:8080 (ConnectRPC)；
         docker 模式无 host 端口映射，经前端 nginx 代理访问（8022）
  PG:    localhost:5433 (docker 映射 127.0.0.1:5433→5432)
  NATS:  localhost:4222
  Redis: localhost:6379

资源约束 (4核/8GB/58GB):
  峰值内存: ~2.5 GB (全部并行)
  Go 测试: -p 2 (4核限制)
  Playwright: 1 worker
```

### Run Commands

```bash
# 全量
cd backend && go test -short -count=1 ./...
cd frontend && npx vitest run
cd tests/e2e && npx playwright test

# 含集成（约定见 Layer 2：build tag 方式）
cd backend && go test -tags=integration -count=1 ./...

# 单包/单测试
cd backend && go test -run TestMarketplaceService_Purchase -count=1 ./internal/marketplace/...

# E2E 调试
cd tests/e2e && npx playwright test --headed --debug

# Lint only
cd backend && golangci-lint run ./...
cd frontend && npx eslint .
```

---

## B.9 Implementation Roadmap

```
Phase R: 止损（Week 0 — 验收前置硬门槛）
  - 修复 internal/interceptor 失败测试（X-Forwarded-For 优先级，按根因）
  - main 全绿：go test -short 0 FAIL、CI 全 job 绿
  - 实现 scripts/check_handler_registry.sh 并接入 CI
  - 本文档（rd.md v2）落地，🔧 项进入跟踪

Phase 1: 门禁对齐（Week 1-2）
  - Layer 0 全量化：golangci-lint / eslint 改全量并清零存量
  - check-file-lines 补 --strict；接入 deadcode / knip / govulncheck / 密钥扫描
  - 集成测试约定统一为 -tags=integration
  - per-block 覆盖率门禁以 B.3.2 现状快照为下限接入（防回退）

Phase 2: P0 密度（Week 2-4）
  - risk-gate / mt-gateway（risksvc/mthub/oms）覆盖率提至 ≥80%
  - marketplace（0.1%）/ agent（5.7%）补至 ≥60%
  - 补齐 P1/P3/P5/P6 管线冒烟（B.4.1 契约断言）
  - 输入边界参数化测试框架落地 P0 块 handler

Phase 3: 首次全项目审计（Week 4-5）
  - 按 Part A 六步执行，报告落 docs/audits/
  - 每个发现 → Part B 预防规则（闭环核验）

Phase 4: E2E + 安全 + 终验（Week 5-6）
  - 核心 E2E 接入 pre-merge；全量 nightly 绿
  - 手动安全审计（鉴权覆盖 / SQL 注入 / proto 校验 / 许可证）
  - 按 B.11 出口标准逐项核验，签字验收
```

---

## B.10 Compliance Checklist

每个 PR 合并前核对：

```
[ ] Static Gates 全部通过
[ ] 新增代码有对应测试（按块的覆盖率目标）
[ ] 无硬编码密钥
[ ] PR 描述含 REUSE:/NEW: 引用
[ ] 无 //nolint, // @ts-ignore, # noqa
[ ] 无 t.Skip()（除非有注释说明原因和恢复条件）
[ ] 无 float64 在价格计算路径
[ ] 新增 proto RPC → handler 已注册 → mux 已挂载
[ ] 新增 handler → 经过 auth interceptor 或标记 public
[ ] 文件行数合规 (Go ≤450, TS ≤375)
[ ] go build ./... 通过
[ ] 无被注释掉的代码块
[ ] 无 workaround/hack/legacy/temporary 注释（Part 0.2）
[ ] 未降低任何既有门槛、未删除/弱化既有测试（B.1.7）
[ ] bug 修复附带回归测试（Part 0.5-3）
[ ] 覆盖率门禁无回退（per-block 基线只升不降）
```

---

## B.11 验收出口标准（Acceptance Exit Criteria）

以下 8 道 Gate **全部通过 = 验收通过**。任一不满足 = 验收不通过，无例外、不谈判。

```
Gate 1 绿:    go test -short ./... 0 FAIL、0 无理由 Skip；
              CI 全部 job 绿；frontend build + vitest 绿
Gate 2 门禁:  本文档所有 🔧 项清零（全部转为 ✅ 或 ❌+书面理由）；
              Layer 0 全量执行（非增量）
Gate 3 覆盖率: per-block 门禁生效；P0 ≥80%、P1 ≥70%、P2 ≥60%、P3 ≥50%；
              无任一块低于 B.3.2 基线快照
Gate 4 管线:  6/6 管线冒烟存在且通过，含 B.4.1 契约断言
Gate 5 E2E:   核心 spec pre-merge 绿；全量 19 spec nightly 绿
Gate 6 审计:  首次全项目审计完成（docs/audits/）；
              CRITICAL/HIGH 发现 = 0；每个已修复发现已转 Part B 预防规则
Gate 7 安全:  handler 鉴权归类 100%；govulncheck/trivy/gosec 零 HIGH/CRITICAL；
              零硬编码密钥；P0 块 handler 边界参数化测试存在；
              密码项豁免审计（Part 0.6.1，现状已达标）
Gate 8 一致:  抽查本文档声称的 CI 检查项与实际 pipeline 一致（文档即事实）
```

**验收期禁令**：
- ❌ 降低任何既有门槛 / 删除或弱化测试以使管线变绿（B.1.7）
- ❌ 为达标编写无断言或弱断言的注水测试（覆盖率达标以有效断言为前提，抽查）
- ❌ 跨范围修改（one task = one scope 仍然适用）

---

# Part C: How Audit & Testing Work Together

```
AUDIT (Part A)                      TESTING (Part B)
────────────────                    ─────────────────
Found: handler 缺 auth              → 新增 CI: check_handler_registry.sh
Found: 死代码在 utils.go            → 新增 CI: deadcode + knip
Found: lint 存量错误                → Layer 0 阻断, 存量清零
Found: 管线 P4 无测试               → Phase 2 补管线冒烟
Found: 边界未校验                   → 输入边界参数化测试框架
Found: proto 字段被删               → CI 加入 buf breaking

每次审计发现 → 写一条预防规则加入 Part B → 同类问题永不复发
```

审计是"找到已经存在的问题"，测试是"防止问题再次出现"。审计跑一次，测试跑一辈子。

---

## References

| 文档 | 路径 |
|------|------|
| 业务方向 | `docs/roadmaps/business-direction.md` |
| 架构总览 | `docs/spec/00-architecture-overview.md` |
| 数据流 | `docs/spec/03-data-flow.md` |
| RPC 契约 | `docs/spec/14-rpc-contracts.md` |
| PG Schema | `docs/spec/09-postgres-schema-catalog.md` |
| 可观测性 | `docs/spec/15-observability.md` |
| SLO | `docs/spec/20-slo.md` |
| 能力目录 | `docs/CAPABILITIES.md` |
| 项目规则 | `CLAUDE.md` / `AGENTS.md` |
| 审计报告 | `docs/audits/` |
