# Strategy Marketplace — Full Roadmap

> **最后更新**：2026-07-20
> **状态**：Phase 0 完成，Phase 1-5 实施文档已就绪
> **关联**：[[project-roadmap]] · [[ADR-0024]]
> **实施文档**：
> - Phase 1: [信任基础设施](plan/marketplace/phase-1-trust-infrastructure.md)
> - Phase 2: [AI 策略供给](plan/marketplace/phase-2-ai-strategy-supply.md)
> - Phase 3: [增长引擎](plan/marketplace/phase-3-growth-engine.md)
> - Phase 4: [平台运营](plan/marketplace/phase-4-platform-ops.md)
> - Phase 5: [护城河](plan/marketplace/phase-5-moat.md)

---

## 市场定位

策略市场是 AlphaForge 触达**散户交易者**的核心渠道——从卖工具转变为卖结果。双边市场模型：

```
策略提供者（开发者 / AI 生成）──→ 策略市场 ──→ 散户交易者
          ↑                          ↓
      获得收益分成              购买 / 订阅策略
          ↑                          ↓
   AlphaForge 收取平台抽成（按策略定价百分比）
```

**核心价值**：
- 对散户：无需编程，浏览器即可发现、购买、运行量化策略
- 对提供者：策略代码不离开平台，杜绝盗版，收入可预测
- 对平台：AI 可直接创造可售卖资产（策略），编译器从产品变成基础设施

---

## Phase 0：已完成（后端核心 + 基础前端）

### 后端 — `backend/internal/marketplace/`（9 文件）

| 功能 | 文件 | 关键实现 |
|------|------|----------|
| 策略上架/下架 | `publish.go` | 原子双表写入（`user_strategy_publishes` + `marketplace_strategies`），所有权校验 |
| 策略列表+搜索 | `publish.go` | pg_trgm 模糊搜索，支持 newest/popular/performance 排序，60s 缓存 |
| 三种定价模型 | `types.go` | `free` / `once` / `subscription`，平台费率从 `system_config` 读取 |
| 付费购买 | `purchase.go` | 原子事务（FOR UPDATE 锁钱包 → 扣款 → 分账 → 创建订阅），幂等键防重 |
| 免费订阅 | `service_subscription.go` | `Subscribe()` 含价格门禁，付费策略必须走 `PurchaseStrategy` |
| 订阅续费 | `service_subscription.go` | 每日午夜批量续费，余额不足自动停用，发布者+平台分账 |
| 退款 | `refund.go` | 全额退款（买家退款 + 发布者扣回 + 平台费处理），发布者余额不足时优雅降级 |
| 评分/评论 | `interactions.go` | UPSERT 评分（1-5），分页评论+用户名关联 |
| 跟单引擎 | `copytrade.go` | ProRata 按权益分仓，信号去重（`copytrade_signals`），并发控制（semaphore 8），风控管线集成 |
| 市场回测 | `backtest.go` | 代码服务端保护（`CanAccessCode`），回测结果流式推送 |
| 发布者统计 | `publish.go` | `GetPublisherStats`：上架数/订阅数/总收入/月收入/平均评分/TOP 策略 |

### 后端 — `backend/internal/connect/marketplace/`（8 文件）

| 功能 | 文件 | 关键实现 |
|------|------|----------|
| 14 个 RPC 端点 | `marketplace_handler.go` + `_subs.go` + `_social.go` | PublishStrategy / Subscribe / Unsubscribe / PurchaseStrategy / ListPublished / ListSubscriptions / RateStrategy / ListRatings / CommentOnStrategy / ListComments / SetStrategyPricing / UnpublishStrategy / GetPublisherStats / RunMarketBacktest |
| 回测 SSE 流 | `marketplace_stream.go` | PG NOTIFY on `backtest_status_change`，30s 轮询兜底，终端状态检测 |
| 认证+鉴权 | 所有 handler | 全部需要认证；SetStrategyPricing/UnpublishStrategy 有 admin guard |

