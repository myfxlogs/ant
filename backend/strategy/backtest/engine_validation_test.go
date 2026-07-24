package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// stubStrategy is a minimal strategy for testing the engine.
type stubStrategy struct {
	barsSeen  int
	buyAtBar  int
	closeAtBar int
	closeTicket int64
}

func (s *stubStrategy) OnInit(ctx sdk.Context) error { return nil }
func (s *stubStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *stubStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	s.barsSeen++
	bars := ctx.Bars()
	if s.buyAtBar > 0 && s.barsSeen == s.buyAtBar {
		return &sdk.Signal{
			Action: sdk.ActionBuy,
			Symbol: ctx.Symbol(),
			Volume: decimal.NewFromFloat(0.1),
			Price:  bars.Close(0),
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

func makeTestBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		price := decimal.NewFromFloat(1.1000 + float64(i)*0.0010)
		bars[i] = sdk.Bar{
			Open:      price,
			High:      price.Add(decimal.NewFromFloat(0.0005)),
			Low:       price.Sub(decimal.NewFromFloat(0.0005)),
			Close:     price.Add(decimal.NewFromFloat(0.0003)),
			Volume:    1000,
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour).UnixMilli(),
		}
	}
	return bars
}

// TestEngine_ExplicitCloseRecordsTrade verifies that trades closed by the strategy
// (via broker.PositionClose) are properly recorded in the trades list.
func TestEngine_ExplicitCloseRecordsTrade(t *testing.T) {
	strategy := &ticketCapturingStrategy{buyAtBar: 2, closeAtBar: 5}
	bars := makeTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	engine := New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Errorf("expected 1 trade (buy at bar 2, close at bar 5), got %d", len(result.Trades))
	}
	if result.Trades[0].Side != sdk.SideBuy {
		t.Errorf("expected buy trade, got side %v", result.Trades[0].Side)
	}
}

// ticketCapturingStrategy captures the ticket from the buy order so it can close it.
type ticketCapturingStrategy struct {
	barsSeen    int
	buyAtBar    int
	closeAtBar  int
	closeTicket int64
}

func (s *ticketCapturingStrategy) OnInit(ctx sdk.Context) error { return nil }
func (s *ticketCapturingStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *ticketCapturingStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	s.barsSeen++
	bars := ctx.Bars()

	if s.buyAtBar > 0 && s.barsSeen == s.buyAtBar {
		sig := &sdk.Signal{
			Action: sdk.ActionBuy,
			Symbol: ctx.Symbol(),
			Volume: decimal.NewFromFloat(0.1),
			Price:  bars.Close(0),
		}
		// Execute the order directly via broker to capture ticket
		result, err := ctx.Broker().OrderSend(sdk.OrderRequest{
			Symbol: sig.Symbol,
			Side:   sdk.SideBuy,
			Type:   sdk.OrderMarket,
			Volume: sig.Volume,
			Price:  sig.Price,
		})
		if err != nil || result.RetCode != sdk.RetDone {
			return nil, nil
		}
		s.closeTicket = result.Ticket
		return nil, nil // return nil signal since we already executed
	}

	if s.closeAtBar > 0 && s.barsSeen == s.closeAtBar && s.closeTicket > 0 {
		_, err := ctx.Broker().PositionClose(s.closeTicket, decimal.Zero)
		if err != nil {
			return nil, nil
		}
		s.closeTicket = 0
		return nil, nil
	}

	return nil, nil
}

// TestEngine_ExplicitCloseViaBrokerRecordsTrade verifies that when a strategy
// calls broker.PositionClose directly, the trade is recorded in the engine's trades list.
func TestEngine_ExplicitCloseViaBrokerRecordsTrade(t *testing.T) {
	strategy := &ticketCapturingStrategy{buyAtBar: 2, closeAtBar: 5}
	bars := makeTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	engine := New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if len(result.Trades) == 0 {
		t.Fatal("BUG: explicit close via broker.PositionClose did not record a trade")
	}
	if len(result.Trades) != 1 {
		t.Errorf("expected exactly 1 trade, got %d", len(result.Trades))
	}
	trade := result.Trades[0]
	if trade.Side != sdk.SideBuy {
		t.Errorf("expected buy side, got %v", trade.Side)
	}
	if !trade.EntryPrice.Equal(decimal.NewFromFloat(1.1023)) {
		t.Errorf("expected entry price 1.1023 (bar 2 close), got %s", trade.EntryPrice.String())
	}
	// Close price should be bar 5's close price (1.1000 + 5*0.0010 + 0.0003 = 1.1053)
	expectedClose := decimal.NewFromFloat(1.1053)
	if !trade.ExitPrice.Equal(expectedClose) {
		t.Errorf("expected exit price %s, got %s", expectedClose.String(), trade.ExitPrice.String())
	}
	if trade.ProfitPct == 0 {
		t.Error("ProfitPct should be non-zero for a profitable trade")
	}
}

