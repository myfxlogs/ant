package interp

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// mockBarSeries implements sdk.BarSeries for testing.
type mockBarSeries struct {
	closes []decimal.Decimal
}

func (m *mockBarSeries) Open(shift int) decimal.Decimal  { return decimal.Zero }
func (m *mockBarSeries) High(shift int) decimal.Decimal  { return decimal.Zero }
func (m *mockBarSeries) Low(shift int) decimal.Decimal   { return decimal.Zero }
func (m *mockBarSeries) Close(shift int) decimal.Decimal { return m.closes[shift] }
func (m *mockBarSeries) Volume(shift int) int64          { return 100 }
func (m *mockBarSeries) Time(shift int) int64            { return time.Now().UnixMilli() }
func (m *mockBarSeries) Len() int                        { return len(m.closes) }
func (m *mockBarSeries) Slice(n int) sdk.BarSeries       { return m }
func (m *mockBarSeries) Timeframe() string               { return "H1" }
func (m *mockBarSeries) Symbol() string                  { return "EURUSD" }

// mockBroker implements sdk.Broker for testing.
type mockBroker struct {
	positions []sdk.Position
	orders    []sdk.PendingOrder
	lastReq   sdk.OrderRequest
}

func (m *mockBroker) OrderSend(req sdk.OrderRequest) (sdk.OrderResult, error) {
	m.lastReq = req
	return sdk.OrderResult{Ticket: 999}, nil
}
func (m *mockBroker) PositionClose(ticket int64, volume decimal.Decimal) (sdk.OrderResult, error) {
	return sdk.OrderResult{}, nil
}
func (m *mockBroker) PositionModify(ticket int64, sl, tp decimal.Decimal) (sdk.OrderResult, error) {
	return sdk.OrderResult{}, nil
}
func (m *mockBroker) OrderDelete(ticket int64) (sdk.OrderResult, error) {
	return sdk.OrderResult{}, nil
}
func (m *mockBroker) Positions(magic int32) []sdk.Position { return m.positions }
func (m *mockBroker) Orders(magic int32) []sdk.PendingOrder { return m.orders }
func (m *mockBroker) HistoryOrders(from, to int64) []sdk.Position { return nil }
func (m *mockBroker) Deals(from, to int64, magic int32) []sdk.Deal { return nil }
func (m *mockBroker) SymbolInfo(symbol string) (sdk.SymbolInfo, error) {
	return sdk.SymbolInfo{}, nil
}
func (m *mockBroker) Account() sdk.AccountInfo {
	return sdk.AccountInfo{
		Balance:    decimal.NewFromInt(10000),
		Equity:     decimal.NewFromInt(10000),
		FreeMargin: decimal.NewFromInt(5000),
		Margin:     decimal.NewFromInt(1000),
		Leverage:   100,
	}
}

// mockContext implements sdk.Context for testing.
type mockContext struct {
	bars   *mockBarSeries
	broker *mockBroker
}

func (m *mockContext) Param(name string, defaultVal interface{}) interface{} { return defaultVal }
func (m *mockContext) ParamDecimal(name string, defaultVal decimal.Decimal) decimal.Decimal {
	return defaultVal
}
func (m *mockContext) ParamInt(name string, defaultVal int) int          { return defaultVal }
func (m *mockContext) ParamString(name, defaultVal string) string        { return defaultVal }
func (m *mockContext) ParamBool(name string, defaultVal bool) bool       { return defaultVal }
func (m *mockContext) Bars() sdk.BarSeries                                { return m.bars }
func (m *mockContext) BarsTF(timeframe string) sdk.BarSeries              { return m.bars }
func (m *mockContext) Symbol() string                                     { return "EURUSD" }
func (m *mockContext) Timeframe() string                                  { return "H1" }
func (m *mockContext) Point() decimal.Decimal                             { return decimal.NewFromFloat(0.00001) }
func (m *mockContext) Pip() decimal.Decimal                               { return decimal.NewFromFloat(0.0001) }
func (m *mockContext) Digits() int32                                      { return 5 }
func (m *mockContext) Ask() decimal.Decimal                               { return decimal.NewFromFloat(1.1000) }
func (m *mockContext) Bid() decimal.Decimal                               { return decimal.NewFromFloat(1.0998) }
func (m *mockContext) Spread() decimal.Decimal                            { return decimal.NewFromInt(2) }
func (m *mockContext) Account() sdk.AccountInfo                           { return m.broker.Account() }
func (m *mockContext) Mode() sdk.AccountMode                              { return sdk.ModeHedging }
func (m *mockContext) Broker() sdk.Broker                                 { return m.broker }
func (m *mockContext) Indicators() sdk.IndicatorSet                       { return nil }
func (m *mockContext) SetTimer(seconds int)                               {}
func (m *mockContext) KillTimer()                                         {}
func (m *mockContext) Log(msg string)                                     {}
func (m *mockContext) ServerTime() int64                                  { return time.Now().UnixMilli() }

