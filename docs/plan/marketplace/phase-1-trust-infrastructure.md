# Phase 1 · 信任基础设施 · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 1
> 冲突时以本文为准。
> 本清单是可勾选执行版；每条任务标注 **文件**、**验收**、**红线**。GLM 逐模块执行，模块间有依赖注明。

## 0. 目标与边界（执行前必读）

- **达成即最优**：用户看到策略后，基于实盘业绩、回测质量门槛、提供者身份，做出理性购买决策。
- **核心原则**：信任不是 UI 层的事——每一层（DB 约束 / 后端校验 / 前端展示）都不可绕过。
- **合规**：proto-only 跨进程交换、decimal 计价、无 REST endpoint、文件行数限制、复用优先（`bash scripts/cap.sh`）。

---

## 1. 依赖图

```
1.5 (价格模型 BUG 修复) ── 无依赖，立即修
1.2 (回测质量门槛)     ── 无依赖，后端独立
1.1 (实盘业绩跟踪)     ── 依赖 1.5（需要正确的订阅关系）
1.3 (提供者验证)       ── 无依赖
1.4 (风险声明合规)     ── 依赖 1.5（需要正确的购买流程）
```

---

## 模块 1.5 · 价格模型 BUG 修复（🔴 阻塞付费订阅）

**前置**：`bash scripts/cap.sh price_model subscription publish` 确认现有实现。

### 后端修复

- [ ] **1.5a `PurchaseStrategy` 价格模型白名单校验**

  **文件**：`backend/internal/marketplace/purchase.go`
  **变更**：在 `PurchaseStrategy` 的 DB 查询之后、钱包扣款之前，新增价格模型白名单校验：

  ```go
  // After: err = tx.QueryRow(... price_model ...).Scan(...)
  // Add:
  if priceModel != PriceModelOnce && priceModel != PriceModelSubscription {
      return nil, fmt.Errorf("marketplace: unsupported price_model %q", priceModel)
  }
  ```

  **验收**：传入 `price_model="monthly"` 的策略购买请求 → 返回明确错误 "unsupported price_model"。

- [ ] **1.5b `Publish` 价格模型白名单校验**

  **文件**：`backend/internal/marketplace/publish.go`
  **变更**：在 `Publish` 方法开头新增校验：

  ```go
  switch params.PriceModel {
  case PriceModelFree, PriceModelOnce, PriceModelSubscription:
      // ok
  default:
      return "", fmt.Errorf("marketplace: invalid price_model %q", params.PriceModel)
  }
  ```

  **验收**：传入 `price_model="monthly"` → Publish 失败；传入 `price_model="subscription"` → 成功。

### 前端修复

- [ ] **1.5c `PublishToMarketModal` 价格模型选项对齐**

  **文件**：`frontend/src/pages/strategy/components/PublishToMarketModal.tsx`
  **变更**：将 `monthly` 改为 `subscription`：

  ```tsx
  // OLD: { label: t('monthly'), value: 'monthly' }
  // NEW: { label: t('subscription'), value: 'subscription' }
  ```

  **同时检查**：该文件的 `PriceModel` 类型定义、i18n 键 `monthly` → `subscription`。

  **验收**：前端选择"订阅制"后，Network tab 看到发送的是 `subscription`。

- [ ] **1.5d 回归测试**

  **文件**：`backend/internal/connect/marketplace/marketplace_test.go`
  **变更**：新增测试用例：
  - `TestPublish_InvalidPriceModel` — Publish 传入 `monthly` → 断言失败
  - `TestPurchase_InvalidPriceModel` — 直接调用 `PurchaseStrategy`（mock DB 返回 `monthly`）→ 断言失败
  - `TestPurchase_Subscription_Success` — 完整链路：Publish(`subscription`) → Purchase → 验证 subKind、expiresAt

  **验收**：`go test ./internal/connect/marketplace/...` 全部通过。

- **Gate 1.5**：`go build ./...` + `go test ./internal/marketplace/... ./internal/connect/marketplace/...`。

---

## 模块 1.2 · 回测质量门槛

**前置**：`bash scripts/cap.sh backtest validate quality sharpe drawdown`。

