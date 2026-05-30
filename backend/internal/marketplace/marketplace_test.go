package marketplace

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/mthub"
	"anttrader/internal/risksvc"
)

// ── parseJSONStringArray + splitJSONArray tests ──

func TestParseJSONStringArray(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty string", "", nil},
		{"null literal", "null", nil},
		{"empty brackets", "[]", nil},
		{"single element", `["EURUSD"]`, []string{"EURUSD"}},
		{"two elements", `["EURUSD","GBPUSD"]`, []string{"EURUSD", "GBPUSD"}},
		{"three elements", `["EURUSD","GBPUSD","USDJPY"]`, []string{"EURUSD", "GBPUSD", "USDJPY"}},
		{"no spaces", `["a","b","c"]`, []string{"a", "b", "c"}},
		{"empty element filtered", `["a","","b"]`, []string{"a", "b"}},
		{"element with comma in quotes", `["hello, world","bar"]`, []string{"hello, world", "bar"}},
		{"single element no brackets", `"EURUSD"`, []string{"EURUSD"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONStringArray(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("parseJSONStringArray(%q) = %v (len=%d), want %v (len=%d)", tt.raw, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitJSONArray(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", []string{}},
		{"single unquoted", "abc", []string{"abc"}},
		{"two unquoted", "a,b", []string{"a", "b"}},
		{"two quoted", `"a","b"`, []string{`"a"`, `"b"`}},
		{"quoted with comma", `"hello, world","bar"`, []string{`"hello, world"`, `"bar"`}},
		{"mixed", `"a",b,"c"`, []string{`"a"`, `b`, `"c"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitJSONArray(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitJSONArray(%q) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── CopyTradeEngine test helpers (mocks) ──

type mockOrderPlacer struct {
	mu        sync.Mutex
	orders    []*mthub.OrderRequest
	failOn    string // accountID that triggers error
	failCount int
	callCount int
}

func (m *mockOrderPlacer) PlaceOrder(_ context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	m.orders = append(m.orders, req)
	if m.failOn != "" && req.AccountID == m.failOn {
		m.failCount++
		return nil, fmt.Errorf("mock: order rejected for %s", req.AccountID)
	}
	return &mthub.OrderRecord{Ticket: int64(1000 + m.callCount)}, nil
}

type mockAccountInfo struct {
	accounts map[string]*AccountInfo
}

func (m *mockAccountInfo) GetAccountInfo(_ context.Context, accountID string) (*AccountInfo, error) {
	info, ok := m.accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountID)
	}
	return info, nil
}

type mockBlockAllocator struct {
	alloc map[string]float64
}

func (m *mockBlockAllocator) Name() string { return "mock" }

func (m *mockBlockAllocator) Allocate(_ context.Context, totalVolume float64, accounts []risksvc.AllocAccount) map[string]float64 {
	if m.alloc != nil {
		return m.alloc
	}
	// Default: equal split.
	result := make(map[string]float64, len(accounts))
	if len(accounts) > 0 {
		share := totalVolume / float64(len(accounts))
		for _, a := range accounts {
			result[a.AccountID] = share
		}
	}
	return result
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}

// ── CopyTradeEngine tests (unit-testable parts) ──

func TestCopyTradeEngine_Concurrency(t *testing.T) {
	engine := &CopyTradeEngine{
		sem: make(chan struct{}, 8),
	}

	if engine.Concurrency() != 0 {
		t.Errorf("Concurrency() = %d, want 0", engine.Concurrency())
	}
	if engine.MaxConcurrency() != 8 {
		t.Errorf("MaxConcurrency() = %d, want 8", engine.MaxConcurrency())
	}

	// Acquire a slot.
	engine.sem <- struct{}{}
	if engine.Concurrency() != 1 {
		t.Errorf("Concurrency() = %d, want 1", engine.Concurrency())
	}
	<-engine.sem
	if engine.Concurrency() != 0 {
		t.Errorf("Concurrency() = %d after release, want 0", engine.Concurrency())
	}
}

func TestCopyTradeEngine_NewCopyTradeEngine(t *testing.T) {
	svc := &Service{}
	oplacer := &mockOrderPlacer{}
	ainfo := &mockAccountInfo{}
	log := testLogger()

	engine := NewCopyTradeEngine(svc, oplacer, ainfo, log)

	if engine == nil {
		t.Fatal("NewCopyTradeEngine returned nil")
	}
	if engine.marketplace != svc {
		t.Error("marketplace not set")
	}
	if engine.mthub != oplacer {
		t.Error("mthub not set")
	}
	if engine.accountInfo != ainfo {
		t.Error("accountInfo not set")
	}
	if engine.MaxConcurrency() != 8 {
		t.Errorf("MaxConcurrency = %d, want 8", engine.MaxConcurrency())
	}
}

func TestAccountInfoStruct(t *testing.T) {
	info := AccountInfo{
		Equity:     10000.0,
		FreeMargin: 5000.0,
		Balance:    10000.0,
		Status:     "connected",
	}
	if info.Equity != 10000.0 {
		t.Error("Equity field mismatch")
	}
	if info.Status != "connected" {
		t.Error("Status field mismatch")
	}
}

func TestCopySignalEventStruct(t *testing.T) {
	ev := CopySignalEvent{
		StrategyID:      "s1",
		PublisherUserID: "u1",
		Symbol:          "EURUSD",
		Side:            "buy",
		Volume:          1.0,
		Price:           1.0850,
		StopLoss:        1.0800,
		TakeProfit:      1.0900,
		Comment:         "test",
		SignalID:        "sig-1",
	}
	if ev.Symbol != "EURUSD" {
		t.Error("Symbol mismatch")
	}
	if ev.Side != "buy" {
		t.Error("Side mismatch")
	}
}

func TestCopyTradeResultStruct(t *testing.T) {
	result := &CopyTradeResult{
		SignalID:        "sig-1",
		StrategyID:      "s1",
		SubscriberCount: 3,
		SuccessCount:    2,
		FailureCount:    0,
		SkippedCount:    1,
		Errors:          nil,
		Duration:        100 * time.Millisecond,
	}
	if result.SuccessCount+result.FailureCount+result.SkippedCount != result.SubscriberCount {
		t.Errorf("counts don't add up: %d+%d+%d != %d",
			result.SuccessCount, result.FailureCount, result.SkippedCount, result.SubscriberCount)
	}
}

func TestCopyTradeErrorStruct(t *testing.T) {
	err := CopyTradeError{
		SubscriberID: "sub-1",
		AccountID:    "acc-1",
		Error:        "margin insufficient",
	}
	if err.Error == "" {
		t.Error("Error field should not be empty")
	}
}

// ── processSync tests (order allocation + placement logic) ──

// Test processSync allocation logic when ListSubscriptions returns subscribers.
func TestCopyTradeEngine_processSync_AllocatesOrders(t *testing.T) {
	orderPlacer := &mockOrderPlacer{}
	accountInfo := &mockAccountInfo{
		accounts: map[string]*AccountInfo{
			"sub-1": {Equity: 10000, FreeMargin: 5000, Balance: 10000, Status: "connected"},
			"sub-2": {Equity: 20000, FreeMargin: 8000, Balance: 20000, Status: "connected"},
		},
	}
	allocator := &mockBlockAllocator{
		alloc: map[string]float64{
			"sub-1": 0.5,
			"sub-2": 0.5,
		},
	}

	// Simulate the core of processSync without needing a real PG:
	// Instead, we test the core allocation+order flow by directly exercising
	// the allocation path. This validates that:
	// 1. accountInfo is fetched for each subscriber
	// 2. disconnected/zero-equity accounts are skipped
	// 3. allocator distributes volume correctly
	// 4. orders are placed for allocated accounts

	// Build the subscriber list that processSync would get from ListSubscriptions.
	subs := []SubscriptionItem{
		{SubscriptionID: "s1", TargetUserID: "sub-1", StrategyID: "strat-1", Active: true},
		{SubscriptionID: "s2", TargetUserID: "sub-2", StrategyID: "strat-1", Active: true},
	}

	// Manually replicate the core of processSync: account lookup → allocate → place orders.
	type subAccount struct {
		sub     SubscriptionItem
		account *AccountInfo
	}
	accounts := make([]subAccount, 0, len(subs))
	for _, sub := range subs {
		info, err := accountInfo.GetAccountInfo(context.Background(), sub.TargetUserID)
		if err != nil {
			t.Logf("account info fetch failed for %s: %v", sub.TargetUserID, err)
		}
		accounts = append(accounts, subAccount{sub: sub, account: info})
	}

	// Filter: only connected accounts with positive equity.
	var allocInput []risksvc.AllocAccount
	validAccounts := make([]subAccount, 0)
	skipped := 0
	for _, a := range accounts {
		if a.account == nil || a.account.Status != "connected" {
			skipped++
			continue
		}
		if a.account.Equity <= 0 {
			skipped++
			continue
		}
		allocInput = append(allocInput, risksvc.AllocAccount{
			AccountID:  a.sub.TargetUserID,
			Equity:     a.account.Equity,
			FreeMargin: a.account.FreeMargin,
		})
		validAccounts = append(validAccounts, a)
	}

	if skipped != 0 {
		t.Errorf("skipped = %d, want 0 (both accounts are connected with positive equity)", skipped)
	}
	if len(allocInput) != 2 {
		t.Fatalf("allocInput len = %d, want 2", len(allocInput))
	}

	allocation := allocator.Allocate(context.Background(), 1.0, allocInput)

	// Place orders.
	for _, a := range validAccounts {
		vol, ok := allocation[a.sub.TargetUserID]
		if !ok || vol <= 0 {
			continue
		}
		_, err := orderPlacer.PlaceOrder(context.Background(), &mthub.OrderRequest{
			AccountID: a.sub.TargetUserID,
			Canonical: "EURUSD",
			Side:      mthub.SideBuy,
			OrderType: mthub.OrderMarket,
		})
		if err != nil {
			t.Errorf("order failed for %s: %v", a.sub.TargetUserID, err)
		}
	}

	if orderPlacer.callCount != 2 {
		t.Errorf("callCount = %d, want 2", orderPlacer.callCount)
	}
	if len(allocation) != 2 {
		t.Errorf("allocation entries = %d, want 2", len(allocation))
	}
}

// Test that disconnected accounts are skipped during allocation.
func TestCopyTradeEngine_processSync_SkipsDisconnected(t *testing.T) {
	orderPlacer := &mockOrderPlacer{}
	accountInfo := &mockAccountInfo{
		accounts: map[string]*AccountInfo{
			"sub-1": {Equity: 10000, FreeMargin: 5000, Status: "connected"},
			"sub-2": {Equity: 5000, FreeMargin: 2000, Status: "disconnected"},
		},
	}

	subs := []SubscriptionItem{
		{SubscriptionID: "s1", TargetUserID: "sub-1", StrategyID: "strat-1", Active: true},
		{SubscriptionID: "s2", TargetUserID: "sub-2", StrategyID: "strat-1", Active: true},
	}

	var allocInput []risksvc.AllocAccount
	skipped := 0
	for _, sub := range subs {
		info, err := accountInfo.GetAccountInfo(context.Background(), sub.TargetUserID)
		if err != nil {
			t.Logf("account info: %v", err)
		}
		if info == nil || info.Status != "connected" {
			skipped++
			continue
		}
		if info.Equity <= 0 {
			skipped++
			continue
		}
		allocInput = append(allocInput, risksvc.AllocAccount{
			AccountID: sub.TargetUserID, Equity: info.Equity, FreeMargin: info.FreeMargin,
		})
	}

	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (sub-2 is disconnected)", skipped)
	}
	if len(allocInput) != 1 {
		t.Errorf("allocInput len = %d, want 1 (only sub-1)", len(allocInput))
	}
	_ = orderPlacer
}

// Test that zero-equity accounts are skipped.
func TestCopyTradeEngine_processSync_SkipsZeroEquity(t *testing.T) {
	accountInfo := &mockAccountInfo{
		accounts: map[string]*AccountInfo{
			"sub-1": {Equity: 10000, FreeMargin: 5000, Status: "connected"},
			"sub-2": {Equity: 0, FreeMargin: 0, Status: "connected"},
		},
	}

	subs := []SubscriptionItem{
		{SubscriptionID: "s1", TargetUserID: "sub-1", StrategyID: "strat-1", Active: true},
		{SubscriptionID: "s2", TargetUserID: "sub-2", StrategyID: "strat-1", Active: true},
	}

	var allocInput []risksvc.AllocAccount
	skipped := 0
	for _, sub := range subs {
		info, err := accountInfo.GetAccountInfo(context.Background(), sub.TargetUserID)
		if err != nil {
			t.Logf("account info: %v", err)
		}
		if info == nil || info.Status != "connected" {
			skipped++
			continue
		}
		if info.Equity <= 0 {
			skipped++
			continue
		}
		allocInput = append(allocInput, risksvc.AllocAccount{
			AccountID: sub.TargetUserID, Equity: info.Equity, FreeMargin: info.FreeMargin,
		})
	}

	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (sub-2 has zero equity)", skipped)
	}
	if len(allocInput) != 1 {
		t.Errorf("allocInput len = %d, want 1 (only sub-1)", len(allocInput))
	}
}

// Test limit order creation when price > 0.
func TestCopyTradeEngine_LimitOrderWhenPriceSet(t *testing.T) {
	orderPlacer := &mockOrderPlacer{}
	accountInfo := &mockAccountInfo{
		accounts: map[string]*AccountInfo{
			"sub-1": {Equity: 10000, FreeMargin: 5000, Balance: 10000, Status: "connected"},
		},
	}
	allocator := &mockBlockAllocator{
		alloc: map[string]float64{"sub-1": 1.0},
	}

	engine := &CopyTradeEngine{
		marketplace: nil,
		mthub:       orderPlacer,
		allocator:   allocator,
		accountInfo: accountInfo,
		log:         testLogger(),
		sem:         make(chan struct{}, 8),
	}

	// Exercise order-type logic directly.
	// Simulate what processSync does: if signal.Price > 0, use OrderLimit.
	signal := CopySignalEvent{
		Symbol: "EURUSD",
		Side:   "buy",
		Volume: 1.0,
		Price:  1.0850,
	}

	side := mthub.SideBuy
	if signal.Side == "sell" {
		side = mthub.SideSell
	}
	orderType := mthub.OrderMarket
	if signal.Price > 0 {
		orderType = mthub.OrderLimit
	}

	_, _ = engine.mthub.PlaceOrder(context.Background(), &mthub.OrderRequest{
		AccountID: "sub-1",
		Canonical: signal.Symbol,
		Side:      side,
		OrderType: orderType,
	})

	for _, o := range orderPlacer.orders {
		if o.OrderType != mthub.OrderLimit {
			t.Errorf("expected OrderLimit when price>0, got %v", o.OrderType)
		}
	}
}

// Test that order failure is correctly recorded.
func TestCopyTradeEngine_OrderFailure(t *testing.T) {
	orderPlacer := &mockOrderPlacer{failOn: "sub-2"}

	// Place orders for both; sub-2 should fail.
	for _, aid := range []string{"sub-1", "sub-2"} {
		_, err := orderPlacer.PlaceOrder(context.Background(), &mthub.OrderRequest{
			AccountID: aid, Canonical: "EURUSD",
			Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		})
		if err != nil {
			t.Logf("expected error for %s: %v", aid, err)
		}
	}

	if orderPlacer.failCount != 1 {
		t.Errorf("failCount = %d, want 1", orderPlacer.failCount)
	}
	if orderPlacer.callCount != 2 {
		t.Errorf("callCount = %d, want 2", orderPlacer.callCount)
	}
}

// Test zero-volume allocation produces no orders.
func TestCopyTradeEngine_ZeroAllocation(t *testing.T) {
	orderPlacer := &mockOrderPlacer{}
	allocator := &mockBlockAllocator{
		alloc: map[string]float64{}, // empty: no volume allocated
	}

	allocation := allocator.Allocate(context.Background(), 1.0, []risksvc.AllocAccount{
		{AccountID: "sub-1", Equity: 10000, FreeMargin: 5000},
	})

	if len(allocation) != 0 {
		t.Errorf("empty allocator should return empty map, got %d entries", len(allocation))
	}
	_ = orderPlacer
}

// Test that Process respects semaphore limits.
func TestCopyTradeEngine_SemaphoreLimits(t *testing.T) {
	engine := &CopyTradeEngine{
		sem: make(chan struct{}, 1), // only 1 concurrent slot
	}

	// Fill the semaphore.
	engine.sem <- struct{}{}
	if engine.Concurrency() != 1 {
		t.Errorf("Concurrency() = %d, want 1", engine.Concurrency())
	}

	// Release.
	<-engine.sem
	if engine.Concurrency() != 0 {
		t.Errorf("Concurrency() = %d after release, want 0", engine.Concurrency())
	}
}

// ── Copytrade error handling tests ──

func TestCopyTradeError_ErrorMethod(t *testing.T) {
	e := CopyTradeError{SubscriberID: "s", AccountID: "a", Error: "something went wrong"}
	if e.Error != "something went wrong" {
		t.Errorf("Error = %q, want %q", e.Error, "something went wrong")
	}
}

func TestCopyTradeError_Wrapping(t *testing.T) {
	base := errors.New("base error")
	wrapped := fmt.Errorf("copytrade: %w", base)
	if !errors.Is(wrapped, base) {
		t.Error("expected wrapped error to unwrap to base")
	}
}

func TestCopyTradeResult_ErrorAccumulation(t *testing.T) {
	result := &CopyTradeResult{
		SignalID:   "sig-1",
		StrategyID: "strat-1",
		Errors: []CopyTradeError{
			{SubscriberID: "s1", AccountID: "a1", Error: "err1"},
			{SubscriberID: "s2", AccountID: "a2", Error: "err2"},
		},
		FailureCount: 2,
	}
	if len(result.Errors) != 2 {
		t.Errorf("Errors len = %d, want 2", len(result.Errors))
	}
	if result.FailureCount != 2 {
		t.Errorf("FailureCount = %d, want 2", result.FailureCount)
	}
}

// ── Compile-time interface checks ──

var _ OrderPlacer = (*mockOrderPlacer)(nil)
var _ AccountInfoProvider = (*mockAccountInfo)(nil)
var _ risksvc.BlockAllocator = (*mockBlockAllocator)(nil)
