package mthub

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

func TestSideToString(t *testing.T) {
	t.Parallel()
	if sideToString(SideBuy) != "buy" {
		t.Fatal("expected buy")
	}
	if sideToString(SideSell) != "sell" {
		t.Fatal("expected sell")
	}
	if sideToString(Side(0)) != "unknown" {
		t.Fatal("expected unknown")
	}
}

func TestOrderTypeToString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input    OrderType
		expected string
	}{
		{OrderMarket, "market"},
		{OrderLimit, "limit"},
		{OrderStop, "stop"},
		{OrderStopLimit, "stop_limit"},
		{OrderType(99), "unknown"},
	}
	for _, tc := range cases {
		if got := orderTypeToString(tc.input); got != tc.expected {
			t.Errorf("orderTypeToString(%v) = %s, want %s", tc.input, got, tc.expected)
		}
	}
}

func TestOrderRequestToIntent(t *testing.T) {
	t.Parallel()
	req := &OrderRequest{
		AccountID:  "acc-1",
		Canonical:  "EURUSD",
		Side:       SideBuy,
		OrderType:  OrderMarket,
		Volume:     dec(0.1),
		Price:      dec(1.085),
		StopLoss:   dec(1.08),
		TakeProfit: dec(1.09),
		Magic:      123,
	}
	intent := orderRequestToIntent(req)
	if intent.AccountId != "acc-1" {
		t.Fatalf("expected acc-1, got %s", intent.AccountId)
	}
	if intent.Symbol != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", intent.Symbol)
	}
	if intent.Side != "buy" {
		t.Fatalf("expected buy, got %s", intent.Side)
	}
	if intent.Type != "market" {
		t.Fatalf("expected market, got %s", intent.Type)
	}
	if intent.Volume != "0.1" {
		t.Fatalf("expected 0.1, got %s", intent.Volume)
	}
}

func TestNewHubCostEstimator(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	def := &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: decimal.Zero}
	e := NewHubCostEstimator(hub, def, nil)
	if e == nil {
		t.Fatal("expected non-nil estimator")
	}
}

func TestNewHubCostEstimator_NilDefault(t *testing.T) {
	t.Parallel()
	e := NewHubCostEstimator(NewHub(), nil, nil)
	if e == nil {
		t.Fatal("expected non-nil estimator")
	}
}

func TestHubCostEstimator_Refresh(t *testing.T) {
	t.Parallel()
	e := NewHubCostEstimator(NewHub(), nil, nil)
	e.Refresh("EURUSD") // should not panic
}

func TestNewOmsWriter(t *testing.T) {
	t.Parallel()
	w := NewOmsWriter(nil, nil)
	if w == nil {
		t.Fatal("expected non-nil writer")
	}
}

func TestOmsWriter_SetOrderEventBroker(t *testing.T) {
	t.Parallel()
	w := NewOmsWriter(nil, nil)
	w.SetOrderEventBroker(NewOrderEventBroker())
}

func TestNewReconciliationLoop(t *testing.T) {
	t.Parallel()
	rl := NewReconciliationLoop(NewHub(), nil, nil, nil, nil)
	if rl == nil {
		t.Fatal("expected non-nil loop")
	}
}
