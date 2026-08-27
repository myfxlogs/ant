# FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION

> 状态: 🟦open  
> 类型: 数据流断裂（死代码 + 字段丢弃）  
> 严重度: P0（业务线端到端验证阻断）  
> 发现于: 实盘端到端验证（账号 904d14e6 / schedule 898035e2）

## 1. 症状

| 症状 | 证据 |
|------|------|
| 前端"订单日志"tab 永远空 | `order_history` 表 0 行；`trade_records` 表 331 行 |
| 前端 Magic 列显示 `-` | `trade_records.magic_number` 全为 0 |
| `trade_records.schedule_id` 全为 NULL | 策略订单无法归属到 schedule |
| `LogService.LogOrder` 死代码 | 全代码库无调用方 |

## 2. 根因（两条独立断裂）

### 断裂 A: `order_history` 表是死表

**根因**: `LogService.LogOrder`（`internal/service/log_service.go:30`）是唯一能写 `order_history` 表的方法，但**全代码库无调用方**。

**数据流对比**:
- 前端"订单日志" → `GetOrderLogHistory` RPC → `LogRepository.GetOrderHistory` → 查 `order_history` 表（0 行）
- 实际订单数据 → `writeClosedTradeRecord` → 写 `trade_records` 表（331 行）
- 手动同步 → `SyncOrderHistory` → 写 `trade_records` 表

**结论**: `order_history` 表是历史遗留死表，实际数据在 `trade_records` 表。前端"订单日志"查的是死表。

### 断裂 B: `writeClosedTradeRecord` 丢弃 Magic + ScheduleID

**根因**: `cmd/server/pipeline_callbacks.go:130-138` 构造 `model.TradeRecord` 时**未设置 `MagicNumber` 和 `ScheduleID` 字段**，尽管 `o.UpdateMagic` 有值且 `mthub.ResolveScheduleID` 可从 magic 反查 schedule_id。

**证据**:
```go
// pipeline_callbacks.go:130-138 — 当前代码（断裂）
rec := &model.TradeRecord{
    UserID: userUUID, AccountID: uid, Ticket: o.UpdateTicket, Symbol: o.UpdateSymbol, OrderType: o.UpdateOrderType,
    Volume: o.UpdateVolume, OpenPrice: o.UpdateOpenPrice,
    ClosePrice: o.UpdateClosePrice, Profit: o.UpdateProfit,
    Swap: o.UpdateSwap, Commission: o.UpdateCommission,
    OpenTime: time.Unix(o.UpdateOpenTime, 0), CloseTime: time.Unix(o.UpdateCloseTime, 0),
    StopLoss: o.UpdateSL, TakeProfit: o.UpdateTP,
    OrderComment: o.UpdateComment, Platform: o.Platform,
    // ❌ 缺失: MagicNumber, ScheduleID
}
```

**对比**: `orderRecordToTradeRecord`（`mthub_service_orders.go:143-166`，SyncOrderHistory 路径）正确设置了这两个字段：
```go
MagicNumber:  int(r.Magic),                                    // ✅
ScheduleID:   mthub.ResolveScheduleID(ctx, resolver, log, accountID, r.Magic),  // ✅
```

**DB 证据**:
```
ticket   | magic_number | schedule_id | symbol  | order_type 
-----------+--------------+-------------+---------+------------
 362095819 |            0 |             | XAUUSDm | buy
 362095130 |            0 |             | XAUUSDm | buy
 ...（全部 0/NULL）
```

## 3. 影响范围

| 影响面 | 描述 |
|--------|------|
| 前端"订单日志"tab | 永远空（查死表 `order_history`） |
| 前端 Magic 列 | 永远显示 `-`（`trade_records.magic_number=0`） |
| 策略订单归属 | `trade_records.schedule_id=NULL`，无法按 schedule 筛选订单 |
| 市场战绩归因 | `marketplace/live_performance.go` 按 schedule_id 聚合战绩，NULL 被丢弃 |
| 分歧检测 | `divergence_handler.go` 按 schedule_id 关联，NULL 被跳过 |
| 账号详情页 | AccountDetail 历史交易 tab 的 Magic 列永远 `-` |

## 4. 修复方案

### 修复 A: 前端"订单日志"改查 `trade_records` 表

**方案**: 改 `LogRepository.GetOrderHistory` 查 `trade_records` 而非 `order_history`，映射到 `model.OrderHistory` 返回。