- [ ] **1.2a 门槛配置项**

  **文件**：`backend/migrations/xxx_marketplace_quality_gates.up.sql` (new)

  ```sql
  INSERT INTO system_config (key, value, value_type, description, enabled)
  VALUES
    ('marketplace.quality.min_sharpe_ratio',   '0.5',  'decimal', 'Publish 最低夏普比率',           true),
    ('marketplace.quality.max_drawdown_pct',   '0.30', 'decimal', 'Publish 最大回撤百分比（0-1）',  true),
    ('marketplace.quality.min_total_trades',   '20',   'int',    'Publish 最低交易次数',             true),
    ('marketplace.quality.min_win_rate',       '0.35', 'decimal', 'Publish 最低胜率（0-1）',         true),
    ('marketplace.quality.enforce_backtest_snapshot', 'true', 'bool', 'Publish 必须携带回测快照', true)
  ON CONFLICT (key) DO NOTHING;
  ```

  **设计决策**：用 `system_config` 而非硬编码常量，Admin 可在运行时调参（无需重新部署）。阈值从宽松起步（夏普 0.5 / 回撤 30%），随市场质量提升再收紧。

- [ ] **1.2b 质量校验函数**

  **文件**：`backend/internal/marketplace/quality.go` (new)

  ```go
  package marketplace

  import (
      "context"
      "fmt"
      "strings"

      antv1 "alphaforge/gen/proto/ant/v1"
      "github.com/shopspring/decimal"
  )

  // QualityViolation describes a single failed quality check.
  type QualityViolation struct {
      Field       string  // e.g. "sharpe_ratio"
      Actual      float64
      Threshold   float64
      Description string  // human-readable
  }

  // ValidateBacktestQuality checks a BacktestSnapshot against configurable thresholds.
  // Returns nil if all checks pass, or a list of violations.
  func (s *Service) ValidateBacktestQuality(ctx context.Context, snap *antv1.BacktestSnapshot) ([]QualityViolation, error) {
      if snap == nil {
          // If enforce_backtest_snapshot is true, missing snapshot is a violation.
          enforce, _ := s.getConfigBool(ctx, "marketplace.quality.enforce_backtest_snapshot")
          if enforce {
              return []QualityViolation{{Field: "backtest_snapshot", Description: "回测快照缺失"}}, nil
          }
          return nil, nil
      }

      checks := []struct {
          key       string
          field     string
          actual    float64
      }{
          {"marketplace.quality.min_sharpe_ratio", "夏普比率", snap.SharpeRatio},
          {"marketplace.quality.max_drawdown_pct", "最大回撤", snap.MaxDrawdown},
          {"marketplace.quality.min_total_trades", "交易次数", float64(snap.TotalTrades)},
          {"marketplace.quality.min_win_rate", "胜率", snap.WinRate},
      }

      var violations []QualityViolation
      for _, c := range checks {
          threshold, err := s.getConfigDecimal(ctx, c.key)
          if err != nil || threshold == "0" {
              continue // disabled or unconfigured
          }
          th, _ := decimal.NewFromString(threshold)
          af, _ := decimal.NewFromFloat(c.actual)
          // For max_drawdown_pct: actual must be <= threshold (smaller is better)
          // For other metrics: actual must be >= threshold (larger is better)
          if c.key == "marketplace.quality.max_drawdown_pct" {
              if af.GreaterThan(th) {
                  violations = append(violations, QualityViolation{
                      Field: c.field, Actual: c.actual,
                      Threshold: th.InexactFloat64(),
                      Description: fmt.Sprintf("%s %.2f 超过上限 %.2f", c.field, c.actual, th.InexactFloat64()),
                  })
              }
          } else {
              if af.LessThan(th) {
                  violations = append(violations, QualityViolation{
                      Field: c.field, Actual: c.actual,
                      Threshold: th.InexactFloat64(),
                      Description: fmt.Sprintf("%s %.2f 低于下限 %.2f", c.field, c.actual, th.InexactFloat64()),
                  })
              }
          }
      }
      return violations, nil
  }

  // Helpers — read typed config values.
  func (s *Service) getConfigDecimal(ctx context.Context, key string) (string, error) { ... }
  func (s *Service) getConfigBool(ctx context.Context, key string) (bool, error) { ... }
  ```

  **红线**：
  - 比较用 `decimal.Decimal`，不用 float64。
  - `max_drawdown_pct` 逻辑反方向（越小越好），其他指标越大越好。
  - `TotalTrades` 是 int32，转为 float64 比较前先检查范围。

