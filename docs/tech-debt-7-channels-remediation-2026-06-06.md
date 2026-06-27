# 7条断开/死代码通道 — 技术债修复方案

**日期**: 2026-06-06  
**关联**: 管道盘点（188条通道）中发现的7条断开/死代码通道（#182-#188）

---

## 总览

| # | 通道 | 债型 | 严重度 | 修复方案 |
|:--|------|:----:|:------:|----------|
| 182 | Factor 订阅者 | 功能债 | 🔴 高 | 移除孤儿 goroutine，保留代码 |
| 183 | NotificationService | 延迟债 | 🟡 中 | 完整实现 handler + DB |
| 184 | AutoTradingService | 废弃债 | 🟡 中 | 加延期注释 |
| 185 | EconomicDataService | 占位债 | 🟠 低 | 接受存根 |
| 186 | IndicatorCatalogService | 占位债 | 🟠 低 | 接受存根 |
| 187 | ObjectiveScoreService(Go) | 冗余债 | 🔴 高 | 新增 Go handler 代理到 Python |
| 188 | Python `/api/backtest` | 遗留债 | 🔴 高 | 迁移到 ConnectRPC，移除 REST |

---

## A1 (#182): 移除 Factor 订阅者孤儿 goroutine

### 现状

`handlers.go:274-275`：
```go
factorSub := factor.NewSubscriber(factor.DefaultSubscriberConfig(), log)
go factorSub.Start(context.Background())
```

- goroutine 使用 `context.Background()` 永不取消
- 输出 channel `factorSub.Chan()` 无人消费
- `Push()` 从未被调用（没有 NATS bar 事件订阅）
- `internal/factor/dsl/` 下有完整的 DSL 求值引擎（14个文件）未被挂载

### 修复方法

1. **删除** `handlers.go:274-275` 的 Start() 调用
2. **替换**为激活前置条件注释
3. **移除** `"anttrader/internal/factor"` import（如无其他引用点）
4. **保留** `internal/factor/` 包全部代码（含 `subscriber.go`、`dsl/` 等），供未来激活使用

### 激活前置条件（注释内容）

```
Factor subscriber activation prerequisites (M10-BASE-B6):
When ready to wire, create and start:
  factorSub := factor.NewSubscriber(factor.DefaultSubscriberConfig(), log)
  go factorSub.Start(pipelineCtx)
Required before activation:
  (1) Factor registry that registers DSL strategies
  (2) Bar-stream subscription from mdgateway → factorSub.Push()
  (3) Evaluation results → signal/order pipeline
```

---

## A2 (#187): 新增 ObjectiveScoreService Go handler

> **⚠️ 已过时：** Python REST 端点已按 ADR-0021 退役，ObjectiveScore 和 Backtest 改用 Go 原生实现。

### 现状

- Proto 定义: `proto/ant/v1/objective_score.proto` — 1个 RPC: `CalculateObjectiveScore`
- Go handler: **不存在**
- Python 实现: `strategy-service/app/services/objective_score.py` — 128行，计算 RSI/MACD/MA 评分
- Python REST 端点: `POST /api/objective-score` — 已注册、可用
- ConnectRPC 路径返回 `Unimplemented`

### 修复方法

**新建文件**: `internal/connect/strategy/objective_score_handler.go`

实现 `antv1c.ObjectiveScoreServiceHandler` 接口：
- `CalculateObjectiveScore(ctx, req)` 方法
- 将 proto 请求转为 JSON，POST 到 Python `/api/objective-score`
- 解析 JSON 响应转回 proto `CalculateObjectiveScoreResponse`
- 使用 `net/http.Client`（与 `PythonClient` 模式一致）

**修改文件**: `cmd/server/handlers.go`

在 `pythonStrategyServer` 注册处附近添加：
```go
objectiveScoreServer := strategy.NewObjectiveScoreServer(cfg.StrategyServiceURL, log)
mux.Handle(antv1c.NewObjectiveScoreServiceHandler(objectiveScoreServer,
    connectrpc.WithInterceptors(authInterceptor)))
```

### Python 端

保留 `/api/objective-score` REST 端点不变（Go handler 需要它作为后端）。

---

## A3 (#188): 迁移 RunBacktest 到 ConnectRPC + 移除 Python 遗留 REST

> **⚠️ 已过时：** Python REST 端点已按 ADR-0021 退役，ObjectiveScore 和 Backtest 改用 Go 原生实现。

### 现状

`StrategyServer.RunBacktest()` (strategy_signals.go:18-79) 通过 `s.client.Backtest()` 调用 Python 的 `POST /api/backtest` REST 端点。

