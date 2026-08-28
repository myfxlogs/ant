# FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP

> 状态: 🟦open
> 类型: 数据流断裂（死表查询残留）
> 严重度: P2（schedule health API 部分字段返回空）
> 发现于: FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S3 验收（2026-08-27）

## 1. 症状

| 症状 | 证据 |
|------|------|
| Schedule health API `latestOrderProfit` 永远空 | `schedule_health_repo.go:136` `GetLatestOrderProfit` 查 `order_history`（0 行） |
| Schedule health API `orders` 列表永远空 | `schedule_health_repo.go:172` `ListOrders` 查 `order_history`（0 行） |

**前端影响**: `GetScheduleHealth` RPC 返回的 `summary.latestOrderTicket`/`latestOrderProfit`/`hasLatestOrderProfit` 全为零值；`orders` 数组为空。用户在 schedule health 面板看不到该 schedule 的订单数据和最新盈亏。

## 2. 根因

S2 修复将 `LogRepository.GetOrderHistory` 改查 `trade_records`，但 `ScheduleHealthRepository` 的 2 个方法未被同步修复——仍查死表 `order_history`。

**数据流对比**:
- 实际订单数据 → `writeClosedTradeRecord` / `SyncOrderHistory` → 写 `trade_records` 表（331 行）
- schedule health `GetLatestOrderProfit` → 查 `order_history` 表（0 行）→ 返回空
- schedule health `ListOrders` → 查 `order_history` 表（0 行）→ 返回空

**调用链**:
```
前端 schedule health 面板
  → GetScheduleHealth RPC (schedule_health_handler.go:54)
    → buildHealthSummary (handler:88)
      → repo.GetLatestOrderProfit (handler:96) ← 查死表
    → fetchHealthOrders (handler:137)
      → repo.ListOrders (handler:138) ← 查死表
```

## 3. schema 对比（order_history vs trade_records）

| 字段 | order_history | trade_records | 兼容性 |
|------|---------------|---------------|--------|
| id | UUID | UUID | ✅ `id::text` cast 仍可用 |
| user_id | UUID NOT NULL | UUID NOT NULL (migration 135) | ✅ |
| schedule_id | UUID (nullable) | UUID NULL (migration 030) | ✅ |
| ticket | BIGINT | BIGINT | ✅ |
| order_type | VARCHAR(20) | VARCHAR(10) | ✅ scan 到 string |
| symbol | VARCHAR(50) | VARCHAR(20) | ✅ scan 到 string |
| profit | DECIMAL(20,8) | DECIMAL(18,4) | ✅ scan 到 decimal.Decimal |
| open_time | TIMESTAMP WITH TIME ZONE (nullable) | TIMESTAMP NOT NULL | ✅ scan 到 *time.Time |
| close_time | TIMESTAMP WITH TIME ZONE (nullable) | TIMESTAMP NOT NULL | ✅ scan 到 *time.Time |

**关键差异**: `trade_records.close_time` 是 `NOT NULL`，`order_history.close_time` 是 nullable。
- `GetLatestOrderProfit` 的 `close_time IS NOT NULL` 过滤条件变为**冗余但无害**（始终为 true）。
- `ListOrders` 的 `ORDER BY COALESCE(close_time, open_time)` 变为**冗余但无害**（close_time 始终非 NULL，COALESCE 总返回 close_time）。

**结论**: 两处 SELECT 只需改 `FROM order_history` → `FROM trade_records`，scan 代码无需变更。冗余条件保留以最小化 diff。

## 4. 修复方案

### S1: 改查 trade_records（2 处）

**文件**: `backend/internal/repository/schedule_health_repo.go`

**S1a: `GetLatestOrderProfit`（:136）**
```go
// before
"SELECT COALESCE(ticket, 0), COALESCE(profit, 0) FROM order_history WHERE user_id = $1 AND schedule_id = $2 AND close_time IS NOT NULL ORDER BY close_time DESC LIMIT 1"
// after
"SELECT COALESCE(ticket, 0), COALESCE(profit, 0) FROM trade_records WHERE user_id = $1 AND schedule_id = $2 AND close_time IS NOT NULL ORDER BY close_time DESC LIMIT 1"
```

**S1b: `ListOrders`（:172）**
```go
// before
"SELECT id::text, ticket, order_type, symbol, profit, open_time, close_time FROM order_history WHERE user_id = $1 AND schedule_id = $2 ORDER BY COALESCE(close_time, open_time) DESC LIMIT $3"
// after
"SELECT id::text, ticket, order_type, symbol, profit, open_time, close_time FROM trade_records WHERE user_id = $1 AND schedule_id = $2 ORDER BY COALESCE(close_time, open_time) DESC LIMIT $3"
```

### 不做（scope 排除）