- [ ] **1.2c 集成到 Publish 流程**

  **文件**：`backend/internal/marketplace/publish.go`
  **变更**：在 `Publish` 方法的 DB 写入之前，新增质量校验步骤：

  ```go
  func (s *Service) Publish(ctx context.Context, params PublishParams) (string, error) {
      // NEW: quality gate (before any DB write)
      if params.BacktestSnapshotProto != nil {
          var snap antv1.BacktestSnapshot
          if err := proto.Unmarshal(params.BacktestSnapshotProto, &snap); err == nil {
              violations, err := s.ValidateBacktestQuality(ctx, &snap)
              if err != nil {
                  return "", fmt.Errorf("marketplace: quality check error: %w", err)
              }
              if len(violations) > 0 {
                  msgs := make([]string, len(violations))
                  for i, v := range violations {
                      msgs[i] = v.Description
                  }
                  return "", fmt.Errorf("marketplace: backtest quality below threshold: %s", strings.Join(msgs, "; "))
              }
          }
      }
      // ... existing publish logic ...
  }
  ```

  **验收**：提供不合格回测数据（夏普 0.1 等）→ Publish 失败，错误消息包含具体不达标指标和阈值。

- [ ] **1.2d Admin 豁免机制**

  **文件**：`backend/migrations/xxx_marketplace_quality_waivers.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_quality_waivers (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
      waived_by UUID NOT NULL REFERENCES users(id),
      reason TEXT NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_quality_waivers_strategy ON marketplace_quality_waivers(strategy_id);
  ```

  **文件**：`backend/internal/marketplace/quality.go`
  **变更**：`ValidateBacktestQuality` 新增参数 `skipWaiver bool`；Admin Publish 调用传 `skipWaiver=true`。

  **验收**：Admin 可以 Publish 一个质量不达标的策略（手动豁免）。

- **Gate 1.2**：`go build ./...` + `go test ./internal/marketplace/...` + 手动 Publish 测试（达标/不达标/豁免）。

---

## 模块 1.1 · 实盘业绩跟踪（事件驱动架构）

**前置依赖**：模块 1.5（订阅关系正确才能关联实盘数据）。

**架构约束**：🔴 禁用 timer/ticker/polling。必须用 push-first：业绩数据由交易事件驱动写入，查询时实时聚合。

### 架构决策

```
┌─────────────────────────────────────────────────────────────────┐
│  交易事件流（已存在）                                              │
│                                                                  │
│  MT Gateway ──→ mthub ──→ OMS (16 状态机) ──→ NATS              │
│                                                  │               │
│  order_filled / position_closed / account_equity_changed         │
│                                                  │               │
│  ┌───────────────────────────────────────────────┘               │
│  │  NATS JetStream (push)                                        │
│  │                                                                │
│  │  LivePerformanceCollector 订阅以下 subject：                    │
│  │    - mt.<broker>.account.<id>.equity   → 净值变动              │
│  │    - mt.<broker>.trade.closed          → 已平仓交易            │
│  └──────────────────────────────────────────────────────────────┐ │
│                          │                                       │ │
│                          ▼                                       │ │
│  marketplace_live_performance  (日粒度快照)                       │ │
│  marketplace_live_performance_summary  (写时增量更新)             │ │
└─────────────────────────────────────────────────────────────────┘
```

**为什么不用定时器**：交易事件已通过 OMS/NATS 实时推送。收盘后的日终结算不需要独立 timer——当日最后一笔交易事件到达时自然就是"日终"状态。如果某天无交易，这一天也不需要写入（无交易=无需展示）。

### 数据模型

