package backtest

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// --- aggregate.go tests ---

func TestTfDuration(t *testing.T) {
	tests := []struct {
		tf   string
		want int64 // milliseconds
	}{
		{"M1", 60_000},
		{"M5", 300_000},
		{"M15", 900_000},
		{"M30", 1_800_000},
		{"H1", 3_600_000},
		{"H4", 14_400_000},
		{"D1", 86_400_000},
		{"W1", 604_800_000},
		{"UNKNOWN", 3_600_000}, // default 1h
	}
	for _, tt := range tests {
		got := tfDuration(tt.tf).Milliseconds()
		if got != tt.want {
			t.Errorf("tfDuration(%q) = %d ms, want %d ms", tt.tf, got, tt.want)
		}
	}
}

func TestAggregateBars_Empty(t *testing.T) {
	if got := aggregateBars(nil, "H1"); got != nil {
		t.Errorf("aggregateBars(nil) = %v, want nil", got)
	}
}

func TestAggregateBars_SameBucket(t *testing.T) {
	bars := []sdk.Bar{
		{Open: d("1.0"), High: d("2.0"), Low: d("0.5"), Close: d("1.5"), Volume: 100, Timestamp: 1_000},
		{Open: d("1.5"), High: d("2.5"), Low: d("0.8"), Close: d("2.0"), Volume: 200, Timestamp: 2_000},
	}
	// M1 bucket = 0-60000ms, both bars in same bucket.
	result := aggregateBars(bars, "M1")
	if len(result) != 1 {
		t.Fatalf("aggregateBars(M1) = %d bars, want 1", len(result))
	}
	if !result[0].High.Equal(d("2.5")) {
		t.Errorf("High = %s, want 2.5", result[0].High)
	}
	if !result[0].Low.Equal(d("0.5")) {
		t.Errorf("Low = %s, want 0.5", result[0].Low)
	}
	if !result[0].Close.Equal(d("2.0")) {
		t.Errorf("Close = %s, want 2.0", result[0].Close)
	}
	if result[0].Volume != 300 {
		t.Errorf("Volume = %d, want 300", result[0].Volume)
	}
}

func TestAggregateBars_MultipleBuckets(t *testing.T) {
	bars := []sdk.Bar{
		{Open: d("1.0"), High: d("2.0"), Low: d("0.5"), Close: d("1.5"), Volume: 100, Timestamp: 1_000},
		{Open: d("1.5"), High: d("2.5"), Low: d("0.8"), Close: d("2.0"), Volume: 200, Timestamp: 2_000},
		{Open: d("2.0"), High: d("3.0"), Low: d("1.5"), Close: d("2.5"), Volume: 300, Timestamp: 70_000},
	}
	// First two bars in M1 bucket 0, third in M1 bucket 60000.
	result := aggregateBars(bars, "M1")
	if len(result) != 2 {
		t.Fatalf("aggregateBars(M1) = %d bars, want 2", len(result))
	}
}

func TestAggregateBars_W1(t *testing.T) {
	// Two bars in different weeks.
	bars := []sdk.Bar{
		{Open: d("1.0"), High: d("2.0"), Low: d("0.5"), Close: d("1.5"), Volume: 100, Timestamp: 1_000},
		{Open: d("1.5"), High: d("2.5"), Low: d("0.8"), Close: d("2.0"), Volume: 200, Timestamp: 604_800_001},
	}
	result := aggregateBars(bars, "W1")
	if len(result) != 2 {
		t.Fatalf("aggregateBars(W1) = %d bars, want 2", len(result))
	}
}

// --- runStrategySignal tests (tick path) ---

type tickStrategyNoCapable struct{}

func (s *tickStrategyNoCapable) OnInit(ctx sdk.Context) error  { return nil }
func (s *tickStrategyNoCapable) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	return nil, nil
}
func (s *tickStrategyNoCapable) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *tickStrategyNoCapable) OnTick(ctx sdk.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	return &sdk.Signal{Action: sdk.ActionBuy}, nil
}

type tickStrategyWithCapable struct {
	hasTick bool
}

func (s *tickStrategyWithCapable) OnInit(ctx sdk.Context) error  { return nil }
func (s *tickStrategyWithCapable) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	return nil, nil
}
func (s *tickStrategyWithCapable) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *tickStrategyWithCapable) OnTick(ctx sdk.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	return &sdk.Signal{Action: sdk.ActionSell}, nil
}
func (s *tickStrategyWithCapable) HasOnTick() bool { return s.hasTick }

func TestEngine_RunStrategySignal_TickNoCapable(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, Spread: d("0.0002")})
	engine := &Engine{
		broker:   broker,
		strategy: &tickStrategyNoCapable{},
		config:   Config{Spread: d("0.0002")},
	}
	btCtx := &backtestContext{}
	sig, err := engine.runStrategySignal(btCtx, sdk.Bar{Close: d("1.1")})
	if err != nil {
		t.Fatalf("runStrategySignal failed: %v", err)
	}
	if sig == nil || sig.Action != sdk.ActionBuy {
		t.Errorf("signal = %v, want ActionBuy", sig)
	}
}

func TestEngine_RunStrategySignal_TickWithCapable_True(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, Spread: d("0.0002")})
	engine := &Engine{
		broker:   broker,
		strategy: &tickStrategyWithCapable{hasTick: true},
		config:   Config{Spread: d("0.0002")},
	}
	btCtx := &backtestContext{}
	sig, err := engine.runStrategySignal(btCtx, sdk.Bar{Close: d("1.1")})
	if err != nil {
		t.Fatalf("runStrategySignal failed: %v", err)
	}
	if sig == nil || sig.Action != sdk.ActionSell {
		t.Errorf("signal = %v, want ActionSell", sig)
	}
}

