# Phase 5 · 护城河 · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 5
> 前置条件：Phase 1-3 完成，用户量达到临界质量
> 优先级：🟢 P3 — 在前四个 Phase 稳定运行后启动

## 0. 目标与边界

- **达成即最优**：建立长期竞争壁垒——跟单网络效应（离开平台就失去跟单能力）、捆绑包（提升客单价）、白标（B2B 收入渠道）。
- **时机判断**：月活跃买方 >1000、上架策略 >200 时启动本 Phase 的投资回报最高。

---

## 模块 5.1 · 策略捆绑包

- [ ] **5.1a 数据模型**

  **文件**：`backend/migrations/xxx_marketplace_bundles.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_bundles (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      title VARCHAR(200) NOT NULL,
      description TEXT,
      strategy_ids UUID[] NOT NULL,       -- 2-10 个策略
      bundle_price NUMERIC(20,8) NOT NULL, -- 捆绑售价
      original_total NUMERIC(20,8) NOT NULL, -- 原价总和（用于显示节省金额）
      discount_pct NUMERIC(5,2),           -- 折扣百分比
      publisher_user_id UUID REFERENCES users(id), -- 捆绑包创建者（可选：平台 or 提供者）
      status VARCHAR(20) DEFAULT 'published',
      -- published / hidden
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

- [ ] **5.1b RPC 定义**

  ```protobuf
  message BundleStrategy {
    PublishedStrategy strategy = 1;
    string individual_price = 2;
  }
  message BundleItem {
    string id = 1;
    string title = 2;
    string description = 3;
    repeated BundleStrategy strategies = 4;
    string bundle_price = 5;
    string original_total = 6;
    double discount_pct = 7;
    string total_subscribers = 8;
  }

  // Add to MarketplaceService:
  // rpc ListBundles(ListBundlesRequest) returns (ListBundlesResponse);
  // rpc PurchaseBundle(PurchaseBundleRequest) returns (PurchaseBundleResponse);
  ```

- [ ] **5.1c `PurchaseBundle` 实现**

  **文件**：`backend/internal/marketplace/bundle.go` (new)

  **逻辑**：
  1. 校验 bundle 下所有策略均为 published
  2. 验重：用户是否已购买 bundle 内所有策略（已购买某个策略的提醒但不阻止）
  3. 扣款 `bundle_price`
  4. 按各策略原价比例分配收入给各提供者
  5. 创建 N 条 subscription 记录（一个 bundle 购买 → N 个策略订阅）

  **关键边界**：
  - 如果 bundle 内某个策略的提供者账号已被禁用 → bundle 仍可购买，但该策略的提供者不分账（平台代持）。
  - bundle 退款：整包退款，按原比例扣回各提供者。

- [ ] **5.1d 前端捆绑包展示**

  **文件**：`frontend/src/pages/marketplace/components/BundleCard.tsx` (new)

  **UI**：卡片展示捆绑包（标题/包含策略数/原价 vs 捆绑价/节省金额），点击展开内嵌策略列表。

- **Gate 5.1**：`go build ./...` + `buf generate` + 捆绑包购买+退款链路测试。

---

## 模块 5.2 · 跟单 UI

- [ ] **5.2a 跟单配置 RPC**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message StartCopyTradeRequest {
    string strategy_id = 1;
    string account_id = 2;       // 跟单目标账户
    string allocation_pct = 3;   // 跟单比例 "50" = 50%
    string max_position_size = 4; // 单笔最大仓位（decimal）
    string stop_loss_pct = 5;    // 可选：跟单止损比例
  }
  message CopyTradeStatus {
    bool is_active = 1;
    string strategy_id = 2;
    string account_id = 3;
    string allocation_pct = 4;
    string total_pnl = 5;        // 跟单累计盈亏
    int32 copied_trades = 6;     // 已跟单笔数
    string started_at = 7;
    string last_signal_at = 8;
  }

  // Add to MarketplaceService:
  // rpc StartCopyTrade(StartCopyTradeRequest) returns (google.protobuf.Empty);
  // rpc StopCopyTrade(StopCopyTradeRequest) returns (google.protobuf.Empty);
  // rpc GetCopyTradeStatus(GetCopyTradeStatusRequest) returns (CopyTradeStatus);
  // rpc ListMyCopyTrades(google.protobuf.Empty) returns (ListMyCopyTradesResponse);
  ```

  **设计决策**：后端 `CopyTradeEngine` 已完成，只需在 `user_subscriptions` 中管理跟单关系（`kind='copy_trade'`）。跟单配置存储在 `subscription` 的 metadata 字段或单独的 `user_copy_trade_configs` 表中。

