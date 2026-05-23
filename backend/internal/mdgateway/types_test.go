package mdgateway

import (
	"testing"
)

func TestMoney_GetValue(t *testing.T) {
	m := &Money{Value: "1.2345"}
	if m.GetValue() != "1.2345" {
		t.Fatalf("expected 1.2345, got %s", m.GetValue())
	}
}

func TestMoney_Nil(t *testing.T) {
	var m *Money
	if m.GetValue() != "" {
		t.Fatal("expected empty for nil Money")
	}
}

func TestTick_GetBid(t *testing.T) {
	tick := &Tick{
		Bid: &Money{Value: "1.1000"},
		Ask: &Money{Value: "1.1005"},
	}
	if tick.GetBid().GetValue() != "1.1000" {
		t.Fatalf("expected bid 1.1000, got %s", tick.GetBid().GetValue())
	}
	if tick.GetAsk().GetValue() != "1.1005" {
		t.Fatalf("expected ask 1.1005, got %s", tick.GetAsk().GetValue())
	}
}

func TestTick_NilFields(t *testing.T) {
	var tick *Tick
	if tick.GetBid() != nil {
		t.Fatal("expected nil from nil Tick")
	}
	if tick.GetAsk() != nil {
		t.Fatal("expected nil from nil Tick")
	}
}

func TestBar_Fields(t *testing.T) {
	b := Bar{
		UserID:        "u1",
		Broker:        "demo",
		SymbolRaw:     "EURUSD",
		Canonical:     "EURUSD",
		Period:        "1m",
		OpenTsUnixMs:  1000,
		CloseTsUnixMs: 1059,
		Open:          1.1000,
		High:          1.1010,
		Low:           1.0990,
		Close:         1.1005,
		Volume:        100.0,
		TickCount:     5,
	}
	if b.UserID != "u1" {
		t.Fatalf("expected UserID=u1, got %s", b.UserID)
	}
	if b.SymbolRaw != "EURUSD" {
		t.Fatalf("expected SymbolRaw=EURUSD, got %s", b.SymbolRaw)
	}
	if b.Open != 1.1000 {
		t.Fatalf("expected Open=1.1000, got %f", b.Open)
	}
}
