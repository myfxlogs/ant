> ⚠️ 已迁移至 docs/blocks/strategy-marketplace/plans/phase-4-platform-ops.md。此文件保留为兼容旧引用。

# Phase 4 · 平台运营 · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 4
> 前置条件：Phase 1 完成（退款/Admin RPC 后端已有，补前端）
> 本清单各模块独立，可与 Phase 3 并行。

## 0. 目标与边界

- **达成即最优**：平台方具备完整的运营管理能力——审核策略、处理退款、查看收入、管理版本、发放优惠券。
- **关键原则**：所有 Admin 操作必须记录 audit log（`admin_audit_logs` 表已有，复用）。

---

## 依赖图

```
4.1 (Admin 策略管理) ── 独立，依赖现有 RPC
4.2 (退款 UI)        ── 独立，依赖 Phase 1.5
4.3 (收入仪表盘)     ── 依赖 Phase 1.1 (实盘数据) + 现有 wallet 交易数据
4.4 (策略版本管理)   ── 独立
4.5 (折扣优惠券)     ── 独立
4.6 (策略提现)       ── 依赖 ADR-0026 HD 钱包冷签名
```

---

## 模块 4.1 · Admin 策略管理面板

- [ ] **4.1a 策略审核列表 RPC**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message AdminListStrategiesRequest {
    string status = 1;     // published / hidden / all
    string keyword = 2;    // 搜索
    int32 limit = 3;
    int32 offset = 4;
  }
  message AdminStrategyItem {
    PublishedStrategy strategy = 1;
    string status = 2;     // published / hidden
    int32 total_sales = 3;
    string total_revenue = 4;  // 该策略产生的总收入（平台+提供者）
    string platform_revenue = 5;
    string last_sale_at = 6;
  }
  message AdminListStrategiesResponse {
    repeated AdminStrategyItem strategies = 1;
    int32 total = 2;
  }

  // 已有 RPC，补前端 UI：
  // rpc SetStrategyPricing(...)   — 已有
  // rpc UnpublishStrategy(...)    — 已有
  // 新增：
  // rpc AdminListStrategies(AdminListStrategiesRequest) returns (AdminListStrategiesResponse);
  // rpc AdminFeatureStrategy(AdminFeatureStrategyRequest) returns (google.protobuf.Empty);
  ```

- [ ] **4.1b 推荐/置顶支持**

  **文件**：`backend/migrations/xxx_marketplace_featured.up.sql` (new)

  ```sql
  ALTER TABLE marketplace_strategies ADD COLUMN is_featured BOOLEAN DEFAULT false;
  ALTER TABLE marketplace_strategies ADD COLUMN featured_until TIMESTAMPTZ;
  ALTER TABLE marketplace_strategies ADD COLUMN featured_priority INT DEFAULT 0;
  ```

  **`ListPublished` 扩展**：置顶策略排在非置顶之前（ORDER BY `is_featured DESC, featured_priority DESC, published_at DESC`）。

- [ ] **4.1c 前端 Admin 策略管理页面**

  **文件**：`frontend/src/pages/admin/MarketplaceManagement.tsx` (new)

  **UI**：
  - Table：策略列表（名称/提供者/状态/价格/销量/收入/上架日期）
  - 操作列：编辑定价 / 隐藏 / 置顶 / 查看详情
  - 置顶弹窗：设置置顶时长 + 优先级
  - 隐藏确认：填写隐藏原因 + 通知提供者（可选）
  - 状态筛选 Tab：全部 / 已发布 / 已隐藏

  **路由**：`/admin/marketplace`

- **Gate 4.1**：`go build ./...` + `buf generate` + 前端构建通过。

---

## 模块 4.2 · 退款 UI

- [ ] **4.2a 退款申请 RPC**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message RequestRefundRequest {
    string subscription_id = 1;
    string reason = 2;           // 用户填写的退款原因
  }
  message RequestRefundResponse {
    string refund_id = 1;
    string status = 2;           // "pending_review"
  }

  message AdminListRefundRequestsRequest {
    string status = 1;           // pending / approved / rejected
    int32 limit = 2;
    int32 offset = 3;
  }
  message RefundRequestItem {
    string refund_id = 1;
    string user_id = 2;
    string user_name = 3;
    string subscription_id = 4;
    string strategy_title = 5;
    string amount = 6;
    string reason = 7;
    string status = 8;
    string created_at = 9;
    string reviewed_by = 10;
    string review_note = 11;
  }
  message AdminListRefundRequestsResponse {
    repeated RefundRequestItem requests = 1;
    int32 total = 2;
  }

  // 已有 RPC（后端 refund.go 已完成），新增前端 RPC：
  // rpc RequestRefund(RequestRefundRequest) returns (RequestRefundResponse);
  // rpc AdminListRefundRequests(AdminListRefundRequestsRequest) returns (AdminListRefundRequestsResponse);
  // rpc AdminProcessRefund(AdminProcessRefundRequest) returns (google.protobuf.Empty);
  ```

  **设计决策**：退款不自动执行——用户提交申请 → Admin 审核 → 手动确认后调用已有 `RefundPurchase`。减少恶意退款风险。

