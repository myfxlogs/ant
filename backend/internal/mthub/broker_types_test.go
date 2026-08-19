package mthub

import (
	"testing"

	"github.com/shopspring/decimal"
)

// --- Broker pub/sub tests ---

func TestPositionSnapshotBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()

	b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	ev := <-ch
	if ev.AccountID != "acc-1" {
		t.Fatalf("expected acc-1, got %s", ev.AccountID)
	}
}

func TestPositionSnapshotBroker_DropsOldestKeepsLatest(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	for i := int64(0); i < 8; i++ {
		b.Publish(&PositionSnapshot{AccountID: "acc-1", Positions: []PositionSnapshotItem{{Ticket: i}}})
	}
	b.Publish(&PositionSnapshot{AccountID: "acc-1", Positions: []PositionSnapshotItem{{Ticket: 99}}})
	foundLatest := false
	for i := 0; i < 8; i++ {
		ev := <-ch
		if len(ev.Positions) == 1 && ev.Positions[0].Ticket == 99 {
			foundLatest = true
		}
	}
	if !foundLatest {
		t.Fatal("latest authoritative snapshot was dropped when subscriber buffer was full")
	}
}

func TestPositionSnapshotBroker_NoSubscriber(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	// Should not block or panic
	b.Publish(&PositionSnapshot{AccountID: "acc-1"})
}

func TestPositionSnapshotBroker_Unsubscribe(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.Subscribe("acc-1")
	cancel()
	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after cancel")
	}
}

func TestBarBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewBarBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()

	b.Publish(&BarUpdate{AccountID: "acc-1", Symbol: "EURUSD"})
	ev := <-ch
	if ev.Symbol != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", ev.Symbol)
	}
}

func TestBarBroker_DroppedBars(t *testing.T) {
	t.Parallel()
	b := NewBarBroker()
	// No subscribers → publish drops
	b.Publish(&BarUpdate{AccountID: "acc-1"})
	b.Publish(&BarUpdate{AccountID: "acc-1"})
	if b.DroppedBars("acc-1") != 0 {
		t.Fatal("no subscribers means no drop count (drop only counts when subscriber buffer is full)")
	}
}

func TestBarBroker_FullBuffer_Drops(t *testing.T) {
	t.Parallel()
	b := NewBarBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	_ = ch

	// Fill the buffer (64) then publish one more → should drop
	for i := 0; i < 65; i++ {
		b.Publish(&BarUpdate{AccountID: "acc-1"})
	}
	// At least 1 drop expected
	if b.DroppedBars("acc-1") < 1 {
		t.Fatal("expected at least 1 dropped bar")
	}
}

func TestAccountStatusBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewAccountStatusBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()

	b.Publish(&AccountStatusEvent{AccountID: "acc-1", Status: "connected"})
	ev := <-ch
	if ev.Status != "connected" {
		t.Fatalf("expected connected, got %s", ev.Status)
	}
}

// --- Pure logic tests ---

func TestHashToNegative(t *testing.T) {
	t.Parallel()
	v := hashToNegative("test-uuid-123")
	if v >= 0 {
		t.Fatalf("expected negative value, got %d", v)
	}
	// Same input → same output
	v2 := hashToNegative("test-uuid-123")
	if v != v2 {
		t.Fatalf("expected deterministic output, got %d vs %d", v, v2)
	}
	// Different input → different output
	v3 := hashToNegative("different-uuid")
	if v == v3 {
		t.Fatal("expected different hash for different input")
	}
}

func TestIdempotencyKey_Deterministic(t *testing.T) {
	t.Parallel()
	k1 := IdempotencyKey("acc-1", "client-1")
	k2 := IdempotencyKey("acc-1", "client-1")
	if k1 != k2 {
		t.Fatalf("same input should produce same key: %s vs %s", k1, k2)
	}
}

func TestIdempotencyKey_RandomForEmptyClientID(t *testing.T) {
	t.Parallel()
	k1 := IdempotencyKey("acc-1", "")
	k2 := IdempotencyKey("acc-1", "")
	if k1 == k2 {
		t.Fatal("empty clientID should produce random (different) keys")
	}
}

func TestAdvisoryLockKey(t *testing.T) {
	t.Parallel()
	k1a, k1b := advisoryLockKey("acc-1", "client-1")
	k2a, k2b := advisoryLockKey("acc-1", "client-1")
	if k1a != k2a || k1b != k2b {
		t.Fatal("same input should produce same key pair")
	}
	// Different input → different key pair
	k3a, k3b := advisoryLockKey("acc-2", "client-1")
	if k1a == k3a && k1b == k3b {
		t.Fatal("different input should produce different key pair")
	}
}