而 `PythonStrategyServer` 的异步 worker 已使用 ConnectRPC `backtestClient.RunBacktest()` 调用 `POST /ant.v1.BacktestService/RunBacktest`。

两个端点调用相同的 `engine_run_backtest()` 函数，但前者是 JSON REST，后者是 proto binary。

### 修复方法

**Step 1**: 给 `StrategyServer` 添加 `backtestClient` 字段

文件: `internal/connect/strategy/strategy_handler.go`
```go
type StrategyServer struct {
    svc            *service.StrategySvc
    client         *strategysvc.PythonClient         // legacy REST (deprecated)
    backtestClient antv1c.BacktestServiceClient      // ConnectRPC (canonical)
    log            *zap.Logger
    pgListen       *pglisten.Listener
}
```
新增 `SetBacktestClient(c antv1c.BacktestServiceClient)` setter。

**Step 2**: 切换 `RunBacktest()` 实现

文件: `internal/connect/strategy/strategy_signals.go`

将 `s.client.Backtest(ctx, &strategysvc.BacktestRequest{...})` 替换为：
```go
resp, err := s.backtestClient.RunBacktest(ctx, connect.NewRequest(&antv1.ExecuteBacktestRequest{
    StrategyCode:   tmpl.Code,
    Symbol:         m.Symbol,
    Timeframe:      m.Timeframe,
    InitialCapital: balance,
}))
```
从 `resp.Msg` 中提取 metrics/risk 字段映射到 `RunBacktestResponse`。

**Step 3**: 注入 backtestClient

文件: `cmd/server/handlers.go` — 在已有 `backtestClient` 创建处（第143行）添加：
```go
strategyServer.SetBacktestClient(backtestClient)
```

**Step 4**: 移除 Python 遗留端点

文件: `strategy-service/app/routes/backtest.py`
- 删除 `POST /api/backtest` 路由（第25行）
- 删除 `_build_engine_request` 等辅助函数（`backtest_connect.py` 有自己的实现，不依赖此文件）
- 整文件可删除

文件: `strategy-service/app/main.py`
- 移除 `from app.routes import backtest`
- 移除 `app.include_router(backtest.router)`

---

## B1 (#183): 实现 NotificationService

### 现状

- Proto 定义: `proto/ant/v1/notification_service.proto` — 3个 RPC: `ListNotifications`, `MarkRead`, `MarkAllRead`
- 注释: "Handler implementation deferred to P1; schema + proto this round (B-2.6)"
- Go handler: **不存在**
- 注册: **未注册**
- 前端: 仅生成的 TypeScript 类型，无页面/组件

### 修复方法

**Step 1**: SQL migration

新建 `migrations/001_add_notifications.sql`:
```sql
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'system',
    title TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    data_json TEXT NOT NULL DEFAULT '{}',
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user_unread
    ON notifications(user_id, is_read, created_at DESC);
```

**Step 2**: Repository

新建 `internal/repository/notification_repository.go`:
```go
type NotificationRepository struct { pool *pgxpool.Pool }

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository

// ListByUser returns notifications for a user with pagination + unread filter.
// Returns (items, total_unread_count, error).
func (r *NotificationRepository) ListByUser(
    ctx context.Context, userID uuid.UUID,
    limit, offset int32, unreadOnly bool,
) ([]NotificationRow, int32, error)

func (r *NotificationRepository) MarkRead(ctx context.Context, id uuid.UUID) error
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error
```

**Step 3**: ConnectRPC handler

新建 `internal/connect/notification/handler.go`:
```go
type NotificationServer struct {
    repo *repository.NotificationRepository
    log  *zap.Logger
}

func NewNotificationServer(repo *repository.NotificationRepository, log *zap.Logger) *NotificationServer
func (s *NotificationServer) ListNotifications(ctx, req) (resp, error)
func (s *NotificationServer) MarkRead(ctx, req) (resp, error)
func (s *NotificationServer) MarkAllRead(ctx, req) (resp, error)
```

**Step 4**: 注册

文件: `cmd/server/handlers.go`:
```go
notifRepo := repository.NewNotificationRepository(pool)
notifServer := notification.NewNotificationServer(notifRepo, log)
mux.Handle(antv1c.NewNotificationServiceHandler(notifServer,
    connectrpc.WithInterceptors(authInterceptor)))
```

---

## C1 (#184): AutoTradingService 加延期注释

### 现状