- [ ] **1.1a 实盘业绩表**

  **文件**：`backend/migrations/xxx_marketplace_live_performance.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_live_performance (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
      account_id  UUID NOT NULL,          -- 提供者绑定的 MT 账户
      date        DATE NOT NULL,           -- 交易日（UTC）
      daily_pnl       NUMERIC(20,8) NOT NULL DEFAULT 0,
      daily_return    NUMERIC(10,6) NOT NULL DEFAULT 0,
      equity          NUMERIC(20,8) NOT NULL,
      drawdown        NUMERIC(10,6) NOT NULL DEFAULT 0,
      total_trades    INT NOT NULL DEFAULT 0,
      winning_trades  INT NOT NULL DEFAULT 0,
      created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

      UNIQUE (strategy_id, account_id, date)
  );
  CREATE INDEX idx_live_perf_strategy_date ON marketplace_live_performance(strategy_id, date DESC);

  -- Summary: 写时增量更新（非触发器，由 Go 层在每次 INSERT 后调用）
  CREATE TABLE marketplace_live_performance_summary (
      strategy_id     UUID PRIMARY KEY REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
      account_id      UUID NOT NULL,
      total_return    NUMERIC(10,6) NOT NULL DEFAULT 0,
      annual_return   NUMERIC(10,6),
      max_drawdown    NUMERIC(10,6) NOT NULL DEFAULT 0,
      sharpe_ratio    NUMERIC(10,6),
      win_rate        NUMERIC(10,6),
      total_trades    INT NOT NULL DEFAULT 0,
      avg_monthly_return NUMERIC(10,6),
      tracking_since  DATE NOT NULL,
      last_updated    DATE NOT NULL,
      updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

  **设计决策**：
  - summary 不使用 PG 触发器——触发器内复杂聚合不可控（写入放大、锁竞争）。改为 Go 层在每次写入日记录后，执行增量更新 SQL。
  - 日粒度写入由交易事件触发，非定时器。同一策略同一日可能多次更新（当天多笔交易），用 UPSERT 语义：`INSERT ... ON CONFLICT (strategy_id, account_id, date) DO UPDATE SET ...`。
  - 年化夏普假设 252 交易日。

- [ ] **1.1b Summary 增量更新（Go 层）**

  **文件**：`backend/internal/marketplace/live_performance.go` (new)

  **函数**：`UpsertDailyPerformance(ctx, tx, record) error`

  ```go
  // UpsertDailyPerformance is called every time a trade closes on a linked account.
  // It upserts the daily performance row, then incrementally updates the summary.
  func (s *Service) UpsertDailyPerformance(ctx context.Context, tx pgx.Tx, record *LivePerformanceRecord) error {
      // 1. UPSERT daily row
      _, err := tx.Exec(ctx, `
          INSERT INTO marketplace_live_performance
              (strategy_id, account_id, date, daily_pnl, daily_return, equity, drawdown, total_trades, winning_trades)
          VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
          ON CONFLICT (strategy_id, account_id, date) DO UPDATE SET
              daily_pnl      = marketplace_live_performance.daily_pnl + EXCLUDED.daily_pnl,
              daily_return   = ... ,  -- 增量计算
              equity         = EXCLUDED.equity,
              drawdown       = EXCLUDED.drawdown,
              total_trades   = marketplace_live_performance.total_trades + EXCLUDED.total_trades,
              winning_trades = marketplace_live_performance.winning_trades + EXCLUDED.winning_trades`,
          record.StrategyID, record.AccountID, record.Date,
          record.DailyPnL, record.DailyReturn, record.Equity,
          record.Drawdown, record.TotalTrades, record.WinningTrades)
      if err != nil { return err }

      // 2. Recompute summary (incremental — only for this strategy)
      _, err = tx.Exec(ctx, `
          INSERT INTO marketplace_live_performance_summary
              (strategy_id, account_id, total_return, max_drawdown, sharpe_ratio,
               win_rate, total_trades, avg_monthly_return, tracking_since, last_updated)
          SELECT strategy_id, account_id,
              EXP(SUM(LN(1 + daily_return))) - 1,
              MAX(drawdown),
              CASE WHEN STDDEV(daily_return) > 0
                   THEN (AVG(daily_return) / STDDEV(daily_return)) * SQRT(252) ELSE NULL END,
              CASE WHEN SUM(total_trades) > 0
                   THEN SUM(winning_trades)::numeric / SUM(total_trades)::numeric ELSE NULL END,
              SUM(total_trades),
              (SELECT AVG(mr) FROM (
                  SELECT SUM(daily_return) as mr FROM marketplace_live_performance
                  WHERE strategy_id = $1 AND account_id = $2
                  GROUP BY DATE_TRUNC('month', date)) sub),
              MIN(date), MAX(date)
          FROM marketplace_live_performance
          WHERE strategy_id = $1 AND account_id = $2
          ON CONFLICT (strategy_id) DO UPDATE SET
              total_return = EXCLUDED.total_return, max_drawdown = EXCLUDED.max_drawdown,
              sharpe_ratio = EXCLUDED.sharpe_ratio, win_rate = EXCLUDED.win_rate,
              total_trades = EXCLUDED.total_trades, avg_monthly_return = EXCLUDED.avg_monthly_return,
              last_updated = EXCLUDED.last_updated, updated_at = now()`,
          record.StrategyID, record.AccountID)
      return err
  }
  ```

  **设计决策**：
  - 增量更新在写入日记录**同一个事务**内完成——保证 summary 与明细始终一致。
  - 聚合 SQL 只扫描单个 strategy_id 的数据（有索引），不影响其他策略。
  - 不使用 PG 触发器（避免隐式逻辑、写入放大、锁竞争）。不使用 Go 定时器（违反项目约束）。

### Proto 定义

- [ ] **1.1c 实盘业绩 RPC**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **变更**：新增 message 和 RPC。

  ```protobuf
  message LivePerformancePoint {
    string date = 1;
    string daily_pnl = 2;         // decimal string
    double daily_return = 3;
    string equity = 4;            // decimal string
    double drawdown = 5;
    int32 total_trades = 6;
    int32 winning_trades = 7;
  }

  message LivePerformanceSummary {
    double total_return = 1;
    double annual_return = 2;
    double max_drawdown = 3;
    double sharpe_ratio = 4;
    double win_rate = 5;
    int32 total_trades = 6;
    double avg_monthly_return = 7;
    string tracking_since = 8;
    string last_updated = 9;
  }

  message GetLivePerformanceRequest {
    string strategy_id = 1;
    int32 days = 2;               // 默认 90
  }

  message GetLivePerformanceResponse {
    repeated LivePerformancePoint points = 1;
    LivePerformanceSummary summary = 2;
    bool has_live_data = 3;
  }

  // Add to MarketplaceService:
  // rpc GetLivePerformance(GetLivePerformanceRequest) returns (GetLivePerformanceResponse);
  ```

### 后端实现

- [ ] **1.1d NATS 事件订阅 + 业绩写入**

  **文件**：`backend/internal/marketplace/live_performance.go` (new)

  **核心逻辑**：

  ```go
  // LivePerformanceCollector subscribes to trade events and writes performance data.
  // No timer, no polling — pure push from existing NATS streams.
  type LivePerformanceCollector struct {
      marketplace *Service
      natsSub     nats.Subscription
      log         *zap.Logger
  }

  // Start begins listening to NATS subjects for trade events.
  func (c *LivePerformanceCollector) Start(ctx context.Context) error {
      // Subscribe to trade-closed events (already published by OMS/mthub)
      sub, err := js.Subscribe("mt.*.trade.closed", c.handleTradeClosed)
      // Subscribe to equity change events
      sub2, _ := js.Subscribe("mt.*.account.*.equity", c.handleEquityChanged)
      ...
  }

  func (c *LivePerformanceCollector) handleTradeClosed(msg *nats.Msg) {
      // 1. Parse trade event → extract account_id, symbol, pnl, volume
      // 2. Look up: is this account linked to any published strategy?
      //    SELECT strategy_id FROM marketplace_strategies WHERE linked_account_id = $1 AND status = 'published'
      // 3. If yes → compute daily metrics → call UpsertDailyPerformance
      // 4. If no → ignore (not a marketplace-linked account)
  }

  func (c *LivePerformanceCollector) handleEquityChanged(msg *nats.Msg) {
      // Update equity/drawdown in the daily record if already exists for today
  }
  ```

  **依赖**：
  - 需要 `marketplace_strategies` 新增 `linked_account_id UUID`——提供者绑定实盘 MT 账户。
  - NATS JetStream 的 trade/equity subject 已存在（OMS/mthub 已发布）。

  **红线**：
  - 采集失败不 panic，记录 error log。
  - 使用 decimal 计算 PnL 和 return。
  - 同一日同一策略多笔交易：用 UPSERT 累积（ON CONFLICT DO UPDATE）。

- [ ] **1.1e `marketplace_strategies` 加绑定字段**

  **文件**：`backend/migrations/xxx_marketplace_linked_account.up.sql` (new)

  ```sql
  ALTER TABLE marketplace_strategies ADD COLUMN linked_account_id UUID;
  ```

  新增 RPC `LinkLiveAccount(strategy_id, account_id)` — 提供者绑定/解绑实盘账户。

- [ ] **1.1f `GetLivePerformance` RPC 实现**

  **文件**：`backend/internal/connect/marketplace/marketplace_handler_performance.go` (new)

  **逻辑**：
  1. 从 `marketplace_live_performance` 查最近 N 天明细。
  2. 从 `marketplace_live_performance_summary` 查聚合总结。
  3. 组装 proto response。

  **验收**：有实盘数据的策略返回完整 response；无数据的策略 `has_live_data=false`。

### 前端实现

- [ ] **1.1g 策略详情页 — 实盘业绩 Tab**

  **文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`
  **变更**：在详情 Modal 中新增 Tab 切换：回测业绩 | 实盘业绩。

  **实盘业绩 Tab 内容**：
  - 净值曲线图（复用 `ProtectedBacktestPanel` 的 Recharts 图表组件）
  - 月度收益热力图（12 列 × N 行，绿色正收益 / 红色负收益）
  - 关键指标卡片（累计收益、年化收益、最大回撤、夏普、胜率）
  - 无数据状态：灰色占位 + "此策略暂未绑定实盘账户，暂无实盘业绩数据"

  **红线**：
  - 回测 Tab 和实盘 Tab 之间必须有明显的视觉区分（颜色/标签）——防止用户混淆回测和实盘。
  - 实盘数据不足 30 天时显示"跟踪时间较短，数据仅供参考"。