### 数据库迁移（20 个 migration）

核心表：`marketplace_strategies`、`user_strategy_publishes`、`user_subscriptions`、`marketplace_ratings`、`marketplace_comments`、`copytrade_signals`、`subscription_plans`、`user_platform_subscriptions`

### 前端 — `frontend/src/pages/marketplace/`

| 组件 | 功能 |
|------|------|
| `MarketplacePage` | 3 Tab 布局：Market / Purchases / Author |
| `MarketTab` | 搜索+筛选+排序+分页卡片网格 |
| `StrategyMarketCard` | 策略卡片（名称/价格/评分/KPI/标签/购买状态） |
| `StrategyDetailModal` | 策略详情（指标+评分+评论+操作按钮） |
| `PaymentModal` | 支付确认（钱包余额检查+不足引导充值） |
| `ProtectedBacktestPanel` | 回测表单+服务端流式结果+净值曲线图 |
| `PurchaseTab` | 已购策略列表 |
| `AuthorTab` | 发布者统计卡片+已发布策略表 |
| `PublishToMarketModal` | 上架表单（策略库入口） |
| i18n | 5 语言（en/zh-cn/zh-tw/vi/ja） |

### 已知缺陷

| # | 问题 | 影响 |
|---|------|------|
| BUG-1 | 前端 `PublishToMarketModal` 价格模型选项为 `free/once/monthly`，后端只认 `free/once/subscription`，`monthly` 会导致付费购买逻辑失效 | 🔴 阻塞付费订阅 |
| BUG-2 | `copy_trade_links` 表（migration 110）未被任何代码引用，CopyTradeEngine 直接查 `user_subscriptions` | 🟡 冗余 schema |
| GAP-1 | 退款流程无前端 UI | 🟡 用户体验缺失 |
| GAP-2 | Admin 策略管理无前端 UI（SetStrategyPricing/Unpublish 只有 RPC） | 🟡 运营能力缺失 |
| GAP-3 | 平台订阅（Free/Pro/Enterprise）与策略市场无实际联动，所有 tier 标记 `marketplace: true` 但无差异化控制 | 🟡 商业化不完整 |

---

## Phase 1：信任基础设施 🔴 P0

> **目标**：让散户看到策略后愿意掏钱。核心解决"这策略真的赚钱吗？"

### 1.1 实盘业绩跟踪

**Why**: 回测可以美化，实盘不能造假。这是信任的基石。

**Tasks**:
- [ ] 新建 `marketplace_live_performance` 表（strategy_id, date, daily_pnl, daily_return, equity, drawdown, total_trades, winning_trades, created_at）
- [ ] 提供者绑定实盘 MT 账户后，平台自动采集日度交易数据写入 performance 表
- [ ] 新增 RPC `GetLivePerformance(strategy_id) → {daily_performance[], summary_stats}`
- [ ] 策略详情页前端展示实盘净值曲线 + 月度收益热力图
- [ ] 区分"回测业绩"和"实盘业绩"两个 Tab

**Files**:
- `backend/migrations/xxx_marketplace_live_performance.up.sql` (new)
- `backend/internal/repository/marketplace_performance_repo.go` (new)
- `backend/internal/connect/marketplace/marketplace_handler_performance.go` (new)
- `proto/ant/v1/marketplace_service.proto` (extend)
- `frontend/src/pages/marketplace/components/LivePerformancePanel.tsx` (new)

### 1.2 回测质量门槛

**Why**: 防止低质量策略泛滥，损害市场信誉。

**Tasks**:
- [ ] 在 `Publish` 方法中新增 `validateBacktestQuality` 步骤
- [ ] 门槛规则可配置（`system_config` 表）：`min_sharpe_ratio`、`max_drawdown_pct`、`min_total_trades`、`min_win_rate`
- [ ] 不达标返回明确错误码 + 中文提示（哪些指标不达标及阈值）
- [ ] Admin 可针对特定提供者/策略豁免门槛（`marketplace_quality_waivers` 表）

