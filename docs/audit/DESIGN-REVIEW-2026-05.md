# DESIGN-REVIEW-2026-05 — ant 项目设计审查

**Date**: 2026-05-23
**Scope**: ant 主仓全量（backend / strategy-service / frontend / proto / docs / docker）
**Author**: 设计审查（自动产出，待人工 review）
**关联**: `@/opt/ant/AGENT.md` · `@/opt/ant/docs/AlfQ功能迁移计划.md`

> **审查口径**：本报告对照 `AGENT.md` 硬性规则、AlfQ 迁移计划、用户体验最佳实践、量化交易系统典型工程化要求，对仓库现状作客观盘点。每条 finding 给出严重度（CR/MR/MN）、证据、影响、建议方向；不做单方面拍板。

---

## 0. 执行摘要

| 维度 | 结论 |
|---|---|
| **仓库实际成熟度** | 远超 `AGENT.md` 自述的"M0 脚手架"。**93k LOC Go + 41k LOC TS + 3.7k LOC Py + 64 migrations + 116 proto + 73 前端页面**，是从 anttrader 继承的成熟代码库 |
| **AGENT.md 自述 vs 实际** | 严重偏差。`AGENT.md` 把它写成"启动期空仓"，实际是"已有大量功能但工程纪律未补齐"的状态 |
| **核心架构亮点** | ① Connect RPC + SSE 已落地；② Python 沙箱三层防御（AST + RestrictedPython + 受限 globals）+ iptables 出口隔离；③ proto 单一源 + buf 生成；④ 多语言 i18n（5 语种） |
| **核心架构隐患** | ① OMS 状态机缺失（订单丢状态/重复下单风险）；② 风控写在 JSONB 字段，不可插拔；③ symbol 是裸字符串，跨 broker 下单会失败；④ Repository 层是手写 sqlx 与 `AGENT.md`「sqlc 优先」相悖 |
| **测试与回归** | ① 后端 26 测试文件 / 93k LOC；② **前端 0 测试**；③ 64 up-migrations 仅 22 down-migrations |
| **代码品质** | ① TS `any` 出现 522 次（strict 形同虚设）；② Go `errors.New("…") / fmt.Errorf("…")` 裸字符串 632 处；③ 多个 Go 文件 >700 行（`AGENT.md` 上限 300）；④ 多个 TSX 文件 >700 行（上限 250） |
| **可观测性** | ① `trace_id` 0 命中（`AGENT.md` 强制必带）；② 无 `/healthz` `/readyz` 实现；③ 已用 `zap` 结构化日志（521 处） |
| **Git 卫生** | 已 git 化，但 ① 41MB + 45MB 二进制文件入仓；② 大量生成 proto 代码入仓；③ proto 生成路径深嵌套 `backend/gen/proto/anttrader/gen/proto/` |
| **UX** | ① App 主路由集中、过载（lazy 30+ 页）；② 无统一错误展示规范；③ 移动端响应式状态未知；④ 无可访问性（a11y）基线 |

**Top-3 立刻该补的工程基线**（详见 §5 优先级建议）：

1. **CR-01 OMS 状态机**：金融系统不能没有 ─ 阻塞 M2
2. **CR-02 移除入仓二进制 + 修 .gitignore**：影响所有协作者拉代码体验
3. **CR-03 测试矩阵 + CI**：当前测试稀薄 + 0 前端测试 + 无 CI Workflow，全部靠人工

---

## 1. 工程化缺陷（按严重度）

### CR — Critical（阻塞生产 / 影响项目整体可走）

#### CR-01｜OMS 状态机缺失，所有订单"裸发"