- [ ] **5.2b 跟单配置表**

  **文件**：`backend/migrations/xxx_user_copy_trade_configs.up.sql` (new)

  ```sql
  CREATE TABLE user_copy_trade_configs (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id),
      account_id UUID NOT NULL,         -- 哪个 MT 账户接收跟单
      allocation_pct NUMERIC(5,2) NOT NULL DEFAULT 100,  -- 跟单比例 1-100
      max_position_size NUMERIC(20,8),  -- 单笔最大仓位，NULL = 不限制
      stop_loss_pct NUMERIC(5,2),       -- 止损比例
      active BOOLEAN DEFAULT true,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

      UNIQUE (user_id, strategy_id, account_id)
  );
  ```

- [ ] **5.2c 前端跟单入口**

  **文件**：`frontend/src/pages/marketplace/components/CopyTradeSetupModal.tsx` (new)

  **UI**：
  - 策略详情页"跟单"按钮
  - 弹窗配置：
    - 选择跟单账户（下拉，列出已连接的 MT 账户）
    - 跟单比例滑块（10%-100%，默认 50%）
    - 单笔最大仓位（可选）
    - 止损比例（可选，如"跟单策略亏损 >20% 自动停止"）
  - 一键开启

  **已购策略列表**：显示跟单状态（跟单中/已暂停）+ 跟单收益 + 停止按钮。

- **Gate 5.2**：`go build ./...` + `buf generate` + 跟单配置 CRUD 测试 + 端到端跟单测试（提供者下单 → 订阅者账户出现订单）。

---

## 模块 5.3 · 阶梯费率

- [ ] **5.3a 提供者等级定义**

  **文件**：`backend/internal/marketplace/tier.go` (new)

  | 等级 | 条件 | 平台费率 |
  |------|------|---------|
  | Bronze | 默认 | 25% |
  | Silver | 月收入 ≥ $500 + 评分 ≥ 4.0 | 20% |
  | Gold | 月收入 ≥ $2000 + 评分 ≥ 4.3 | 12% |
  | Platinum | 月收入 ≥ $5000 + 评分 ≥ 4.5 | 8% |

  **实现**：
  ```go
  type ProviderTier struct {
      Name         string
      MinRevenue   decimal.Decimal
      MinRating    float64
      PlatformFee  decimal.Decimal  // 0.25, 0.20, 0.12, 0.08
  }
  ```

  **等级计算 — 惰性求值（禁用定时任务）**：
  - 🔴 禁止月度 cron。等级在以下时机按需计算：
    1. **购买时**：`PurchaseStrategy` 从 DB 实时查询提供者过去 30 天收入 + 当前评分 → 映射等级 → 应用对应费率。计算逻辑是纯 SQL 聚合（已有数据），单次查询 <10ms。
    2. **Dashboard 展示时**：`GetPublisherStats` 返回当前等级 + 升级进度条。
    3. **等级变更记录**：购买时如果计算出的等级与 `users.provider_tier` 不同，UPDATE 并写 `provider_tier_history`。
  - 为什么惰性求值可行：等级判断只依赖已有数据（`wallet_transactions` 过去 30 天 SUM + `marketplace_ratings` AVG），纯只读查询，无需预计算。

