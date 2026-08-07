package strategy

import (
	"testing"

	"github.com/google/uuid"
)

func TestStrategyOrderClientID_Deterministic(t *testing.T) {
	t.Parallel()
	runID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	k1 := strategyOrderClientID(runID, 1700000000000, "buy")
	k2 := strategyOrderClientID(runID, 1700000000000, "buy")
	if k1 != k2 {
		t.Fatalf("same inputs should produce same ClientID: %s vs %s", k1, k2)
	}
}

func TestStrategyOrderClientID_DifferentRunID(t *testing.T) {
	t.Parallel()
	run1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	run2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	k1 := strategyOrderClientID(run1, 1700000000000, "buy")
	k2 := strategyOrderClientID(run2, 1700000000000, "buy")
	if k1 == k2 {
		t.Fatal("different runID should produce different ClientID")
	}
}

func TestStrategyOrderClientID_DifferentBarTime(t *testing.T) {
	t.Parallel()
	runID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	k1 := strategyOrderClientID(runID, 1700000000000, "buy")
	k2 := strategyOrderClientID(runID, 1700000060000, "buy")
	if k1 == k2 {
		t.Fatal("different bar time should produce different ClientID")
	}
}

func TestStrategyOrderClientID_DifferentSignalType(t *testing.T) {
	t.Parallel()
	runID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	k1 := strategyOrderClientID(runID, 1700000000000, "buy")
	k2 := strategyOrderClientID(runID, 1700000000000, "sell")
	if k1 == k2 {
		t.Fatal("different signal type should produce different ClientID")
	}
}

func TestStrategyOrderClientID_Format(t *testing.T) {
	t.Parallel()
	runID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	k := strategyOrderClientID(runID, 1700000000000, "buy_limit")
	expected := "strat-550e8400-e29b-41d4-a716-446655440000-1700000000000-buy_limit"
	if k != expected {
		t.Fatalf("ClientID format: want %s, got %s", expected, k)
	}
}