**Files**:
- `backend/internal/marketplace/publish.go` (extend)
- `backend/migrations/xxx_marketplace_quality_waivers.up.sql` (new)

### 1.3 提供者身份验证

**Why**: 让用户知道策略背后是人还是 AI，是否有真实身份验证。

**Tasks**:
- [ ] `users` 表新增 `verified_provider` boolean，默认 false
- [ ] 提供者提交验证材料（Admin 审核流程）
- [ ] `PublishedStrategy` proto 新增 `provider_verified` 和 `provider_type`（human/ai/hybrid）字段
- [ ] 前端策略卡片 + 详情页显示提供者认证徽章 + 类型标签

### 1.4 风险声明与合规

**Why**: 避免法律风险，建立专业形象。

**Tasks**:
- [ ] 策略详情页底部强制风险提示组件（不可折叠）
- [ ] 首次购买弹窗二次确认：显示风险声明 + 勾选"我已知晓风险"
- [ ] 策略页底部显示策略免责：过往业绩不代表未来表现
- [ ] 参考 MQL5 Market / TradingView 合规文案

### 1.5 价格模型修复 [BUG-1]

**Why**: 当前前端 `monthly` 与后端 `subscription` 不匹配，付费订阅功能实际不可用。

**Tasks**:
- [ ] 前端 `PublishToMarketModal` 将 `monthly` 改为 `subscription`
- [ ] 后端 `PurchaseStrategy` 增加输入校验，拒绝不识别的 price_model
- [ ] 回归测试覆盖 free/once/subscription 三种发布+购买流程

---

## Phase 2：AI 策略供给 🔴 P0

> **目标**：用 AI 批量生产高质量策略，解决冷启动。串联现有 agent-engine → mql-compiler → backtest-engine 管线。

### 2.1 AI 一键生成 + 自动上架

**Why**: 这是平台最大的差异化能力。用户可以零代码获得可售策略。

**Tasks**:
- [ ] 新建 `AutoGenerateStrategy` RPC：接收自然语言需求描述 → 调用 agent-engine 生成策略 → mql-compiler 编译 → backtest-engine 回测 → 达标自动 `Publish`
- [ ] 全流程 SSE 推送进度（生成中 → 编译中 → 回测中 → 评估中 → 已上架/不达标）
- [ ] 前端新建 `AutoGeneratePanel` 组件：需求输入 + 进度条 + 结果预览 + 一键发布
- [ ] 用户可在发布前编辑策略标题、描述、定价
- [ ] 失败时返回具体原因（编译失败/回测不达标/超时）

**Files**:
- `backend/internal/connect/marketplace/marketplace_handler_autogen.go` (new)
- `proto/ant/v1/marketplace_service.proto` (extend)
- `frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx` (new)

### 2.2 批量策略生成队列

**Why**: 平台侧主动扩充策略库，覆盖主流品种×周期组合。

**Tasks**:
- [ ] 后台异步任务：扫描品种（EURUSD, BTCUSD, XAUUSD...）× 周期（M15, H1, H4, D1）× 策略类型（趋势/均值回归/突破）
- [ ] 每个组合生成 3-5 个策略变体，写入 `auto_generated_strategies` 表（待审核状态）
- [ ] Admin 审核面板：预览回测结果 → 批量批准/拒绝 → 批准后自动 Publish
- [ ] 每日限额控制，避免 API 费用失控

**Files**:
- `backend/internal/marketplace/batch_generator.go` (new)
- `backend/migrations/xxx_auto_generated_strategies.up.sql` (new)

### 2.3 策略参数模板

**Why**: 提供"填空式"策略创建，降低策略构思门槛。

**Tasks**:
- [ ] 预置策略模板（trend_following / mean_reversion / breakout / arbitrage），每个模板定义可配置参数
- [ ] `ListStrategyTemplates` RPC：返回可用模板列表 + 参数说明
- [ ] 前端模板选择器：卡片式模板浏览 → 选品种+周期 → AI 填充参数 → 生成回测 → 预览 → 上架

