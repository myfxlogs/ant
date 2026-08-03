package runner

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// mockExecutor is a test OrderExecutor that records calls and returns canned responses.
type mockExecutor struct {
	orders        []sdk.Position
	pendingOrders []sdk.PendingOrder
	accountInfo   sdk.AccountInfo
	symbolInfo    sdk.SymbolInfo
	placeErr      error
	closeErr      error
	modifyErr     error
	cancelErr     error

	lastPlace struct {
		Symbol string
		Side   sdk.PositionSide
		Type   sdk.OrderType
		Volume decimal.Decimal
		Price  decimal.Decimal
		SL     decimal.Decimal
		TP     decimal.Decimal
	}
}

func (m *mockExecutor) PlaceOrder(ctx context.Context, symbol string, side sdk.PositionSide,
	orderType sdk.OrderType, volume, price, sl, tp decimal.Decimal,
	comment string, magic int32) (int64, error) {
	m.lastPlace.Symbol = symbol
	m.lastPlace.Side = side
	m.lastPlace.Type = orderType
	m.lastPlace.Volume = volume
	m.lastPlace.Price = price
	m.lastPlace.SL = sl
	m.lastPlace.TP = tp
	if m.placeErr != nil {
		return 0, m.placeErr
	}
	return 42, nil
}

func (m *mockExecutor) CloseOrder(ctx context.Context, ticket int64, volume decimal.Decimal) error {
	return m.closeErr
}

func (m *mockExecutor) ModifyOrder(ctx context.Context, ticket int64, sl, tp decimal.Decimal) error {
	return m.modifyErr
}

func (m *mockExecutor) CancelOrder(ctx context.Context, ticket int64) error {
	return m.cancelErr
}

func (m *mockExecutor) OpenedOrders(ctx context.Context) ([]sdk.Position, error) {
	return m.orders, nil
}

func (m *mockExecutor) PendingOrders(ctx context.Context) ([]sdk.PendingOrder, error) {
	return m.pendingOrders, nil
}

func (m *mockExecutor) Account() sdk.AccountInfo {
	return m.accountInfo
}

func (m *mockExecutor) SymbolInfo(symbol string) (sdk.SymbolInfo, error) {
	return m.symbolInfo, nil
}

// --- Runner lifecycle tests ---

func TestNew_Defaults(t *testing.T) {
	r := New(Config{Symbol: "EURUSD", Timeframe: "M5"})
	if r.ctx.Symbol() != "EURUSD" {
		t.Errorf("Symbol() = %q, want EURUSD", r.ctx.Symbol())
	}
	if r.ctx.Timeframe() != "M5" {
		t.Errorf("Timeframe() = %q, want M5", r.ctx.Timeframe())
	}
}

func TestRunner_Init_NilStrategy(t *testing.T) {
	r := New(Config{})
	if err := r.Init(context.Background()); err != nil {
		t.Errorf("Init with nil strategy should return nil, got %v", err)
	}
}

func TestRunner_OnBar_NilStrategy(t *testing.T) {
	r := New(Config{})
	sig, err := r.OnBar(context.Background(), nil, "M5")
	if err != nil {
		t.Errorf("OnBar with nil strategy should return nil error, got %v", err)
	}
	if sig != nil {
		t.Errorf("OnBar with nil strategy should return nil signal")
	}
}

func TestRunner_Deinit_NilStrategy(t *testing.T) {
	r := New(Config{})
	if err := r.Deinit(context.Background(), "test"); err != nil {
		t.Errorf("Deinit with nil strategy should return nil, got %v", err)
	}
}

func TestRunner_OnTick_NonTickStrategy(t *testing.T) {
	r := New(Config{})
	r.SetStrategy(&barOnlyStrategy{})
	sig, err := r.OnTick(context.Background(), dec("1.1"), dec("1.2"))
	if err != nil {
		t.Errorf("OnTick on non-TickStrategy should return nil error, got %v", err)
	}
	if sig != nil {
		t.Errorf("OnTick on non-TickStrategy should return nil signal")
	}
}

