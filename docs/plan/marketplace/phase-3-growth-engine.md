# Phase 3 · 增长引擎 · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 3
> 前置条件：Phase 1 + Phase 2 完成（有策略可卖、有信任基础）
> 本清单各模块相对独立，可并行实施。

## 0. 目标与边界

- **达成即最优**：构建双边网络效应——买方越多 → 提供者越多 → 策略越多 → 买方更多。
- **核心手段**：排行榜（发现） + 免费试用（转化） + 策略对比（决策） + 通知（留存）。

---

## 依赖图

```
3.1 (排行榜) ──── 依赖 Phase 1.1 (实盘业绩数据)
3.2 (免费试用) ── 依赖 Phase 1.5 (订阅模型正确)
3.3 (策略对比) ── 独立，纯前端+聚合查询
3.4 (通知系统) ── 依赖现有 SSE 管道
3.5 (社交分享) ── 独立，纯前端+SEO
```

---

## 模块 3.1 · 策略排行榜

- [ ] **3.1a RPC 定义**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  enum LeaderboardType {
    LEADERBOARD_TYPE_UNSPECIFIED = 0;
    LEADERBOARD_TYPE_RETURN = 1;      // 收益榜
    LEADERBOARD_TYPE_POPULAR = 2;     // 人气榜（按订阅数）
    LEADERBOARD_TYPE_NEW = 3;         // 新锐榜（最近30天上架）
    LEADERBOARD_TYPE_COPYTRADE = 4;   // 跟单榜
  }

  enum LeaderboardPeriod {
    LEADERBOARD_PERIOD_UNSPECIFIED = 0;
    LEADERBOARD_PERIOD_WEEK = 1;
    LEADERBOARD_PERIOD_MONTH = 2;
    LEADERBOARD_PERIOD_QUARTER = 3;
    LEADERBOARD_PERIOD_ALL = 4;
  }

  message LeaderboardEntry {
    int32 rank = 1;
    PublishedStrategy strategy = 2;
    string metric_value = 3;   // 排名依据的指标值（收益率/订阅数/评分）
  }

  message ListLeaderboardRequest {
    LeaderboardType type = 1;
    LeaderboardPeriod period = 2;
    string asset_class = 3;    // 可选筛选
    int32 limit = 4;           // 默认 20，最大 50
  }
  message ListLeaderboardResponse {
    repeated LeaderboardEntry entries = 1;
  }

  // Add to MarketplaceService:
  // rpc ListLeaderboard(ListLeaderboardRequest) returns (ListLeaderboardResponse);
  ```

- [ ] **3.1b 排行榜查询实现**

  **文件**：`backend/internal/marketplace/leaderboard.go` (new)

  **四种榜单的排序逻辑**：

  | 榜单 | 排序依据 | 数据源 |
  |------|---------|--------|
  | 收益榜 | 实盘累计收益率（无实盘的不参与） | `marketplace_live_performance_summary` |
  | 人气榜 | `total_subscribers` | `marketplace_strategies` |
  | 新锐榜 | 上架 30 天内 + 实盘收益率 | `marketplace_strategies.published_at` + `marketplace_live_performance_summary` |
  | 跟单榜 | 跟单人数（`kind='copy_trade'` 订阅数） | `user_subscriptions WHERE kind='copy_trade' GROUP BY target_strategy_id` |

  **缓存策略**：排行榜缓存 5 分钟（比列表页的 60s 长——计算成本更高，用户能接受 5 分钟延迟）。

- [ ] **3.1c 前端排行榜页面**

  **文件**：`frontend/src/pages/marketplace/LeaderboardPage.tsx` (new)

  **UI**：
  - 顶部 Tab 切换榜单类型（收益/人气/新锐/跟单）
  - 时间周期下拉：本周 / 本月 / 本季 / 全部
  - 资产类别筛选 Chip 组（全部 / 外汇 / 加密 / 指数 / 商品）
  - 排行列表：前三名大卡片（🏆🥈🥉）+ 4-20 名列表
  - 列表项：排名徽章 + 策略卡片摘要（标题/提供者/指标/价格）
  - 点击跳转策略详情

  **路由**：`/marketplace/leaderboard`

- **Gate 3.1**：`go build ./...` + `buf generate` + 前端构建通过。

---

## 模块 3.2 · 免费试用

- [ ] **3.2a 试用表**

  **文件**：`backend/migrations/xxx_marketplace_trials.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_trials (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id),
      started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      expires_at TIMESTAMPTZ NOT NULL,         -- started_at + 7 days
      status VARCHAR(20) NOT NULL DEFAULT 'active',
      -- active / expired / cancelled
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

      UNIQUE (user_id, strategy_id)  -- 每个用户对一个策略只能试用一次
  );
  CREATE INDEX idx_trials_user ON marketplace_trials(user_id);
  CREATE INDEX idx_trials_expires ON marketplace_trials(expires_at) WHERE status = 'active';
  ```

- [ ] **3.2b RPC 定义**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message StartTrialRequest {
    string strategy_id = 1;
  }
  message StartTrialResponse {
    string trial_id = 1;
    string expires_at = 2;   // ISO 8601
    int32 trial_days = 3;
  }

  message CheckTrialStatusRequest {
    string strategy_id = 1;
  }
  message CheckTrialStatusResponse {
    bool can_start_trial = 1;      // 是否可以开始试用
    string active_trial_id = 2;    // 如果已有活跃试用
    string active_trial_expires = 3;
    bool has_previously_trialed = 4;  // 是否已经试用过（不可再试）
  }

  // Add to MarketplaceService:
  // rpc StartTrial(StartTrialRequest) returns (StartTrialResponse);
  // rpc CheckTrialStatus(CheckTrialStatusRequest) returns (CheckTrialStatusResponse);
  ```

