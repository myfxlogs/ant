# AlphaForge — 项目总路线图

> **最后更新**：2026-07-20
> **状态**：M12 完成，P0-P7 全部完成；策略市场路线图已拆分至 [[strategy-marketplace]]

---

## 项目愿景

**"让普通人都能用上量化交易系统"** — 用户用自然语言描述交易想法，AI 自动生成策略代码，在回测/仿真/实盘环境中运行。

核心价值链：

```
行情数据 → 因子 → 信号 → 回测验证 → 仿真交易 → 风控 → 下单 → 持仓 → 复盘
                                                ↑
                                         AI 策略生成（自然语言 → 策略代码）
```

---

## 已完成里程碑

### M7-M10：基础设施 + 金融语义

- ✅ MT4/MT5 gRPC 适配层（mtapi.io 双层代理）
- ✅ 行情网关（tick dedup、CircuitBreaker、PgWriter、PostgreSQL 存储）
- ✅ 策略执行引擎（MQL → AST → Bytecode VM，328 个内置函数）
- ✅ 回测引擎（SimBroker，统一回测/实盘代码路径）
- ✅ 风控引擎（RiskGate，6 条预检规则 + Capability 模型）
- ✅ 订单状态机 + 持仓管理
- ✅ 仿真交易（Paper Trading）
- ✅ 多用户基础设施（user_id + account_id + broker 三维隔离）
- ✅ ConnectRPC + SSE 实时推送架构

### M11：MQL→VM 单一执行管线

- ✅ 清除旧解释器执行路径
- ✅ 清除 Go 代码生成路径（gen.go 仅保留 CLI 调试用）
- ✅ VM 内置函数完善（328 + 24 MQL5 指标）
- ✅ 新增 compile_py.go：Python 子集 → IR → Bytecode VM（Agent 生成路径）
- ✅ 前端修复

### M12：Agent-Native 策略平台

- ✅ AI 策略生成 Agent（Go 进程内 Agent + 双编译前端：MQL/Python 子集 → Bytecode VM）
- ✅ Agent 循环（观察 → 思考 → 行动 → 迭代）
- ✅ 多语言 Agent 提示词（en, zh-cn, zh-tw, ja, vi）
- ✅ Plan Mode 交互（先出方案 → 用户确认 → 生成代码）
- ✅ 策略工作区（Workspace + 代码编辑器 + 回测面板）
- ✅ 策略模板库 + 调度执行
- ✅ 实盘策略管理页（LiveStrategyPage + 信号 SSE 流）
- ✅ 策略市场基础实现（发布/订阅/购买/评分/跟单） → 📋 详见 [[strategy-marketplace]]
- ✅ 自动交易设置
- ✅ 三层记忆系统（全局领域知识 + 用户手写 + Agent 自动写）
- ✅ 模型降级链 + Fail-closed 安全

### 品牌与增长基础设施

- ✅ AlphaForge 品牌改名（前后端全量）
- ✅ SEO 基础设施（prerendered HTML、sitemap、robots、og-image）
- ✅ Umami 自托管分析
- ✅ nginx 资产缓存优化

---

## 进行中 / 近期

### 策略市场深化（P1-P5）

> 📋 详细路线图：[[strategy-marketplace]]

策略市场已从 M12 的基础实现进入深化阶段。核心定位从"卖工具"转为"卖结果"——双边市场模型连接策略提供者与散户交易者。

| Phase | 主题 | 状态 | 说明 |
|-------|------|------|------|
| Phase 1 | 信任基础设施 | ⏳ 待实施 | 实盘业绩跟踪、回测质量门槛、提供者验证、风险声明合规 |
| Phase 2 | AI 策略供给 | ⏳ 待实施 | AI 一键生成+自动上架、批量生成队列、参数模板 |
| Phase 3 | 增长引擎 | ⏳ 待实施 | 排行榜、免费试用、策略对比、通知系统、社交分享 |
| Phase 4 | 平台运营 | ⏳ 待实施 | Admin 管理面板、退款 UI、收入仪表盘、版本管理、优惠券 |
| Phase 5 | 护城河 | ⏳ 待实施 | 捆绑包、跟单 UI、阶梯费率、白标/API |

### P0：上线准备 ✅ 已完成

