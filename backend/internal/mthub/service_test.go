package mthub

import (
	"context"
	"testing"
	"time"
)

func TestMtHubService_Setters(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}

	// All setters should be nil-safe and not panic
	svc.SetAccountStateProvider(nil)
	svc.SetGate(nil)
	svc.SetGuard(nil)
	svc.SetOmsWriter(nil)
	svc.SetKillSwitch(nil)
	svc.SetBrokerRegistry(nil)
	svc.SetAccountOwnerVerifier(nil)
	svc.SetLogger(nil)
	svc.SetBarBroker(nil)
	svc.SetTickBroker(nil)
	svc.SetTradeBroker(nil)
	svc.SetStatusBroker(nil)
}

func TestMtHubService_BrokerRegistry(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	if svc.BrokerRegistry() != nil {
		t.Fatal("expected nil by default")
	}
}

func TestMtHubService_PublishBar_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	svc.PublishBar(&BarUpdate{AccountID: "acc-1"}) // should not panic
}

func TestMtHubService_SubscribeBarUpdates_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	ch, cancel := svc.SubscribeBarUpdates("acc-1")
	defer cancel()
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel when no broker")
	}
}

func TestMtHubService_PublishBar_WithBroker(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{barBroker: NewBarBroker()}
	ch, cancel := svc.SubscribeBarUpdates("acc-1")
	defer cancel()
	svc.PublishBar(&BarUpdate{AccountID: "acc-1", Symbol: "EURUSD"})
	ev := <-ch
	if ev.Symbol != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", ev.Symbol)
	}
}

func TestMtHubService_PublishTick_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	svc.PublishTick(&TickUpdate{AccountID: "acc-1"}) // should not panic
}

func TestMtHubService_SubscribeTickUpdates_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	ch, cancel := svc.SubscribeTickUpdates("acc-1")
	defer cancel()
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel when no broker")
	}
}

func TestMtHubService_PublishTradeEvent_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	svc.PublishTradeEvent(&BrokerTradeEvent{AccountID: "acc-1"})
}

func TestMtHubService_SubscribeTradeEvents_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	ch, cancel := svc.SubscribeTradeEvents("acc-1")
	defer cancel()
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel")
	}
}

func TestMtHubService_PublishAccountStatus_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	svc.PublishAccountStatus(&AccountStatusEvent{AccountID: "acc-1"})
}

func TestMtHubService_SubscribeAccountStatus_Nil(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	ch, cancel := svc.SubscribeAccountStatus("acc-1")
	defer cancel()
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel")
	}
}

func TestMtHubService_Platform_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	if svc.Platform("no-such") != "" {
		t.Fatal("expected empty string for no session")
	}
}

func TestMtHubService_OpenedOrders_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	orders, err := svc.OpenedOrders(context.Background(), "no-such")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("expected empty list, got %d", len(orders))
	}
}

func TestMtHubService_OrderHistory_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	orders, err := svc.OrderHistory(context.Background(), "no-such", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("expected empty list, got %d", len(orders))
	}
}