### 2.4 提供者工具面板增强

**Why**: 当前 `AuthorTab` 只有基础统计，需要完整的提供者工作站。

**Tasks**:
- [ ] 收益趋势图（日/周/月维度，销售收入 + 订阅收入分开展示）
- [ ] 订阅者分析（总数、新增、流失、续费率）
- [ ] 单策略分析（每个策略的订阅数趋势、评分分布、收入贡献）
- [ ] 提现入口（跳转钱包提现页）

---

## Phase 3：增长引擎 🟡 P1

> **目标**：构建双边网络效应 — 买方越多 → 提供者越多 → 策略越多 → 买方更多。

### 3.1 策略排行榜

**Why**: 给用户一个"从哪里开始"的入口，给提供者一个竞争激励。

**Tasks**:
- [ ] 新增 RPC `ListLeaderboard(type, period, limit)`：type ∈ {return, popular, new, copytrade}，period ∈ {week, month, quarter, all}
- [ ] 收益榜按实盘业绩排序（无实盘的策略不参与）
- [ ] 人气榜按订阅数排序
- [ ] 新锐榜按最近 30 天上架的业绩排序
- [ ] 前端独立 Leaderboard 页面，Tab 切换榜单类型
- [ ] 策略卡片在榜单中显示排名徽章

### 3.2 免费试用

**Why**: 降低购买决策门槛，付费转化率的直接杠杆。

**Tasks**:
- [ ] 新建 `marketplace_trials` 表（id, user_id, strategy_id, started_at, expires_at, status）
- [ ] `StartTrial` RPC：7 天免费试用，一个用户同一策略只能试用一次
- [ ] 后台 `CheckTrialExpiry` 定时任务：到期自动取消试用，恢复代码访问限制
- [ ] `CanAccessCode` 扩展：试用期内可访问代码
- [ ] 前端策略详情页显示"免费试用"按钮（未试用过 + 策略支持试用）
- [ ] 试用到期前 24h 邮件/站内通知

### 3.3 策略对比工具

**Why**: 帮助用户在多个策略之间做出理性选择。

**Tasks**:
- [ ] `CompareStrategies(strategy_ids[])` RPC：批量返回标准化对比数据
- [ ] 前端对比组件：并排表格（回测指标、实盘业绩、费率、风险指标），高亮最优值
- [ ] 添加到对比的快捷操作（策略卡片上的复选框）

### 3.4 通知系统

**Why**: 激活沉默用户，提升留存。

**Tasks**:
- [ ] 新建 `marketplace_notifications` 表（id, user_id, type, title, body, strategy_id, is_read, created_at）
- [ ] SSE 推送通知到前端导航栏 Bell 图标
- [ ] 触发场景：新策略上架（关注品种）、价格变动、订阅即将到期、策略业绩大幅异动、收到新评分/评论
- [ ] 通知偏好设置（用户可选择接收哪些类型的通知）

### 3.5 社交分享

**Why**: 免费获客渠道，利用用户社交网络传播。

**Tasks**:
- [ ] 策略详情页生成 SEO 友好的 OpenGraph 标签（标题、描述、回测摘要缩略图）
- [ ] 分享按钮（复制链接 / Twitter / Telegram / 微信）
- [ ] 分享落地页（非登录用户可浏览策略基本信息 + 注册 CTA）

---

## Phase 4：平台运营 🟢 P2

> **目标**：平台方具备完整的运营能力，商业化可管理。

### 4.1 Admin 策略管理面板

**Why**: 运营人员需要一个统一的后台来管理市场内容。

**Tasks**:
- [ ] 策略审核列表（待审核/已发布/已隐藏/违规下架），支持批量操作
- [ ] 策略详情查看（完整元数据 + 回测结果 + 销售数据）
- [ ] 推荐/置顶策略（`marketplace_strategies` 加 `is_featured`、`featured_until` 字段）
- [ ] Admin 修改定价 + 平台费率（已有 RPC，补前端 UI）
- [ ] 违规策略下架 + 下架原因记录 + 通知提供者