- **不加 `magic_number` 到 `ScheduleHealthOrder` proto**——proto 当前无此字段，前端 schedule health 面板未展示 Magic 列。加字段 = scope creep。
- **不删冗余的 `close_time IS NOT NULL` / `COALESCE(close_time, open_time)`**——保留以最小化 diff，冗余条件无害。
- **不改 scan 代码**——`*time.Time` / `decimal.Decimal` / `int64` / `string` 类型均兼容。
- **不删 `order_history` 表**——S3 已删写入路径，表本身仍存在（migration 未回滚），删表是 DB migration 任务，超出本 fix scope。

## 5. 对抗证明

**测试文件**: `backend/internal/repository/schedule_health_repo_test.go`（新建）

**测试策略**: 文件内容扫描（镜像 S3 `dead_code_removal_test.go` 模式），因查询是 inline 字符串未提取为 builder。

**helper**: `internal/repository` 包无 `mustReadFile`（S3 的 helper 在 `internal/connect/system/`，不同包不可直接复用）。测试文件需内联定义本地 `mustReadFile`：
```go
func mustReadFile(t *testing.T, relPath string) string {
    t.Helper()
    wd, err := os.Getwd()
    if err != nil { t.Fatalf("getwd: %v", err) }
    // wd is .../backend/internal/repository → backend root is ../..
    backendRoot := filepath.Join(wd, "..", "..")
    data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
    if err != nil { t.Fatalf("read %s: %v", relPath, err) }
    return string(data)
}
```
**注意**: `internal/repository` 的 wd 是 `.../backend/internal/repository`（比 `internal/connect/system` 少一层），所以 `backendRoot` 是 `../..` 而非 `../../..`。

**T1: `TestScheduleHealthRepoQueriesTradeRecords`**
```go
func TestScheduleHealthRepoQueriesTradeRecords(t *testing.T) {
    content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
    if !strings.Contains(content, "FROM trade_records") {
        t.Fatal("schedule_health_repo.go must query trade_records")
    }
    if strings.Contains(content, "FROM order_history") {
        t.Fatal("schedule_health_repo.go must NOT query order_history (dead table)")
    }
}
```

**RED**: 将两处 `FROM trade_records` 改回 `FROM order_history` → T1 RED（`must NOT query order_history`）。
**restore**: 改回 `FROM trade_records` → GREEN。

**T2: `TestGetLatestOrderProfitQueriesTradeRecords`**（精确断言）
```go
func TestGetLatestOrderProfitQueriesTradeRecords(t *testing.T) {
    content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
    // 定位 GetLatestOrderProfit 方法体，断言含 trade_records
    idx := strings.Index(content, "func (r *ScheduleHealthRepository) GetLatestOrderProfit")
    if idx < 0 { t.Fatal("GetLatestOrderProfit not found") }
    methodBody := content[idx:]
    if !strings.Contains(methodBody, "FROM trade_records") {
        t.Fatal("GetLatestOrderProfit must query trade_records")
    }
}
```

**T3: `TestListOrdersQueriesTradeRecords`**（精确断言）
```go
func TestListOrdersQueriesTradeRecords(t *testing.T) {
    content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
    idx := strings.Index(content, "func (r *ScheduleHealthRepository) ListOrders")
    if idx < 0 { t.Fatal("ListOrders not found") }
    methodBody := content[idx:]
    if !strings.Contains(methodBody, "FROM trade_records") {
        t.Fatal("ListOrders must query trade_records")
    }
}
```

**T4: `TestGetScheduleStatsStillQueriesScheduleRunLogs`**（回归守卫）
```go
// 确保 GetScheduleStats 仍查 schedule_run_logs（不应被误改）
func TestGetScheduleStatsStillQueriesScheduleRunLogs(t *testing.T) {
    content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
    if !strings.Contains(content, "FROM schedule_run_logs") {
        t.Fatal("GetScheduleStats must still query schedule_run_logs")
    }
}
```

## 6. 验收门禁

- `go build ./...`
- `go vet ./...`
- `go test ./internal/repository/... ./internal/connect/system/...`
- `go test -race -count=3 ./internal/repository/...`
- `go run ./tools/check-file-lines --strict`（0 errors）
- `git diff --check` clean
- 对抗证明独立重跑：RED→restore→GREEN

## 7. 风险

- **TIMESTAMP without time zone vs WITH TIME ZONE**: `trade_records.close_time` 是 `TIMESTAMP`（without TZ），`order_history.close_time` 是 `TIMESTAMP WITH TIME ZONE`。pgx scan 到 `*time.Time` 时，without TZ 的值使用连接时区设置。S2 的 `GetOrderHistory` 已建立此模式（同样 scan trade_records.close_time 到 `*time.Time`），无运行时问题报告。风险可接受。
- **schedule_id NULL 行**: `trade_records.schedule_id` 可 NULL（migration 030）。S1 修复后 `GetLatestOrderProfit` 和 `ListOrders` 按 `schedule_id = $2` 过滤，NULL 行自动排除。无影响。
- **性能**: `trade_records` 有 `idx_trade_records_schedule_id`（migration 030）和 `idx_trade_records_user_id`（migration 135）。两处查询的 `WHERE user_id = $1 AND schedule_id = $2` 可命中索引。无性能退化。