- **证据**：`backend/internal/service/strategy_schedule_runner_loop.go`（503 行）+ `execution_gateway.go`（531 行）直接调 broker；无 `internal/oms/` 目录；orders 表无 `state` 字段（`migration 001_init.up.sql`）
- **影响**：服务重启 / 网络抖动后订单状态无法重建；重复下单、漏单、风控旁路全部可能；`AlfQ 迁移计划 §M2` 已明确点名
- **建议**：M2 阶段移植 AlfQ `oms.OrderExecutor` + `state machine`；orders 表加 `state int` + `broker_symbol_raw`；老订单默认 `SUBMITTED`

#### CR-02｜入仓二进制 + 入仓生成代码

- **证据**：
  - `backend/main` 41 MB、`backend/server` 45 MB 已被 `git` 跟踪（编译产物）
  - `backend/gen/proto/anttrader/gen/proto/*.pb.go` 全部入仓（生成代码）
  - `backend/mt4/mt4.pb.go` 405 KB、`backend/mt5/mt5.pb.go` 496 KB 入仓
- **影响**：① 仓库膨胀，clone 慢；② 历史无法清理（每次重新构建产物 commit hash 都变）；③ 把生成代码当源码 review 浪费精力；④ 与 `AGENT.md` "代码生成优先" 纪律冲突
- **建议**：
  - 立即 `git rm --cached backend/main backend/server`
  - 把 `backend/gen/` `frontend/src/gen/` `**/*.pb.go` `**/*_pb.ts` 加入 `.gitignore`
  - CI 跑 `make proto` 重生成；本地开发同此
  - 历史清理可选：`git filter-repo` 或留作"既成事实"，新仓约束

#### CR-03｜测试矩阵稀薄 + 无 CI

- **证据**：
  - 后端 `*_test.go` = 15 个跟踪文件 / 93k LOC（覆盖率估 < 5%）
  - **前端 `*.test.* / *.spec.*` = 0 个 / 41k LOC**（完全无测试）
  - strategy-service `tests/` 11 个 Python 测试（占比相对最高）
  - `.github/workflows/` 内容未充实（需进一步确认）
  - `AGENT.md` 第 7 条要求 `go test -race`、CI 检测，但仓库无 CI 矩阵
- **影响**：M1-M5 任意改动都可能静默回归；与 `AGENT.md` 防偷懒第 5 条「禁止删/弱化测试」配套的"基线"无从对比
- **建议**：
  - M0.X 立即建 `.github/workflows/ci.yml`：lint + test + build + buf breaking + migrate dry-run
  - 前端引入 vitest + Testing Library，先把 `auth` `accounts` `dashboard` 三个核心页面覆盖
  - 后端先把 `service/` 下 ≥300 行的 10 个 fat-service 加 happy-path 测试

#### CR-04｜Repository 层与 `AGENT.md` 「sqlc 优先，不引入 ORM」 矛盾

- **证据**：`backend/internal/repository/*.go` 36 个文件全部手写 `sqlx`；无 `sqlc.yaml`、无 sqlc 生成产物
- **影响**：手写 SQL 类型安全弱，重构数据库 schema 时纯人工同步；与 AGENT.md 工程纪律明文冲突
- **建议**：两条路二选一（**需决策**）：
  - **(A)** 走 ADR 0006，接受现状改写成「使用 sqlx，配合 strong testing」；并删除 `AGENT.md` 关于 sqlc 的承诺
  - **(B)** 立项 ADR 0006，引入 sqlc，新模块先用，旧模块按 M2-M5 顺序迁移
  - 推荐 B，但 M0 先决策

### MR — Major（强烈建议解决，但短期可缓）

#### MR-01｜风控不可插拔（JSONB 字段 + 硬编码逻辑）

- **证据**：`Strategy.RiskControl JSONB` + `auto_trading_service_risk.go` 等硬编码；无 `internal/risksvc/`
- **影响**：每加一条规则要改代码 + 改 JSON 模板；风控决策不可审计、不可单测
- **建议**：M3 阶段移植 AlfQ `risksvc.Engine + Rule` 接口；新增 `user_risk_profiles` 表

#### MR-02｜symbol 是裸字符串，跨 broker 不可移植