- **Gate 1.1**：`go build ./...` + `buf generate` + 前端编译通过 + `npm run test`。

---

## 模块 1.3 · 提供者身份验证

- [ ] **1.3a DB 迁移**

  **文件**：`backend/migrations/xxx_provider_verification.up.sql` (new)

  ```sql
  ALTER TABLE users ADD COLUMN verified_provider BOOLEAN NOT NULL DEFAULT false;
  ALTER TABLE users ADD COLUMN provider_type VARCHAR(20) NOT NULL DEFAULT 'human';
  -- provider_type: 'human' | 'ai' | 'hybrid'

  CREATE TABLE provider_verification_requests (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      display_name VARCHAR(100) NOT NULL,      -- 对外显示名称
      contact_email VARCHAR(255),               -- 联系方式
      proof_url TEXT,                           -- 证明材料链接（个人网站/LinkedIn/其他）
      status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending/approved/rejected
      reviewed_by UUID REFERENCES users(id),
      review_note TEXT,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      reviewed_at TIMESTAMPTZ
  );
  ```

  **设计决策**：
  - 验证请求和用户表分离——`users.verified_provider` 是审核后的结果字段，`provider_verification_requests` 是审核流程记录。
  - `provider_type` 在 `users` 而非 `marketplace_strategies`——因为这是提供者级别的属性，不随策略变化。