func TestRunner_OnTrade_NonTradeStrategy(t *testing.T) {
	r := New(Config{})
	r.SetStrategy(&barOnlyStrategy{})
	sig, err := r.OnTrade(context.Background(), sdk.TradeEvent{})
	if err != nil {
		t.Errorf("OnTrade on non-TradeStrategy should return nil error, got %v", err)
	}
	if sig != nil {
		t.Errorf("OnTrade on non-TradeStrategy should return nil signal")
	}
}

func TestRunner_OnTimerTick_NonTimerStrategy(t *testing.T) {
	r := New(Config{})
	r.SetStrategy(&barOnlyStrategy{})
	sig, err := r.OnTimerTick(context.Background())
	if err != nil {
		t.Errorf("OnTimerTick on non-TimerStrategy should return nil error, got %v", err)
	}
	if sig != nil {
		t.Errorf("OnTimerTick on non-TimerStrategy should return nil signal")
	}
}

func TestRunner_OnTradeTransaction_NonImpl(t *testing.T) {
	r := New(Config{})
	r.SetStrategy(&barOnlyStrategy{})
	sig, err := r.OnTradeTransaction(context.Background())
	if err != nil {
		t.Errorf("OnTradeTransaction on non-impl should return nil error, got %v", err)
	}
	if sig != nil {
		t.Errorf("OnTradeTransaction on non-impl should return nil signal")
	}
}

func TestRunner_OnBookEvent_NonImpl(t *testing.T) {
	r := New(Config{})
	r.SetStrategy(&barOnlyStrategy{})
	sig, err := r.OnBookEvent(context.Background())
	if err != nil {
		t.Errorf("OnBookEvent on non-impl should return nil error, got %v", err)
	}
	if sig != nil {
		t.Errorf("OnBookEvent on non-impl should return nil signal")
	}
}

func TestRunner_HasOnTradeTransaction(t *testing.T) {
	r := New(Config{})
	r.SetStrategy(&barOnlyStrategy{})
	if r.HasOnTradeTransaction() {
		t.Error("HasOnTradeTransaction should be false for barOnlyStrategy")
	}
	if r.HasOnBookEvent() {
		t.Error("HasOnBookEvent should be false for barOnlyStrategy")
	}
}

func TestRunner_HasOnTradeTransaction_NilStrategy(t *testing.T) {
	r := New(Config{})
	if r.HasOnTradeTransaction() {
		t.Error("HasOnTradeTransaction should be false with nil strategy")
	}
	if r.HasOnBookEvent() {
		t.Error("HasOnBookEvent should be false with nil strategy")
	}
}

func TestRunner_UpdateLiveState(t *testing.T) {
	r := New(Config{})
	positions := []sdk.Position{{Ticket: 1, Symbol: "EURUSD"}}
	r.UpdateLiveState("10000", "10500", positions)

	// Verify via broker.Account() in harness mode (no executor).
	info := r.broker.Account()
	if !info.Balance.Equal(dec("10000")) {
		t.Errorf("Balance = %s, want 10000", info.Balance)
	}
	if !info.Equity.Equal(dec("10500")) {
		t.Errorf("Equity = %s, want 10500", info.Equity)
	}
}

func TestRunner_UpdateTickState(t *testing.T) {
	r := New(Config{})
	r.UpdateTickState(dec("1.1"), dec("1.2"))
	if !r.ctx.Ask().Equal(dec("1.2")) {
		t.Errorf("Ask() = %s, want 1.2", r.ctx.Ask())
	}
	if !r.ctx.Bid().Equal(dec("1.1")) {
		t.Errorf("Bid() = %s, want 1.1", r.ctx.Bid())
	}
}

