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

// T1: TestScheduleHealthRepoQueriesTradeRecords — global guard.
// Verifies FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP: the file
// must query trade_records and must NOT query the dead order_history table.
// Adversarial proof: revert either SQL back to "FROM order_history" → RED.
func TestScheduleHealthRepoQueriesTradeRecords(t *testing.T) {
	content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
	if !strings.Contains(content, "FROM trade_records") {
		t.Fatal("schedule_health_repo.go must query trade_records")
	}
	if strings.Contains(content, "FROM order_history") {
		t.Fatal("schedule_health_repo.go must NOT query order_history (dead table)")
	}
}

// T2: TestGetLatestOrderProfitQueriesTradeRecords — precise assertion.
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

// T3: TestListOrdersQueriesTradeRecords — precise assertion.
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

// T4: TestGetScheduleStatsStillQueriesScheduleRunLogs — regression guard.
// Ensures the unrelated schedule_run_logs query was not accidentally changed.
func TestGetScheduleStatsStillQueriesScheduleRunLogs(t *testing.T) {
	content := mustReadFile(t, "internal/repository/schedule_health_repo.go")
	if !strings.Contains(content, "FROM schedule_run_logs") {
		t.Fatal("GetScheduleStats must still query schedule_run_logs (unrelated table)")
	}
}
