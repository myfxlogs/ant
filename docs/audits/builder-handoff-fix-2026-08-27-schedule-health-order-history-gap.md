# 施工提示词 — FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP

> **设计 SSOT**: `docs/spec/fix-2026-08-27-schedule-health-order-history-gap.md`
> **base commit**: `c50212fe`
> **scope**: 单 commit，仅改 `schedule_health_repo.go` + 新建 1 测试文件
> **不做**: 不加 proto 字段、不改 scan、不删 order_history 表、不删冗余 SQL 条件

## 立项背景

**触发**: FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S3 验收时发现——S2 将 `LogRepository.GetOrderHistory` 改查 `trade_records`，但 `ScheduleHealthRepository` 的 2 个方法未同步，仍查死表 `order_history`（0 行）。

**证据链**:
- `schedule_health_repo.go:136` `GetLatestOrderProfit` → `FROM order_history`（0 行）
- `schedule_health_repo.go:172` `ListOrders` → `FROM order_history`（0 行）
- 被 `schedule_health_handler.go:96,138` 调用 → 前端 schedule health 面板 `latestOrderProfit`/`orders` 永远空

**约束**: 镜像 S2 改法（`FROM order_history` → `FROM trade_records`），scan 代码不变（schema 兼容性已在 spec §3 验证）。

## S1: 改查 trade_records（2 处）

**文件**: `backend/internal/repository/schedule_health_repo.go`

**S1a: `GetLatestOrderProfit`（:136）**

将 SQL 字符串中的 `FROM order_history` 改为 `FROM trade_records`：
```go
// before (:136)
"SELECT COALESCE(ticket, 0), COALESCE(profit, 0) FROM order_history WHERE user_id = $1 AND schedule_id = $2 AND close_time IS NOT NULL ORDER BY close_time DESC LIMIT 1"
// after
"SELECT COALESCE(ticket, 0), COALESCE(profit, 0) FROM trade_records WHERE user_id = $1 AND schedule_id = $2 AND close_time IS NOT NULL ORDER BY close_time DESC LIMIT 1"
```

**S1b: `ListOrders`（:172）**

将 SQL 字符串中的 `FROM order_history` 改为 `FROM trade_records`：
```go
// before (:172)
"SELECT id::text, ticket, order_type, symbol, profit, open_time, close_time FROM order_history WHERE user_id = $1 AND schedule_id = $2 ORDER BY COALESCE(close_time, open_time) DESC LIMIT $3"
// after
"SELECT id::text, ticket, order_type, symbol, profit, open_time, close_time FROM trade_records WHERE user_id = $1 AND schedule_id = $2 ORDER BY COALESCE(close_time, open_time) DESC LIMIT $3"
```

**不改**: scan 代码、`close_time IS NOT NULL`（冗余但无害）、`COALESCE(close_time, open_time)`（冗余但无害）、其他方法。

## T1-T4: 对抗证明

**新建文件**: `backend/internal/repository/schedule_health_repo_test.go`

**helper（内联定义，`internal/repository` 包无既有 `mustReadFile`）**:
```go
package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadFile(t *testing.T, relPath string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd is .../backend/internal/repository → backend root is ../..
	backendRoot := filepath.Join(wd, "..", "..")
	data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}
```

**注意**: `internal/repository` 的 wd 是 `.../backend/internal/repository`（比 `internal/connect/system` 少一层），所以 `backendRoot` 是 `../..` 而非 `../../..`。

**T1: `TestScheduleHealthRepoQueriesTradeRecords`**（全局守卫）
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

**T2: `TestGetLatestOrderProfitQueriesTradeRecords`**（精确断言）
```go
func TestGetLatestOrderProfitQueriesTradeRecords(t *testing.T) {
	content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
	idx := strings.Index(content, "func (r *ScheduleHealthRepository) GetLatestOrderProfit")
	if idx < 0 {
		t.Fatal("GetLatestOrderProfit not found")
	}
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
	if idx < 0 {
		t.Fatal("ListOrders not found")
	}
	methodBody := content[idx:]
	if !strings.Contains(methodBody, "FROM trade_records") {
		t.Fatal("ListOrders must query trade_records")
	}
}
```

**T4: `TestGetScheduleStatsStillQueriesScheduleRunLogs`**（回归守卫——确保 schedule_run_logs 查询未被误改）
```go
func TestGetScheduleStatsStillQueriesScheduleRunLogs(t *testing.T) {
	content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
	if !strings.Contains(content, "FROM schedule_run_logs") {
		t.Fatal("GetScheduleStats must still query schedule_run_logs (unrelated table)")
	}
}
```

## 对抗证明 RED→restore→GREEN

**RED**: 将 S1a/S1b 的 `FROM trade_records` 改回 `FROM order_history` → T1 RED（`must NOT query order_history`）+ T2 RED + T3 RED。
**restore**: 改回 `FROM trade_records` → 全 4 测试 GREEN。

## 验收门禁

```bash
cd backend
go build ./...
go vet ./...
go test ./internal/repository/... ./internal/connect/system/...
go test -race -count=3 ./internal/repository/...
go run ./tools/check-file-lines --strict   # 0 errors
git diff --check                             # clean
```

## 红队自审 A-F

- **A 架构**: 镜像 S2 改法，复用 trade_records 既有 schema + 索引，无逆向依赖
- **B 实现**: 纯字符串替换是最简方案，scan 代码不变
- **C 洁净**: check-lines 0 errors / 无死代码 / 无 TODO / 无调试残留
- **D 正确性**: T4 回归守卫确保 schedule_run_logs 未被误改；冗余 SQL 条件无害
- **E 合规**: 与 AGENTS.md §1 一致
- **F 文档**: registry/STATE/handover 同步

## 收工

更新 `docs/audits/tech-debt-registry.md`（`FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP` 条目状态）+ `docs/handoff/STATE.md` + `docs/audits/handover-audit-plan.md`（追加变更日志）。

**勿部署，停手等 Devin CLI 复审。**