**代码坐标**:
- `internal/repository/order_history_repository.go:47-66` — `GetOrderHistory` 查询
- `internal/repository/order_history_repository.go:68-89` — `buildOrderHistoryFilters`
- `internal/connect/system/log_handler.go:53-71` — `orderHistoryToProto` 映射

**字段映射** (`trade_records` → `OrderHistoryRecord` proto):
| proto 字段 | trade_records 列 | 备注 |
|------------|------------------|------|
| id | id | uuid |
| account_id | account_id | uuid |
| schedule_id | schedule_id | uuid (nullable) |
| ticket | ticket | int64 |
| symbol | symbol | text |
| order_type | order_type | text |
| lots | volume | numeric → string |
| open_price | open_price | numeric → string |
| close_price | close_price | numeric → string |
| profit | profit | numeric → string |
| open_time | open_time | timestamp → timestamppb |
| close_time | close_time | timestamp → timestamppb |
| **magic_number** (新增) | magic_number | bigint → int64 (需 proto 加字段) |

**proto 变更**: `OrderHistoryRecord` 新增 `magic_number` 字段（field 13），前端列定义新增 Magic 列。

### 修复 B: `writeClosedTradeRecord` 补齐 Magic + ScheduleID

**方案**: 在 `writeClosedTradeRecord` 构造 `model.TradeRecord` 时补齐 `MagicNumber` 和 `ScheduleID`。

**代码坐标**: `cmd/server/pipeline_callbacks.go:118-150`

**变更**:
1. `buildOnOrderUpdate` 签名增加 `scheduleResolver mthub.ScheduleResolver` 参数
2. `writeClosedTradeRecord` 签名增加 `resolver mthub.ScheduleResolver`
3. 构造 `rec` 时补齐:
```go
MagicNumber: int(o.UpdateMagic),
ScheduleID:  mthub.ResolveScheduleID(ctx, resolver, log, uid, o.UpdateMagic),
```
4. `pipeline.go:74` 传入 resolver（从 `mthubSvc` 或 `accountSyncSvc` 获取）

**依赖**: `mthubSvc` 已有 `ScheduleResolver`（`main.go:132` 设置了 `accountSyncSvc.SetScheduleResolver`）。需确认 `mthubSvc` 是否暴露 resolver，或直接注入 `repository.StrategyScheduleRepository`。

### 修复 C: 清理死代码 `WriteClosedTrade` + `ClosedTradeParams`

**方案**: 删除 `mthub_service_orders.go:108-141` 的 `WriteClosedTrade` 和 `ClosedTradeParams`（全代码库无调用方）。

## 5. 验收标准