func TestRunner_UpdateExtraBars(t *testing.T) {
	r := New(Config{Symbol: "EURUSD", Timeframe: "M5"})
	r.UpdateExtraBars(map[string][]sdk.Bar{
		"GBPUSD": {{Close: dec("1.3"), Timestamp: 1000}},
	})
	got := r.ctx.BarsForSymbol("GBPUSD", "")
	if got.Len() != 1 {
		t.Fatalf("BarsForSymbol(GBPUSD).Len() = %d, want 1", got.Len())
	}
	if !got.Close(0).Equal(dec("1.3")) {
		t.Errorf("BarsForSymbol(GBPUSD).Close(0) = %s, want 1.3", got.Close(0))
	}
}

// --- Broker tests ---

func TestBrokerImpl_OrderSend_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	_, err := r.broker.OrderSend(sdk.OrderRequest{Symbol: "EURUSD"})
	if err == nil {
		t.Error("OrderSend with no executor should return error")
	}
}

func TestBrokerImpl_OrderSend_Success(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	res, err := r.broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: dec("0.1"), Price: dec("1.1"),
	})
	if err != nil {
		t.Fatalf("OrderSend failed: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want %v", res.RetCode, sdk.RetDone)
	}
	if res.Ticket != 42 {
		t.Errorf("Ticket = %d, want 42", res.Ticket)
	}
	if exec.lastPlace.Symbol != "EURUSD" {
		t.Errorf("lastPlace.Symbol = %q, want EURUSD", exec.lastPlace.Symbol)
	}
}

func TestBrokerImpl_OrderSend_Error(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{placeErr: context.DeadlineExceeded}
	r.broker.executor = exec

	res, err := r.broker.OrderSend(sdk.OrderRequest{Symbol: "EURUSD"})
	if err == nil {
		t.Error("OrderSend should return error")
	}
	if res.RetCode != sdk.RetRejected {
		t.Errorf("RetCode = %v, want %v", res.RetCode, sdk.RetRejected)
	}
}

func TestBrokerImpl_PositionClose_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	_, err := r.broker.PositionClose(1, decimal.Zero)
	if err == nil {
		t.Error("PositionClose with no executor should return error")
	}
}

func TestBrokerImpl_PositionClose_Success(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	res, err := r.broker.PositionClose(1, dec("0.1"))
	if err != nil {
		t.Fatalf("PositionClose failed: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want %v", res.RetCode, sdk.RetDone)
	}
}

func TestBrokerImpl_PositionClose_Error(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{closeErr: context.Canceled}
	r.broker.executor = exec

	_, err := r.broker.PositionClose(1, decimal.Zero)
	if err == nil {
		t.Error("PositionClose should return error")
	}
}

func TestBrokerImpl_PositionCloseBy_Success(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	res, err := r.broker.PositionCloseBy(1, 2)
	if err != nil {
		t.Fatalf("PositionCloseBy failed: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want %v", res.RetCode, sdk.RetDone)
	}
}

func TestBrokerImpl_PositionCloseBy_Error(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{closeErr: context.Canceled}
	r.broker.executor = exec

	_, err := r.broker.PositionCloseBy(1, 2)
	if err == nil {
		t.Error("PositionCloseBy should return error on first close failure")
	}
}

func TestBrokerImpl_PositionModify_Success(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	res, err := r.broker.PositionModify(1, dec("1.0"), dec("2.0"))
	if err != nil {
		t.Fatalf("PositionModify failed: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want %v", res.RetCode, sdk.RetDone)
	}
}

func TestBrokerImpl_PositionModify_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	_, err := r.broker.PositionModify(1, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Error("PositionModify with no executor should return error")
	}
}

func TestBrokerImpl_PositionModify_Error(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{modifyErr: context.DeadlineExceeded}
	r.broker.executor = exec

	_, err := r.broker.PositionModify(1, dec("1.0"), dec("2.0"))
	if err == nil {
		t.Error("PositionModify should return error")
	}
}

func TestBrokerImpl_OrderDelete_Success(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	res, err := r.broker.OrderDelete(1)
	if err != nil {
		t.Fatalf("OrderDelete failed: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want %v", res.RetCode, sdk.RetDone)
	}
}

func TestBrokerImpl_OrderDelete_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	_, err := r.broker.OrderDelete(1)
	if err == nil {
		t.Error("OrderDelete with no executor should return error")
	}
}

