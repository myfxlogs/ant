# 施工提示词 — FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION

> **开工指令：本文件落档，等待 Devin CLI 发开工指令后施工。**

## 立项背景

**触发**: 实盘端到端验证（账号 904d14e6 / schedule 898035e2）发现前端"订单日志"tab 永远空、Magic 列永远显示 `-`。

**证据链**:
1. `order_history` 表 0 行；`trade_records` 表 331 行（实际订单数据在这里）
2. `LogService.LogOrder`（`log_service.go:30`）全代码库无调用方 → `order_history` 是死表
3. `writeClosedTradeRecord`（`pipeline_callbacks.go:130-138`）构造 `model.TradeRecord` 时未设置 `MagicNumber` 和 `ScheduleID` → `trade_records.magic_number=0`、`schedule_id=NULL`
4. 前端 `logApi.getOrderHistory` → `LogRepository.GetOrderHistory` → 查死表 `order_history` → 永远空
5. 前端 Magic 列渲染 `trade.magicNumber ? trade.magicNumber : '-'`，magic=0 → 显示 `-`

**设计 SSOT**: `docs/spec/fix-2026-08-27-order-history-magic-attribution.md`（含自我审计 A1-A7）

## 约束与目标

- 修复 B 优先：补齐 `writeClosedTradeRecord` 的 `MagicNumber` + `ScheduleID`，立即修复数据
- 修复 A 次之：改 `GetOrderHistory` 查 `trade_records`，proto 加 `magic_number` 字段
- 修复 C 收尾：清理死代码
- hash chain 安全：`computeTradeEntryHash` 不含 magic_number/schedule_id，修复 B 不破坏 hash chain
- `trade_records` 有 `idx_trade_records_schedule_id` 和 `idx_trade_records_user_id` 索引

## 边界 / 不做

- 不删除 `order_history` 表（向后兼容）
- 不改 `SyncOrderHistory` 路径（`mthub_service_orders.go:82-106`，已正确设置 Magic + ScheduleID）
- 不改 `tradingApi.getOrderHistory`（trading.ts → MtHubServer.OrderHistory → broker 实时拉取，已有 magic）
- 不改前端 `AccountDetail.shared.tsx`（Magic 渲染逻辑正确，数据源修复后自动生效）
- 不改前端 `expandedRowColumns.tsx`（Magic 渲染逻辑正确）

---

## S1: 修复 B — `writeClosedTradeRecord` 补齐 Magic + ScheduleID

**目标**: 实时关闭订单写入 `trade_records` 时，补齐 `MagicNumber` 和 `ScheduleID` 字段。

**代码坐标**:

1. `cmd/server/pipeline.go:30-53` — `mdGatewayPipelineDeps` 结构体
   - 加字段: `scheduleResolver mthub.ScheduleResolver`

2. `cmd/server/main.go` — 构造 `mdGatewayPipelineDeps` 处
   - 注入: `scheduleResolver: repository.NewStrategyScheduleRepository(pool)`
   - 参考 `main.go:132` 已有 `accountSyncSvc.SetScheduleResolver(repository.NewStrategyScheduleRepository(pool))`

3. `cmd/server/pipeline.go:74` — `buildOnOrderUpdate` 调用处
   - 传入: `d.scheduleResolver`

4. `cmd/server/pipeline_callbacks.go:22-46` — `buildOnOrderUpdate` 函数签名
   - 加参数: `resolver mthub.ScheduleResolver`

5. `cmd/server/pipeline_callbacks.go:118-150` — `writeClosedTradeRecord` 函数
   - 加参数: `resolver mthub.ScheduleResolver`
   - 构造 `rec` 时补齐:
     ```go
     MagicNumber: int(o.UpdateMagic),
     ScheduleID:  mthub.ResolveScheduleID(ctx, resolver, log, uid, o.UpdateMagic),
     ```
   - import `alphaforge/internal/mthub`（如未有）

**落点**: `writeClosedTradeRecord` 写入的 `trade_records` 行将携带正确的 `magic_number` 和 `schedule_id`。

**对抗测试（先红后绿）**:
- RED: 在 `writeClosedTradeRecord` 故意省略 `MagicNumber` → 测试断言 `rec.MagicNumber == int(o.UpdateMagic)` 失败
- GREEN: 恢复 `MagicNumber: int(o.UpdateMagic)` → 测试通过
- 测试文件: `cmd/server/pipeline_callbacks_test.go`（新建）

---

## S2: 修复 A — 前端"订单日志"改查 `trade_records`

**目标**: `LogRepository.GetOrderHistory` 改查 `trade_records` 表，`OrderHistoryRecord` proto 新增 `magic_number` 字段。

**代码坐标**:

### S2.1: Proto 变更

1. `proto/ant/v1/log_order.proto` — `OrderHistoryRecord` message
   - 新增字段: `int64 magic_number = 13;`（field number 13，紧接 close_time=11、schedule_id=12 之后）
   - 注意：field 12 是 `schedule_id`，所以 magic_number 用 field 13

2. 重新生成 proto 代码:
   - `cd backend && go generate ./...` 或 `make proto`
   - 确认 `gen/proto/ant/v1/log_order.pb.go` 生成 `MagicNumber int64` 字段
   - 确认 `frontend/src/gen/ant/v1/log_order_pb.ts` 生成 `magicNumber` 字段

