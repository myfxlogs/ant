package interp

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// mockIndicatorSet implements sdk.IndicatorSet for testing.
type mockIndicatorSet struct{}

func (m *mockIndicatorSet) MA(period, shift int, method string) decimal.Decimal {
	return decimal.NewFromFloat(1.1050)
}
func (m *mockIndicatorSet) EMA(period, shift int) decimal.Decimal { return decimal.Zero }
func (m *mockIndicatorSet) RSI(period, shift int) decimal.Decimal { return decimal.NewFromInt(55) }
func (m *mockIndicatorSet) MACD(fast, slow, signal, shift int) decimal.Decimal {
	return decimal.NewFromFloat(0.0005)
}
func (m *mockIndicatorSet) MACDSignal(fast, slow, signal, shift int) decimal.Decimal {
	return decimal.Zero
}
func (m *mockIndicatorSet) ATR(period, shift int) decimal.Decimal { return decimal.NewFromFloat(0.0020) }
func (m *mockIndicatorSet) Bollinger(period int, dev decimal.Decimal, shift int) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) Stochastic(k, d, slowing, shift int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) CCI(period, shift int) decimal.Decimal      { return decimal.Zero }
func (m *mockIndicatorSet) ADX(period, shift int) decimal.Decimal      { return decimal.Zero }
func (m *mockIndicatorSet) MFI(period, shift int) decimal.Decimal      { return decimal.Zero }
func (m *mockIndicatorSet) OBV(shift int) decimal.Decimal              { return decimal.Zero }
func (m *mockIndicatorSet) SAR(step, max decimal.Decimal, shift int) decimal.Decimal {
	return decimal.Zero
}
func (m *mockIndicatorSet) StdDev(period, shift int) decimal.Decimal { return decimal.Zero }
func (m *mockIndicatorSet) WPR(period, shift int) decimal.Decimal    { return decimal.Zero }
func (m *mockIndicatorSet) Momentum(period, shift int) decimal.Decimal {
	return decimal.Zero
}
func (m *mockIndicatorSet) ICustom(name string, params []decimal.Decimal, buffer, shift int) decimal.Decimal {
	return decimal.Zero
}
func (m *mockIndicatorSet) Alligator(jaw, jawS, teeth, teethS, lips, lipsS int, method string, ap, shift int) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) Ichimoku(tenkan, kijun, senkou, shift int) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) Envelopes(period int, dev decimal.Decimal, method string, ap, shift int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) DeMarker(period, shift int) decimal.Decimal { return decimal.Zero }
func (m *mockIndicatorSet) OsMA(fast, slow, signal, ap, shift int) decimal.Decimal {
	return decimal.Zero
}
func (m *mockIndicatorSet) RVI(period, shift int) decimal.Decimal { return decimal.Zero }
func (m *mockIndicatorSet) Force(period int, method string, ap, shift int) decimal.Decimal {
	return decimal.Zero
}
func (m *mockIndicatorSet) Fractals(shift int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) Gator(jaw, jawS, teeth, teethS, lips, lipsS int, method string, ap, shift int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (m *mockIndicatorSet) AC(shift int) decimal.Decimal                          { return decimal.Zero }
func (m *mockIndicatorSet) AD(shift int) decimal.Decimal                          { return decimal.Zero }
func (m *mockIndicatorSet) AO(shift int) decimal.Decimal                          { return decimal.Zero }
func (m *mockIndicatorSet) BearsPower(period int, ap, shift int) decimal.Decimal  { return decimal.Zero }
func (m *mockIndicatorSet) BullsPower(period int, ap, shift int) decimal.Decimal  { return decimal.Zero }
func (m *mockIndicatorSet) BWMFI(shift int) decimal.Decimal                      { return decimal.Zero }
func (m *mockIndicatorSet) AMA(period, fast, slow, shift int) decimal.Decimal    { return decimal.Zero }
func (m *mockIndicatorSet) DEMA(period, shift int) decimal.Decimal               { return decimal.Zero }
func (m *mockIndicatorSet) TEMA(period, shift int) decimal.Decimal               { return decimal.Zero }
func (m *mockIndicatorSet) FrAMA(period, shift int) decimal.Decimal              { return decimal.Zero }
func (m *mockIndicatorSet) VIDyA(cmoP, cmoS, maP, maS, shift int) decimal.Decimal { return decimal.Zero }
func (m *mockIndicatorSet) TriX(period, shift int) decimal.Decimal               { return decimal.Zero }
func (m *mockIndicatorSet) ADXWilder(period, shift int) decimal.Decimal          { return decimal.Zero }
func (m *mockIndicatorSet) Chaikin(fast, slow, shift int) decimal.Decimal        { return decimal.Zero }
func (m *mockIndicatorSet) Volumes(shift int) decimal.Decimal                    { return decimal.Zero }

// mockContextWithIndicators extends mockContext with indicator support.
type mockContextWithIndicators struct {
	mockContext
	ind sdk.IndicatorSet
}

func (m *mockContextWithIndicators) Indicators() sdk.IndicatorSet { return m.ind }

func TestInterpreter_OnBar_MQL4_Simple(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{
				Kind: StmtExpr,
				Expr: &Expr{
					Kind: ExprCall,
					Name: "OrderSend",
					Args: []Expr{
						{Kind: ExprCall, Name: "Symbol"},
						{Kind: ExprConst, Name: "OP_BUY"},
						{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(0.1))},
						{Kind: ExprCall, Name: "Ask"},
						{Kind: ExprLiteral, Val: IntVal(10)},
						{Kind: ExprLiteral, Val: DecimalVal(decimal.Zero)},
						{Kind: ExprLiteral, Val: DecimalVal(decimal.Zero)},
						{Kind: ExprLiteral, Val: StringVal("test")},
						{Kind: ExprLiteral, Val: IntVal(12345)},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	broker := &mockBroker{}
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: broker,
	}

	err := it.OnInit(ctx)
	if err != nil {
		t.Fatalf("OnInit failed: %v", err)
	}

	sig, err := it.OnBar(ctx, "H1")
	if err != nil {
		t.Fatalf("OnBar failed: %v", err)
	}

	// Should have sent an order
	if broker.lastReq.Symbol != "EURUSD" {
		t.Errorf("OrderSend symbol = %s, want EURUSD", broker.lastReq.Symbol)
	}
	if broker.lastReq.Side != sdk.SideBuy {
		t.Errorf("OrderSend side = %v, want SideBuy", broker.lastReq.Side)
	}
	if !broker.lastReq.Volume.Equal(decimal.NewFromFloat(0.1)) {
		t.Errorf("OrderSend volume = %s, want 0.1", broker.lastReq.Volume)
	}
	if broker.lastReq.Magic != 12345 {
		t.Errorf("OrderSend magic = %d, want 12345", broker.lastReq.Magic)
	}

	_ = sig // signal may be nil (OrderSend doesn't set signal in current impl)
}

func TestInterpreter_OnBar_IfCondition(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{
				Kind: StmtIf,
				Cond: &Expr{
					Kind: ExprBinary,
					Op:   ">",
					Args: []Expr{
						{Kind: ExprSubscript, Name: "Close", Index: &Expr{Kind: ExprLiteral, Val: IntVal(1)}},
						{Kind: ExprCall, Name: "iMA", Args: []Expr{
							{Kind: ExprLiteral, Val: IntVal(14)},
							{Kind: ExprLiteral, Val: IntVal(0)},
							{Kind: ExprLiteral, Val: IntVal(0)}, // MODE_SMA
						}},
					},
				},
				Body: []Statement{
					{
						Kind: StmtExpr,
						Expr: &Expr{
							Kind: ExprCall,
							Name: "OrderSend",
							Args: []Expr{
								{Kind: ExprCall, Name: "Symbol"},
								{Kind: ExprConst, Name: "OP_BUY"},
								{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(0.1))},
								{Kind: ExprCall, Name: "Ask"},
								{Kind: ExprLiteral, Val: IntVal(10)},
								{Kind: ExprLiteral, Val: DecimalVal(decimal.Zero)},
								{Kind: ExprLiteral, Val: DecimalVal(decimal.Zero)},
								{Kind: ExprLiteral, Val: StringVal("buy")},
								{Kind: ExprLiteral, Val: IntVal(12345)},
							},
						},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	broker := &mockBroker{}
	ctx := &mockContextWithIndicators{
		mockContext: mockContext{
			bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1200)}},
			broker: broker,
		},
		ind: &mockIndicatorSet{},
	}

	err := it.OnInit(ctx)
	if err != nil {
		t.Fatalf("OnInit failed: %v", err)
	}

	_, err = it.OnBar(ctx, "H1")
	if err != nil {
		t.Fatalf("OnBar failed: %v", err)
	}

	// Close[1] = 1.1200, iMA returns 1.1050, so 1.1200 > 1.1050 → should buy
	if broker.lastReq.Side != sdk.SideBuy {
		t.Errorf("Expected buy order, got side=%v", broker.lastReq.Side)
	}
}

func TestInterpreter_OnBar_ForLoop(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{
				Kind: StmtFor,
				Init: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprAssignment, Name: "i", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}},
				},
				Cond: &Expr{
					Kind: ExprBinary,
					Op:   "<",
					Args: []Expr{
						{Kind: ExprVar, Name: "i"},
						{Kind: ExprCall, Name: "OrdersTotal"},
					},
				},
				Update: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprUpdate, Name: "i", Op: "++"},
				},
				Body: []Statement{
					{
						Kind: StmtExpr,
						Expr: &Expr{
							Kind: ExprCall,
							Name: "OrderSelect",
							Args: []Expr{
								{Kind: ExprVar, Name: "i"},
								{Kind: ExprConst, Name: "SELECT_BY_POS"},
								{Kind: ExprConst, Name: "MODE_TRADES"},
							},
						},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	broker := &mockBroker{
		positions: []sdk.Position{
			{Ticket: 100, Symbol: "EURUSD", Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.1), OpenTime: time.Now()},
			{Ticket: 200, Symbol: "GBPUSD", Side: sdk.SideSell, Volume: decimal.NewFromFloat(0.2), OpenTime: time.Now()},
		},
	}
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: broker,
	}

	err := it.OnInit(ctx)
	if err != nil {
		t.Fatalf("OnInit failed: %v", err)
	}

	_, err = it.OnBar(ctx, "H1")
	if err != nil {
		t.Fatalf("OnBar failed: %v", err)
	}

	// After the loop, the pool should have been iterated
	// The last OrderSelect should have selected index 1 (second position)
	if it.orderPool.Ticket() != 200 {
		t.Errorf("After loop, selected ticket = %d, want 200", it.orderPool.Ticket())
	}
}

func TestInterpreter_MarketData(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{
				Kind: StmtExpr,
				Expr: &Expr{Kind: ExprAssignment, Name: "ask", Args: []Expr{{Kind: ExprCall, Name: "Ask"}}},
			},
			{
				Kind: StmtExpr,
				Expr: &Expr{Kind: ExprAssignment, Name: "bid", Args: []Expr{{Kind: ExprCall, Name: "Bid"}}},
			},
			{
				Kind: StmtExpr,
				Expr: &Expr{Kind: ExprAssignment, Name: "sym", Args: []Expr{{Kind: ExprCall, Name: "Symbol"}}},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}

	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	askVal := it.getVar("ask")
	if !askVal.ToDecimal().Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("ask = %s, want 1.1000", askVal.ToDecimal())
	}

	bidVal := it.getVar("bid")
	if !bidVal.ToDecimal().Equal(decimal.NewFromFloat(1.0998)) {
		t.Errorf("bid = %s, want 1.0998", bidVal.ToDecimal())
	}

	symVal := it.getVar("sym")
	if symVal.ToString() != "EURUSD" {
		t.Errorf("symbol = %s, want EURUSD", symVal.ToString())
	}
}