- [ ] **1.3b Proto 扩展**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **变更**：扩展 `PublishedStrategy` 消息，新增 `RpcRequestVerification`。

  ```protobuf
  message PublishedStrategy {
    // ... existing fields ...
    bool provider_verified = 20;
    string provider_type = 21;  // "human" | "ai" | "hybrid"
  }

  message RequestVerificationRequest {
    string display_name = 1;
    string contact_email = 2;
    string proof_url = 3;
  }
  message RequestVerificationResponse {
    string request_id = 1;
    string status = 2;  // "pending"
  }

  // Admin RPCs:
  message ListVerificationRequestsRequest { ... }
  message ReviewVerificationRequest { string request_id = 1; string status = 2; string note = 3; }

  // Add to MarketplaceService:
  // rpc RequestVerification(RequestVerificationRequest) returns (RequestVerificationResponse);
  // rpc ListVerificationRequests(ListVerificationRequestsRequest) returns (...);  // Admin
  // rpc ReviewVerificationRequest(ReviewVerificationRequest) returns (...);       // Admin
  ```

- [ ] **1.3c `ListPublished` 扩展**

  **文件**：`backend/internal/marketplace/publish.go`
  **变更**：`ListPublished` 的 SQL JOIN 扩展——从 `users` 表取 `verified_provider` 和 `provider_type` 字段，填充到 `PublishedStrategy`。

  **验收**：已验证提供者的策略在列表 API 中 `provider_verified=true`。

