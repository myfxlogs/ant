package backtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// signalStrategy returns configurable buy/close signals at specified bars.
type signalStrategy struct {
	barsSeen    int
	buyAtBar    int
	closeAtBar  int
	closeTicket int64
	stopLoss    decimal.Decimal
	takeProfit  decimal.Decimal
}

func (s *signalStrategy) OnInit(ctx sdk.Context) error                  { return nil }
func (s *signalStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *signalStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	s.barsSeen++
	if s.buyAtBar > 0 && s.barsSeen == s.buyAtBar {
		return &sdk.Signal{
			Action:     sdk.ActionBuy,
			Symbol:     ctx.Symbol(),
			Volume:     decimal.NewFromFloat(0.1),
			StopLoss:   s.stopLoss,
			TakeProfit: s.takeProfit,
		}, nil
	}
	if s.closeAtBar > 0 && s.barsSeen == s.closeAtBar && s.closeTicket > 0 {
		return &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: s.closeTicket,
		}, nil
	}
	return nil, nil
}

// makeExecTestBars creates bars with distinct open/close prices for testing
// execution timing. Bar i has Open = 1.1000 + i*0.01, Close = Open + 0.005.
func makeExecTestBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		open, _ := decimal.NewFromString(toFixed(float64(1.1000+float64(i)*0.01), 4))
		closeP, _ := decimal.NewFromString(toFixed(float64(1.1000+float64(i)*0.01+0.005), 4))
		bars[i] = sdk.Bar{
			Open:      open,
			High:      open.Add(decimal.NewFromFloat(0.008)),
			Low:       open.Sub(decimal.NewFromFloat(0.002)),
			Close:     closeP,
			Volume:    1000,
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour).UnixMilli(),
		}
	}
	return bars
}

func toFixed(f float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, f)
}

// Test 1: next_bar_open buy signal fills at next bar's open, not current bar's close.
func TestNextBarOpen_BuyFillsAtNextBarOpen(t *testing.T) {
	strategy := &signalStrategy{buyAtBar: 2}
	bars := makeExecTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SignalTiming:   "next_bar_open",
	}

	engine := New(cfg, strategy, bars)
	_, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	positions := engine.Broker().Positions(0)
	if len(positions) != 1 {
		t.Fatalf("expected 1 open position, got %d", len(positions))
	}

	// Bar 2 (0-indexed from makeExecTestBars): Open=1.1200, Close=1.1250
	// Bar 3: Open=1.1300, Close=1.1350
	// next_bar_open → buy at bar 3's open = 1.1300
	expectedEntry, _ := decimal.NewFromString("1.1300")
	if !positions[0].OpenPrice.Equal(expectedEntry) {
		t.Errorf("next_bar_open: entry price = %s, want %s (next bar open)",
			positions[0].OpenPrice.String(), expectedEntry.String())
	}
}

// Test 2: same_bar_close buy signal fills at current bar's close.
func TestSameBarClose_BuyFillsAtBarClose(t *testing.T) {
	strategy := &signalStrategy{buyAtBar: 2}
	bars := makeExecTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SignalTiming:   "same_bar_close",
	}

	engine := New(cfg, strategy, bars)
	_, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	positions := engine.Broker().Positions(0)
	if len(positions) != 1 {
		t.Fatalf("expected 1 open position, got %d", len(positions))
	}

	// Bar 2: Close = 1.1250
	expectedEntry, _ := decimal.NewFromString("1.1250")
	if !positions[0].OpenPrice.Equal(expectedEntry) {
		t.Errorf("same_bar_close: entry price = %s, want %s (bar close)",
			positions[0].OpenPrice.String(), expectedEntry.String())
	}
}

// Test 3: next_bar_open close signal fills at next bar's open, not current bar's close.
// This is the core bug fix — delayed close signals were using bar.Close instead of next bar.Open.
func TestNextBarOpen_CloseFillsAtNextBarOpen(t *testing.T) {
	// Strategy buys at bar 2, closes at bar 4.
	// With next_bar_open: buy fills at bar 3 open, close fills at bar 5 open.
	strategy := &signalCloseStrategy{buyAtBar: 2, closeAtBar: 4}
	bars := makeExecTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SignalTiming:   "next_bar_open",
	}

	engine := New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}

	// Buy signal at bar 2 → fills at bar 3 open = 1.1300
	expectedEntry, _ := decimal.NewFromString("1.1300")
	if !result.Trades[0].EntryPrice.Equal(expectedEntry) {
		t.Errorf("entry price = %s, want %s", result.Trades[0].EntryPrice.String(), expectedEntry.String())
	}

	// Close signal at bar 4 → fills at bar 5 open = 1.1500
	// BUG was: close fills at bar 4 close = 1.1450
	expectedExit, _ := decimal.NewFromString("1.1500")
	if !result.Trades[0].ExitPrice.Equal(expectedExit) {
		t.Errorf("next_bar_open close: exit price = %s, want %s (next bar open, not bar close)",
			result.Trades[0].ExitPrice.String(), expectedExit.String())
	}
}

// signalCloseStrategy buys via signal, then closes via signal (not direct broker call).
type signalCloseStrategy struct {
	barsSeen    int
	buyAtBar    int
	closeAtBar  int
	closeTicket int64
}