- **证据**：`Strategy.Symbol string`；无 `canonical_symbols` / `broker_symbols` 表；`migration 001` 起的 schema
- **影响**：BTCUSDm vs BTCUSD vs BTCUSDpro 等 broker-specific 命名导致下单失败 / 信号丢失
- **建议**：M1 阶段移植 AlfQ `symbolsync` + `SymbolResolver`；新增 3 张表 + 数据迁移

#### MR-03｜文件复杂度严重超 `AGENT.md` 上限

- **证据**：
  - Go ≥300 行：`debate_v2_service.go (722)` `admin_service.go (661)` `python_strategy_service.go (622)` `strategy_schedule_runner_loop.go (503)` 等 10+ 文件
  - TS ≥250 行：`DebatePageV2Steps.tsx (793)` `StrategyTemplatePage.tsx (750)` `StrategySchedulePage.tsx (683)` `Summary.tsx (641)` 等 10+ 文件
  - i18n 资源文件巨大：`zh-cn/strategy.ts` 615 行、`en/index.legacy.ts` 752 行
- **影响**：CR-CI 真启用后会卡死所有 PR；阅读和测试成本高
- **建议**：
  - i18n 资源文件可豁免（数据文件不应参与复杂度上限）
  - 业务代码列入 M0/M1 重构清单：fat service 拆 sub-service；fat page 拆 components/hooks

#### MR-04｜Go 错误处理违反 `AGENT.md` 「errs 包，禁裸字符串」

- **证据**：`errors.New("…") / fmt.Errorf("…")` 裸字符串 632 处；无 `internal/errs/` 包；前端无错误码映射
- **影响**：错误码无法做 i18n、不可程序化判别；用户看到原始英文/中文混杂
- **建议**：建立 `internal/errs/` 错误码 enum + 中文/英文 message map；ADR 0010 立项；逐步替换

#### MR-05｜可观测性缺口（trace_id / health / metrics）

- **证据**：
  - `trace_id / TraceID / traceID` 全仓 0 命中（`AGENT.md` 「日志必带 trace_id」）
  - `/healthz` 仅出现在 `system_ai_adapter.go`（非真 health 端点）；docker-compose `backend` 容器无 healthcheck
  - 已用 zap 结构化日志（521 处带字段），方向对，但未贯穿请求链路
- **影响**：线上排障靠 grep；compose 重启策略形同虚设；与 SLO 体系不接轨
- **建议**：
  - ADR 0011 / 0012 立项：① Connect interceptor 注入 trace_id（OpenTelemetry 兼容格式）；② 标准 `/healthz` `/readyz` `/metrics` 三端点；③ Prometheus exporter

#### MR-06｜数据迁移不可回滚率 66%

- **证据**：64 `.up.sql` / 22 `.down.sql` ─ **42 个 migration 缺 rollback**
- **影响**：生产数据迁移出问题无法回滚；与 AlfQ 迁移计划第 7 条直接冲突
- **建议**：
  - 历史 migration（编号 < 064）：低优先级补 down，新生事物（M1+ 新加的）必须配套 down
  - CI 加 lint：新 PR 中任意 `*.up.sql` 必须配 `*.down.sql`

#### MR-07｜docker-compose 无 backend healthcheck，frontend healthcheck 路径未对齐

- **证据**：
  - `docker-compose.yml` `backend` 块无 `healthcheck`
  - `frontend/Dockerfile` healthcheck 走 `/health`，nginx 配置需确认
  - `strategy-service` 也无 healthcheck
- **影响**：`depends_on: condition: service_healthy` 形同虚设；启动顺序不确定
- **建议**：与 MR-05 一并 ADR；compose 三业务容器都要 healthcheck

### MN — Minor（值得做，但不阻塞）

#### MN-01｜TypeScript `any` 522 处，strict mode 形同虚设
- 集中在 `StrategySchedulePage` 等大页面
- 建议：CI 引入 `eslint --max-warnings=0` + `@typescript-eslint/no-explicit-any: error`；新代码禁止，存量打 TODO

