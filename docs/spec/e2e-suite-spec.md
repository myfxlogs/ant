# E2E 测试套件施工 Spec

> **功能块**：跨块（frontend 全链路 + api-gateway + 后端）。**关联**：`launch-readiness-assessment.md`（上线缺口 ①）；REUSE `playwright-backtest` skill（ConnectRPC 登录 + 模板创建 + UI 登录 + workspace + 回测流程）。**状态**：🏗 待施工。**日期**：2026-08-08

---

## 1. 背景

**零 E2E**：无 `playwright.config`、无 `.spec.ts`、无 e2e 目录。核心用户旅程只靠手工回归——购买→实盘、登录→回测→看报告 等主链路无自动化守护。`playwright-backtest` skill 已沉淀了 ConnectRPC API 登录 + 回测的成熟模式（REUSE 入口）。

> **命名消歧**：本 spec = **浏览器/前端全链路 E2E**（Playwright）。与 `p0-e2e-param-test-spec.md`（后端参数链集成测试，Go）是**不同范围**，互不替代——后者守 mql2go 参数链，本 spec 守用户旅程端到端。

## 2. 目标 / 非目标

**目标**：Playwright E2E 套件覆盖**最高风险旅程**，能对 dev/staging 跑、进 CI。重点是钱路径 + 购买→实盘（审计反复验证的核心链路）。

**非目标**：不全量覆盖每个页面；不做视觉回归；单测由 `frontend-test-baseline-spec.md` 负责。E2E 只守"端到端通不通 + 钱路径对不对"。

## 3. 设计

**工具**：Playwright（skill 已选型；支持 ConnectRPC API 速登 + 浏览器双模式）。

**双模式登录**（REUSE skill）：
- **API 速登**（多数测试）：ConnectRPC `Login` 拿 token → 注入 → `storageState` 复用。快，避每测走 UI。
- **UI 登录**（仅登录旅程本身）：真点表单，验登录流。

**测试环境**：dev（`vite` 前端 + 后端）或专用 staging；DB 用 seed 脚本造测试账户/策略/MT mock 账户。MT 网关用 mock（不打真 broker）。

## 4. 旅程（按风险优先）

| # | 旅程 | 为什么最高优先 | 模式 |
|---|------|--------------|------|
| 1 | **登录 → 落地 dashboard** | 鉴权链路；后续旅程前置 | UI 登录 |
| 2 | **marketplace 浏览 → 购买策略 → "我的策略"可见** | 钱路径 + entitlement；购买是收入入口 | API 登录 + UI 操作 |
| 3 | **购买→实盘：调度购买策略 → 看到 live 信号/持仓** | **审计核心链路**（ARCH-3/FEAT-1 修复点）；最易在接缝断 | API 登录 + mock MT |
| 4 | **回测：配参 → 跑 → 看报告** | skill 已有模式，直接 REUSE | skill 全流程 |
| 5 | **账户：加 MT 账户 → 连接 → 看 balance/持仓** | mt-gateway 接入；多状态机 | API 登录 + mock MT |

> 起步先做 1-3（最高风险）；4 复用 skill；5 视进度。

## 5. 实现任务

| # | 任务 | 产出 |
|----|------|------|
| 1 | 装 `@playwright/test` + `playwright.config.ts`（`baseURL`、`webServer` 启 frontend dev + 后端、`storageState`） | `frontend/playwright.config.ts` |
| 2 | Test fixtures：DB seed（测试账户/策略/MT mock）、auth helper（ConnectRPC 登录 → 存 storageState，REUSE skill） | `frontend/e2e/fixtures/` |
| 3 | 旅程 1-3（登录 / 购买 / 购买→实盘） | `frontend/e2e/*.spec.ts` |
| 4 | 旅程 4 回测（REUSE skill 全流程） | `frontend/e2e/backtest.spec.ts` |
| 5 | CI：`npx playwright test`（对 test env；独立 job，不拖慢主 CI） | CI workflow |

> **复用核对**：`playwright-backtest` skill（登录 + 回测全流程，REUSE 主体，勿重写）、ConnectRPC client（`src/client/*.ts`）。NEW：config + fixtures + spec 文件。

## 6. 验收 + 对抗证明

- `npx playwright test` 绿（旅程 1-3 起步）；CI 跑。
- **对抗证明**：在旅程 3 故意回退 ARCH-3 取码源 bug（schedule 读错表）→ E2E 必红（购买策略起不来）；故意退 LIVE-2 幂等（重复信号双发）→ 持仓数翻倍，E2E 断言必红。E2E 必须真能抓住回归，不是只跑 happy path。

## 7. 边界

- MT 网关 mock：不打真 broker，注入 fake quote/order stream。后端 OMS/风控/Gate 全真跑（E2E 的价值在验真实后端链路）。
- 数据隔离：每测用唯一账户或测后清理，防交叉污染。
- 速度：API 速登 + 只验关键断言；E2E 不做细粒度 UI 断言。

## 8. 完工回填

`launch-readiness-assessment.md` 缺口 ① 划掉 + handover 变更日志 + 对抗证明。CI 绿为底线，不自行宣告完成。