func TestMQL4OrderPool_Lifecycle(t *testing.T) {
	pool := &MQL4OrderPool{}
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{
			positions: []sdk.Position{
				{
					Ticket:    1001,
					Symbol:    "EURUSD",
					Side:      sdk.SideBuy,
					Volume:    decimal.NewFromFloat(0.1),
					OpenPrice: decimal.NewFromFloat(1.0950),
					StopLoss:  decimal.NewFromFloat(1.0900),
					TakeProfit: decimal.NewFromFloat(1.1050),
					Profit:    decimal.NewFromFloat(50),
					Magic:     12345,
					Comment:   "test",
					OpenTime:  time.Now(),
				},
			},
		},
	}

	pool.Reset(ctx)

	if pool.Total() != 1 {
		t.Errorf("Total() = %d, want 1", pool.Total())
	}

	// Select by position
	if !pool.Select(0) {
		t.Error("Select(0) should succeed")
	}
	if pool.Ticket() != 1001 {
		t.Errorf("Ticket() = %d, want 1001", pool.Ticket())
	}
	if pool.Symbol() != "EURUSD" {
		t.Errorf("Symbol() = %s, want EURUSD", pool.Symbol())
	}
	if pool.Type() != 0 {
		t.Errorf("Type() = %d, want 0 (OP_BUY)", pool.Type())
	}
	if !pool.Lots().ToDecimal().Equal(decimal.NewFromFloat(0.1)) {
		t.Errorf("Lots() = %s, want 0.1", pool.Lots().ToDecimal())
	}
	if pool.MagicNumber() != 12345 {
		t.Errorf("MagicNumber() = %d, want 12345", pool.MagicNumber())
	}

	// Select by ticket
	if !pool.SelectByTicket(1001) {
		t.Error("SelectByTicket(1001) should succeed")
	}
	if pool.Ticket() != 1001 {
		t.Errorf("Ticket() = %d, want 1001", pool.Ticket())
	}

	// Out of range
	if pool.Select(1) {
		t.Error("Select(1) should fail (only 1 order)")
	}
}

func TestMQL5PositionPool_Lifecycle(t *testing.T) {
	pool := &MQL5PositionPool{}
	ctx := &mockContext{
		bars: &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{
			positions: []sdk.Position{
				{
					Ticket:    2001,
					Symbol:    "GBPUSD",
					Side:      sdk.SideSell,
					Volume:    decimal.NewFromFloat(0.2),
					OpenPrice: decimal.NewFromFloat(1.2500),
					StopLoss:  decimal.NewFromFloat(1.2600),
					TakeProfit: decimal.NewFromFloat(1.2400),
					Profit:    decimal.NewFromFloat(-30),
					Magic:     54321,
					Comment:   "sell_test",
					OpenTime:  time.Now(),
				},
			},
		},
	}

	pool.Reset(ctx)

	if pool.Total() != 1 {
		t.Errorf("Total() = %d, want 1", pool.Total())
	}

	// GetTicket selects by index
	ticket := pool.GetTicket(0)
	if ticket != 2001 {
		t.Errorf("GetTicket(0) = %d, want 2001", ticket)
	}

	// Position properties
	if pool.Symbol() != "GBPUSD" {
		t.Errorf("Symbol() = %s, want GBPUSD", pool.Symbol())
	}

	vol := pool.GetDouble(0) // POSITION_VOLUME
	if !vol.ToDecimal().Equal(decimal.NewFromFloat(0.2)) {
		t.Errorf("GetDouble(VOLUME) = %s, want 0.2", vol.ToDecimal())
	}

	if pool.GetInteger(1) != 54321 { // POSITION_MAGIC
		t.Errorf("GetInteger(MAGIC) = %d, want 54321", pool.GetInteger(1))
	}

	if pool.GetString(1) != "sell_test" { // POSITION_COMMENT
		t.Errorf("GetString(COMMENT) = %s, want sell_test", pool.GetString(1))
	}

	// SelectByTicket
	if !pool.SelectByTicket(2001) {
		t.Error("SelectByTicket(2001) should succeed")
	}
}

func TestInterpreter_AccountFunctions(t *testing.T) {
	it := &Interpreter{
		globals: make(map[string]Value),
		scopes:  []map[string]Value{make(map[string]Value)},
	}
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.ctx = ctx

	// Test AccountBalance
	v, ok := it.callTrade("AccountBalance", nil)
	if !ok {
		t.Error("AccountBalance should be handled")
	}
	if !v.ToDecimal().Equal(decimal.NewFromInt(10000)) {
		t.Errorf("AccountBalance = %s, want 10000", v.ToDecimal())
	}

	// Test AccountLeverage
	v, ok = it.callTrade("AccountLeverage", nil)
	if !ok {
		t.Error("AccountLeverage should be handled")
	}
	if v.ToInt() != 100 {
		t.Errorf("AccountLeverage = %d, want 100", v.ToInt())
	}
}