- [ ] **4.2b 退款申请表**

  **文件**：`backend/migrations/xxx_marketplace_refund_requests.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_refund_requests (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      subscription_id UUID NOT NULL REFERENCES user_subscriptions(id),
      reason TEXT NOT NULL,
      status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending / approved / rejected
      reviewed_by UUID REFERENCES users(id),
      review_note TEXT,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      reviewed_at TIMESTAMPTZ
  );
  ```

- [ ] **4.2c 前端退款入口**

  **文件**：`frontend/src/pages/marketplace/PurchaseTab.tsx`
  **变更**：已购策略列表的操作列新增"申请退款"按钮。

  **退款条件**（前端展示规则）：
  - 购买后 7 天内可申请
  - 一次购买只能申请一次
  - 已申请待审核 → 显示"退款审核中"
  - 退款已批准 → 显示"已退款"

  **Admin 退款审核面板**：`frontend/src/pages/admin/RefundManagement.tsx` (new)

- **Gate 4.2**：`go build ./...` + `buf generate` + 前端构建通过 + 退款流程手动测试。

---

## 模块 4.3 · 收入仪表盘 (Admin)

- [ ] **4.3a 聚合查询 RPC**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message GetMarketplaceAnalyticsRequest {
    string period = 1;  // "7d" / "30d" / "90d" / "all"
  }
  message MarketplaceAnalytics {
    // 收入指标
    string total_gmv = 1;
    string platform_revenue = 2;     // 平台抽成收入
    string provider_revenue = 3;     // 提供者总收入
    int32 total_transactions = 4;

    // 用户指标
    int32 active_buyers = 5;         // 期间内至少购买一次的用户
    int32 new_subscribers = 6;
    string arpu = 7;                 // 人均消费

    // 策略指标
    int32 total_strategies = 8;
    int32 new_strategies = 9;        // 期间内新上架

    // 健康指标
    string refund_rate = 10;         // 退款率
    string renewal_rate = 11;        // 续费率（订阅制）

    // 趋势数据（用于前端折线图）
    repeated DailyAnalytics daily = 12;
  }
  message DailyAnalytics {
    string date = 1;
    string gmv = 2;
    int32 transactions = 3;
    int32 new_subscribers = 4;
  }

  message TopItem {
    string id = 1;
    string name = 2;
    string value = 3;
    int32 rank = 4;
  }
  message TopStrategiesResponse {
    repeated TopItem by_revenue = 1;
    repeated TopItem by_subscribers = 2;
  }
  message TopProvidersResponse {
    repeated TopItem by_revenue = 1;
    repeated TopItem by_strategies = 2;
  }

  // Add to MarketplaceService // Admin only:
  // rpc GetMarketplaceAnalytics(GetMarketplaceAnalyticsRequest) returns (MarketplaceAnalytics);
  // rpc GetTopStrategies(google.protobuf.Empty) returns (TopStrategiesResponse);
  // rpc GetTopProviders(google.protobuf.Empty) returns (TopProvidersResponse);
  ```

- [ ] **4.3b 查询实现**

  **文件**：`backend/internal/marketplace/analytics.go` (new)

  **数据源**：
  - GMV / 平台收入：`wallet_transactions WHERE tx_type IN ('purchase','platform_fee')`
  - 退款率：`COUNT(tx_type='refund') / COUNT(tx_type='purchase')`
  - 续费率：`COUNT(subscriptions renewed) / COUNT(subscriptions expired)`

  **设计决策**：不做实时聚合。Admin Dashboard 查询频率极低（每天几次），直接查 PG 即可，无需额外缓存。

- [ ] **4.3c 前端 Admin 收入面板**

  **文件**：`frontend/src/pages/admin/MarketplaceAnalytics.tsx` (new)

  **UI**：
  - 顶部统计卡片（GMV / 平台收入 / 活跃买家 / ARPU / 退款率）
  - GMV 趋势折线图（Recharts）
  - TOP 策略 / TOP 提供者 表格
  - 时间范围选择器（7天 / 30天 / 90天 / 全部）

- **Gate 4.3**：`go build ./...` + `buf generate` + 前端构建通过。

---

## 模块 4.4 · 策略版本管理

- [ ] **4.4a 版本表**

  **文件**：`backend/migrations/xxx_marketplace_strategy_versions.up.sql` (new)

  ```sql
  -- 版本号字段
  ALTER TABLE marketplace_strategies ADD COLUMN version VARCHAR(20) NOT NULL DEFAULT '1.0.0';

  -- 版本历史
  CREATE TABLE marketplace_strategy_versions (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id),
      version VARCHAR(20) NOT NULL,   -- SemVer: "1.0.0", "1.1.0", "2.0.0"
      changelog TEXT NOT NULL DEFAULT '',
      -- 快照该版本的策略源码引用（指向 strategy_templates 的某个版本）
      template_snapshot_id UUID,       -- 可选：引用特定 template 版本
      backtest_snapshot BYTEA,         -- 该版本的回测快照
      published_by UUID NOT NULL REFERENCES users(id),
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

      UNIQUE (strategy_id, version)
  );
  ```

  **设计决策**：
  - 版本号使用 SemVer：MAJOR（策略逻辑大改）/ MINOR（参数优化）/ PATCH（bug 修复）。
  - 每个版本关联一个回测快照——用户可以对比不同版本的回测表现。
  - `template_snapshot_id` 可选——策略源码可能在 `strategy_templates` 中有自己的版本管理。

- [ ] **4.4b RPC 定义**

  ```protobuf
  message ListStrategyVersionsRequest {
    string strategy_id = 1;
  }
  message StrategyVersion {
    string id = 1;
    string version = 2;
    string changelog = 3;
    string published_at = 4;
    BacktestSnapshot backtest = 5;
  }
  message ListStrategyVersionsResponse {
    repeated StrategyVersion versions = 1;
    string current_version = 2;
  }

  message UpdateStrategyVersionRequest {
    string strategy_id = 1;
    string new_version = 2;       // "2.0.0"
    string changelog = 3;
    bytes backtest_snapshot = 4;  // 新版本的回测快照（proto）
  }

  // Add to MarketplaceService:
  // rpc ListStrategyVersions(ListStrategyVersionsRequest) returns (ListStrategyVersionsResponse);
  // rpc UpdateStrategyVersion(UpdateStrategyVersionRequest) returns (google.protobuf.Empty);
  ```

- [ ] **4.4c `UpdateStrategyVersion` 实现**

  **文件**：`backend/internal/marketplace/version.go` (new)

  **逻辑**：
  1. 校验 `new_version > current_version`（字符串比较 SemVer）
  2. 写 `marketplace_strategy_versions`（版本快照）
  3. UPDATE `marketplace_strategies.version = new_version`
  4. 已购买用户**不自动升级**——在策略详情页显示"新版本可用 (v2.0.0)"提示，用户手动操作

- [ ] **4.4d 前端版本历史**

  **文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`
  **扩展**：在详情页加入"版本历史"区域（Collapse 面板）。

  - 时间线样式（v2.0.0 — 最新 → v1.0.0 — 初始发布）
  - 每个版本展开显示 changelog + 回测指标对比
  - 已购用户看到"有新版本可用"提示 → 点击查看新版本回测 → 选择是否升级