| 任务 | 状态 | 说明 |
|------|------|------|
| Google Search Console | ✅ | 域名级验证通过，sitemap 已提交 |
| Cloudflare 缓存策略 | ✅ | index.html DYNAMIC，不被 CDN 缓存 |
| 域名切换 | ✅ | alfq.org 通过 Cloudflare 正常访问 |
| Umami 验证 | ✅ | /umami/script.js 正常返回 JS |
| OG image | ✅ | /og-image.svg 返回 image/svg+xml |
| robots.txt | ✅ | 正常返回 |
| sitemap.xml | ✅ | 正常返回，5 个 URL |
| SEO meta 标签 | ✅ | og/twitter/keywords 全部正确 |

### P1：用户体验完善

| 任务 | 优先级 | 说明 |
|------|--------|------|
| 策略分享页优化 | 高 | ✅ 完成 — 暗色模式适配 + 响应式图表高度 + profitFactor 边界修复 + 分享管理弹窗移动端适配 |
| 前端 i18n 完整性审计 | 中 | ✅ 完成 — 修复 8 个文件的硬编码字符串：VersionHistoryDrawer/LiveSchedulesTab/AICodeReviseChat/AlgoSubmitForm/TradePasswordModal/AIGatewayModals/SystemConfigEditModal/MarketplacePage/MonitoringPage |
| 账户连接向导 UX | 中 | ✅ 完成 — Step2 增加别名输入框、Step3 增加服务器地址显示 + 错误恢复"修改凭据"链接、5 语言 i18n 键补全 |
| 错误提示友好化 | 中 | ✅ 完成 — 补全 ConnectRPC Code→i18n 映射（6 个新增）、添加 wallet 余额不足 i18n 键（5 语言）、修复 transport.ts 中文 defaultValue→英文、getErrorMessage 增加 ConnectError code fallback |

### P2：策略能力扩展

| 任务 | 优先级 | 说明 |
|------|--------|------|
| Python 子集策略语言完善 | 高 | ✅ 完成 — compile_py.go 将 Python subset 编译为 Bytecode VM 执行，非独立 Python 运行时 |
| 策略回测增强 | 中 | ✅ 完成 |
| 多品种策略支持 | 中 | ✅ 完成 |
| 策略版本管理 | 低 | ✅ 完成 |

### P3：平台商业化

| 任务 | 优先级 | 说明 |
|------|--------|------|
| 用户计费体系 | 高 | ✅ 完成 — WalletService 扣费 + AITokenUsageRepository token 用量统计 + MonthlyCost/MonthlyRuntimeMinutes |
| 订阅计划 | 高 | ✅ 完成 — SubscriptionService: Free/Pro/Enterprise、月付/年付、自动续费、按比例退款换计划 |
| 策略市场交易 | 中 | ✅ 完成 — MarketplaceService: 发布/订阅/购买策略、免费/付费模式 |
| 用户注册流程完善 | 中 | ✅ 完成 — email_verification.go + VerifyEmail 前端页 + WelcomeModal 3 步引导 |
| 管理后台完善 | 中 | ✅ 完成 — UserManagement + BillingManagement + StrategyManagement + ShareManagement + MonitoringPage(SSE) + AdminSettingsPage |

### P4：技术债务 & 可靠性

| 任务 | 优先级 | 说明 |
|------|--------|------|
| Go module path 重命名 | 低 | ✅ 完成 — `anttrader/` → `alphaforge/`（全量 import 重写 + binary rename + buf regenerate） |
| Proto i18n 品牌同步 | 中 | ✅ 完成 — textproto/proto 源文件 AntTrader → AlphaForge，buf regenerate |
| Docker 容器名统一 | 低 | ✅ 完成 — `ant-backend` → `alphaforge-backend` 等（volume 名保留，需数据迁移时再改） |
| E2E 测试覆盖 | 中 | ✅ 新增 4 个测试文件：login-bind-account（登录+绑定向导 5 步）、i18n-language-switch（语言切换+登录页 i18n）、error-i18n（ConnectRPC 错误码→用户提示）、share-page（分享页面渲染+无效 token） |
| 监控告警 | 中 | ✅ 完成 — Prometheus /metrics 端点 + promauto counters/histograms + MonitoringPage SSE 实时面板 |

### P5：代码质量加固

