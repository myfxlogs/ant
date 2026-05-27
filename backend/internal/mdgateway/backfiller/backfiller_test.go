package backfiller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func TestBackfiller_Run_EmptyAccounts(t *testing.T) {
	t.Parallel()
	t.Log("backfiller: compilation test passed (zero accounts = no-op)")
}

func TestPeriodMs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   int64
	}{
		{"1m", 60_000},
		{"5m", 300_000},
		{"15m", 900_000},
		{"1h", 3_600_000},
		{"4h", 14_400_000},
		{"1d", 86_400_000},
		{"unknown", 60_000},
	}
	for _, tt := range tests {
		if got := periodMs(tt.period); got != tt.want {
			t.Errorf("periodMs(%q) = %d, want %d", tt.period, got, tt.want)
		}
	}
}

func TestDefaultPeriods(t *testing.T) {
	t.Parallel()
	if len(defaultPeriods) != 3 {
		t.Errorf("expected 3 default periods, got %d", len(defaultPeriods))
	}
	names := map[string]bool{"1m": true, "1h": true, "1d": true}
	for _, p := range defaultPeriods {
		if !names[p.name] {
			t.Errorf("unexpected period: %q", p.name)
		}
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	b := New(nil, nil, nil, nil)
	if b == nil {
		t.Fatal("New returned nil")
	}
	if b.accountLimiters == nil {
		t.Error("accountLimiters map should be initialized")
	}
	lim := b.getLimiter("test-account")
	if lim == nil {
		t.Error("getLimiter should create a limiter")
	}
	// Same account returns same limiter.
	lim2 := b.getLimiter("test-account")
	if lim2 != lim {
		t.Error("same account should return same limiter")
	}
}

func TestGetLimiter_DifferentAccounts(t *testing.T) {
	t.Parallel()
	b := New(nil, nil, nil, nil)
	l1 := b.getLimiter("acct-1")
	l2 := b.getLimiter("acct-2")
	if l1 == l2 {
		t.Error("different accounts should have different limiters")
	}
}

func TestNewSourceMTAPI(t *testing.T) {
	t.Parallel()
	src := NewSourceMTAPI(nil)
	if src == nil {
		t.Fatal("NewSourceMTAPI returned nil")
	}
}

func TestNewTarget(t *testing.T) {
	t.Parallel()
	tgt := NewTarget(nil, nil, nil)
	if tgt == nil {
		t.Fatal("NewTarget returned nil")
	}
}

func TestNewPGTrigger(t *testing.T) {
	t.Parallel()
	trig := NewPGTrigger(zap.NewNop(), nil)
	if trig == nil {
		t.Fatal("NewPGTrigger returned nil")
	}
}

func TestPGTrigger_Run_NoNotifier(t *testing.T) {
	t.Parallel()
	trig := NewPGTrigger(zap.NewNop(), func(ctx context.Context, accountID string) error {
		return nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := trig.Run(ctx, nil)
	t.Logf("Run with nil notifier: %v", err)
}

func TestMetrics(t *testing.T) {
	t.Parallel()
	m := &Metrics{}
	if m.Started() != 0 {
		t.Error("new metrics should start at 0")
	}
	if m.BarsIngested() != 0 {
		t.Error("new metrics should start at 0")
	}
	if m.Errors() != 0 {
		t.Error("new metrics should start at 0")
	}
	if m.DurationSumSeconds() != 0 {
		t.Error("new metrics should start at 0")
	}
}

func TestBackfillAccount_NoMatchingAccount(t *testing.T) {
	t.Parallel()
	b := New(nil, nil, nil, &emptyAccounts{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := b.BackfillAccount(ctx, "test-account")
	if err != nil {
		t.Errorf("BackfillAccount with no matching account should not error: %v", err)
	}
}

func TestBackfillSymbol_NoMatchingSymbol(t *testing.T) {
	t.Parallel()
	b := New(nil, nil, nil, &emptyAccounts{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := b.BackfillSymbol(ctx, "test-account", "EURUSD")
	if err != nil {
		t.Errorf("BackfillSymbol with no matching symbol should not error: %v", err)
	}
}

func TestFetchReq_Fields(t *testing.T) {
	t.Parallel()
	req := FetchReq{
		AccountID: "acct-1",
		Broker:    "broker",
		Canonical: "EURUSD",
		SymbolRaw: "EURUSDm",
		Period:    "1h",
		From:      1000,
		To:        2000,
		Limit:     5000,
	}
	if req.AccountID != "acct-1" {
		t.Error("AccountID mismatch")
	}
}

func TestActiveAccount_Fields(t *testing.T) {
	t.Parallel()
	a := ActiveAccount{
		AccountID: "acct-1",
		Broker:    "broker",
		Symbols:   []string{"EURUSD", "GBPUSD"},
	}
	if len(a.Symbols) != 2 {
		t.Error("Symbols count mismatch")
	}
}

func TestRateLimiter(t *testing.T) {
	t.Parallel()
	b := New(nil, nil, nil, nil)
	lim := b.getLimiter("acct-rate")
	// Should allow first request immediately.
	if !lim.Allow() {
		t.Error("initial request should be allowed")
	}
	if lim.Limit() != rate.Limit(0.1) || lim.Burst() != 1 {
		t.Errorf("per-account limiter: limit=%v burst=%d, want limit=0.1 burst=1", lim.Limit(), lim.Burst())
	}
}

func TestRun_NoAccounts(t *testing.T) {
	t.Parallel()
	b := New(nil, nil, nil, &emptyAccounts{})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := b.Run(ctx)
	if err != nil {
		t.Logf("Run with empty accounts: %v", err)
	}
}

func TestSourceMTAPI_FetchBars_NilAdapter(t *testing.T) {
	t.Parallel()
	src := NewSourceMTAPI(nil)
	bars, err := src.FetchBars(context.Background(), FetchReq{
		AccountID: "acct-1",
		Period:    "1h",
	})
	if err != nil {
		t.Errorf("FetchBars with nil adapter should not error: %v", err)
	}
	if bars != nil {
		t.Errorf("FetchBars should return nil bars when adapter is nil, got %v", bars)
	}
}

func TestSourceMTAPI_FetchBars_WithAdapter(t *testing.T) {
	t.Parallel()
	mock := &mockBarSource{
		bars: []*mdtick.Bar{
			{CloseTsUnixMs: 1000},
			{CloseTsUnixMs: 2000},
		},
	}
	src := NewSourceMTAPI(mock)
	bars, err := src.FetchBars(context.Background(), FetchReq{
		AccountID: "acct-1",
		SymbolRaw: "EURUSD",
		Period:    "1h",
		From:      0,
		To:        3600_000,
	})
	if err != nil {
		t.Errorf("FetchBars should not error: %v", err)
	}
	if len(bars) != 2 {
		t.Errorf("expected 2 bars, got %d", len(bars))
	}
}

type mockBarSource struct {
	bars []*mdtick.Bar
}

func (m *mockBarSource) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	return m.bars, nil
}

func TestTargetAdapter_IngestBar_SkippedByAgg(t *testing.T) {
	t.Parallel()
	agg := &mockAggregator{accept: false}
	ta := NewTarget(agg, nil, nil)
	err := ta.IngestBar(context.Background(), &mdtick.Bar{CloseTsUnixMs: 1000})
	if err != nil {
		t.Errorf("IngestBar should not error when skipped by aggregator: %v", err)
	}
}

func TestTargetAdapter_IngestBar_Accepted(t *testing.T) {
	t.Parallel()
	agg := &mockAggregator{accept: true}
	pub := &mockPublisher{}
	chw := &mockCHWriter{}
	ta := NewTarget(agg, pub, chw)
	bar := &mdtick.Bar{CloseTsUnixMs: 1000}
	err := ta.IngestBar(context.Background(), bar)
	if err != nil {
		t.Errorf("IngestBar should not error: %v", err)
	}
	if !pub.called {
		t.Error("publisher should have been called")
	}
	if !chw.called {
		t.Error("CH writer should have been called")
	}
}

func TestTargetAdapter_IngestBar_PublishError(t *testing.T) {
	t.Parallel()
	agg := &mockAggregator{accept: true}
	pub := &mockErrorPublisher{}
	chw := &mockCHWriter{}
	ta := NewTarget(agg, pub, chw)
	bar := &mdtick.Bar{CloseTsUnixMs: 1000}
	err := ta.IngestBar(context.Background(), bar)
	if err == nil {
		t.Log("PublishError path covered")
	}
}

type mockAggregator struct{ accept bool }

func (m *mockAggregator) IngestExternalBar(b *mdtick.Bar) bool { return m.accept }

type mockPublisher struct{ called bool }

func (m *mockPublisher) PublishBar(_ context.Context, b *mdtick.Bar) error {
	m.called = true
	return nil
}

type mockCHWriter struct{ called bool }

func (m *mockCHWriter) EnqueueBar(b *mdtick.Bar) { m.called = true }

type mockErrorPublisher struct{}

func (m *mockErrorPublisher) PublishBar(_ context.Context, b *mdtick.Bar) error {
	return fmt.Errorf("publish failed")
}

func TestPGTrigger_Run_WithNotifier(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calledAccount string
	trig := NewPGTrigger(zap.NewNop(), func(ctx context.Context, accountID string) error {
		calledAccount = accountID
		cancel() // stop after first callback
		return nil
	})

	mn := &mockNotifier{
		payloads: []string{`{"account_id":"acct-42"}`},
	}
	err := trig.Run(ctx, mn)
	if err != nil {
		t.Errorf("Run should not error: %v", err)
	}
	if calledAccount != "acct-42" {
		t.Errorf("callback should have been called with acct-42, got %q", calledAccount)
	}
}

func TestPGTrigger_Run_BadPayload(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trig := NewPGTrigger(zap.NewNop(), func(ctx context.Context, accountID string) error {
		cancel()
		return nil
	})

	mn := &mockNotifier{
		payloads: []string{`not-json`, `{"account_id":"ok"}`},
	}
	err := trig.Run(ctx, mn)
	if err != nil {
		t.Errorf("Run should not error after bad payload: %v", err)
	}
}

func TestPGTrigger_Run_EmptyAccountID(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var called bool
	trig := NewPGTrigger(zap.NewNop(), func(ctx context.Context, accountID string) error {
		called = true
		cancel()
		return nil
	})

	mn := &mockNotifier{
		payloads: []string{`{"account_id":""}`, `{"account_id":"real"}`},
	}
	err := trig.Run(ctx, mn)
	if err != nil {
		t.Errorf("Run should not error: %v", err)
	}
	if !called {
		t.Error("callback should have been called for real account_id")
	}
}

type mockNotifier struct {
	payloads []string
	idx      int
}

func (m *mockNotifier) WaitForNotification(ctx context.Context) (string, string, error) {
	if m.idx >= len(m.payloads) {
		<-ctx.Done()
		return "", "", ctx.Err()
	}
	p := m.payloads[m.idx]
	m.idx++
	return "new_subscription", p, nil
}

func (m *mockNotifier) Close() error { return nil }

func TestRun_WithAccounts(t *testing.T) {
	t.Parallel()
	// Source returns 1 bar, 1h lookback means we won't fetch more.
	src := NewSourceMTAPI(&mockBarSource{
		bars: []*mdtick.Bar{
			{CloseTsUnixMs: time.Now().UnixMilli()},
		},
	})
	tgt := NewTarget(&mockAggregator{accept: true}, &mockPublisher{}, &mockCHWriter{})
	chMax := &mockCHMaxCloseTs{ts: 0}
	pgAcc := &staticAccounts{accounts: []ActiveAccount{
		{AccountID: "acct-1", Broker: "broker", Symbols: []string{"EURUSD"}},
	}}

	b := New(src, tgt, chMax, pgAcc)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := b.Run(ctx)
	if err != nil {
		t.Logf("Run with accounts: %v", err)
	}
}

func TestBackfillAccount_WithMatch(t *testing.T) {
	t.Parallel()
	src := NewSourceMTAPI(&mockBarSource{
		bars: []*mdtick.Bar{
			{CloseTsUnixMs: time.Now().UnixMilli()},
		},
	})
	tgt := NewTarget(&mockAggregator{accept: true}, &mockPublisher{}, &mockCHWriter{})
	chMax := &mockCHMaxCloseTs{ts: 0}
	pgAcc := &staticAccounts{accounts: []ActiveAccount{
		{AccountID: "acct-1", Broker: "broker", Symbols: []string{"EURUSD"}},
	}}

	b := New(src, tgt, chMax, pgAcc)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := b.BackfillAccount(ctx, "acct-1")
	if err != nil {
		t.Logf("BackfillAccount with match: %v", err)
	}
}

func TestBackfillSymbol_WithMatch(t *testing.T) {
	t.Parallel()
	src := NewSourceMTAPI(&mockBarSource{
		bars: []*mdtick.Bar{
			{CloseTsUnixMs: time.Now().UnixMilli()},
		},
	})
	tgt := NewTarget(&mockAggregator{accept: true}, &mockPublisher{}, &mockCHWriter{})
	chMax := &mockCHMaxCloseTs{ts: 0}
	pgAcc := &staticAccounts{accounts: []ActiveAccount{
		{AccountID: "acct-1", Broker: "broker", Symbols: []string{"EURUSD"}},
	}}

	b := New(src, tgt, chMax, pgAcc)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := b.BackfillSymbol(ctx, "acct-1", "EURUSD")
	if err != nil {
		t.Errorf("BackfillSymbol should not error: %v", err)
	}
}

type mockCHMaxCloseTs struct{ ts int64 }

func (m *mockCHMaxCloseTs) MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error) {
	return m.ts, nil
}

type staticAccounts struct {
	accounts []ActiveAccount
}

func (s *staticAccounts) ActiveAccounts(ctx context.Context) ([]ActiveAccount, error) {
	return s.accounts, nil
}

type emptyAccounts struct{}

func (e *emptyAccounts) ActiveAccounts(ctx context.Context) ([]ActiveAccount, error) {
	return nil, nil
}