func TestParamsToCostModel_ExactMatch(t *testing.T) {
	t.Parallel()
	params := []*SymbolParam{
		{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
	}
	model := paramsToCostModel("EURUSD", params)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.Symbol != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", model.Symbol)
	}
}

func TestParamsToCostModel_SuffixMatch(t *testing.T) {
	t.Parallel()
	params := []*SymbolParam{
		{Canonical: "EURUSDm", Digits: 5, PointValue: dec(1)},
	}
	model := paramsToCostModel("EURUSD", params)
	if model == nil {
		t.Fatal("expected non-nil model via suffix match")
	}
}

func TestParamsToCostModel_NotFound(t *testing.T) {
	t.Parallel()
	params := []*SymbolParam{
		{Canonical: "GBPUSD", Digits: 5, PointValue: dec(1)},
	}
	model := paramsToCostModel("EURUSD", params)
	if model != nil {
		t.Fatal("expected nil for not-found symbol")
	}
}

func TestParamsToCostModel_DefaultPointValue(t *testing.T) {
	t.Parallel()
	params := []*SymbolParam{
		{Canonical: "EURUSD", Digits: 5, PointValue: decimal.Zero},
	}
	model := paramsToCostModel("EURUSD", params)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// With zero point value, default 0.10 is used
	if !model.PipValue.Equal(dec(1)) {
		t.Fatalf("expected pip value 1 (0.10*10), got %s", model.PipValue.String())
	}
}

func TestParamsToCostModel_3DigitJPY(t *testing.T) {
	t.Parallel()
	params := []*SymbolParam{
		{Canonical: "USDJPY", Digits: 3, PointValue: dec(1)},
	}
	model := paramsToCostModel("USDJPY", params)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// 3 digits → pipInPoints=10
	if !model.PipValue.Equal(dec(10)) {
		t.Fatalf("expected pip value 10 for JPY pair, got %s", model.PipValue.String())
	}
}

func TestSymbolVariants(t *testing.T) {
	t.Parallel()
	variants := symbolVariants("EURUSD")
	if len(variants) != 5 {
		t.Fatalf("expected 5 variants, got %d", len(variants))
	}
	if variants[0] != "EURUSD" {
		t.Fatalf("expected first variant EURUSD, got %s", variants[0])
	}
	if variants[1] != "EURUSDm" {
		t.Fatalf("expected second variant EURUSDm, got %s", variants[1])
	}
}

func dec(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		side      Side
		orderType OrderType
		want      string
	}{
		{"buy market", SideBuy, OrderMarket, "BUY"},
		{"sell market", SideSell, OrderMarket, "SELL"},
		{"buy limit", SideBuy, OrderLimit, "BUY_LIMIT"},
		{"sell limit", SideSell, OrderLimit, "SELL_LIMIT"},
		{"buy stop", SideBuy, OrderStop, "BUY_STOP"},
		{"sell stop", SideSell, OrderStop, "SELL_STOP"},
		{"buy stop_limit", SideBuy, OrderStopLimit, "BUY_STOP_LIMIT"},
		{"sell stop_limit", SideSell, OrderStopLimit, "SELL_STOP_LIMIT"},
		{"balance", SideBuy, OrderBalance, "BALANCE"},
		{"credit", SideBuy, OrderCredit, "CREDIT"},
		{"unknown", SideBuy, OrderType(99), "BUY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &OrderRecord{Side: tt.side, OrderType: tt.orderType}
			got := r.OrderTypeString()
			if got != tt.want {
				t.Errorf("OrderTypeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPositionSnapshotBroker_SubscribeAll(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.SubscribeAll()
	defer cancel()

	b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	ev := <-ch
	if ev.AccountID != "acc-1" {
		t.Fatalf("expected acc-1, got %s", ev.AccountID)
	}

	// Verify all-subscriber receives from all accounts
	b.Publish(&PositionSnapshot{AccountID: "acc-2"})
	ev2 := <-ch
	if ev2.AccountID != "acc-2" {
		t.Fatalf("expected acc-2, got %s", ev2.AccountID)
	}
}

func TestPositionSnapshotBroker_SubscribeAll_Unsubscribe(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.SubscribeAll()
	cancel()
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after cancel")
	}
}
