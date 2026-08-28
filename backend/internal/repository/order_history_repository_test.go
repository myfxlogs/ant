package repository

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// TestBuildOrderHistoryFiltersQueriesTradeRecords verifies FIX-2026-08-27
// 修复 A: GetOrderHistory must query trade_records (the live write target
// with magic_number + schedule_id), not the dead order_history table.
// Adversarial proof: change baseQ back to "FROM order_history" → this test
// turns RED.
func TestBuildOrderHistoryFiltersQueriesTradeRecords(t *testing.T) {
	baseQ, _, _ := buildOrderHistoryFilters(uuid.New(), nil)
	if !strings.Contains(baseQ, "FROM trade_records") {
		t.Fatalf("base query must query trade_records, got: %s", baseQ)
	}
	if strings.Contains(baseQ, "order_history") {
		t.Fatalf("base query must NOT reference order_history, got: %s", baseQ)
	}
}

// TestBuildOrderHistoryFiltersScheduleID verifies the schedule_id filter is
// applied when provided.
func TestBuildOrderHistoryFiltersScheduleID(t *testing.T) {
	sid := uuid.New()
	params := &model.LogQueryParams{ScheduleID: sid.String()}
	baseQ, args, idx := buildOrderHistoryFilters(uuid.New(), params)
	if !strings.Contains(baseQ, "schedule_id = $") {
		t.Fatalf("expected schedule_id filter, got: %s", baseQ)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args (userID + scheduleID), got %d", len(args))
	}
	if args[1] != sid {
		t.Fatalf("schedule_id arg: expected %s, got %v", sid, args[1])
	}
	if idx != 3 {
		t.Fatalf("expected idx=3 after schedule_id filter, got %d", idx)
	}
}

// TestBuildOrderHistoryFiltersAllFilters verifies all supported filters.
func TestBuildOrderHistoryFiltersAllFilters(t *testing.T) {
	sid := uuid.New()
	aid := uuid.New()
	params := &model.LogQueryParams{
		ScheduleID: sid.String(),
		AccountID:  aid.String(),
		Symbol:     "EURUSD",
		Type:       "buy",
		StartDate:  "2026-01-01",
		EndDate:    "2026-12-31",
	}
	baseQ, args, idx := buildOrderHistoryFilters(uuid.New(), params)
	for _, expected := range []string{"schedule_id = $", "account_id = $", "symbol = $", "order_type = $", "open_time >= $", "open_time <= $"} {
		if !strings.Contains(baseQ, expected) {
			t.Fatalf("expected filter %q in query, got: %s", expected, baseQ)
		}
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
	if idx != 8 {
		t.Fatalf("expected idx=8 after all filters, got %d", idx)
	}
}