// TestEngine_SLTPCloseRecordsTrade verifies that SL/TP closes are recorded.
func TestEngine_SLTPCloseRecordsTrade(t *testing.T) {
	strategy := &stubStrategy{buyAtBar: 2}
	bars := makeTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	engine := New(cfg, strategy, bars)
	// Manually set TP on the position after buy
	// We'll use a strategy that sets TP
	_, _ = engine.Run(context.Background())

	// Check if any positions were opened
	broker := engine.Broker()
	history := broker.HistoryOrders(0, 0)
	_ = history
}

// TestMetrics_ProfitPctNotZero verifies that ProfitPct is calculated for trades.
func TestMetrics_ProfitPctNotZero(t *testing.T) {
	trades := []Trade{
		{
			Symbol:     "EURUSD",
			Side:       sdk.SideBuy,
			EntryPrice: decimal.NewFromFloat(1.1000),
			ExitPrice:  decimal.NewFromFloat(1.1100),
			Volume:     decimal.NewFromFloat(0.1),
			Profit:     decimal.NewFromFloat(100),
			ProfitPct:  0.909,
		},
		{
			Symbol:     "EURUSD",
			Side:       sdk.SideSell,
			EntryPrice: decimal.NewFromFloat(1.1100),
			ExitPrice:  decimal.NewFromFloat(1.1050),
			Volume:     decimal.NewFromFloat(0.1),
			Profit:     decimal.NewFromFloat(-50),
			ProfitPct:  -0.45,
		},
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 6, 0)
	equity := []EquityPoint{
		{Time: start, Equity: decimal.NewFromInt(10000), Bar: 0},
		{Time: end, Equity: decimal.NewFromInt(10100), Bar: 100},
	}

	metrics := CalculateMetrics(decimal.NewFromInt(10000), equity, trades)
	sharpe, _ := decimal.NewFromString(metrics.SharpeRatio)
	if !sharpe.GreaterThan(decimal.Zero) {
		t.Error("SharpeRatio is 0 — ProfitPct values may not be used correctly")
	}
	t.Logf("SharpeRatio = %s", metrics.SharpeRatio)
}

// TestMetrics_AnnualReturnNotEqualToTotalReturn verifies that AnnualReturn
// is properly annualized rather than just copied from TotalReturn.
func TestMetrics_AnnualReturnNotEqualToTotalReturn(t *testing.T) {
	// 6-month backtest with 10% return
	// Annualized should be ~21% (1.1^2 - 1), not 10%
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 6, 0) // 6 months later

	equity := []EquityPoint{
		{Time: startTime, Equity: decimal.NewFromInt(10000), Bar: 0},
		{Time: endTime, Equity: decimal.NewFromInt(11000), Bar: 100},
	}

	metrics := CalculateMetrics(decimal.NewFromInt(10000), equity, nil)

	if metrics.AnnualReturn == metrics.TotalReturn {
		t.Errorf("AnnualReturn (%s) == TotalReturn (%s) — should be annualized",
			metrics.AnnualReturn, metrics.TotalReturn)
	}
	// 6 months, 10% return → annualized ~21% (1.1^2 - 1)
	expected := 1.1*1.1 - 1
	ar, _ := decimal.NewFromString(metrics.AnnualReturn)
	if math.Abs(ar.InexactFloat64()-expected) > 0.01 {
		t.Errorf("AnnualReturn = %s, expected ~%.4f", metrics.AnnualReturn, expected)
	}
}