func TestMtHubService_SymbolParams_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	_, err := svc.SymbolParams(context.Background(), "no-such", []string{"EURUSD"})
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestMtHubService_PriceHistory_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	_, err := svc.PriceHistory(context.Background(), "no-such", "EURUSD", "M1", 0, 0, 100)
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestMtHubService_SymbolList_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	_, err := svc.SymbolList(context.Background(), "no-such")
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestMtHubService_SubscribeSymbols_NoSession(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{hub: NewHub()}
	err := svc.SubscribeSymbols(context.Background(), "no-such", []string{"EURUSD"})
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestMtHubService_PublishPositionSnapshot(t *testing.T) {
	t.Parallel()
	broker := NewPositionSnapshotBroker()
	svc := &MtHubService{snapshotBroker: broker}
	ch, cancel := svc.SubscribePositionSnapshots(context.Background(), "acc-1")
	defer cancel()
	svc.PublishPositionSnapshot(&PositionSnapshot{AccountID: "acc-1"})
	ev := <-ch
	if ev.AccountID != "acc-1" {
		t.Fatalf("expected acc-1, got %s", ev.AccountID)
	}
}

func TestMtHubService_PublishAccountProfit(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{accountBroker: NewAccountProfitBroker()}
	ch, cancel := svc.SubscribeAccountProfit(context.Background(), "acc-1")
	defer cancel()
	svc.PublishAccountProfit(&AccountProfitEvent{AccountID: "acc-1", Profit: dec(100)})
	ev := <-ch
	if !ev.Profit.Equal(dec(100)) {
		t.Fatalf("expected profit 100, got %s", ev.Profit.String())
	}
}

func TestMtHubService_SubscribeUserOrderEvents(t *testing.T) {
	t.Parallel()
	broker := NewOrderEventBroker()
	svc := &MtHubService{broker: broker}
	ch, cancel := svc.SubscribeUserOrderEvents(context.Background(), "user-1")
	defer cancel()
	_ = ch // just verify no panic
}

func TestPlatform_NoSession(t *testing.T) {
	t.Parallel()
	if platform("no-such", NewHub()) != "" {
		t.Fatal("expected empty string for no session")
	}
}

func TestMtHubService_SetCostEstimator(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	svc.SetCostEstimator(nil) // nil-safe
}

func TestMtHubService_SetUserLimiter(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	svc.SetUserLimiter(nil) // nil-safe
}

// --- TickBroker and TradeBroker pub/sub ---

func TestTickBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewTickBroker(64, nil)
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	b.Publish(&TickUpdate{AccountID: "acc-1", Bid: dec(1.085), Ask: dec(1.0851)})
	ev := <-ch
	if !ev.Bid.Equal(dec(1.085)) {
		t.Fatalf("expected bid 1.085, got %s", ev.Bid.String())
	}
}

func TestTickBroker_LatestTick(t *testing.T) {
	t.Parallel()
	b := NewTickBroker(64, nil)

	// No tick published yet → nil.
	if got := b.LatestTick("acc-1", "EURUSD"); got != nil {
		t.Fatal("expected nil before Publish")
	}

	// Publish a tick → LatestTick returns it.
	b.Publish(&TickUpdate{AccountID: "acc-1", Symbol: "EURUSD", Bid: dec(1.085), Ask: dec(1.0851)})
	got := b.LatestTick("acc-1", "EURUSD")
	if got == nil {
		t.Fatal("expected non-nil after Publish")
	}
	if !got.Bid.Equal(dec(1.085)) {
		t.Fatalf("expected bid 1.085, got %s", got.Bid.String())
	}

	// Different symbol → nil.
	if got := b.LatestTick("acc-1", "GBPUSD"); got != nil {
		t.Fatal("expected nil for unpublished symbol")
	}

	// Overwrite with newer tick.
	b.Publish(&TickUpdate{AccountID: "acc-1", Symbol: "EURUSD", Bid: dec(1.090), Ask: dec(1.0901)})
	got = b.LatestTick("acc-1", "EURUSD")
	if !got.Bid.Equal(dec(1.090)) {
		t.Fatalf("expected updated bid 1.090, got %s", got.Bid.String())
	}
}

func TestTradeBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewTradeBroker(64, nil)
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	b.Publish(&BrokerTradeEvent{AccountID: "acc-1", Ticket: 123})
	ev := <-ch
	if ev.Ticket != 123 {
		t.Fatalf("expected ticket 123, got %d", ev.Ticket)
	}
}

func TestAccountProfitBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewAccountProfitBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	b.Publish(&AccountProfitEvent{AccountID: "acc-1", Profit: dec(50)})
	ev := <-ch
	if !ev.Profit.Equal(dec(50)) {
		t.Fatalf("expected profit 50, got %s", ev.Profit.String())
	}
}

func TestOrderEventBroker_PubSub(t *testing.T) {
	t.Parallel()
	b := NewOrderEventBroker()
	ch, cancel := b.Subscribe("user-1")
	defer cancel()
	b.PublishEvent("user-1", &OrderEvent{AccountID: "acc-1"})
	ev := <-ch
	if ev.AccountID != "acc-1" {
		t.Fatalf("expected acc-1, got %s", ev.AccountID)
	}
}
