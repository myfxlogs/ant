> ⚠️ 已迁移至 docs/blocks/strategy-marketplace/plans/phase-1-trust-infrastructure.md。此文件保留为兼容旧引用。

# Phase 1 · 信任基础设施 + SEO · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 1
> 冲突时以本文为准。

## 0. 目标与边界

- **达成即最优**：用户看到策略后，基于实盘业绩、回测质量门槛、提供者身份，做出理性购买决策。
- **同时完成 SEO 基础建设**：补全 `Seo.tsx` 缺失标签，扩展关键词覆盖（broker 名 + AI 词），新建 `/brokers` 页。
- **合规**：proto-only、decimal 计价、无 REST、push-first（禁 timer/cron）、复用优先。

---

## 依赖图

```
1.5 价格模型 BUG 修复 ── 无依赖，立即修，阻塞所有付费流程
1.2 回测质量门槛     ── 无依赖
1.1 实盘业绩跟踪     ── 依赖 1.5
1.3 提供者验证       ── 无依赖
1.4 风险声明合规     ── 依赖 1.5
S1  Seo.tsx 补全     ── 无依赖，与所有模块并行
S3  SEO 关键词落地   ── 无依赖，与所有模块并行
```

---

## 模块 1.5 · 价格模型 BUG 修复

> 🔴 阻塞付费订阅。前后端 `monthly` vs `subscription` 不匹配。

- [ ] **1.5a 后端 `PurchaseStrategy` 加白名单校验**

  **文件**：`backend/internal/marketplace/purchase.go`
  DB 查询后、扣款前插入：
  ```go
  if priceModel != PriceModelOnce && priceModel != PriceModelSubscription {
      return nil, fmt.Errorf("marketplace: unsupported price_model %q", priceModel)
  }
  ```

- [ ] **1.5b 后端 `Publish` 加白名单校验**

  **文件**：`backend/internal/marketplace/publish.go`
  `Publish` 开头 switch `PriceModelFree/Once/Subscription`，其他返回 error。

- [ ] **1.5c 前端对齐**

  **文件**：`frontend/src/pages/strategy/components/PublishToMarketModal.tsx`
  `monthly` → `subscription`（包括 i18n 键和类型定义）。

- [ ] **1.5d 回归测试**

  **文件**：`backend/internal/connect/marketplace/marketplace_test.go`
  新增 `TestPublish_InvalidPriceModel`、`TestPurchase_InvalidPriceModel`、`TestPurchase_Subscription_Success`。

- **Gate 1.5**：`go build ./...` + `go test ./internal/marketplace/... ./internal/connect/marketplace/...`

---

## 模块 1.2 · 回测质量门槛

**Why**：防止过拟合策略上架（R-1）。不用纯阈值门——那是过拟合策略最会刷的指标。叠加 walk-forward/purged CV。

- [ ] **1.2a 门槛配置（`system_config`）**

  **文件**：`backend/migrations/xxx_marketplace_quality_gates.up.sql` (new)
  ```sql
  INSERT INTO system_config (key, value, value_type, description, enabled) VALUES
    ('marketplace.quality.min_sharpe_ratio',   '0.5',  'decimal', '最低夏普比率', true),
    ('marketplace.quality.max_drawdown_pct',   '0.30', 'decimal', '最大回撤百分比', true),
    ('marketplace.quality.min_total_trades',   '20',   'int',    '最低交易次数', true),
    ('marketplace.quality.min_win_rate',       '0.35', 'decimal', '最低胜率', true),
    ('marketplace.quality.max_is_oos_degradation', '0.5', 'decimal', 'IS vs OOS 最大劣化比', true),
    ('marketplace.quality.enforce_backtest_snapshot', 'true', 'bool', '必须携带回测快照', true)
  ON CONFLICT (key) DO NOTHING;
  ```