- [ ] **3.2c 试用逻辑实现**

  **文件**：`backend/internal/marketplace/trial.go` (new)

  **关键方法**：

  ```go
  func (s *Service) StartTrial(ctx context.Context, userID, strategyID string) (string, time.Time, error) {
      // 1. 查策略是否为付费策略（免费策略不需要试用）
      // 2. 查是否已经购买过（已购买不需要试用）
      // 3. 查是否已经试用过（UNIQUE 约束保证每个用户一次）
      // 4. INSERT INTO marketplace_trials (expires_at = now() + 7 days)
      // 5. 返回 trial_id + expires_at
  }
  ```

  **试用过期 — 惰性求值（禁用定时任务）**：
  - 🔴 禁止 timer 扫描过期。在 `CanAccessCode` 查询时做 inline 过期判断：
    ```go
    // 查询试用状态时自动清理过期记录
    _, _ = s.pg.Exec(ctx,
        `UPDATE marketplace_trials SET status='expired'
         WHERE user_id=$1 AND strategy_id=$2 AND status='active' AND expires_at <= now()`,
        userID, strategyID)
    // 然后正常查询
    var trialActive bool
    s.pg.QueryRow(ctx,
        `SELECT EXISTS(SELECT 1 FROM marketplace_trials
         WHERE user_id=$1 AND strategy_id=$2 AND status='active' AND expires_at > now())`,
        userID, strategyID).Scan(&trialActive)
    ```

  **设计决策**：试用过期只在用户下次访问时结算——惰性求值，零定时器。`CanAccessCode` 每次调用先清理过期试用，再判断权限。过期试用的用户看到"试用已过期，¥XX 购买"。

- [ ] **3.2d 前端试用按钮**

  **文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`

  **按钮状态机**：
  ```
  未登录 → "登录后可试用"
  已购买 → 不显示（直接显示"已购买"）
  试用中 → "试用中（还剩 X 天）"
  试用过期 → "试用已过期，¥XX 购买"
  未试用过 → "🎁 免费试用 7 天"
  ```

  **`PaymentModal` 扩展**：首次购买弹窗增加"先试用 7 天"按钮，引导用户试用。

- **Gate 3.2**：`go build ./...` + `buf generate` + 测试演示（试用→过期→购买 完整流程）。

---

## 模块 3.3 · 策略对比

- [ ] **3.3a RPC 定义**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message CompareStrategiesRequest {
    repeated string strategy_ids = 1;  // 2-4 个
  }
  message StrategyCompareRow {
    string strategy_id = 1;
    string title = 2;
    string provider_name = 3;
    // 回测指标
    string backtest_total_return = 4;
    string backtest_max_drawdown = 5;
    string backtest_sharpe = 6;
    string backtest_win_rate = 7;
    // 实盘指标
    string live_total_return = 8;
    string live_max_drawdown = 9;
    string live_tracking_months = 10;
    // 费用
    string price_model = 11;
    string price_amount = 12;
    // 其他
    string risk_level = 13;
    int32 total_subscribers = 14;
    double avg_rating = 15;
  }
  message CompareStrategiesResponse {
    repeated StrategyCompareRow rows = 1;
  }

  // Add to MarketplaceService:
  // rpc CompareStrategies(CompareStrategiesRequest) returns (CompareStrategiesResponse);
  ```

- [ ] **3.3b 后端聚合查询**

  **文件**：`backend/internal/marketplace/compare.go` (new)

  **逻辑**：一次查询 JOIN `marketplace_strategies` + `marketplace_live_performance_summary` + `marketplace_ratings`，返回标准化对比行。不做复杂计算——所有指标已在 summary 表中。

- [ ] **3.3c 前端对比组件**

  **文件**：`frontend/src/pages/marketplace/components/StrategyCompareModal.tsx` (new)

  **UI**：
  - 触发：策略卡片上的复选框 + 底部浮现的"对比已选（N）"按钮
  - Modal 内并排表格：行=指标，列=策略；最优值绿色高亮
  - 支持拖拽调整策略列顺序
  - 底部按钮："查看详情"（跳转到对应策略）