### S2.2: 后端查询改 `trade_records`

3. `internal/repository/order_history_repository.go:47-66` — `GetOrderHistory` 方法
   - 改查 `trade_records` 表
   - SELECT 字段映射:
     ```
     id, user_id, account_id, schedule_id, ticket, symbol, order_type, volume,
     open_price, close_price, profit, open_time, close_time, magic_number
     ```
   - WHERE: `user_id = $1` + 可选 `schedule_id`/`account_id`/`symbol`/`order_type`/`start_date`/`end_date`
   - ORDER BY: `open_time DESC`
   - 返回 `[]*model.OrderHistory`（需从 `trade_records` 行映射）

4. `internal/repository/order_history_repository.go:68-89` — `buildOrderHistoryFilters`
   - 改 base query: `FROM trade_records WHERE user_id = $1`
   - `schedule_id` 过滤: `AND schedule_id = $N`（注意 nullable，用 `IS NOT DISTINCT FROM` 或确认传入非 Nil）

5. `internal/connect/system/log_handler.go:53-71` — `orderHistoryToProto` 函数
   - 补齐 `MagicNumber: o.MagicNumber`（proto 新字段）
   - 字段映射:
     - `o.ID` → `Id`
     - `o.AccountID` → `AccountId`
     - `o.ScheduleID` → `ScheduleId`（需处理 `*uuid.UUID` → string）
     - `o.Ticket` → `Ticket`
     - `o.Symbol` → `Symbol`
     - `o.OrderType` → `OrderType`
     - `o.Volume` → `Lots`（`.String()`）
     - `o.OpenPrice` → `OpenPrice`（`.String()`）
     - `o.ClosePrice` → `ClosePrice`（`.String()`）
     - `o.Profit` → `Profit`（`.String()`）
     - `o.OpenTime` → `OpenTime`（`timestamppb.New`）
     - `o.CloseTime` → `CloseTime`（`timestamppb.New`，注意 `trade_records.close_time` not null）
     - `o.MagicNumber` → `MagicNumber`（新增）

6. `internal/model/logs.go:65-87` — `OrderHistory` struct
   - 确认 `MagicNumber` 字段类型为 `int64`（与 `trade_records.magic_number` bigint 一致）
   - 当前 `MagicNumber int64` — 已匹配

**落点**: `logApi.getOrderHistory` 返回 `trade_records` 数据，含 `magic_number`。

### S2.3: 前端列定义

7. `frontend/src/pages/strategy/scheduleLogColumns.tsx:102-118` — `buildOrderColumns`
   - 新增 Magic 列:
     ```tsx
     { title: 'Magic', dataIndex: 'magicNumber', key: 'magicNumber', width: 110, render: (v: unknown) => {
       const n = typeof v === 'number' ? v : Number(v);
       return n ? <Text type="secondary">{n}</Text> : <Text type="secondary">-</Text>;
     }},
     ```

**对抗测试（先红后绿）**:
- RED: `GetOrderHistory` 查 `order_history` → 测试断言返回 `trade_records` 数据失败
- GREEN: 改查 `trade_records` → 测试通过
- 测试文件: `internal/repository/order_history_repository_test.go`（新建或扩展）

---

## S3: 修复 C — 清理死代码

**目标**: 删除 `order_history` 表的写入和更新路径（全无调用方）。

**代码坐标**:

1. `internal/connect/system/mthub_service_orders.go:108-141` — 删除 `ClosedTradeParams` struct 和 `WriteClosedTrade` 方法

2. `internal/service/log_service.go:30-40` — 删除 `LogOrder` 和 `UpdateOrderHistoryClose` 方法

3. `internal/repository/order_history_repository.go:14-45` — 删除 `CreateOrderHistory` 和 `UpdateOrderHistoryClose` 方法

4. `internal/model/logs.go:65-87` — `OrderHistory` struct 保留（`GetOrderHistory` 仍用它做映射目标）

**落点**: `order_history` 表变为纯只读历史表（不再有写入路径）。

**验证**: `go build ./...` 通过，无未使用导入/方法警告。

---

## 验收

### 机检五件套
- [ ] `cd backend && go build ./...` 通过
- [ ] `cd backend && go vet ./...` 通过
- [ ] `cd backend && go test ./internal/repository/... ./internal/connect/system/... ./cmd/server/...` 通过
- [ ] `cd backend && go test -race ./internal/repository/... ./cmd/server/...` ×3 通过
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零警告

### 对抗测试（先红后绿）
- [ ] S1 RED: `writeClosedTradeRecord` 省略 `MagicNumber` → 测试失败
- [ ] S1 GREEN: 恢复 → 测试通过
- [ ] S2 RED: `GetOrderHistory` 查 `order_history` → 测试失败
- [ ] S2 GREEN: 改查 `trade_records` → 测试通过

### 运行时证据（部署后验证）
- [ ] `trade_records.magic_number` 非零（策略订单）
- [ ] `trade_records.schedule_id` 非空（策略订单）
- [ ] 前端"订单日志"tab 显示数据
- [ ] 前端 Magic 列显示数字（策略订单）

---

## 勿部署，停手等 Devin CLI 复审。禁 `--no-verify`。