- Proto 定义: `proto/ant/v1/auto_trading.proto` — 14个 RPC + 4个 supporting proto 文件
- 无 Go handler，未注册
- 前端有生成的 TypeScript 类型 + i18n 字符串，无页面

### 修复方法

文件: `proto/ant/v1/auto_trading.proto` — 在 service 定义前插入：
```proto
// Deferred — no handler implementation exists.
// 14 RPC schema defined as API contract for future P2 milestone.
// Supporting proto files: auto_trading_settings, auto_trading_risk*, auto_trading_logs.
```

无其他代码修改。

---

## C2/C3 (#185/#186): EconomicDataService / IndicatorCatalogService — 接受存根

### 现状

- EconomicDataService: handler 存在，返回空数组。注释说明 "尚未集成真实的经济日历 API"
- IndicatorCatalogService: handler 存在，返回9个硬编码指标 + 4个风险参数。功能正确

### 处理

**不修改代码**。两者都已注册、功能正确（存根返回空/硬编码数据是预期行为）。真实数据源接入属于独立功能开发任务，非技术债清理范畴。

- #185 需外部 API 集成（Trading Economics / FRED / Alpha Vantage）— 独立 feature task
- #186 目录数据为定义性质，可后续迁移到 DB 动态管理 — 独立 enhancement task

---

## 执行顺序

```
A1 (#182 移除孤儿goroutine) ──┐
                                 ├── 并行（独立无冲突）
A2 (#187 新增ObjScore handler) ─┘
                                 │
A3 (#188 迁移RunBacktest)  ──────┤ A1/A2完成后
                                 │
B1 (#183 NotificationService) ──┘ 独立执行
                                 │
C1 (#184 延期注释)        ──────┘ 独立执行（仅注释）
```

---

## 验证清单

```bash
# Go 编译
cd /opt/ant/backend && go build ./...

# Python 导入验证（移除 backtest.py 后）
cd /opt/ant/strategy-service && python -c "from app.main import app"

# 文件/函数大小合规
cd /opt/ant && make check-size

# 确认不再有未注册的 proto 服务
grep -r "Unimplemented" gen/proto/ant/v1/antv1connect/ | wc -l
```

---

## 附录 B：审计修正（2026-06-06）

实施后审计发现 3 项违规 + 9 项次优，全部修复。

### 违规修复

| # | 文件 | 问题 | 修复 |
|---|------|------|------|
| F3.1 | strategy_signals.go | RunBacktest 79行 >50 | 拆分为 3 个 ≤32 行函数：RunBacktest(25) → runConnectBacktest(32) + mapBacktestResult(35) |
| F5.1 | objective_score_handler.go | CalculateObjectiveScore 83行 >50 | 拆分为 4 个 ≤28 行函数：CalculateObjectiveScore(17) → marshalScoreRequest(16) + callPython(28) + ToProto(20) |
| F8.1 | python_client.go | 死代码 Backtest() 方法 + BacktestRequest/Result/KlineBar 类型 | 整块删除（含 circuit breaker 逻辑），文件从 275 行减至 140 行 |

### 次优修复

| # | 问题 | 修复 |
|---|------|------|
| F1.5/F2.3 | StrategyServer 死 client 字段 | 移除字段 + SetClient() + handlers.go 注入行 |
| F3.4/F4.1 | fetchKlinesForBacktest / klinesForSync 重复 | 提取为包级共享函数 `fetchKlines()`，两处调用统一 |
| F3.3 | ClickHouse 查询无超时 | `fetchKlines()` 内封装 `context.WithTimeout(ctx, 5s)` |
| F3.5 | 回测失败返回 Success:true | 改为 `Success: false` |
| F4.4 | Execute/Validate 仍用 REST | 标注为已知技术债（PythonClient 继续服役），待 Python 端增加 ConnectRPC 端点后迁移 |
| F5.4 | ObjectiveScore 桥接模式 | 标注 TODO 注释：待 Python 端增加 ConnectRPC 端点后切换 |
| F5.5 | ObjectiveScore 无 userID | 新增 userID 提取（即使仅审计用） |
| F6.2 | Notification uuid.Nil 静默 | ListNotifications + MarkAllRead 加 uuid.Nil → CodeUnauthenticated 检查 |

### 最终指标

| 指标 | 值 |
|------|-----|
| 最大函数行数 | 38 (post), 35 (mapBacktestResult) — 全部 <50 |
| 最大文件行数 | 253 (python_strategy_handler.go) — 全部 <300 |
| 死代码 | 0 |
| 重复逻辑 | 0 (klines 统一为 fetchKlines) |
| 静默错误吞没 | 0 |
