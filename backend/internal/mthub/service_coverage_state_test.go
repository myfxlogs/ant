package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
)

func TestOmsTransition_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.omsTransition(context.Background(), "ord-1", "acc-1", OMSStateNew, OMSStateValidated)
}

// --- DerivedState recalculate with data ---

func TestDerivedState_Recalculate_WithData(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.1), Price: dec(1.085),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	computer := NewDerivedComputer(cache, 1*time.Minute)
	computer.recalculate()
	ds := computer.State()
	acc := ds.GetAccount("acc-1")
	if acc == nil {
		t.Fatal("expected acc-1 derived state")
	}
	if !acc.Exposure.GreaterThan(decimal.Zero) {
		t.Fatal("expected positive exposure")
	}
}

// --- saturatedLimiter helper: creates a UserLimiter pre-saturated for a user ---

func TestIsTerminalOrderState(t *testing.T) {
	t.Parallel()
	terminals := []string{"CLOSED", "FILLED", "CANCELLED", "EXPIRED", "FAILED", "REJECTED"}
	for _, s := range terminals {
		if !isTerminalOrderState(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []string{"WORKING", "SUBMITTED", "NEW", "VALIDATED", "RISK_APPROVED"}
	for _, s := range nonTerminals {
		if isTerminalOrderState(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestCostToProto(t *testing.T) {
	t.Parallel()
	est := costsvc.CostBreakdown{
		SpreadCost:   dec(1.5),
		Commission:   dec(7),
		SlippageCost: dec(0.5),
		SwapCost:     dec(0),
		TotalCost:    dec(9),
	}
	p := costToProto(&est)
	if p.SpreadCost != "1.5" {
		t.Fatalf("expected 1.5, got %s", p.SpreadCost)
	}
}

// --- Broker publish/subscribe coverage ---

func TestStateCache_PositionIncrease_WeightedAvg(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Buy 0.1 at 1.0850
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.1), Price: dec(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos := c.GetPosition("acc-1", "EURUSD")
	if !pos.AvgPrice.Equal(dec(1.0850)) {
		t.Fatalf("expected avg price 1.0850, got %s", pos.AvgPrice)
	}

	// Buy 0.05 more at 1.0900 → position increases, weighted avg
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.05), Price: dec(1.0900),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos = c.GetPosition("acc-1", "EURUSD")
	// Weighted: (1.0850*0.1 + 1.0900*0.05) / 0.15 = (0.10850 + 0.05450) / 0.15 = 0.16300 / 0.15 = 1.08667...
	expected := dec(1.0850).Mul(dec(0.1)).Add(dec(1.0900).Mul(dec(0.05))).Div(dec(0.15))
	if !pos.AvgPrice.Equal(expected) {
		t.Fatalf("expected weighted avg %s, got %s", expected, pos.AvgPrice)
	}
}

// --- StateCache: short position increase weighted average ---

func TestStateCache_ShortPositionIncrease_WeightedAvg(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Sell 0.1 at 1.0850 → short
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "SELL", Volume: dec(0.1), Price: dec(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos := c.GetPosition("acc-1", "EURUSD")
	if !pos.NetVolume.Equal(dec(-0.1)) {
		t.Fatalf("expected -0.1, got %s", pos.NetVolume)
	}

	// Sell 0.05 more at 1.0900 → short increases
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "SELL", Volume: dec(0.05), Price: dec(1.0900),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos = c.GetPosition("acc-1", "EURUSD")
	if !pos.NetVolume.Equal(dec(-0.15)) {
		t.Fatalf("expected -0.15, got %s", pos.NetVolume)
	}
	expected := dec(1.0850).Mul(dec(0.1)).Add(dec(1.0900).Mul(dec(0.05))).Div(dec(0.15))
	if !pos.AvgPrice.Equal(expected) {
		t.Fatalf("expected weighted avg %s, got %s", expected, pos.AvgPrice)
	}
}

// --- PlaceOrder with guard and sell side ---

func TestIsValidOMSTransition(t *testing.T) {
	t.Parallel()
	valid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateValidated},
		{OMSStateValidated, OMSStateRiskApproved},
		{OMSStateValidated, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateSubmitted},
		{OMSStateRiskApproved, OMSStateFailed},
		{OMSStateSubmitted, OMSStateWorking},
		{OMSStateSubmitted, OMSStateFilled},
		{OMSStateSubmitted, OMSStateCancelled},
		{OMSStateWorking, OMSStatePartiallyFilled},
		{OMSStateWorking, OMSStateFilled},
		{OMSStateUnknown, OMSStateReconciling},
		{OMSStateReconciling, OMSStateWorking},
		{OMSStateRequoted, OMSStateRiskApproved},
		{OMSStateMarginCall, OMSStateCancelled},
	}
	for _, tr := range valid {
		if !isValidOMSTransition(tr.from, tr.to) {
			t.Errorf("%s → %s should be valid", tr.from, tr.to)
		}
	}
	invalid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateSubmitted},
		{OMSStateFilled, OMSStateWorking},
		{OMSStateCancelled, OMSStateWorking},
		{OMSStateRejected, OMSStateNew},
		{OMSStateExpired, OMSStateWorking},
	}
	for _, tr := range invalid {
		if isValidOMSTransition(tr.from, tr.to) {
			t.Errorf("%s → %s should be invalid", tr.from, tr.to)
		}
	}
}

// --- Pure function: hashToNegative ---

func TestHashToNegative_Coverage(t *testing.T) {
	t.Parallel()
	v := hashToNegative("test-order-id")
	if v >= 0 {
		t.Fatalf("expected negative value, got %d", v)
	}
	v2 := hashToNegative("test-order-id")
	if v != v2 {
		t.Fatal("hashToNegative should be deterministic")
	}
	v3 := hashToNegative("different-id")
	if v == v3 {
		t.Fatal("different inputs should produce different hashes")
	}
}

// --- Pure function: advisoryLockKey ---

func TestAdvisoryLockKey_Coverage(t *testing.T) {
	t.Parallel()
	k1, k2 := advisoryLockKey("acc-1", "client-1")
	_ = k1
	_ = k2
	k3, k4 := advisoryLockKey("acc-1", "client-1")
	if k1 != k3 || k2 != k4 {
		t.Fatal("advisoryLockKey should be deterministic")
	}
	k5, k6 := advisoryLockKey("acc-2", "client-1")
	if k1 == k5 && k2 == k6 {
		t.Fatal("different accountID should produce different keys")
	}
}

// --- PlaceOrder with kill switch ---