- [ ] **5.3b DB 扩展**

  **文件**：`backend/migrations/xxx_provider_tiers.up.sql` (new)

  ```sql
  ALTER TABLE users ADD COLUMN provider_tier VARCHAR(20) DEFAULT 'bronze';
  -- bronze / silver / gold / platinum

  CREATE TABLE provider_tier_history (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      old_tier VARCHAR(20),
      new_tier VARCHAR(20) NOT NULL,
      monthly_revenue NUMERIC(20,8),
      avg_rating DOUBLE PRECISION,
      calculated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

- [ ] **5.3c `PurchaseStrategy` 集成**

  **变更**：购买时从 `users.provider_tier` 读取提供者当前等级 → 映射 `platform_fee_rate` → 计算分账。

  **当前逻辑**：`platform_fee_rate` 从 `marketplace_strategies` 表读取（策略上架时固定）。

  **新逻辑**：如果 `marketplace_strategies.platform_fee_rate` 为空或为默认值（表示"使用提供者等级费率"），则从 `provider_tier` 映射动态计算。如果策略上架时手动指定了费率，优先使用手动指定值。

  **设计决策**：不强制改写已有策略的费率——向后兼容。新上架策略默认使用等级费率。

- [ ] **5.3d 前端等级展示**

  **文件**：`frontend/src/pages/marketplace/AuthorTab.tsx`
  **扩展**：提供者 Dashboard 显示当前等级徽章 + 升到下一级的条件进度条。

- **Gate 5.3**：`go build ./...` + 等级计算逻辑单测 + 不同等级分账验证。

---

## 模块 5.4 · 白标 / API

- [ ] **5.4a 策略市场公开 API**

  **设计决策**：策略市场当前是平台内闭环（用户在平台内浏览+购买）。白标场景需要 broker 等第三方在自己的平台展示策略。三种方案：

  | 方案 | 优点 | 缺点 |
  |------|------|------|
  | A: iframe 嵌入 | 零开发，平台风格可控 | 用户体验受限，跨域问题 |
  | B: ConnectRPC API | 标准 proto，broker 自行集成 | broker 需要理解 proto |
  | C: REST API（仅此场景豁免）| broker 最容易集成 | 项目禁止 REST |

  **选择方案 B + iframe 兜底**：
  - 优先提供 iframe 嵌入方案（一行 `<script>` 标签），broker 零集成成本
  - 提供 ConnectRPC API 供有能力的 broker 深度集成（自定义 UI）
  - REST API 仅作为最后的 fallback（如果 broker 无法处理 gRPC-web）

- [ ] **5.4b iframe 嵌入 SDK**

  **文件**：`frontend/public/embed/marketplace-widget.js` (new)

  **功能**：一行脚本加载策略市场 widget：
  ```html
  <script src="https://alfq.org/embed/marketplace-widget.js"
          data-broker-id="xxx"
          data-theme="light"
          data-asset-class="forex">
  </script>
  <div id="alphaforge-marketplace"></div>
  ```

  **Widget 内部**：iframe 加载 `/embed/marketplace?broker_id=xxx&asset_class=forex`，使用 PostMessage 与父页面通信（购买事件通知 broker）。

- [ ] **5.4c 对外 ConnectRPC API**

  **新增 RPC**（broker 专用）：
  ```protobuf
  // 公开 API（不需要用户登录，需要 broker API key）
  // rpc PublicListStrategies(PublicListStrategiesRequest) returns (PublicListStrategiesResponse);
  // rpc GetPublicStrategy(GetPublicStrategyRequest) returns (GetPublicStrategyResponse);
  ```

  **认证**：Broker API Key 通过 Header 传入（`X-Broker-API-Key`），在 ConnectRPC interceptor 中校验。

- [ ] **5.4d Broker 管理后台**

  **文件**：`backend/migrations/xxx_marketplace_brokers.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_brokers (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name VARCHAR(100) NOT NULL,
      api_key VARCHAR(64) UNIQUE NOT NULL,
      revenue_share_pct NUMERIC(5,2) DEFAULT 10,  -- broker 引入的用户，broker 分成比例
      allowed_asset_classes TEXT[] DEFAULT '{}',
      custom_css TEXT,         -- 自定义 iframe 样式
      logo_url TEXT,
      enabled BOOLEAN DEFAULT true,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

- **Gate 5.4**：`go build ./...` + iframe widget 在第三方页面加载测试 + API key 认证测试。

---

## Phase 5 完成检验

```bash
go build ./...
buf generate
go test ./internal/marketplace/...
cd frontend && npm run build
bash scripts/gen_capability_map.sh
```

**关键验收场景**：
1. 捆绑包购买 → N 个策略同时订阅 → 提供者按比例分账
2. 跟单一键开启 → 提供者下单 → 订阅者账户出现按比例缩放的订单
3. 提供者月度收入达标 → 等级自动升级 → 新购买使用更低平台费率
4. broker 页面加载 iframe widget → 策略列表正确展示 → 购买事件通知 broker