| 任务 | 优先级 | 说明 |
|------|--------|------|
| float64 清理 | 高 | ✅ 完成 — strategy_templates.go 3 个策略模板（MA/RSI/Bollinger）全部改用 decimal.Decimal，移除 InexactFloat64() |
| E2E 测试实跑验证 | 高 | ✅ 完成 — 31/33 passed（1 pre-existing backtest-venus UI 选择器问题，1 flaky share-page 单独跑通过） |
| TODO/FIXME 清零 | 中 | ✅ 完成 — 唯一真实 TODO（feedbackSystemTemplate i18n）已修复：添加英文模板 + Locale 字段 + 本地化 hints/gate failure/user message |
| PG LISTEN 连接池 | 中 | ✅ 已实现 — pglisten.Listener shared-listener fan-out：每 channel 一个 PG 连接，多 SSE 订阅者共享，无需修改 |

---

### P6：工程级加固

| 任务 | 优先级 | 说明 |
|------|--------|------|
| connect/ 层单元测试 | 高 | ✅ 完成 — marketplace/user/strategy 38+ 测试覆盖 auth 验证、错误码映射、边界条件、parseDecimal |
| golangci-lint v2 + ESLint | 高 | ✅ 完成 — golangci-lint v2.12.2 CI 集成，ESLint flat config CI 集成 |
| Sentry 错误追踪 | 中 | ✅ 完成 — 后端 sentry-go（HTTP panic recovery + ConnectRPC error interceptor），前端 @sentry/react，env: SENTRY_DSN/SENTRY_ENVIRONMENT/SENTRY_RELEASE |
| API rate limiting | 中 | ✅ 完成 — RateLimitInterceptor 扩展至 6 端点（Login/Register/GenerateStrategy/Plan/Analyze/Complete），per-IP token bucket |
| 前端 Vitest 测试 | 中 | ✅ 完成 — 73 测试覆盖 amount/price/streamErrors/accountStatus/paramLabel 核心工具模块 |

---

### P7：生产安全加固

| 任务 | 优先级 | 说明 |
|------|--------|------|
| 端口暴露修复 | 🔴 高 | ✅ 完成 — PG/Redis/NATS/Umami/Prometheus/AlertManager 从 0.0.0.0 改为 127.0.0.1 绑定，仅前端 8022 对外（经 Cloudflare tunnel） |
| Umami healthcheck 修复 | 中 | ✅ 完成 — localhost → 127.0.0.1（IPv6 解析导致 wget 连接失败） |
| Sentry DSN 配置 | 低 | ⏳ 待配置 — 代码已就绪，需在 .env 设置 SENTRY_DSN 后重建后端激活 |

---

## 技术架构概览

```
┌──────────────────────────────────────────────────────────────────┐
│  Frontend (React + Ant Design + ConnectRPC + SSE)                │
│  ┌──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┐      │
│  │ Auth │ Dash │ AI   │ Str  │ Live │ Mkt  │ Trade │ Admin│      │
│  └──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┘      │
└──────────────────────────┬───────────────────────────────────────┘
                           │ ConnectRPC + SSE
┌──────────────────────────┴───────────────────────────────────────┐
│  Backend (Go)                                                     │
│  ┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐         │
│  │ AI  │ Str │ Mkt │Paper│ OMS │Risk │ Not │ Sys │User │         │
│  └─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┘         │
│  ┌──────────────────────────────────────────────────┐            │
│  │  MT4/MT5 Adapter (gRPC) │ MD Gateway │ MQL VM    │            │
│  └──────────────────────────────────────────────────┘            │
└───────┬──────────────┬──────────────┬──────────────┬─────────────┘
        │              │              │              │
   PostgreSQL                     Redis         NATS JS
   (业务+行情数据)                 (缓存)        (事件流)
```

---

## 关键 ADR 索引

| ADR | 标题 | 状态 |
|-----|------|------|
| 0003 | 直连 mtapi 不封装 | Accepted |
| 0012 | 统一回测/实盘代码路径 | Accepted |
| 0022 | MQL 盲区架构 | Accepted |
| 0023 | AST 解释器 + MQL 源码为唯一真实来源 | Accepted |
| 0024 | Agent-Native 策略平台 | Accepted |
| 0025 | Agent UX 与自我进化 | Accepted |

---

## 详细路线图索引

| 文档 | 说明 |
|------|------|
| [[strategy-marketplace]] | 策略市场完整路线图（6 Phase，含现状分析、依赖关系、指标） |
| [[live-strategy-user-facing]] | 实盘策略面向用户的剩余工作 |