func TestBrokerImpl_OrderDelete_Error(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{cancelErr: context.Canceled}
	r.broker.executor = exec

	_, err := r.broker.OrderDelete(1)
	if err == nil {
		t.Error("OrderDelete should return error")
	}
}

func TestBrokerImpl_Positions_NoExecutor_HarnessMode(t *testing.T) {
	r := New(Config{})
	r.UpdateLiveState("1000", "1100", []sdk.Position{
		{Ticket: 1, Magic: 100},
		{Ticket: 2, Magic: 200},
		{Ticket: 3, Magic: 100},
	})

	// magic=0 returns all.
	all := r.broker.Positions(0)
	if len(all) != 3 {
		t.Errorf("Positions(0) returned %d, want 3", len(all))
	}

	// magic=100 returns filtered.
	filtered := r.broker.Positions(100)
	if len(filtered) != 2 {
		t.Errorf("Positions(100) returned %d, want 2", len(filtered))
	}
}

func TestBrokerImpl_Positions_WithExecutor(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{
		orders: []sdk.Position{
			{Ticket: 1, Magic: 100},
			{Ticket: 2, Magic: 200},
		},
	}
	r.broker.executor = exec

	all := r.broker.Positions(0)
	if len(all) != 2 {
		t.Errorf("Positions(0) returned %d, want 2", len(all))
	}

	filtered := r.broker.Positions(100)
	if len(filtered) != 1 {
		t.Errorf("Positions(100) returned %d, want 1", len(filtered))
	}
}

func TestBrokerImpl_Orders_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	if r.broker.Orders(0) != nil {
		t.Error("Orders with no executor should return nil")
	}
}

func TestBrokerImpl_Orders_WithExecutor(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{
		pendingOrders: []sdk.PendingOrder{
			{Ticket: 1, Magic: 100},
			{Ticket: 2, Magic: 200},
		},
	}
	r.broker.executor = exec

	all := r.broker.Orders(0)
	if len(all) != 2 {
		t.Errorf("Orders(0) returned %d, want 2", len(all))
	}

	filtered := r.broker.Orders(100)
	if len(filtered) != 1 {
		t.Errorf("Orders(100) returned %d, want 1", len(filtered))
	}
}

func TestBrokerImpl_HistoryOrders_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	if r.broker.HistoryOrders(0, 0) != nil {
		t.Error("HistoryOrders with no executor should return nil")
	}
}

func TestBrokerImpl_Deals_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	if r.broker.Deals(0, 0, 0) != nil {
		t.Error("Deals with no executor should return nil")
	}
}

func TestBrokerImpl_SymbolInfo_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.broker.executor = nil
	si, err := r.broker.SymbolInfo("EURUSD")
	if err != nil {
		t.Errorf("SymbolInfo with no executor should not error, got %v", err)
	}
	if si.Spread != 0 {
		t.Errorf("SymbolInfo.Spread = %d, want 0 (zero value)", si.Spread)
	}
}

func TestBrokerImpl_SymbolInfo_WithExecutor(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{symbolInfo: sdk.SymbolInfo{Digits: 5, Spread: 10}}
	r.broker.executor = exec

	si, err := r.broker.SymbolInfo("EURUSD")
	if err != nil {
		t.Fatalf("SymbolInfo failed: %v", err)
	}
	if si.Digits != 5 {
		t.Errorf("Digits = %d, want 5", si.Digits)
	}
}

func TestBrokerImpl_Account_NoExecutor(t *testing.T) {
	r := New(Config{})
	r.UpdateLiveState("5000", "5500", nil)

	info := r.broker.Account()
	if !info.Balance.Equal(dec("5000")) {
		t.Errorf("Balance = %s, want 5000", info.Balance)
	}
	if !info.Equity.Equal(dec("5500")) {
		t.Errorf("Equity = %s, want 5500", info.Equity)
	}
}