### 机检五件套
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./internal/repository/... ./internal/connect/system/... ./cmd/server/...` 通过
- [ ] `go test -race ./internal/repository/... ./cmd/server/...` ×3 通过
- [ ] `go run ./tools/check-file-lines --strict` 零警告

### 对抗测试（先红后绿）
- [ ] **RED**: 在 `writeClosedTradeRecord` 故意省略 `MagicNumber` → 测试断言 `rec.MagicNumber == 0` 失败
- [ ] **GREEN**: 恢复 `MagicNumber: int(o.UpdateMagic)` → 测试通过
- [ ] **RED**: `GetOrderHistory` 查 `order_history` → 测试断言返回 `trade_records` 数据失败
- [ ] **GREEN**: 改查 `trade_records` → 测试通过

### 运行时证据
- [ ] 部署后 `trade_records.magic_number` 非零（策略订单）
- [ ] `trade_records.schedule_id` 非空（策略订单）
- [ ] 前端"订单日志"tab 显示数据
- [ ] 前端 Magic 列显示数字（策略订单）

## 6. 不做

- 不删除 `order_history` 表（向后兼容，未来可能用于手动订单）
- 不改 `SyncOrderHistory` 路径（已正确设置 Magic + ScheduleID）
- 不改前端 `AccountDetail.shared.tsx`（Magic 渲染逻辑正确，数据源修复后自动生效）

## 7. 施工顺序

1. **修复 B**（`writeClosedTradeRecord` 补齐字段）— 阻断最小，立即修复数据
2. **修复 A**（前端"订单日志"改查 `trade_records`）— 需要 proto 变更 + 前端联调
3. **修复 C**（清理死代码）— 收尾

## 8. 自我审计发现（2026-08-27）

审计发现方案初版有 5 个遗漏/错误，已修正：

### A1: 两条 getOrderHistory 路径混淆（方案 A 范围确认）

前端有两个不同的 `getOrderHistory`：
- `tradingApi.getOrderHistory`（trading.ts）→ `MtHubServer.OrderHistory` → broker 实时拉取 → **有 magic** → AccountDetail/QuickTrade 用这个
- `logApi.getOrderHistory`（log.ts）→ `LogServer.GetOrderLogHistory` → DB 查 `order_history` 死表 → **无 magic** → ScheduleLogsModal/LogManagement 用这个

**修正**: 方案 A 只修 `logApi.getOrderHistory` 路径（LogService），正确。AccountDetail 的 Magic 列不在方案 A 范围内（它用 tradingApi，从 broker 实时拉取，已有 magic）。

### A2: LogManagement.tsx 也受影响（方案 A 影响面扩大）

`LogManagement.tsx:50-51` 的 `execution` 和 `orders` tab 都用 `logApi.getOrderHistory`。改后端 `GetOrderHistory` 查 `trade_records` 后，这两个 tab 也会显示 `trade_records` 数据。

**修正**: 这是期望行为（`order_history` 是死表，`trade_records` 是真实数据）。无需额外处理。

### A3: 方案 B 依赖确认 — mthubSvc 不暴露 ScheduleResolver

`*mthub.MtHubService` 没有 `SetScheduleResolver`。只有 `*system.MtHubServer` 和 `*service.AccountSyncService` 有。

**修正**: `buildOnOrderUpdate` 需要直接注入 `mthub.ScheduleResolver` 接口。`mdGatewayPipelineDeps` 有 `pool`，可在 `pipeline.go` 构造 `repository.NewStrategyScheduleRepository(d.pool)` 传入。或在 `mdGatewayPipelineDeps` 加 `scheduleResolver` 字段，由 `main.go` 注入。

**选择**: 在 `mdGatewayPipelineDeps` 加 `scheduleResolver mthub.ScheduleResolver` 字段，`main.go` 构造时注入（与 `accountSyncSvc.SetScheduleResolver` 一致）。

### A4: 方案 B hash chain 安全性确认

`computeTradeEntryHash`（`trade_record_repository.go:314`）只用 `prev_hash, seq, account_id, ticket, symbol, volume, open_price, close_price, profit, open_time, close_time`——**不包含 magic_number 和 schedule_id**。

**结论**: 修复 B 补齐 `MagicNumber` 和 `ScheduleID` 不会破坏 hash chain。历史数据的 hash chain 也不受影响（只影响新写入的行）。

### A5: 方案 C 范围扩大 — UpdateOrderHistoryClose 也是死代码

`LogService.UpdateOrderHistoryClose`（`log_service.go:35`）和 `LogRepository.UpdateOrderHistoryClose`（`order_history_repository.go:30`）全代码库无调用方。

**修正**: 方案 C 清理范围扩大为：
- `WriteClosedTrade` + `ClosedTradeParams`（mthub_service_orders.go:108-141）
- `LogService.LogOrder`（log_service.go:30-32）
- `LogService.UpdateOrderHistoryClose`（log_service.go:34-40）
- `LogRepository.CreateOrderHistory`（order_history_repository.go:14-27）
- `LogRepository.UpdateOrderHistoryClose`（order_history_repository.go:29-45）

### A6: 索引确认

`trade_records` 有 `idx_trade_records_schedule_id`（btree schedule_id）和 `idx_trade_records_user_id`（btree user_id）。按 `user_id + schedule_id` 过滤查询有索引支持。

### A7: 字段类型差异确认

| 字段 | order_history | trade_records | 影响 |
|------|---------------|---------------|------|
| close_time | timestamptz nullable | timestamp not null | 无（proto Timestamp 不区分 nullable） |
| is_auto_trade | boolean | 无 | 无（proto 无此字段） |
| magic_number | bigint | bigint | 需 proto 加字段 |

## 9. 相关文件

| 文件 | 角色 |
|------|------|
| `cmd/server/pipeline_callbacks.go:118-150` | 断裂 B 修复点 |
| `cmd/server/pipeline.go:74` | resolver 注入点 |
| `internal/repository/order_history_repository.go:47-89` | 断裂 A 修复点 |
| `internal/connect/system/log_handler.go:53-71` | proto 映射 |
| `internal/connect/system/mthub_service_orders.go:108-141` | 死代码清理点 |
| `proto/ant/v1/log_order.proto` | proto 字段新增 |
| `frontend/src/pages/strategy/scheduleLogColumns.tsx` | 前端 Magic 列新增 |
| `frontend/src/pages/accounts/components/AccountDetail.shared.tsx` | Magic 渲染（无需改） |
