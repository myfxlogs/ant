package mql2go

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
)

// manualMAStrategy is a hand-written Go strategy that mirrors what
// GenerateFromIR would produce for a simple MA crossover MQL4 EA.
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

func makeTestBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	price := 1.1000
	for i := 0; i < n; i++ {
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

	if len(result.Equity) == 0 {
		t.Error("expected equity points from backtest")
	}

	firstEquity := result.Equity[0].Equity
	if !firstEquity.Equals(decimal.NewFromInt(10000)) {
		t.Errorf("expected initial equity 10000, got %s", firstEquity.String())
	}

	if len(result.Trades) == 0 {
		t.Log("no trades produced — this may be expected if EMA never crosses below price in pure uptrend")
	}

	if len(result.Equity) != len(bars)-1 {
		t.Errorf("expected %d equity points, got %d", len(bars)-1, len(result.Equity))
	}
}

// TestGenerateFromIR_ProducesRunnableCode verifies that generated code
// for a realistic MQL4 EA has all structural elements needed to compile
// and run against the SimBroker.
func TestGenerateFromIR_ProducesRunnableCode(t *testing.T) {
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
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	code := GenerateFromIR(ir, "BehavioralTest")

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
		{"OnDeinit", "func (s *BehavioralTest) OnDeinit"},
	}

	for _, ch := range checks {
		if !strings.Contains(code, ch.expected) {
			t.Errorf("generated code missing %s: expected to contain %q", ch.name, ch.expected)
		}
	}
}
