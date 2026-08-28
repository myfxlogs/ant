package system

import (
	"strings"
	"testing"
)

// TestDeadCodeRemoved_WriteClosedTrade verifies FIX-2026-08-27 修复 C:
// WriteClosedTrade and ClosedTradeParams (dead code, zero callers) are
// removed from mthub_service_orders.go.
// Adversarial proof: re-add WriteClosedTrade → this test turns RED
// (the symbol reappears in the file content).
func TestDeadCodeRemoved_WriteClosedTrade(t *testing.T) {
	files := map[string]string{
		"mthub_service_orders.go": mustReadFile(t, "internal/connect/system/mthub_service_orders.go"),
	}
	for name, content := range files {
		if strings.Contains(content, "func (s *MtHubServer) WriteClosedTrade") {
			t.Fatalf("%s: WriteClosedTrade should be removed, but found in file", name)
		}
		if strings.Contains(content, "type ClosedTradeParams struct") {
			t.Fatalf("%s: ClosedTradeParams should be removed, but found in file", name)
		}
	}
}

// TestDeadCodeRemoved_LogOrder verifies LogService.LogOrder is removed.
func TestDeadCodeRemoved_LogOrder(t *testing.T) {
	content := mustReadFile(t, "internal/service/log_service.go")
	if strings.Contains(content, "func (s *LogService) LogOrder(") {
		t.Fatal("log_service.go: LogService.LogOrder should be removed")
	}
}

// TestDeadCodeRemoved_UpdateOrderHistoryClose verifies both
// LogService.UpdateOrderHistoryClose and LogRepository.UpdateOrderHistoryClose
// are removed.
func TestDeadCodeRemoved_UpdateOrderHistoryClose(t *testing.T) {
	logSvc := mustReadFile(t, "internal/service/log_service.go")
	if strings.Contains(logSvc, "func (s *LogService) UpdateOrderHistoryClose") {
		t.Fatal("log_service.go: LogService.UpdateOrderHistoryClose should be removed")
	}
	logRepo := mustReadFile(t, "internal/repository/order_history_repository.go")
	if strings.Contains(logRepo, "func (r *LogRepository) UpdateOrderHistoryClose") {
		t.Fatal("order_history_repository.go: LogRepository.UpdateOrderHistoryClose should be removed")
	}
}

// TestDeadCodeRemoved_CreateOrderHistory verifies
// LogRepository.CreateOrderHistory is removed.
func TestDeadCodeRemoved_CreateOrderHistory(t *testing.T) {
	content := mustReadFile(t, "internal/repository/order_history_repository.go")
	if strings.Contains(content, "func (r *LogRepository) CreateOrderHistory") {
		t.Fatal("order_history_repository.go: LogRepository.CreateOrderHistory should be removed")
	}
}

// TestGetOrderHistoryStillPresent verifies the live method is NOT removed
// (regression guard — don't accidentally delete the wrong method).
func TestGetOrderHistoryStillPresent(t *testing.T) {
	logRepo := mustReadFile(t, "internal/repository/order_history_repository.go")
	if !strings.Contains(logRepo, "func (r *LogRepository) GetOrderHistory") {
		t.Fatal("order_history_repository.go: GetOrderHistory must still be present")
	}
	logSvc := mustReadFile(t, "internal/service/log_service.go")
	if !strings.Contains(logSvc, "func (s *LogService) GetOrderHistory") {
		t.Fatal("log_service.go: LogService.GetOrderHistory must still be present")
	}
}
