# 前端与 Agent 引擎审计

> 前端：marketplace 相关页面完工度。Agent-engine：成本控制与模型路由。

---

## 前端 · Marketplace 完工度

### 后端——Phase 交付物

| Phase | 后端文件 | 状态 |
|-------|---------|------|
| 1.1 实盘跟踪 | `marketplace/live_performance.go` | ✅ |
| 1.2 质量门槛 | `marketplace/quality.go` | ✅ |
| 2.1 AI 自动生成 | `marketplace/batch_generator.go` + `_admin.go` | ✅ |
| 2.2 批量队列 | 同上 | ✅ |
| 3.1 排行榜 | `marketplace/leaderboard.go` | ✅ |
| 3.2 免费试用 | `marketplace/trial.go` | ✅ |
| 3.3 策略对比 | `marketplace/compare.go` | ✅ |
| 4.1 Admin 面板 | — | ❌ 未开始 |
| 4.2 退款 UI | — | ❌ 未开始 |
| 4.3 收入仪表盘 | `marketplace/analytics.go` | ❌ 未开始 |
| 4.4 版本管理 | `marketplace/version.go` | ❌ 未开始 |
| 4.5 优惠券 | `marketplace/coupon.go` | ❌ 未开始 |
| 5.1 AI 迭代 | `marketplace/decay_detector.go` + `strategy_optimizer.go` | ❌ 未开始 |

**Phase 1-3 后端完成。Phase 4-5 未开始。**

### 前端——组件清单

| 组件 | 行数 | 对应 Phase | 状态 |
|------|------|-----------|------|
| `LivePerformanceTab` | 171 | 1.1 | ✅ |
| `AutoGeneratePanel` | 351 | 2.1 | ✅ |
| `AutoGenerateProgress` | — | 2.1 | ✅ |
| `AutoGenerateResult` | 99 | 2.1 | ✅ |
| `TemplateSelector` | 210 | 2.3 | ✅ |
| `LeaderboardTab` | 131 | 3.1 | ✅ |
| `CompareModal` | 162 | 3.3 | ✅ |
| `ShareButtons` | — | 3.5 | ✅ |
| `StrategySharePage` | 169 | SEO | ✅ |
| `PaymentModal` | 127 | 1.4 | ✅ |
| `StrategyDetailModal` | 331 | 1.1/3.2 | ✅ |
| `ProtectedBacktestPanel` | 212 | 市场回测 | ✅ |
| `MarketTab` | — | 市场浏览 | ✅ |
| `AuthorTab` | 143 | 作者面板 | ✅ |
| `PurchaseTab` | — | 已购列表 | ✅ |
| `StrategyMarketCard` | 129 | 策略卡片 | ✅ |
| `MarketplacePage` | 105 | 主布局 | ✅ |
| Admin marketplace 页面 | — | 4.1-4.5 | ❌ 未开始 |

**Phase 1-3 前端完成。Phase 4-5 未开始。**

### 文件大小标记

| 文件 | 行数 | 问题 |
|------|------|------|
| `AutoGeneratePanel.tsx` | 351 | 🔴 超 TS 硬性红线 (250)，+40% |
| `StrategyDetailModal.tsx` | 331 | 🔴 超 TS 硬性红线 (250)，+32% |

需拆分为更小组件。

---

## Agent 引擎 · 成本审计

### 已实现

| 能力 | 状态 | 说明 |
|------|------|------|
| Token 用量追踪 | ✅ | `AITokenUsageRepository` 记录每次调用，`GetTokenUsage` RPC 返回明细 |
| Token 扣费 | ✅ | `RecordTokenUsage` 区分 `paid_by=system`（扣平台钱包）和 `paid_by=user`（扣用户） |
| 用户自带 API key | ✅ | `system_ai_configs` 表支持用户配置自己的 provider/api_key/base_url/models |
| Provider 故障转移 | ✅ | `chat_failover.go` — `ai_circuit_breaker` 表记录连续失败，超阈值自动切换 |
| 配额检查 | ✅ | `QuotaChecker` 按用户订阅计划检查 `max_ai_tokens_monthly` |
| 模型降级链 | ✅ | 每个 provider 可配置 `models` 列表 + `default_model` |

### 缺口

| 缺口 | 影响 | 方案 |
|------|------|------|
| 无 **API 成本仪表盘** | Admin 不知道每天花了多少钱 | 加 Admin 页：按 provider/按用户/按日 展示 token 消耗和费用 |
| Provider 故障转移是**静默的** | Admin 不知道发生了故障转移 | 故障转移时写 admin_audit_log |
| 无 **per-provider 费用差异分析** | 选哪个 provider 只看价格表，缺少实际数据支撑 | 在 token usage 表中加 `provider` 字段，用于事后对比成本 |
| `AutoGeneratePanel.tsx` 351 行 | 超红线 | 拆为 AutoGeneratePanel + AutoGenerateForm + AutoGenerateProgress 三个组件 |
| `StrategyDetailModal.tsx` 331 行 | 超红线 | Tab 内容独立组件化（PerformanceTab / DiscussionTab / PricingTab） |

---

## 结论

**Phase 1-3 全完工。Phase 4-5 零开始。** GLM 当前进度和文档规划一致。

**Agent 引擎成本侧已完整**——token 追踪、扣费、自带 key、故障转移、配额检查全有。缺的是**成本可见性**（Admin 不知道花了多少钱）和**故障转移透明度**（Admin 不知道发生了转移）。

**前端两个文件超红线**：`AutoGeneratePanel.tsx` 和 `StrategyDetailModal.tsx`——已在 GLM 总清单 P0-1c，GLM 完工后拆分。