func TestBrokerImpl_Account_WithExecutor(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{accountInfo: sdk.AccountInfo{
		Balance: dec("20000"), Equity: dec("21000"), Leverage: 100,
	}}
	r.broker.executor = exec

	info := r.broker.Account()
	if !info.Balance.Equal(dec("20000")) {
		t.Errorf("Balance = %s, want 20000", info.Balance)
	}
	if info.Leverage != 100 {
		t.Errorf("Leverage = %d, want 100", info.Leverage)
	}
}

func TestMustDecimal_Empty(t *testing.T) {
	r := New(Config{})
	if !r.broker.mustDecimal("").Equal(decimal.Zero) {
		t.Error("mustDecimal(\"\") should return zero")
	}
}

func TestMustDecimal_Invalid(t *testing.T) {
	r := New(Config{})
	if !r.broker.mustDecimal("abc").Equal(decimal.Zero) {
		t.Error("mustDecimal(\"abc\") should return zero")
	}
}

func TestMustDecimal_Valid(t *testing.T) {
	r := New(Config{})
	got := r.broker.mustDecimal("123.45")
	if !got.Equal(dec("123.45")) {
		t.Errorf("mustDecimal(\"123.45\") = %s, want 123.45", got)
	}
}

// --- Context tests ---

func TestContextImpl_Param(t *testing.T) {
	r := New(Config{Params: map[string]string{"foo": "bar"}})
	if got := r.ctx.Param("foo", "default"); got != "bar" {
		t.Errorf("Param(foo) = %v, want bar", got)
	}
	if got := r.ctx.Param("missing", "default"); got != "default" {
		t.Errorf("Param(missing) = %v, want default", got)
	}
}

func TestContextImpl_ParamDecimal(t *testing.T) {
	r := New(Config{Params: map[string]string{"lot": "0.5"}})
	got := r.ctx.ParamDecimal("lot", decimal.Zero)
	if !got.Equal(dec("0.5")) {
		t.Errorf("ParamDecimal(lot) = %s, want 0.5", got)
	}

	got = r.ctx.ParamDecimal("missing", dec("1.0"))
	if !got.Equal(dec("1.0")) {
		t.Errorf("ParamDecimal(missing) = %s, want 1.0", got)
	}

	// Invalid decimal falls back to default.
	r2 := New(Config{Params: map[string]string{"bad": "xyz"}})
	got = r2.ctx.ParamDecimal("bad", dec("2.0"))
	if !got.Equal(dec("2.0")) {
		t.Errorf("ParamDecimal(bad) = %s, want 2.0", got)
	}
}

func TestContextImpl_ParamInt(t *testing.T) {
	r := New(Config{Params: map[string]string{"period": "14"}})
	got := r.ctx.ParamInt("period", 7)
	if got != 14 {
		t.Errorf("ParamInt(period) = %d, want 14", got)
	}

	got = r.ctx.ParamInt("missing", 7)
	if got != 7 {
		t.Errorf("ParamInt(missing) = %d, want 7", got)
	}
}

func TestContextImpl_ParamString(t *testing.T) {
	r := New(Config{Params: map[string]string{"name": "test"}})
	if got := r.ctx.ParamString("name", "default"); got != "test" {
		t.Errorf("ParamString(name) = %q, want test", got)
	}
	if got := r.ctx.ParamString("missing", "default"); got != "default" {
		t.Errorf("ParamString(missing) = %q, want default", got)
	}
}

func TestContextImpl_ParamBool(t *testing.T) {
	r := New(Config{Params: map[string]string{"flag": "true", "num": "1"}})
	if !r.ctx.ParamBool("flag", false) {
		t.Error("ParamBool(flag) = false, want true")
	}
	if !r.ctx.ParamBool("num", false) {
		t.Error("ParamBool(num) = false, want true")
	}
	if r.ctx.ParamBool("missing", false) {
		t.Error("ParamBool(missing) = true, want false")
	}
	if !r.ctx.ParamBool("missing", true) {
		t.Error("ParamBool(missing, true) = false, want true")
	}
}