#### MN-02｜frontend `console.log` 仅 3 处，但无统一日志层
- 浏览器侧无 trace_id 关联；建议封装 `frontend/src/utils/logger.ts`，对接后端 `/api/log/client`（如需）

#### MN-03｜proto 生成产物路径异常
- `backend/gen/proto/anttrader/gen/proto/...pb.go` 四级嵌套，原因是 `paths=import` + go module 路径 `anttrader`
- 建议：`buf.gen.yaml` 改 `paths=source_relative` + 出 `backend/gen/proto/`，或调整 `go_package` option

#### MN-04｜.env.example vs .env 不同步风险
- `.env` 195 字节，`.env.example` 1780 字节（差距 9×）—— example 比实际多很多 key？还是 example 包含说明？
- 建议：CI 加 `make env-check`，diff `.env.example` 与运行时所需变量

#### MN-05｜i18n 中存在 `.legacy.ts` 文件（752 行）
- `frontend/src/i18n/resources/{en,ja}/index.legacy.ts`
- 建议：M0 末期清理 legacy；统一到 namespace 切分（按域：auth/account/strategy/...）

#### MN-06｜Sandbox iptables 规则全部 `|| true`
- `strategy-service/docker-entrypoint.sh:6-19` 所有 iptables 规则失败时静默通过
- 建议：把 `OUTPUT DROP` 改为强制；失败应让容器 fail-fast；或显式探测 NET_ADMIN 能力

#### MN-07｜Go module 路径与项目名不一致
- `backend/go.mod` `module anttrader`，但项目叫 `ant`
- 影响：长期可能引起命名混淆（IDE goto 跳转、ADR 引用）
- 建议：M0 末期一次性 `gomod rename` 至 `ant`，会破坏所有 import 路径——慎重；或保持 `anttrader` 不动并 ADR 记录"历史包名"

---

## 2. UX 缺陷（按用户感知度）

### CR — 直接劝退用户

#### UX-CR-01｜首屏与路由集中加载 → 白屏期长 / 错误难恢复
- `App.tsx` 集中 import 30+ lazy 路由 + 5 套 antd locale + 5 套 dayjs locale；初始加载即拉取
- 无显式 ErrorBoundary（仅看到 `Suspense` fallback `<Spin>`）
- 建议：① 路由按域拆分子 router；② 全局 ErrorBoundary + 上报；③ 语言包按需加载（仅当前 locale）

#### UX-CR-02｜AI 生成策略全过程的进度可见性不足
- `debate_v2` SSE 事件存在，但前端是否有"思考流式输出 + 取消按钮 + 失败重试"的统一交互未确认
- 建议：在 ADR 中沉淀 AI 长任务交互规范（SSE 心跳、断流重连、taken/canceled/failed 三态）

### MR — 影响留存

#### UX-MR-01｜错误信息无 i18n + 无统一展示
- 后端 632 处裸字符串错误，前端拿到的 `message` 既有中文也有英文
- 建议：与 MR-04 联动；前端建立 `<ErrorMessage code="..." />` 组件

#### UX-MR-02｜移动端 / 响应式未知
- 73 个页面，全部基于 `antd` Pro 风格 ─ `antd` 的 PC 桌面默认布局，移动端体验未做针对性设计
- 建议：M0 阶段 决策"是否做 PWA / 移动 H5"；做则单独 ADR；不做则在 README 写明"PC 优先"

#### UX-MR-03｜无可访问性（a11y）基线
- 未发现 `aria-*` 显式使用（需 grep 验证），无键盘导航测试
- 建议：决策是否纳入 NFR；不纳入则明文写"暂不保证 a11y"