func (s *signalCloseStrategy) OnInit(ctx sdk.Context) error                  { return nil }
func (s *signalCloseStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *signalCloseStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	s.barsSeen++
	if s.buyAtBar > 0 && s.barsSeen == s.buyAtBar {
		return &sdk.Signal{
			Action: sdk.ActionBuy,
			Symbol: ctx.Symbol(),
			Volume: decimal.NewFromFloat(0.1),
		}, nil
	}
	if s.closeAtBar > 0 && s.barsSeen == s.closeAtBar {
		// Find open position to close
		positions := ctx.Broker().Positions(0)
		if len(positions) > 0 {
			return &sdk.Signal{
				Action:      sdk.ActionClose,
				OrderTicket: positions[0].Ticket,
			}, nil
		}
	}
	return nil, nil
}

// Test 4: FillRule=bar_close does not apply spread to market orders.
func TestFillRule_BarClose_NoSpread(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Spread:         decimal.NewFromFloat(0.0002),
		FillRule:       "bar_close",
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))

	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatal("OrderSend failed")
	}

	for _, pos := range broker.positions {
		if pos.Ticket == res.Ticket {
			expected := decimal.NewFromFloat(1.1000)
			if !pos.Price.Equal(expected) {
				t.Errorf("bar_close: fill price = %s, want %s (no spread)",
					pos.Price.String(), expected.String())
			}
			return
		}
	}
	t.Fatal("position not found")
}

// Test 5: FillRule=market applies spread to market orders.
func TestFillRule_Market_AppliesSpread(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Spread:         decimal.NewFromFloat(0.0002),
		FillRule:       "market",
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))

	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatal("OrderSend failed")
	}

	for _, pos := range broker.positions {
		if pos.Ticket == res.Ticket {
			expected := decimal.NewFromFloat(1.1002) // 1.1000 + 0.0002 spread
			if !pos.Price.Equal(expected) {
				t.Errorf("market: fill price = %s, want %s (with spread)",
					pos.Price.String(), expected.String())
			}
			return
		}
	}
	t.Fatal("position not found")
}

// Test 6: FillRule=bar_close does not apply spread on position close.
func TestFillRule_BarClose_CloseNoSpread(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Spread:         decimal.NewFromFloat(0.0002),
		FillRule:       "bar_close",
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))

	// Open position
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
	})

	// Close at 1.1200
	broker.SetBarPrice(decimal.NewFromFloat(1.1200))
	closeRes, err := broker.PositionClose(res.Ticket, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}

	// bar_close: no spread → close at raw 1.1200
	expected := decimal.NewFromFloat(1.1200)
	if !closeRes.Price.Equal(expected) {
		t.Errorf("bar_close close: price = %s, want %s (no spread)",
			closeRes.Price.String(), expected.String())
	}
}

// Test 7: Empty SignalTiming defaults to next_bar_open (delayed execution).
func TestEmptySignalTiming_DefaultsToNextBarOpen(t *testing.T) {
	strategy := &signalStrategy{buyAtBar: 2}
	bars := makeExecTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SignalTiming:   "", // empty = default to next_bar_open
	}

	engine := New(cfg, strategy, bars)
	_, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	// Empty SignalTiming → next_bar_open → buy at bar 3 open = 1.1300
	positions := engine.Broker().Positions(0)
	if len(positions) != 1 {
		t.Fatalf("expected 1 open position, got %d", len(positions))
	}
	expectedEntry, _ := decimal.NewFromString("1.1300")
	if !positions[0].OpenPrice.Equal(expectedEntry) {
		t.Errorf("empty SignalTiming: entry price = %s, want %s (next bar open default)",
			positions[0].OpenPrice.String(), expectedEntry.String())
	}
}

// Test 8: FillRule=market applies spread on position close.
func TestFillRule_Market_CloseAppliesSpread(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Spread:         decimal.NewFromFloat(0.0002),
		FillRule:       "market",
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))

	// Open buy position
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
	})

	// Close: closing a buy = sell → receive bid (no spread on sell side)
	broker.SetBarPrice(decimal.NewFromFloat(1.1200))
	closeRes, err := broker.PositionClose(res.Ticket, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}

	// Closing buy position: isBuy=false (selling) → applySpreadToFill returns price as-is (bid)
	// So close price = 1.1200 (no spread added for sell side)
	expected := decimal.NewFromFloat(1.1200)
	if !closeRes.Price.Equal(expected) {
		t.Errorf("market close: price = %s, want %s", closeRes.Price.String(), expected.String())
	}
}

// Test 9: Config values are preserved in Result for ExecutionAssumptions reporting.
func TestConfig_PreservedInResult(t *testing.T) {
	strategy := &signalStrategy{buyAtBar: 2}
	bars := makeExecTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SignalTiming:   "next_bar_open",
		FillRule:       "market",
		SimulationMode: "KLINE_RANGE",
	}

	engine := New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if result.Config.SignalTiming != "next_bar_open" {
		t.Errorf("result.Config.SignalTiming = %q, want %q", result.Config.SignalTiming, "next_bar_open")
	}
	if result.Config.FillRule != "market" {
		t.Errorf("result.Config.FillRule = %q, want %q", result.Config.FillRule, "market")
	}
	if result.Config.SimulationMode != "KLINE_RANGE" {
		t.Errorf("result.Config.SimulationMode = %q, want %q", result.Config.SimulationMode, "KLINE_RANGE")
	}
}