- [ ] **1.2b 质量校验 + walk-forward 验证**

  **文件**：`backend/internal/marketplace/quality.go` (new)

  核心函数 `ValidateBacktestQuality(ctx, snap) ([]QualityViolation, error)`：
  - 检查 `BacktestSnapshot` 是否存在
  - 逐项对比阈值（夏普/回撤/交易次数/胜率），用 `decimal.Decimal` 比较
  - `max_drawdown_pct` 反向比较（越小越好）
  - Walk-forward：按时间切分 IS/OOS，计算两段指标差，超过 `max_is_oos_degradation` 则拒绝
  - 不达标返回具体指标名 + 实际值 vs 阈值

- [ ] **1.2c 集成到 `Publish` 流程**

  **文件**：`backend/internal/marketplace/publish.go`
  DB 写入前：unmarshal `BacktestSnapshotProto` → `ValidateBacktestQuality` → 不达标拒绝（含具体原因）。

- [ ] **1.2d Admin 豁免**

  **文件**：`backend/migrations/xxx_marketplace_quality_waivers.up.sql` (new)
  ```sql
  CREATE TABLE marketplace_quality_waivers (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
      waived_by UUID NOT NULL REFERENCES users(id),
      reason TEXT NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

- **Gate 1.2**：`go build ./...` + `go test ./internal/marketplace/...` + 手动 Publish（达标/不达标/豁免）

---

## 模块 1.1 · 实盘业绩跟踪（事件驱动）

**依赖 1.5**。🔴 禁用 timer——交易事件通过 NATS push 触发写入。

- [ ] **1.1a DB 表**

  **文件**：`backend/migrations/xxx_marketplace_live_performance.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_live_performance (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
      account_id UUID NOT NULL,
      date DATE NOT NULL,
      daily_pnl NUMERIC(20,8) NOT NULL DEFAULT 0,
      daily_return NUMERIC(10,6) NOT NULL DEFAULT 0,
      equity NUMERIC(20,8) NOT NULL,
      drawdown NUMERIC(10,6) NOT NULL DEFAULT 0,
      total_trades INT NOT NULL DEFAULT 0,
      winning_trades INT NOT NULL DEFAULT 0,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      UNIQUE (strategy_id, account_id, date)
  );

  CREATE TABLE marketplace_live_performance_summary (
      strategy_id UUID PRIMARY KEY REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
      account_id UUID NOT NULL,
      total_return NUMERIC(10,6) NOT NULL DEFAULT 0,
      annual_return NUMERIC(10,6),
      max_drawdown NUMERIC(10,6) NOT NULL DEFAULT 0,
      sharpe_ratio NUMERIC(10,6),
      win_rate NUMERIC(10,6),
      total_trades INT NOT NULL DEFAULT 0,
      avg_monthly_return NUMERIC(10,6),
      tracking_since DATE NOT NULL,
      last_updated DATE NOT NULL,
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

  Summary 由 Go 层在同一事务内增量更新（不用 PG 触发器——避免隐式写入放大和锁竞争）。

- [ ] **1.1b NATS 事件订阅 + 写入**

  **文件**：`backend/internal/marketplace/live_performance.go` (new)

  `LivePerformanceCollector` 订阅 NATS subject `mt.*.trade.closed` 和 `mt.*.account.*.equity`，匹配 `linked_account_id` → 调用 `UpsertDailyPerformance(tx, record)` 做 UPSERT + summary 增量更新。

- [ ] **1.1c Proto + RPC**

  新增 `GetLivePerformance` RPC：返回 `LivePerformancePoint[]` + `LivePerformanceSummary`。`daily_pnl`/`equity` 用 string（decimal），其他用 double。

- [ ] **1.1d 绑定字段**

  **文件**：`backend/migrations/xxx_marketplace_linked_account.up.sql` (new)
  ```sql
  ALTER TABLE marketplace_strategies ADD COLUMN linked_account_id UUID;
  ```
  新增 RPC `LinkLiveAccount(strategy_id, account_id)`。

- [ ] **1.1e 前端——策略详情页实盘 Tab**

  **文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`
  回测业绩 | 实盘业绩 Tab 切换。实盘 Tab：净值曲线（Recharts）+ 月度收益热力图 + 关键指标卡片。无数据 → "暂未绑定实盘账户"。

- **Gate 1.1**：`go build ./...` + `buf generate` + 前端 compile

---

## 模块 1.3 · 提供者身份验证

- [ ] **1.3a DB**

  **文件**：`backend/migrations/xxx_provider_verification.up.sql` (new)
  ```sql
  ALTER TABLE users ADD COLUMN verified_provider BOOLEAN NOT NULL DEFAULT false;
  ALTER TABLE users ADD COLUMN provider_type VARCHAR(20) NOT NULL DEFAULT 'human';  -- human/ai/hybrid
  CREATE TABLE provider_verification_requests (...);  -- pending/approved/rejected
  ```

- [ ] **1.3b Proto**：`PublishedStrategy` 加 `provider_verified` + `provider_type`。新增 `RequestVerification` RPC。

- [ ] **1.3c `ListPublished` SQL JOIN `users` 取验证字段。**

- [ ] **1.3d 前端**：`StrategyMarketCard` 显示认证徽章 + AI/Human 标签。

- **Gate 1.3**：`go build ./...` + `buf generate` + 前端 compile

---

## 模块 1.4 · 风险声明

- [ ] **1.4a 策略详情页底部 `Alert` 组件**（不可折叠，5 语言 i18n）
- [ ] **1.4b 首次购买弹窗加 Checkbox**："我已阅读并理解风险提示"
- [ ] **1.4c `marketplace_strategies.disclaimer_text` 字段**（提供者可自定义，空则用默认）
- **Gate 1.4**：5 语言 i18n 键完整 + 前端 compile

---

## 模块 S1 · SEO 标签补全

> 详见 `docs/plan/marketplace/seo-strategy.md` 模块 S1。与所有模块并行。

- [ ] **S1a `Seo.tsx` 扩展 props**：`keywords`、`ogImage`、`ogType`（默认 `"website"`）、`twitterCard`（默认 `"summary_large_image"`）

- [ ] **S1b 各页面传入 keywords**：
  - LandingPage: `AI trading, MT4, MT5, MetaTrader, algorithmic trading, automated trading, forex EA, IC Markets, Pepperstone, XM, Exness, OANDA`
  - MarketplacePage: `strategy marketplace, buy forex EA, MT4 strategies, MT5 strategies, trading robots, AI trading strategies`
  - SharePerformancePage: 策略名 + 品种 + 周期 + `trading performance, verified track record`

- [ ] **S1c 策略分享页定制 og-image**：后端新增 `/share/:token/og-image` 端点，Publish 时预生成 SVG（叠加策略指标），存静态文件。

---

## 模块 S3 · SEO 关键词落地页

> 详见 `docs/plan/marketplace/seo-strategy.md` 模块 S3。与所有模块并行。

- [ ] **S3a Landing page 自然嵌入关键词**（不是 spam 列表）：
  - "兼容 30+ MT4/MT5 broker（IC Markets, Pepperstone, XM, Exness, OANDA…）"
  - "AI 驱动策略生成与自动优化"
  - "MQL4/MQL5 策略一键编译到云端执行"

- [ ] **S3b Marketplace 页 `Seo` description**：
  `"发现并购买 MT4/MT5 交易策略。支持 IC Markets, Pepperstone, XM 等 30+ broker。AI 辅助策略生成与优化。回测验证、实盘跟踪。"`

- [ ] **S3c 新建 `/brokers` 页**

  **文件**：`frontend/src/pages/landing/BrokersPage.tsx` (new)
  列出所有兼容 broker（名称 + MT4/MT5 标识 + 一句话描述）。这是 SEO 金矿——Google 能看到 30+ broker 名。

---

## Phase 1 完成检验

```bash
go build ./...
buf generate
go test ./internal/marketplace/... ./internal/connect/marketplace/...
cd frontend && npm run build
bash scripts/gen_capability_map.sh
```

**关键验收**：
1. 正确/错误价格模型发布购买 → 成功/明确错误
2. 回测不达标 → 拒绝 + 指出指标
3. 已绑实盘 → 详情页显示实盘曲线；未绑 → "暂无数据"
4. 已验证提供者 → 徽章显示
5. 首次购买 → 风险确认弹窗
6. `/brokers` 页正常渲染；Landing/Marketplace 页含 broker 名 + AI 关键词