#### UX-MR-04｜国际化覆盖不均
- 5 套 locale 但 `*.legacy.ts` 暗示历史断层；`vi`（越南语）资源是否完整需验证
- 建议：M0 出 i18n 治理 ADR，明确支持语种 + 退化策略（缺翻译时 fallback 到 zh-CN 还是 en）

### MN

#### UX-MN-01｜空数据 / 加载中 / 错误三态组件未抽象
- 估计在各 page 重复实现；建议封装 `<DataState />`

#### UX-MN-02｜表单验证错误反馈风格不一致
- 多个 strategy 编辑页，验证规则散落；建议接入 `react-hook-form` + `zod`

#### UX-MN-03｜AI 配置页 `SystemAI.tsx` 646 行 → 单页过载
- 建议拆成 tab + sub-page

---

## 3. 安全 / 合规（专项）

| # | 项 | 严重度 | 说明 |
|---|---|---|---|
| SEC-1 | `.env` 195 字节，**已 gitignore**（`@/opt/ant/.gitignore:34`） | OK | 但 `.env.example` 须包含全量 placeholder |
| SEC-2 | JWT_SECRET 走 `.env` 注入，但 secret rotation 机制？ | MR | ADR 0007 "用户中心架构" 同期决策 |
| SEC-3 | 沙箱 iptables `|| true` 失败静默 | MN-06 | 同上 |
| SEC-4 | 用户 Python 代码运行在 `strategy-service` 进程内（in-process） | MR | 同进程隔离弱：① CPU 时间 / 内存 / FD 限制有无？② 多用户代码并发是否互相影响？ADR 0004 沙箱方案需深化 |
| SEC-5 | 策略市场（M5）作者刷数据风险 | 大 | AlfQ 迁移计划已识别（locked_hash 设计） |
| SEC-6 | API 限流 / 防爬 / 防刷 | 未知 | 未发现 ratelimit interceptor；ADR 待立 |
| SEC-7 | broker secret（账号密码）存储 | 未知 | `internal/pkg/secretbox/` 看起来在做加密；需要审 ADR-0008 |

---

## 4. 文档 / 治理（专项）

| # | 问题 | 建议 |
|---|---|---|
| DOC-1 | `AGENT.md` 自述 M0，实际是后期项目 → Agent 容易据此误判 | 本审查后立即修订 `AGENT.md "当前阶段" §`：实际处于"功能完整、工程纪律待补"的 M0.5 |
| DOC-2 | 86 篇 docs 散落在 `docs/`、`docs/专项设计/`、`docs/remediation-2026-05/`、`docs/进度/` | M0 末期建立 `docs/README.md` 导航 + 弃置 obsolete |
| DOC-3 | `docs/remediation-2026-05/` 14 个文件用途不明 | 决策：保留（旧补救计划，归档）/ 删除 |
| DOC-4 | `AGENTS.md`（含 `@RTK.md`）位置歧义 | `AGENTS.md` 是大写复数，`AGENT.md` 是单数；多数 AI 工具识别 `AGENTS.md`；建议统一 |
| DOC-5 | ADR 体系 0 篇 | 本次产出 ADR 0001-0009 骨架（M0 任务） |

---

## 5. 优先级建议（顶层路线）

下表把 finding 映射到 milestone（与 AlfQ 迁移计划对齐 + 补 M0.X 工程基线）。