### 4.2 退款 UI [GAP-1]

**Why**: 后端退款逻辑已完成，缺少用户入口。

**Tasks**:
- [ ] 前端已购策略列表增加"申请退款"按钮（条件：购买后 < 7 天）
- [ ] 退款申请表单（原因选择 + 补充说明）
- [ ] Admin 退款审核面板（待处理/已批准/已拒绝）
- [ ] 退款处理结果通知用户

### 4.3 收入仪表盘（Admin）

**Why**: 平台方需要实时了解市场运行状态。

**Tasks**:
- [ ] `GetMarketplaceAnalytics` RPC：GMV、平台收入、付费用户数、ARPU、退款率、续费率
- [ ] 按时间维度（日/周/月/自定义）聚合
- [ ] TOP 策略（按收入）、TOP 提供者（按收入）、策略数量趋势
- [ ] 前端 Admin Dashboard 新增"策略市场"模块

### 4.4 策略版本管理

**Why**: 策略会迭代优化，已购用户需要知道版本变化。

**Tasks**:
- [ ] `marketplace_strategies` 新增 `version` 字段（SemVer，默认 1.0.0）
- [ ] 提供者更新策略时创建新版本记录（`marketplace_strategy_versions` 表）
- [ ] 已购用户可在策略详情页查看版本历史 + changelog
- [ ] 已购用户可选择升级到新版本（免费升级 or 补差价，视定价策略而定）
- [ ] 前端展示"v1.2.0 · 3 天前更新"

### 4.5 折扣/优惠券

**Why**: 促销活动是拉动 GMV 的标准手段。

**Tasks**:
- [ ] 新建 `marketplace_coupons` 表（code, discount_type[percentage/fixed], discount_value, min_purchase, max_uses, used_count, expires_at, applicable_strategies[]）
- [ ] `ValidateCoupon` RPC：校验优惠券有效性 + 计算折后价格
- [ ] `PurchaseStrategy` 扩展：接受 coupon_code 参数，应用折扣后扣款
- [ ] Admin 优惠券管理面板（创建/启用/停用/查看使用统计）
- [ ] 前端支付弹窗增加优惠券输入框

### 4.6 策略提现

**Why**: 提供者赚到的钱需要能提出来。

**Tasks**:
- [ ] 复用 HD 钱包系统（ADR-0026）的冷签名提现流程
- [ ] 提供者在 Author Dashboard 发起提现（输入金额 + 目标地址）
- [ ] 平台审核 → 冷签名 → 广播交易
- [ ] 提现记录 + 状态追踪

---

## Phase 5：护城河 🟢 P3

> **目标**：建立长期竞争壁垒，拓展收入来源。

### 5.1 策略捆绑包

**Why**: 提升客单价，促进交叉销售。

**Tasks**:
- [ ] 新建 `marketplace_bundles` 表（id, title, description, strategy_ids[], bundle_price, original_total, discount_pct）
- [ ] `ListBundles` / `PurchaseBundle` RPC
- [ ] 前端捆绑包卡片（显示包含策略数 + 原价 vs 捆绑价 + 节省金额）

### 5.2 跟单 UI [引擎已有]

**Why**: CopyTradeEngine 已完成，缺少让用户使用的 UI。

**Tasks**:
- [ ] 策略详情页新增"跟单"按钮
- [ ] 跟单配置弹窗：选择跟单账户 → 设置跟单比例（10%-100%） → 最大仓位限制 → 止损设置
- [ ] `StartCopyTrade` / `StopCopyTrade` RPC
- [ ] 已购策略列表显示跟单状态（跟单中/已暂停）+ 跟单收益

### 5.3 阶梯费率

**Why**: 激励高质量提供者，提升平台收入。

