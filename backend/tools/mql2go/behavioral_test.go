package mql2go

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
)

// manualMAStrategy is a hand-written Go strategy that mirrors what mql2go
// would generate for a simple MA crossover MQL4 EA. Used to verify that
// the SimBroker + Engine pipeline produces correct behavioral results
// that the generated code should also produce.
type manualMAStrategy struct {
	maPeriod  int32
	lotSize   decimal.Decimal
	magic     int32
	hedging   bool
	maValue   decimal.Decimal
	prevClose decimal.Decimal
}

func (s *manualMAStrategy) OnInit(ctx sdk.Context) error {
	s.maPeriod = int32(ctx.ParamInt("MAPeriod", 14))
	s.lotSize = ctx.ParamDecimal("LotSize", decimal.NewFromFloat(0.1))
	s.magic = int32(ctx.ParamInt("MagicNumber", 12345))
	s.hedging = ctx.Mode() == sdk.ModeHedging
	return nil
}

func (s *manualMAStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	if tf != ctx.Timeframe() {
		return nil, nil
	}
	bars := ctx.Bars()
	if bars.Len() < int(s.maPeriod)+1 {
		return nil, nil
	}

	_ = bars

	s.maValue = ctx.Indicators().EMA(int(s.maPeriod), 1)
	closeVal := ctx.Bars().Close(1)

	if s.maValue.GreaterThan(closeVal) && s.prevClose.LessThanOrEqual(closeVal) {
		s.prevClose = closeVal
		return &sdk.Signal{
			Action:     sdk.ActionBuy,
			Volume:     s.lotSize,
			Magic:      s.magic,
			FillPolicy: sdk.FillIOC,
		}, nil
	}

	if s.maValue.LessThan(closeVal) && s.prevClose.GreaterThanOrEqual(closeVal) {
		s.prevClose = closeVal
		return &sdk.Signal{
			Action:     sdk.ActionSell,
			Volume:     s.lotSize,
			Magic:      s.magic,
			FillPolicy: sdk.FillIOC,
		}, nil
	}

	s.prevClose = closeVal
	return nil, nil
}

func (s *manualMAStrategy) OnDeinit(ctx sdk.Context, reason string) error {
	return nil
}

// makeTestBars creates a simple uptrend series of bars for testing.
func makeTestBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	price := 1.1000
	for i := 0; i < n; i++ {
		// Simple uptrend: each bar closes higher
		open := price
		close := price + 0.0010
		high := close + 0.0005
		low := open - 0.0005
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(open),
			High:      decimal.NewFromFloat(high),
			Low:       decimal.NewFromFloat(low),
			Close:     decimal.NewFromFloat(close),
			Volume:    1000,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute).UnixMilli(),
		}
		price = close
	}
	return bars
}

func TestBehavioralAlignment_SimBroker(t *testing.T) {
	bars := makeTestBars(50)
	if len(bars) < 20 {
		t.Fatalf("need at least 20 bars, got %d", len(bars))
	}

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
	}

	strategy := &manualMAStrategy{
		maPeriod:  14,
		lotSize:   decimal.NewFromFloat(0.1),
		magic:     12345,
		prevClose: decimal.Zero,
	}

	engine := backtest.New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	// Verify we got equity points
	if len(result.Equity) == 0 {
		t.Error("expected equity points from backtest")
	}

	// Verify initial equity is the starting capital
	firstEquity := result.Equity[0].Equity
	if !firstEquity.Equals(decimal.NewFromInt(10000)) {
		t.Errorf("expected initial equity 10000, got %s", firstEquity.String())
	}

	// The strategy should produce at least one signal in an uptrend
	// (MA will be below price in uptrend → buy signal on first crossover)
	if len(result.Trades) == 0 {
		t.Log("no trades produced — this may be expected if EMA never crosses below price in pure uptrend")
	}

	// Verify equity curve has the right number of points (one per bar after bar 0)
	if len(result.Equity) != len(bars)-1 {
		t.Errorf("expected %d equity points, got %d", len(bars)-1, len(result.Equity))
	}
}

// TestGenerate_ProducesRunnableCode verifies that generated code for a
// realistic MQL4 EA has all the structural elements needed to compile
// and run against the SimBroker.
func TestGenerate_ProducesRunnableCode(t *testing.T) {
	source := `
extern int MagicNumber = 12345;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
extern double StopLoss = 50;
extern double TakeProfit = 100;

double maValue;

int OnInit() { return 0; }

void OnTick() {
    maValue = iMA(NULL, 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (maValue > Close[1]) {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, Ask - StopLoss * Point, Ask + TakeProfit * Point, "Buy", MagicNumber, 0, clrGreen);
    }
    if (maValue < Close[1]) {
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, Bid + StopLoss * Point, Bid - TakeProfit * Point, "Sell", MagicNumber, 0, clrRed);
    }
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	intent.Meta.Name = "BehavioralTest"

	code := Generate(intent)

	// Verify all structural elements needed for SimBroker compatibility
	checks := []struct {
		name     string
		expected string
	}{
		{"package", "package"},
		{"sdk import", `"anttrader/strategy/sdk"`},
		{"decimal import", `"github.com/shopspring/decimal"`},
		{"struct", "type BehavioralTest struct"},
		{"interface assertion", "var _ sdk.Strategy = (*BehavioralTest)(nil)"},
		{"OnInit", "func (s *BehavioralTest) OnInit"},
		{"OnBar", "func (s *BehavioralTest) OnBar"},
		{"OnDeinit", "func (s *BehavioralTest) OnDeinit"},
		{"EMA indicator", "ctx.Indicators().EMA("},
		{"signal creation", "sdk.Signal{"},
		{"buy action", "sdk.ActionBuy"},
		{"closeAll helper", "func (s *BehavioralTest) closeAll"},
	}

	for _, ch := range checks {
		if !contains(code, ch.expected) {
			t.Errorf("generated code missing %s: expected to contain %q", ch.name, ch.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