func TestContextImpl_BarsTF(t *testing.T) {
	r := New(Config{Symbol: "EURUSD", Timeframe: "M5"})
	bars := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.1")}})
	r.ctx.setBars(bars)

	got := r.ctx.BarsTF("H1")
	if got.Len() != 1 {
		t.Errorf("BarsTF(H1).Len() = %d, want 1 (falls back to primary)", got.Len())
	}
}

func TestContextImpl_BarsForSymbol_Primary(t *testing.T) {
	r := New(Config{Symbol: "EURUSD", Timeframe: "M5"})
	bars := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.1")}})
	r.ctx.setBars(bars)

	got := r.ctx.BarsForSymbol("EURUSD", "M5")
	if got.Len() != 1 {
		t.Errorf("BarsForSymbol(EURUSD, M5).Len() = %d, want 1", got.Len())
	}

	// Empty symbol falls back to primary.
	got = r.ctx.BarsForSymbol("", "")
	if got.Len() != 1 {
		t.Errorf("BarsForSymbol(\"\", \"\").Len() = %d, want 1", got.Len())
	}
}

func TestContextImpl_BarsForSymbol_UnknownSymbol(t *testing.T) {
	r := New(Config{Symbol: "EURUSD"})
	got := r.ctx.BarsForSymbol("UNKNOWN", "")
	if got.Len() != 0 {
		t.Errorf("BarsForSymbol(UNKNOWN).Len() = %d, want 0", got.Len())
	}
}

func TestContextImpl_PointAndDigits(t *testing.T) {
	r := New(Config{Symbol: "EURUSD"})
	exec := &mockExecutor{symbolInfo: sdk.SymbolInfo{Digits: 5, Point: dec("0.00001")}}
	r.broker.executor = exec

	if !r.ctx.Point().Equal(dec("0.00001")) {
		t.Errorf("Point() = %s, want 0.00001", r.ctx.Point())
	}
	if r.ctx.Digits() != 5 {
		t.Errorf("Digits() = %d, want 5", r.ctx.Digits())
	}
}

func TestContextImpl_Pip(t *testing.T) {
	r := New(Config{Symbol: "EURUSD"})
	exec := &mockExecutor{symbolInfo: sdk.SymbolInfo{Point: dec("0.00001")}}
	r.broker.executor = exec

	if !r.ctx.Pip().Equal(dec("0.0001")) {
		t.Errorf("Pip() = %s, want 0.0001", r.ctx.Pip())
	}
}

func TestContextImpl_AskBid_FromTick(t *testing.T) {
	r := New(Config{})
	r.ctx.setTick(dec("1.1"), dec("1.2"))

	if !r.ctx.Bid().Equal(dec("1.1")) {
		t.Errorf("Bid() = %s, want 1.1", r.ctx.Bid())
	}
	if !r.ctx.Ask().Equal(dec("1.2")) {
		t.Errorf("Ask() = %s, want 1.2", r.ctx.Ask())
	}
}

func TestContextImpl_AskBid_FromBars(t *testing.T) {
	r := New(Config{})
	bars := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.5")}})
	r.ctx.setBars(bars)

	if !r.ctx.Ask().Equal(dec("1.5")) {
		t.Errorf("Ask() from bars = %s, want 1.5", r.ctx.Ask())
	}
	if !r.ctx.Bid().Equal(dec("1.5")) {
		t.Errorf("Bid() from bars = %s, want 1.5", r.ctx.Bid())
	}
}

func TestContextImpl_AskBid_Empty(t *testing.T) {
	r := New(Config{})
	if !r.ctx.Ask().Equal(decimal.Zero) {
		t.Errorf("Ask() with no data = %s, want 0", r.ctx.Ask())
	}
	if !r.ctx.Bid().Equal(decimal.Zero) {
		t.Errorf("Bid() with no data = %s, want 0", r.ctx.Bid())
	}
}

func TestContextImpl_Spread(t *testing.T) {
	r := New(Config{})
	r.ctx.setTick(dec("1.1"), dec("1.3"))
	if !r.ctx.Spread().Equal(dec("0.2")) {
		t.Errorf("Spread() = %s, want 0.2", r.ctx.Spread())
	}
}