- **Gate 4.4**：`go build ./...` + `buf generate` + 前端构建通过。

---

## 模块 4.5 · 折扣/优惠券

- [ ] **4.5a 优惠券表**

  **文件**：`backend/migrations/xxx_marketplace_coupons.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_coupons (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      code VARCHAR(50) UNIQUE NOT NULL,        -- 优惠码，如 "LAUNCH20"
      discount_type VARCHAR(10) NOT NULL,       -- "percentage" / "fixed"
      discount_value NUMERIC(20,8) NOT NULL,    -- 0-100 (percentage) or 金额 (fixed)
      min_purchase_amount NUMERIC(20,8) DEFAULT 0,  -- 最低消费门槛
      max_uses INT DEFAULT 0,                   -- 0 = 无限
      used_count INT DEFAULT 0,
      expires_at TIMESTAMPTZ,                   -- NULL = 永不过期
      applicable_strategy_ids UUID[] DEFAULT '{}',
      -- 空数组 = 全局可用
      created_by UUID REFERENCES users(id),
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      enabled BOOLEAN DEFAULT true
  );
  CREATE INDEX idx_coupons_code ON marketplace_coupons(code) WHERE enabled = true;
  ```

- [ ] **4.5b RPC 定义**

  ```protobuf
  message ValidateCouponRequest {
    string code = 1;
    string strategy_id = 2;
    string purchase_amount = 3;     // 原价
  }
  message ValidateCouponResponse {
    bool valid = 1;
    string discount_type = 2;
    string discount_amount = 3;     // 折扣金额（decimal）
    string final_amount = 4;        // 折后价（decimal）
    string error_message = 5;       // valid=false 时的原因
  }

  // Admin RPCs:
  // rpc CreateCoupon(CreateCouponRequest) returns (CreateCouponResponse);
  // rpc ListCoupons(ListCouponsRequest) returns (ListCouponsResponse);
  // rpc DisableCoupon(DisableCouponRequest) returns (google.protobuf.Empty);

  // Add to MarketplaceService:
  // rpc ValidateCoupon(ValidateCouponRequest) returns (ValidateCouponResponse);
  ```