| Milestone | 持续 | 关键卡片（按本审查编号） | 验收 |
|---|---|---|---|
| **M0.0 现状** | — | 已完成：git 化、docker 构建通、版本基线升级（PG18/Go 1.26/Py 3.14/Node 24/TS 5.9） | ✅ |
| **M0.1 工程基线** | 1 周 | CR-02（去二进制+gitignore）、CR-03（CI workflow）、MR-06（migration .down 政策）、MR-07（healthcheck） | CI 绿色；clone 后 `make verify` 跑通 |
| **M0.2 文档与 ADR** | 1 周 | DOC-1/4/5；ADR 0001-0009 全部 Accepted | `docs/adr/README.md` 列全 ADR |
| **M0.3 复杂度与品质红线** | 1 周 | MR-03（拆 fat-file）、MR-04（errs 包雏形）、MN-01（eslint any 报错）、MR-05（trace_id interceptor） | golangci-lint + eslint 全绿 |
| **M1 canonical symbol** | 2 周 | MR-02 + AlfQ §M1 | broker 绑定后 `SELECT * FROM broker_symbols` 非空 |
| **M2 OMS 状态机** | 2 周 | CR-01 + AlfQ §M2 + CR-04（sqlc 引入决策） | 100% 订单经 OMS；`risk_events` 审计入库 |
| **M3 风控引擎** | 1.5 周 | MR-01 + AlfQ §M3 | 8 条规则单测全过；老 RiskControl 数据迁移完成 |
| **M4 AI 自然语言→canonical** | 1 周 | UX-CR-02 + AlfQ §M4 | 10 条 NL 样例 100% 输出 canonical |
| **M5 策略市场** | 4-6 周 | AlfQ §M5 + UX-MR-02（移动端决策） | 跟单端到端 demo 通过 |
| **M6 anttrader 退役** | 2 周 | AlfQ §M6 | 数据迁移 dry-run 通过 |

---

## 6. 待用户决策清单（产出 ROADMAP / ADR 之前）

| ID | 决策点 | 选项 | 我的倾向 |
|---|---|---|---|
| D-01 | sqlc 引入策略（CR-04） | A. 全面 sqlc，迁现有 / B. 接受 sqlx 现状，删 sqlc 承诺 / C. 新模块用 sqlc，存量保 sqlx | **A**，但 M0 不动，列入 M2 一并 |
| D-02 | 入仓二进制 / 生成代码清理（CR-02） | A. `git rm` + `filter-repo` 清历史 / B. 仅 `git rm --cached` 保历史 | **B**（不破坏历史） |
| D-03 | go module 名（MN-07） | A. 改名 ant，破坏 import / B. 保留 anttrader，文档化 | **B** |
| D-04 | 移动端支持（UX-MR-02） | A. PWA / B. 单独 H5 / C. 仅 PC | 需你定 |
| D-05 | a11y 基线（UX-MR-03） | A. 纳入 NFR / B. 暂不保证 | **B** 短期 |
| D-06 | 沙箱进程模型（SEC-4） | A. 保持 in-process / B. fork-per-execution / C. nsjail 子进程 / D. WASM | 需你定（M3 末期前必须） |
| D-07 | 错误码体系（MR-04） | A. 自建 errs 包 + i18n / B. 用 Connect Code + grpc-status / C. 混合 | **A**（散户产品，前端友好） |
| D-08 | i18n 支持语种（UX-MR-04） | 当前 5 语种是否全部保留？ | 需你定 |
| D-09 | M5 策略市场首期商业模式 | A. 月租 / B. 一次性买断 / C. 利润分成 / D. 全部 | AlfQ 计划倾向 A+B |
| D-10 | broker 拓展计划 | A. 仅 MT4/MT5 / B. + 币安等加密 / C. + A 股券商 | 需你定 |

---

## 7. 限制与未审查项

- **未跑代码 / 测试**：本审查全部基于静态扫描；动态行为（实际撮合、SSE 流稳定性、内存压力）未验证
- **未审 anttrader 生产数据**：`/opt/anttrader/` 实际运行数据 schema 与 ant 的差异需 M6 单独评估
- **未深度审 AI prompt**：`debate_v2_prompts.go` 等内容未读，可能影响 M4 决策
- **未跑 dependency tree**：Go / npm / pip 依赖白名单（依赖白名单待建立）的完整盘点未做

---

> **后续**：基于本审查，将产出 `docs/plan/ROADMAP.md` + `docs/adr/0001-0009-*.md` 骨架；每个 ADR 把上面 D-01..D-10 中相关决策作为 `Status: Proposed` 列出选项，等待人工拍板。
