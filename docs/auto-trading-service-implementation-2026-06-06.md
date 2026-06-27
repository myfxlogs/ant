# AutoTradingService 实施方案

**日期**: 2026-06-06  
**状态**: 已审计 → 实施中  
**关联**: C1 (#184) 技术债修复 — AutoTradingService 从延期到实施

---

## 1. 背景

`AutoTradingService` 是 ant 平台「策略→自动执行」闭环的核心服务。Proto 定义完整（14 RPC / 5 领域），DB 表已建，Repository 已备，**仅缺 ConnectRPC handler + 注册**。

### 现有资产

| 层级 | 路径 | 状态 |
|------|------|:----:|
| Proto | `proto/ant/v1/auto_trading.proto` + 4 个 supporting proto | ✅ |
| Migration | `migrations/010_auto_trading.up.sql` (5 张表) | ✅ |
| Model | `internal/model/auto_trading.go` (GlobalSettings, RiskConfig, TradingLog, StrategySchedule, etc.) | ✅ |
| Repository | `internal/repository/auto_trading_repository.go` + `auto_trading_settings.go` + `auto_trading_risk.go` | ✅ |
| Repository | `internal/repository/strategy_schedule_repository.go` | ✅ |
| **Handler** | — | ❌ 缺失（6 文件，~600 行） |
| **注册** | `cmd/server/handlers.go` | ❌ 缺失 |
| **Migration** | `011_global_settings_add_risk_columns.up.sql` | ❌ 待建（修复表缺列） |

---

## 2. 14 RPC 分域

### 2.1 全局设置 (3 RPC) — 文件: `auto_trade_settings_handler.go`

| RPC | 输入 | 输出 | 实现 |
|-----|------|------|------|
| `GetGlobalSettings` | (empty) | `GlobalSettings` | `repo.GetGlobalSettingsByUserID(ctx, uid)` → proto |
| `UpdateGlobalSettings` | `UpdateGlobalSettingsRequest` | `GlobalSettings` | 查现有→合并 optional 字段→`repo.UpdateGlobalSettings(ctx)` |
| `ToggleAutoTrade` | `user_id, enabled` | `success+message` | `repo.UpdateAutoTradeEnabled(ctx, uid, enabled)` |

**约 90 行**

### 2.2 风控 (4 RPC) — 文件: `auto_trade_risk_handler.go`

| RPC | 输入 | 输出 | 实现 |
|-----|------|------|------|
| `GetRiskConfig` | `user_id, account_id` | `RiskConfig` | `repo.GetRiskConfigByAccountID(ctx, aid)` → proto |
| `UpdateRiskConfig` | `UpdateRiskConfigRequest` | `RiskConfig` | 查现有→合并 optional→`repo.UpdateRiskConfig(ctx)` |
| `CheckRiskLimits` | `CheckRiskLimitsRequest` | `CheckRiskLimitsResponse` | 查 risk config → 比较持仓/亏损/回撤 → 返回 allowed + reason |
| `CalculatePositionSize` | `account_id, symbol, balance, sl_pips, risk%` | `volume, risk_amount, pip_value` | 标准仓位计算公式: `risk_amount = balance * risk%`, `volume = risk_amount / (sl_pips * pip_value)` |

**约 160 行**

### 2.3 策略调度 (5 RPC) — 文件: `auto_trade_schedule_handler.go`

**⚠️ 审计发现：StrategyService（strategy.proto）已有完整 Schedule CRUD 实现。这 5 个 RPC 是重复定义。**

| RPC | 实现 |
|-----|------|
| CreateSchedule / GetSchedule / UpdateSchedule / DeleteSchedule / ToggleSchedule | 返回 `CodeUnimplemented` + 注释引导调用 `StrategyService` |

**约 60 行**

### 2.4 状态+日志 (3 RPC) — 文件: `auto_trade_status_handler.go`

| RPC | 输入 | 输出 | 实现 |
|-----|------|------|------|
| `GetAutoTradingStatus` | (empty) | `AutoTradingStatus` | 查 global_settings + 统计活跃 schedules + 今日成交 |
| `GetTradingLogs` | `user_id, page, page_size, log_type, dates` | `logs + total` | `repo.GetTradingLogs(ctx, uid, params)` → proto |
| `GetRecentTradingLogs` | `user_id, limit` | `logs` | `repo.GetRecentTradingLogs(ctx, uid, limit)` → proto |

**约 110 行**

---

## 3. 架构设计

### 3.1 文件组织

```
internal/connect/autotrading/
├── server.go                        # AutoTradingServer struct + constructor (~30行)
├── auto_trade_settings_handler.go   # 3 RPC (~90行)
├── auto_trade_risk_handler.go       # 4 RPC (~160行)
├── auto_trade_schedule_handler.go   # 5 RPC (~180行)
├── auto_trade_status_handler.go     # 3 RPC (~110行)
└── auto_trade_converters.go         # proto↔model 转换函数 (~120行)
```

总计 ~690 行，拆分为 6 个文件，全部 ≤180 行。

### 3.2 Server 结构体

```go
type AutoTradingServer struct {
    autoRepo   *repository.AutoTradingRepository
    riskPipe   *risksvc.SignalPipeline    // 注入现有风控管线 (CheckRiskLimits)
    log        *zap.Logger
}
```

- `riskPipe` 复用 `handlers.go:229` 已创建的 `risksvc.SignalPipeline`（6 阶段：Capability→HardLimit→Platform→Engine→Sizer→Allocator）
- 实现 `antv1c.AutoTradingServiceHandler` 接口
- 编译期检查: `var _ antv1c.AutoTradingServiceHandler = (*AutoTradingServer)(nil)`

### 3.3 注册

```go
// handlers.go — 在 pipeline 创建后注册（pipeline 已在 line 229 创建）
autoTradingRepo := repository.NewAutoTradingRepository(pool)
autoTradingServer := autotrading.NewAutoTradingServer(autoTradingRepo, pipeline, log)
mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
    connectrpc.WithInterceptors(authInterceptor)))
```

---

## 4. 数据转换

### 4.1 Proto → Model (写入)

| Proto field | Model field | 转换 |
|-------------|-------------|------|
| `string user_id` | `uuid.UUID UserID` | `uuid.Parse()` |
| `string account_id` | `uuid.UUID AccountID` | `uuid.Parse()` |
| `double` 类型 | `float64` | 直接赋值 |
| `int32` 类型 | `int` | `int(v)` |
| `optional` 字段 | 检查 `req.Msg.HasXxx()` | 仅非 nil 时覆盖 |

### 4.2 Model → Proto (读取)

| Model field | Proto field | 转换 |
|-------------|-------------|------|
| `uuid.UUID` | `string` | `.String()` |
| `float64` | `double` | 直接赋值 |
| `decimal.Decimal` | `double` | `d.InexactFloat64()` — 注意：仅用于显示，非计算路径 |
| `time.Time` | `google.protobuf.Timestamp` | `timestamppb.New(t)` |
| `*time.Time` (nilable) | `Timestamp` | if nil → nil; else `timestamppb.New(*t)` |

### 4.3 关键约束

- ⚠️ `RiskConfig.MaxDailyLoss` 和 `DailyLossUsed` 在 model 中是 `decimal.Decimal`，proto 中是 `double`
- ⚠️ 转换时用 `InexactFloat64()` **仅限响应**（非计算路径），符合 "display/transport OK" 规则
- ✅ 所有价格计算发生在 Go 引擎侧，Go 侧仅透传

---

## 5. 实施步骤

| # | 步骤 | 文件 | 预计行数 |
|---|------|------|----------|
| 1 | 创建 `server.go` | 结构体+构造函数 | ~30 |
| 2 | 创建 `auto_trade_converters.go` | proto↔model 转换 | ~120 |
| 3 | 创建 `auto_trade_settings_handler.go` | 3 RPC | ~90 |
| 4 | 创建 `auto_trade_risk_handler.go` | 4 RPC | ~160 |
| 5 | 创建 `auto_trade_schedule_handler.go` | 5 RPC | ~180 |
| 6 | 创建 `auto_trade_status_handler.go` | 3 RPC | ~110 |
| 7 | 修改 `handlers.go` | import + 注册 | +6 |
| 8 | 构建验证 | `go build ./...` | — |

---

## 6. 不在此次范围

| 项目 | 原因 |
|------|------|
| 调度执行引擎 (cron/interval runner) | StrategyService 已有完整调度 CRUD + Watch；AutoTradingService 5 RPC 返回 Unimplemented |
| 前端 UI 页面 | 后端 API 先行，前端独立迭代 |
| TradingLog 表扩展（加 strategy_id/details 列） | 当前 trade_logs 表字段充足；proto TradingLog 的 strategy_id/details 映射到已有字段 |
| Proto 中 `double` → model `decimal.Decimal` 改造 | 改 proto 字段类型是 breaking change；当前 transport 层 double 可接受 |

---

## 7. 验证

```bash
cd /opt/ant/backend && go build ./...
# 确认 14 个 RPC 注册成功 — 不再返回 Unimplemented
# 文件大小: 每个 ≤180 行, 函数 ≤30 行
```

---

## 8. 最优解确认

| 决策 | 方案 | 确认依据 |
|------|------|----------|
| 文件拆分 | 按 RPC 域拆 6 文件，单一 `AutoTradingServer` | 匹配 `StrategyServer` 模式（strategy_handler.go + 多个 domain 文件） |
| 复用 Repository | `AutoTradingRepository` + `StrategyScheduleRepository` | 已存在完整 CRUD，零额外开发 |
| CheckRiskLimits | 接入现有 `risksvc.SignalPipeline`（6 阶段） | 优于静态查表；pipeline 已在 handlers.go 创建并注入 mthubSvc |
| decimal → proto | `InexactFloat64()` 仅响应传输层 | 全代码库 30+ 处使用一致模式（admin/system 等） |
| userID 提取 | helper 方法返回 `uuid.UUID`，Nil 哨兵 → CodeUnauthenticated | 匹配 NotificationServer + StrategyServer 模式 |
| 转换函数 | 独立 `auto_trade_converters.go`，包级函数 | 匹配 `strategy_converters.go` + `python_strategy_converters.go` 模式 |
| Schedule RPC | 返回 Unimplemented + 引导注释 | StrategyService 已有完整实现，避免双重写入同一表 |
| 错误代码 | InvalidArgument / NotFound / Internal / Unauthenticated | 匹配全代码库一致规范 |

---

## 附录 A：审计修正（2026-06-06）

审计发现 **1 项致命缺陷 + 2 项架构冲突 + 3 项数据不匹配**。全部纳入修正。

### A.1 🔴 致命：global_settings 表缺列

**问题**：`global_settings` 表（010 迁移）只有 8 列（id, user_id, auto_trade_enabled, notification_enabled, email_notification, sms_notification, created_at, updated_at），但 `model.GlobalSettings` 和 `AutoTradingRepository` 读写 `max_risk_percent`、`max_positions`、`max_lot_size`、`max_daily_loss`、`max_drawdown_percent` — 全部不存在。任何 `GetGlobalSettings`/`UpdateGlobalSettings` 调用立即崩溃。

**修正**：新增 migration `011_global_settings_add_risk_columns.up.sql`：

```sql
ALTER TABLE global_settings
  ADD COLUMN IF NOT EXISTS max_risk_percent DOUBLE PRECISION DEFAULT 2.0,
  ADD COLUMN IF NOT EXISTS max_positions INTEGER DEFAULT 10,
  ADD COLUMN IF NOT EXISTS max_lot_size DOUBLE PRECISION DEFAULT 100.0,
  ADD COLUMN IF NOT EXISTS max_daily_loss DECIMAL(18,2) DEFAULT 5000.00,
  ADD COLUMN IF NOT EXISTS max_drawdown_percent DOUBLE PRECISION DEFAULT 10.0;
```

**备选分析**：是否应删除 risk 字段使其成为纯开关表？评估：proto `GlobalSettings` 已定义这些字段为 API contract；`RiskConfig` 是**按账户**配置，`GlobalSettings` 是**用户级默认值**。语义不同，保留分离。

### A.2 🔴 架构冲突：Schedule RPC 与 StrategyService 重复

**问题**：`StrategyService`（strategy.proto）已有 7 个 Schedule RPC（CreateSchedule, GetSchedule, UpdateSchedule, DeleteSchedule, ToggleSchedule, ListSchedules, WatchSchedules），由 `StrategyServer` 完整实现。`AutoTradingService` 的 5 个 Schedule RPC 是完全重复。

**修正**：从 AutoTradingService 实施范围**移除全部 5 个 Schedule RPC**。`auto_trade.proto` 中保留 proto 定义（向后兼容），但 handler 返回 `CodeUnimplemented` + 注释引导调用方使用 `StrategyService`。

实施范围缩小为 **9 RPC**（原 14 - 5 调度）。

### A.3 🟡 模型不匹配

| 问题 | 修正 |
|------|------|
| TradingLog.StrategyID/Details 不在 trade_logs 表中 | 新建 migration 加列，或从 TradingLog proto 映射中省略这两个字段。**选择后者** — proto TradingLog 的 strategy_id/details 映射到 trade_logs 的已有等效字段（action/message），不扩展表 |
| GlobalSettings proto 缺 notification_enabled 等字段 | **不修改 proto** — UpdateGlobalSettings proto 无这些字段属有意设计。仅 GetGlobalSettings 响应包含它们，从 model 映射时加入 |
| StrategySchedule.parameters 是 proto `map<string,string>` vs model `JSONB` | 不再需要处理（Schedule RPC 已移除） |

### A.4 🟢 增强：CheckRiskLimits 接入现有 SignalPipeline

**修正**：给 `AutoTradingServer` 注入 `*risksvc.SignalPipeline`（已在 handlers.go:229 创建）。`CheckRiskLimits` 构建 `SignalRequest` → 调用 `pipeline.Process()` → 映射 `SignalResult` 到响应。优于原方案中的"静态查 risk_configs 表"。

### A.5 🟢 补全：错误处理 + userID 模式

**userID 模式**（匹配 NotificationServer/StrategyServer）：
```go
func (s *AutoTradingServer) userID(ctx context.Context) uuid.UUID {
    raw := interceptor.GetUserID(ctx)
    if raw == "" {
        s.log.Warn("autotrading: userID not in context")
        return uuid.Nil
    }
    id, err := uuid.Parse(raw)
    if err != nil {
        s.log.Warn("autotrading: userID parse failed", zap.String("raw", raw), zap.Error(err))
        return uuid.Nil
    }
    return id
}
```

**错误代码规范**：
- 输入验证/UUID 解析失败 → `CodeInvalidArgument`
- 实体未找到 → `CodeNotFound`
- 存储库错误 → `CodeInternal`（前记录 `s.log.Error(...)`）
- userID == uuid.Nil → `CodeUnauthenticated`

### A.6 修正后实施范围

| 文件 | RPC | 预计行数 |
|------|-----|----------|
| `server.go` | 结构体+构造函数 | ~35 |
| `auto_trade_converters.go` | proto↔model 转换 | ~100 |
| `auto_trade_settings_handler.go` | GetGlobalSettings, UpdateGlobalSettings, ToggleAutoTrade (3 RPC) | ~100 |
| `auto_trade_risk_handler.go` | GetRiskConfig, UpdateRiskConfig, CheckRiskLimits, CalculatePositionSize (4 RPC) | ~180 |
| `auto_trade_status_handler.go` | GetAutoTradingStatus, GetTradingLogs, GetRecentTradingLogs (3 RPC) | ~120 |
| `auto_trade_schedule_handler.go` | 5 RPC 返回 Unimplemented + 引导注释 | ~60 |

新增 migration + handlers.go 注册。

总计 **10 RPC 真实实现 + 5 RPC Unimplemented 桩**，约 600 行，6 文件。