func TestEngine_RunStrategySignal_TickWithCapable_False(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	engine := &Engine{
		broker:   broker,
		strategy: &tickStrategyWithCapable{hasTick: false},
		config:   Config{},
	}
	btCtx := &backtestContext{}
	sig, err := engine.runStrategySignal(btCtx, sdk.Bar{Close: d("1.1")})
	if err != nil {
		t.Fatalf("runStrategySignal failed: %v", err)
	}
	// HasOnTick=false → falls back to OnBar → returns nil.
	if sig != nil {
		t.Errorf("signal = %v, want nil (OnBar path)", sig)
	}
}

func TestEngine_RunStrategySignal_BarOnly(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	engine := &Engine{
		broker:   broker,
		strategy: &barOnlyStrat{},
		config:   Config{},
	}
	btCtx := &backtestContext{}
	sig, err := engine.runStrategySignal(btCtx, sdk.Bar{Close: d("1.1")})
	if err != nil {
		t.Fatalf("runStrategySignal failed: %v", err)
	}
	if sig != nil {
		t.Errorf("signal = %v, want nil", sig)
	}
}

// --- WalkForward tests ---

func TestRunWalkForward_TooFewBars(t *testing.T) {
	bars := make([]sdk.Bar, 5)
	_, err := RunWalkForward(context.Background(), Config{}, &barOnlyStrat{}, bars)
	if err == nil {
		t.Error("RunWalkForward with <10 bars should return error")
	}
}

func TestRunWalkForward_TooFewAfterSplit(t *testing.T) {
	bars := make([]sdk.Bar, 11) // 11 * 0.7 = 7 IS, 4 OOS → OOS < 5
	for i := range bars {
		bars[i] = sdk.Bar{Close: d("1.0"), Timestamp: int64(i * 1000)}
	}
	_, err := RunWalkForward(context.Background(), Config{}, &barOnlyStrat{}, bars)
	if err == nil {
		t.Error("RunWalkForward with too few OOS bars should return error")
	}
}

func TestRunWalkForward_Success(t *testing.T) {
	bars := make([]sdk.Bar, 20)
	for i := range bars {
		bars[i] = sdk.Bar{
			Open:  d("1.0"),
			High:  d("1.1"),
			Low:   d("0.9"),
			Close: d("1.0"),
			Volume: 100,
			Timestamp: int64(i * 3600_000),
		}
	}
	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: d("100000"),
		Leverage:       100,
		ContractSize:   d("100000"),
	}
	result, err := RunWalkForward(context.Background(), cfg, &barOnlyStrat{}, bars)
	if err != nil {
		t.Fatalf("RunWalkForward failed: %v", err)
	}
	if result == nil || result.IS == nil || result.OOS == nil {
		t.Error("RunWalkForward should return non-nil IS and OOS metrics")
	}
}

// --- bar_source.go test ---

func TestBtBarSource_Len(t *testing.T) {
	bars := []sdk.Bar{
		{Close: d("1.0")},
		{Close: d("1.1")},
	}
	src := &btBarSource{bars: bars}
	if src.Len() != 2 {
		t.Errorf("Len() = %d, want 2", src.Len())
	}
}

func TestBtBarSource_Access(t *testing.T) {
	bars := []sdk.Bar{
		{Open: d("1.0"), High: d("2.0"), Low: d("0.5"), Close: d("1.5"), Volume: 100},
		{Open: d("1.5"), High: d("2.5"), Low: d("1.0"), Close: d("2.0"), Volume: 200},
	}
	src := &btBarSource{bars: bars}
	// Index 0 = most recent (last element).
	if !src.Close(0).Equal(d("2.0")) {
		t.Errorf("Close(0) = %s, want 2.0", src.Close(0))
	}
	if !src.Open(0).Equal(d("1.5")) {
		t.Errorf("Open(0) = %s, want 1.5", src.Open(0))
	}
	if src.Volume(0) != 200 {
		t.Errorf("Volume(0) = %d, want 200", src.Volume(0))
	}
	// Index 1 = older bar.
	if !src.Close(1).Equal(d("1.5")) {
		t.Errorf("Close(1) = %s, want 1.5", src.Close(1))
	}
	// Out of bounds returns zero.
	if !src.Close(10).Equal(decimal.Zero) {
		t.Errorf("Close(10) out of bounds = %s, want 0", src.Close(10))
	}
	if src.Volume(10) != 0 {
		t.Errorf("Volume(10) out of bounds = %d, want 0", src.Volume(10))
	}
}

// --- metrics.go safeDecimal test ---

func TestSafeDecimal(t *testing.T) {
	if !safeDecimal(1.5).Equal(d("1.5")) {
		t.Error("safeDecimal(1.5) should return 1.5")
	}
	if !safeDecimal(float64NaN()).Equal(decimal.Zero) {
		t.Error("safeDecimal(NaN) should return 0")
	}
}

func float64NaN() float64 {
	var z float64
	return z / z
}

// --- helper strategy ---

type barOnlyStrat struct{}

func (s *barOnlyStrat) OnInit(ctx sdk.Context) error  { return nil }
func (s *barOnlyStrat) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	return nil, nil
}
func (s *barOnlyStrat) OnDeinit(ctx sdk.Context, reason string) error { return nil }