- [ ] **4.5c `ValidateCoupon` 逻辑**

  **文件**：`backend/internal/marketplace/coupon.go` (new)

  ```go
  func (s *Service) ValidateCoupon(ctx context.Context, code, strategyID, amount string) (*CouponResult, error) {
      // 1. SELECT * FROM marketplace_coupons WHERE code=$1 AND enabled=true
      // 2. 校验过期
      // 3. 校验 max_uses
      // 4. 校验 applicable_strategy_ids（为空则全局，否则必须在列表内）
      // 5. 校验 min_purchase_amount（amount >= min_purchase_amount）
      // 6. 计算折扣：percentage → amount * discount_value / 100；fixed → discount_value（不超过 amount）
      // 7. 返回折后价
  }
  ```

  **`PurchaseStrategy` 扩展**：接受可选 `coupon_code` 参数，先 `ValidateCoupon` 再扣款，扣款金额为折后价。扣款成功后 `UPDATE used_count = used_count + 1`。

- [ ] **4.5d 前端优惠券输入框**

  **文件**：`frontend/src/pages/marketplace/components/PaymentModal.tsx`

  **UI**：支付弹窗在价格显示下方增加 "优惠码" 输入框 + "应用" 按钮。应用成功后显示：~~¥100~~ → ¥80 (已优惠 ¥20)。

  **Admin 优惠券管理页面**：`frontend/src/pages/admin/CouponManagement.tsx` (new)

- **Gate 4.5**：`go build ./...` + `buf generate` + 优惠券计算逻辑单测。

---

## 模块 4.6 · 策略提现

- [ ] **4.6a 提现申请流程**

  **设计决策**：复用 ADR-0026 的 HD 钱包冷签名提现流程。
  - 策略提供者收入进入 `user_wallets`（已实现——`PurchaseStrategy` 中 `AdjustBalanceTx` 写入 `TxTypeSale`）
  - 提现 = 提供者从 `user_wallets` 转出到外部地址
  - 走冷签名流程：Admin 审批 → 构建 UnsignedTx → USB 传冷签名机 → 签名 → 广播

  **文件**：后端复用现有提现管线，只需新增：
  - 前端 Author Dashboard 的"提现"按钮 + 提现表单（金额 + 目标地址）
  - 提现记录查询 RPC（已有 wallet 交易记录，扩展过滤 `tx_type='withdrawal'`）

  **优先级说明**：此模块严重依赖 ADR-0026 的完成度。如果冷签名管线未就绪，先做成"提供者提交提现申请 → Admin 手动处理"的简化流程。

- **Gate 4.6**：`go build ./...` + 提现申请流程手动测试。

---

## Phase 4 完成检验

```bash
go build ./...
buf generate
go test ./internal/marketplace/...
go test ./internal/connect/marketplace/...
cd frontend && npm run build
bash scripts/gen_capability_map.sh
```

**关键验收场景**：
1. Admin 查看策略列表 → 隐藏策略 → 策略从市场消失
2. 用户申请退款 → Admin 审核批准 → 钱包退款到账 → 订阅停用
3. Admin 收入面板：GMV/平台收入/ARPU 数据正确
4. 提供者更新策略版本 → 已购用户看到"新版本可用"
5. 用户输入优惠码 → 价格正确折扣 → 购买成功 → used_count+1