- **Gate 3.3**：`go build ./...` + 前端构建通过。

---

## 模块 3.4 · 通知系统

- [ ] **3.4a 通知表**

  **文件**：`backend/migrations/xxx_marketplace_notifications.up.sql` (new)

  ```sql
  CREATE TABLE marketplace_notifications (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      notification_type VARCHAR(30) NOT NULL,
      -- new_strategy / price_change / subscription_expiring / performance_alert / new_rating / new_comment
      title TEXT NOT NULL,
      body TEXT NOT NULL,
      strategy_id UUID,          -- 关联策略（可选）
      is_read BOOLEAN DEFAULT false,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_notifications_user_unread ON marketplace_notifications(user_id, created_at DESC)
      WHERE is_read = false;
  ```

- [ ] **3.4b 触发规则**

  | 事件 | 接收者 | 触发时机 |
  |------|--------|---------|
  | `new_strategy` | 关注了该品种/提供者的用户 | 新策略上架 |
  | `subscription_expiring` | 订阅者 | 订阅到期前 3 天/1 天 |
  | `performance_alert` | 订阅者 | 策略日回撤超过 5% |
  | `new_rating` | 策略提供者 | 收到新评分 |
  | `new_comment` | 策略提供者 | 收到新评论 |
  | `price_change` | 订阅者（已购买用户） | 策略价格变动 |

  **实现方式**：在对应的业务方法中（`Publish`、`Rate`、`Comment`、`SetPricing`、`RenewSubscriptions`）插入通知创建逻辑。用 goroutine 异步写通知，不阻塞主流程。

- [ ] **3.4c RPC 定义**

  ```protobuf
  message ListNotificationsRequest {
    int32 limit = 1;  // 默认 20
    int32 offset = 2;
  }
  message NotificationItem {
    string id = 1;
    string type = 2;
    string title = 3;
    string body = 4;
    string strategy_id = 5;
    bool is_read = 6;
    string created_at = 7;
  }
  message ListNotificationsResponse {
    repeated NotificationItem notifications = 1;
    int32 unread_count = 2;
  }

  // Add to MarketplaceService:
  // rpc ListNotifications(ListNotificationsRequest) returns (ListNotificationsResponse);
  // rpc MarkNotificationRead(MarkNotificationReadRequest) returns (google.protobuf.Empty);
  // rpc MarkAllNotificationsRead(MarkAllNotificationsReadRequest) returns (google.protobuf.Empty);
  ```

- [ ] **3.4d 前端通知 Bell**

  **文件**：`frontend/src/components/layout/NotificationBell.tsx` (new)

  **UI**：
  - 顶部导航栏 Bell 图标 + 未读红点数字
  - 下拉菜单：最新 5 条通知 + "查看全部"链接
  - 点击通知跳转到对应策略详情页
  - 新通知通过 SSE 实时推送（ConnectRPC server-stream，复用现有 SSE 管道，禁止轮询）

- **Gate 3.4**：`go build ./...` + `buf generate` + 前端构建通过。

---

## 模块 3.5 · 社交分享

- [ ] **3.5a 策略分享页 SEO**

  **文件**：`frontend/src/pages/share/StrategySharePage.tsx` (new)

  **关键实现**：
  - 服务端 prerender 生成策略详情的 OG 标签（`<meta property="og:title" content="EMA Crossover — 年化 35%">`）
  - OG image 动态生成（复用现有 `/og-image.svg` 模式——策略名称 + 关键指标叠加到 SVG 模板）
  - 分享落地页：非登录用户可见策略基本信息 + "注册/登录查看详情" CTA 按钮

- [ ] **3.5b 分享按钮**

  **文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`

  **分享渠道**：
  - 复制链接（`navigator.clipboard.writeText`）
  - Twitter（`https://twitter.com/intent/tweet?url=...&text=...`）
  - Telegram（`https://t.me/share/url?url=...&text=...`）

- **Gate 3.5**：前端构建通过 + 分享链接测试（OG 标签正确渲染）。

---

## Phase 3 完成检验

```bash
go build ./...
buf generate
go test ./internal/marketplace/...
go test ./internal/connect/marketplace/...
cd frontend && npm run build
bash scripts/gen_capability_map.sh
```

**关键验收场景**：
1. 排行榜切换类型/周期/资产类别 → 数据正确排序
2. 免费试用 → 7 天后自动过期 → 代码访问恢复限制
3. 策略对比 → 选 3 个策略 → 并排展示 → 最优值高亮
4. 通知触发 → Bell 红点 → 点击查看 → 标记已读
5. 分享链接 → 打开 → OG 标签正确 → 非登录用户看到 CTA