**Tasks**:
- [ ] 提供者等级制度（Bronze/Silver/Gold/Platinum），按月收入/评分/订阅数自动升级
- [ ] 不同等级对应不同 `platform_fee_rate`（如 Bronze 25% → Gold 10%）
- [ ] 新提供者默认 Bronze，每月 1 号重新计算等级
- [ ] 前端 Author Dashboard 显示当前等级 + 升级条件进度条

### 5.4 白标 / API

**Why**: B2B 收入渠道，让 broker 集成策略市场。

**Tasks**:
- [ ] 策略市场公开 API（RESTful 风格，虽然项目禁止 REST 但这是外部集成场景，需评估）
- [ ] Broker 可通过 API 获取策略列表 + 嵌入到自己平台的 iframe
- [ ] 自定义品牌（Logo、主题色）
- [ ] 收入分账（Broker 引入的用户购买策略，Broker 获得分成）

---

## 依赖关系

```
Phase 1 (信任)
  ├── 1.5 价格模型修复 ← 立即修，阻塞付费闭环
  ├── 1.2 回测门槛 ← 依赖现有 backtest-engine
  └── 1.1 实盘跟踪 ← 需要提供者绑定 MT 账户
        │
Phase 2 (AI 供给) ← 依赖 Phase 1.2 (门槛)
  ├── 2.1 AI 生成 ← 串联 agent-engine + mql-compiler + backtest-engine
  └── 2.3 参数模板 ← 依赖 2.1 的管线
        │
Phase 3 (增长) ← 依赖 Phase 1+2 有足够策略和用户
  ├── 3.2 试用 ← 依赖 Phase 1.1 (实盘跟踪作为试用期对比)
  └── 3.1 排行榜 ← 依赖 Phase 1.1 (实盘数据排序)
        │
Phase 4 (运营) ← 可与 Phase 3 并行
  └── 4.1 Admin ← 可用时即做，不阻塞其他 Phase
        │
Phase 5 (护城河) ← 用户量达到临界质量后启动
```

---

## 关键指标

| 指标 | 现状 | Phase 1 目标 | Phase 2 目标 | Phase 3 目标 |
|------|------|-------------|-------------|-------------|
| 上架策略数 | ~10 | 30+ (含回测验证) | 200+ (含 AI 生成) | 500+ |
| 月活跃买方 | <50 | 200 | 1000 | 5000+ |
| 月 GMV | — | $2K | $20K | $100K+ |
| 平台月收入 | — | $500 | $5K | $25K+ |
| 策略平均评分 | — | ≥4.0 | ≥4.2 | ≥4.3 |
| 付费转化率 | — | 3% | 5% | 8%+ |
| 订阅续费率 | — | — | ≥60% | ≥70% |
| AI 策略占比 | 0% | 30% | ≥60% | ≥70% |

---

## 复用清单（Reuse Preflight）

实现每个 Phase 前必须验证以下现有能力可复用，避免重复造轮子：

| 能力 | 位置 | 被 Phase 使用 |
|------|------|-------------|
| 策略编译管线 | `mql-compiler` → IR → Bytecode VM | Phase 2 (AI 生成) |
| AI 策略生成 | `agent-engine` | Phase 2 (AI 生成) |
| 回测引擎 | `backtest-engine` / SimBroker | Phase 1.2 (门槛), Phase 2 (验证) |
| 钱包交易 | `walletRepo.AdjustBalanceTx` (hash chain) | Phase 4.5 (优惠券), Phase 4.6 (提现) |
| HD 钱包提现 | ADR-0026 冷签名流程 | Phase 4.6 (提现) |
| SSE 推送管道 | ConnectRPC server-stream + PG NOTIFY | Phase 1.1 (实盘推送), Phase 3.4 (通知) |
| pg_trgm 搜索 | migration 172 (GIN index) | Phase 1 (搜索增强) |
| 风控管线 | `risk-gate` 6 门管线 + OMS | Phase 5.2 (跟单 UI) |
| i18n 框架 | 5 语言已部署 | 所有前端新页面 |
| 平台订阅 | Free/Pro/Enterprise tiers | GAP-3 联动修复 |