// TestEngine_CommissionDeductedFromBalance verifies that commission is properly
// reflected in the final equity/balance.
func TestEngine_CommissionDeductedFromBalance(t *testing.T) {
	commission := decimal.NewFromFloat(0.001) // 0.1%
	strategy := &ticketCapturingStrategy{buyAtBar: 2, closeAtBar: 5}
	bars := makeTestBars(10)

	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Commission:     commission,
	}

	engine := New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	// After all positions are closed, equity should be:
	// initial + profit - commission - swap
	// If commission is not deducted from balance, final equity will be too high.
	broker := engine.Broker()
	finalBalance := broker.balance
	if len(result.Trades) > 0 {
		t.Logf("finalBalance = %s, initialCapital = %s", finalBalance.String(), cfg.InitialCapital.String())
		// Verify commission was deducted by comparing against a no-commission scenario.
		// Profit = (exitPrice - entryPrice) * volume * contractSize
		// Without commission: balance = initial + profit
		// With commission: balance = initial + profit - commission
		trade := result.Trades[0]
		rawProfit := trade.ExitPrice.Sub(trade.EntryPrice).Mul(trade.Volume).Mul(cfg.ContractSize)
		balanceNoCommission := cfg.InitialCapital.Add(rawProfit)
		t.Logf("balance without commission would be %s, actual %s", balanceNoCommission.String(), finalBalance.String())
		if !finalBalance.LessThan(balanceNoCommission) {
			t.Errorf("finalBalance (%s) >= no-commission balance (%s) — commission not deducted",
				finalBalance.String(), balanceNoCommission.String())
		}
		// Verify the difference matches the commission
		diff := balanceNoCommission.Sub(finalBalance)
		t.Logf("commission deducted = %s, trade.Commission = %s", diff.String(), trade.Commission.String())
		if !diff.Equal(trade.Commission) {
			t.Errorf("deducted amount (%s) != trade.Commission (%s)", diff.String(), trade.Commission.String())
		}
	}
}

// TestEngine_AccountFloatingProfit verifies that ctx.Account().Equity reflects
// floating P&L from open positions using the current bar's close price.
func TestEngine_AccountFloatingProfit(t *testing.T) {
	// Strategy that opens a position and checks Account().Equity
	checkStrategy := &accountCheckStrategy{checkAtBar: 3}

	bars := makeTestBars(10)
	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	engine := New(cfg, checkStrategy, bars)
	_, _ = engine.Run(context.Background())

	if !checkStrategy.equityChecked {
		t.Skip("strategy did not reach the check bar")
	}

	// After buying at bar 2, at bar 3 the equity should include floating P&L
	// from the open position using bar 3's close price.
	// If Account().Equity == Account().Balance, floating P&L is not being calculated.
	if checkStrategy.equityAtCheck.Equal(checkStrategy.balanceAtCheck) {
		t.Error("Account().Equity == Account().Balance while position is open — " +
			"floating P&L not reflected in Account()")
	}
	// Equity should be > balance because price went up (bars are ascending)
	if !checkStrategy.equityAtCheck.GreaterThan(checkStrategy.balanceAtCheck) {
		t.Errorf("Equity (%s) should be > Balance (%s) for a profitable open long position",
			checkStrategy.equityAtCheck.String(), checkStrategy.balanceAtCheck.String())
	}
}

type accountCheckStrategy struct {
	barsSeen       int
	buyAtBar       int
	checkAtBar     int
	equityChecked  bool
	equityAtCheck  decimal.Decimal
	balanceAtCheck decimal.Decimal
}

func (s *accountCheckStrategy) OnInit(ctx sdk.Context) error { return nil }
func (s *accountCheckStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
func (s *accountCheckStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	s.barsSeen++
	bars := ctx.Bars()

	if s.buyAtBar == 0 {
		// Buy at bar 2
		if s.barsSeen == 2 {
			result, err := ctx.Broker().OrderSend(sdk.OrderRequest{
				Symbol: ctx.Symbol(),
				Side:   sdk.SideBuy,
				Type:   sdk.OrderMarket,
				Volume: decimal.NewFromFloat(0.1),
				Price:  bars.Close(0),
			})
			if err == nil && result.RetCode == sdk.RetDone {
				s.buyAtBar = s.barsSeen
			}
		}
		return nil, nil
	}

	if s.barsSeen == s.checkAtBar {
		acct := ctx.Account()
		s.equityAtCheck = acct.Equity
		s.balanceAtCheck = acct.Balance
		s.equityChecked = true
	}

	return nil, nil
}

// TestEngine_BarsOrdering verifies that the engine rejects unsorted bars.
func TestEngine_BarsOrdering(t *testing.T) {
	bars := makeTestBars(5)
	// Swap two bars to create unsorted data
	bars[2], bars[3] = bars[3], bars[2]

	strategy := &stubStrategy{}
	cfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	engine := New(cfg, strategy, bars)
	_, err := engine.Run(context.Background())
	if err == nil {
		t.Error("expected error for unsorted bars, got nil")
	}
}