func TestContextImpl_Account(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{accountInfo: sdk.AccountInfo{Balance: dec("1000"), Mode: sdk.ModeHedging}}
	r.broker.executor = exec

	info := r.ctx.Account()
	if !info.Balance.Equal(dec("1000")) {
		t.Errorf("Account().Balance = %s, want 1000", info.Balance)
	}
}

func TestContextImpl_Mode(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{accountInfo: sdk.AccountInfo{Mode: sdk.ModeNetting}}
	r.broker.executor = exec

	if r.ctx.Mode() != sdk.ModeNetting {
		t.Errorf("Mode() = %v, want %v", r.ctx.Mode(), sdk.ModeNetting)
	}
}

func TestContextImpl_SetKillTimer(t *testing.T) {
	r := New(Config{})
	r.ctx.SetTimer(10)
	if !r.ctx.timerSet {
		t.Error("SetTimer should set timerSet to true")
	}
	r.ctx.KillTimer()
	if r.ctx.timerSet {
		t.Error("KillTimer should set timerSet to false")
	}
}

func TestContextImpl_ServerTime(t *testing.T) {
	r := New(Config{})
	bars := sdk.BarsToSlice([]sdk.Bar{{Timestamp: 9999}})
	r.ctx.setBars(bars)

	if r.ctx.ServerTime() != 9999 {
		t.Errorf("ServerTime() = %d, want 9999", r.ctx.ServerTime())
	}
}

func TestContextImpl_ServerTime_Empty(t *testing.T) {
	r := New(Config{})
	if r.ctx.ServerTime() != 0 {
		t.Errorf("ServerTime() with no bars = %d, want 0", r.ctx.ServerTime())
	}
}

func TestContextImpl_GoContext(t *testing.T) {
	r := New(Config{})
	if r.ctx.GoContext() != context.Background() {
		t.Error("GoContext() should default to context.Background()")
	}

	ctx := context.WithValue(context.Background(), ctxKey{}, "test")
	r.ctx.setGoContext(ctx)
	got := r.ctx.GoContext()
	if v, ok := got.Value(ctxKey{}).(string); !ok || v != "test" {
		t.Errorf("GoContext() should return set context")
	}
}

// --- Full lifecycle integration test ---

func TestRunner_FullLifecycle(t *testing.T) {
	r := New(Config{
		Symbol:    "EURUSD",
		Timeframe: "M5",
		Params:    map[string]string{"period": "14"},
	})
	exec := &mockExecutor{
		symbolInfo:  sdk.SymbolInfo{Digits: 5, Point: dec("0.00001")},
		accountInfo: sdk.AccountInfo{Balance: dec("10000"), Equity: dec("10000")},
	}
	r.broker.executor = exec

	strat := &barOnlyStrategy{}
	r.SetStrategy(strat)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	bars := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.1"), Timestamp: 1000}})
	sig, err := r.OnBar(context.Background(), bars, "M5")
	if err != nil {
		t.Fatalf("OnBar failed: %v", err)
	}
	if sig != nil {
		t.Errorf("OnBar signal = %v, want nil", sig)
	}

	if err := r.Deinit(context.Background(), "test"); err != nil {
		t.Fatalf("Deinit failed: %v", err)
	}

	if !strat.initCalled {
		t.Error("OnInit was not called")
	}
	if !strat.barCalled {
		t.Error("OnBar was not called")
	}
	if !strat.deinitCalled {
		t.Error("OnDeinit was not called")
	}
}

// --- Test helpers ---

type ctxKey struct{}

type barOnlyStrategy struct {
	initCalled   bool
	barCalled    bool
	deinitCalled bool
}

func (s *barOnlyStrategy) OnInit(ctx sdk.Context) error {
	s.initCalled = true
	return nil
}

func (s *barOnlyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	s.barCalled = true
	return nil, nil
}

func (s *barOnlyStrategy) OnDeinit(ctx sdk.Context, reason string) error {
	s.deinitCalled = true
	return nil
}

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}