- [ ] **1.3d 前端认证徽章**

  **文件**：`frontend/src/pages/marketplace/components/StrategyMarketCard.tsx`
  **变更**：
  - 已验证提供者 → 策略卡片右上角显示蓝色盾牌徽章 ✓
  - AI 生成策略 → 标签 `🤖 AI Generated`
  - 混合策略 → 标签 `🤖+👤 Hybrid`

- **Gate 1.3**：`go build ./...` + `buf generate` + 前端编译通过。

---

## 模块 1.4 · 风险声明与合规

**前置依赖**：模块 1.5（购买流程正确才能加确认步骤）。

- [ ] **1.4a 策略详情页风险提示**

  **文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`
  **变更**：在 Modal 底部（操作按钮上方）新增不可折叠的 `Alert` 组件：

  ```tsx
  <Alert type="warning" showIcon
    message={t('riskWarning.title')}
    description={t('riskWarning.description')}
  />
  ```

  **i18n 键**（5 语言）：
  - `riskWarning.title`: "⚠️ 风险提示"
  - `riskWarning.description`: "过往业绩不代表未来表现。策略交易存在本金损失风险，请根据自身风险承受能力谨慎决策。本平台展示的所有回测及实盘数据仅供参考，不构成投资建议。"

- [ ] **1.4b 首次购买确认弹窗**

  **文件**：`frontend/src/pages/marketplace/components/PaymentModal.tsx`
  **变更**：在确认支付按钮之前，新增：

  ```tsx
  <Checkbox checked={confirmed} onChange={...}>
    {t('riskWarning.confirmCheckbox')}
    {/* "我已阅读并理解风险提示，自愿承担策略交易可能带来的本金损失" */}
  </Checkbox>
  <Button disabled={!confirmed} onClick={handleConfirmPayment}>
    {t('confirmPayment')}
  </Button>
  ```

  **设计决策**：首次购买才弹（后端查此用户是否有过任何购买记录），之后不重复。降低摩擦同时满足合规。

- [ ] **1.4c `marketplace_strategies` 免责声明字段**

  **文件**：`backend/migrations/xxx_marketplace_disclaimer.up.sql` (new)

  ```sql
  ALTER TABLE marketplace_strategies ADD COLUMN disclaimer_text TEXT;
  -- 提供者可自定义免责声明，为空则使用平台默认文本
  ```

  **前端**：策略详情页底部显示（平台默认 or 提供者自定义）。

- **Gate 1.4**：5 语言 i18n 键完整 + 前端编译通过。

---

## Phase 1 完成检验

全部模块通过后：

```bash
go build ./...                              # 后端编译
buf generate                                # proto 重新生成
go test ./internal/marketplace/...          # 核心逻辑测试
go test ./internal/connect/marketplace/...  # RPC 层测试
cd frontend && npm run build                # 前端编译
bash scripts/gen_capability_map.sh          # 能力表刷新
```

**关键验收场景**：
1. 用正确价格模型发布策略 → 成功
2. 用错误价格模型发布/购买 → 明确错误
3. 回测不达标发布 → 拒绝 + 指出具体指标
4. 已绑实盘账户的策略 → 详情页显示实盘曲线
5. 未绑实盘账户的策略 → 显示"暂无数据"
6. 已验证提供者 → 徽章显示
7. 首次购买 → 风险确认弹窗
