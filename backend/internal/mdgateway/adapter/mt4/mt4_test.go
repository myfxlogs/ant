package mt4

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNew(t *testing.T) {
	t.Parallel()
	cfg := mdtick.AccountConfig{AccountID: "acct-1", Platform: "mt4"}
	gw := New(cfg, zap.NewNop())
	if gw == nil {
		t.Fatal("New returned nil")
	}
	if gw.Platform() != "mt4" {
		t.Errorf("Platform() = %q, want %q", gw.Platform(), "mt4")
	}
	if gw.AccountID() != "acct-1" {
		t.Errorf("AccountID() = %q, want %q", gw.AccountID(), "acct-1")
	}
}

func TestHealthCheck_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if err := gw.HealthCheck(context.Background()); err == nil {
		t.Error("HealthCheck should fail when not connected")
	}
}

func TestSessionID_Empty(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if gw.SessionID() != "" {
		t.Errorf("SessionID() = %q, want empty", gw.SessionID())
	}
}

func TestMT4Client_Nil(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if gw.MT4Client() != nil {
		t.Error("MT4Client() should be nil when not connected")
	}
}

func TestStrToInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"digits only", "12345", 12345},
		{"with letters", "abc999def", 999},
		{"empty", "", 0},
		{"no digits", "abc", 0},
		{"mixed", "user42", 42},
		{"zero", "0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strToInt(tt.input)
			if got != tt.want {
				t.Errorf("strToInt(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestMinDuration(t *testing.T) {
	t.Parallel()
	if got := minDuration(1*time.Second, 5*time.Minute); got != 1*time.Second {
		t.Errorf("minDuration(1s, 5m) = %v, want 1s", got)
	}
	if got := minDuration(10*time.Second, 2*time.Second); got != 2*time.Second {
		t.Errorf("minDuration(10s, 2s) = %v, want 2s", got)
	}
	if got := minDuration(1*time.Second, 1*time.Second); got != 1*time.Second {
		t.Errorf("minDuration(1s, 1s) = %v, want 1s", got)
	}
}

func TestPlaceOrder_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.PlaceOrder(context.Background(), nil)
	if err == nil {
		t.Error("PlaceOrder should return error")
	}
}

func TestCloseOrder_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.CloseOrder(context.Background(), 0, decimal.Decimal{})
	if err == nil {
		t.Error("CloseOrder should return error")
	}
}

func TestModifyOrder_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.ModifyOrder(context.Background(), 0, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{})
	if err == nil {
		t.Error("ModifyOrder should return error")
	}
}

func TestFetchSymbolParams_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchSymbolParams(context.Background(), nil)
	if err == nil {
		t.Error("FetchSymbolParams should return error")
	}
}

func TestSubscribeOrderEvents_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.SubscribeOrderEvents(context.Background(), nil)
	if err == nil {
		t.Error("SubscribeOrderEvents should return error")
	}
}

func TestFetchOpenedOrders_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchOpenedOrders(context.Background())
	if err == nil {
		t.Error("FetchOpenedOrders should fail when not connected")
	}
}

func TestFetchOrderHistory_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchOrderHistory(context.Background(), time.Now(), time.Now())
	if err == nil {
		t.Error("FetchOrderHistory should fail when not connected")
	}
}

func TestSubscribe_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.Subscribe(context.Background(), nil, nil)
	if err == nil {
		t.Error("Subscribe should fail when not connected")
	}
}

func TestSubscribeProfit_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.SubscribeProfit(context.Background(), nil)
	if err == nil {
		t.Error("SubscribeProfit should fail when not connected")
	}
}

func TestSubscribeOrderUpdate_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.SubscribeOrderUpdate(context.Background(), nil)
	if err == nil {
		t.Error("SubscribeOrderUpdate should fail when not connected")
	}
}

func TestMT4UpdateActionLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		action pb.UpdateAction
		want   string
	}{
		{"position open", pb.UpdateAction_UpdateAction_PositionOpen, "open"},
		{"position close", pb.UpdateAction_UpdateAction_PositionClose, "close"},
		{"position modify", pb.UpdateAction_UpdateAction_PositionModify, "modify"},
		{"pending open", pb.UpdateAction_UpdateAction_PendingOpen, "pending_open"},
		{"pending close", pb.UpdateAction_UpdateAction_PendingClose, "pending_close"},
		{"pending modify", pb.UpdateAction_UpdateAction_PendingModify, "pending_modify"},
		{"pending fill", pb.UpdateAction_UpdateAction_PendingFill, "open"},
		{"balance", pb.UpdateAction_UpdateAction_Balance, "unknown"},
		{"credit", pb.UpdateAction_UpdateAction_Credit, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt4UpdateActionLabel(tt.action); got != tt.want {
				t.Errorf("mt4UpdateActionLabel(%v) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestMT4OrderOpLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   pb.Op
		want string
	}{
		{"buy", pb.Op_Op_Buy, "buy"},
		{"sell", pb.Op_Op_Sell, "sell"},
		{"buy limit", pb.Op_Op_BuyLimit, "buy_limit"},
		{"sell limit", pb.Op_Op_SellLimit, "sell_limit"},
		{"buy stop", pb.Op_Op_BuyStop, "buy_stop"},
		{"sell stop", pb.Op_Op_SellStop, "sell_stop"},
		{"balance (defaults to buy)", pb.Op_Op_Balance, "buy"},
		{"credit (defaults to buy)", pb.Op_Op_Credit, "buy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt4OrderOpLabel(tt.op); got != tt.want {
				t.Errorf("mt4OrderOpLabel(%v) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestMT4PeriodToTimeframe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   pb.Timeframe
		ok     bool
	}{
		{"1m", pb.Timeframe_Timeframe_M1, true},
		{"5m", pb.Timeframe_Timeframe_M5, true},
		{"15m", pb.Timeframe_Timeframe_M15, true},
		{"30m", pb.Timeframe_Timeframe_M30, true},
		{"1h", pb.Timeframe_Timeframe_H1, true},
		{"4h", pb.Timeframe_Timeframe_H4, true},
		{"1d", pb.Timeframe_Timeframe_D1, true},
		{"2m", 0, false},
		{"1w", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got, ok := mt4PeriodToTimeframe(tt.period)
			if ok != tt.ok {
				t.Errorf("mt4PeriodToTimeframe(%q) ok=%v, want %v", tt.period, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("mt4PeriodToTimeframe(%q) = %v, want %v", tt.period, got, tt.want)
			}
		})
	}
}

func TestConvertMT4Bars_Empty(t *testing.T) {
	t.Parallel()
	bars := convertMT4Bars(nil, "acct-1", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from nil, got %d", len(bars))
	}
	bars = convertMT4Bars([]*pb.Bar{}, "acct-1", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from empty, got %d", len(bars))
	}
}

func TestConvertMT4Bars_WithData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, Open: 1.1000, High: 1.1050, Low: 1.0990, Close: 1.1020, Volume: 100},
		{Time: ts, Open: 1.1020, High: 1.1080, Low: 1.1010, Close: 1.1060, Volume: 200},
	}
	bars := convertMT4Bars(pbBars, "acct-1", "1h")
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].AccountID != "acct-1" {
		t.Errorf("AccountID = %q, want acct-1", bars[0].AccountID)
	}
	if bars[0].Period != "1h" {
		t.Errorf("Period = %q, want 1h", bars[0].Period)
	}
	if !bars[0].Open.Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("Open = %s, want 1.1000", bars[0].Open)
	}
	if !bars[0].High.Equal(decimal.NewFromFloat(1.1050)) {
		t.Errorf("High = %s, want 1.1050", bars[0].High)
	}
	if !bars[0].Close.Equal(decimal.NewFromFloat(1.1020)) {
		t.Errorf("Close = %s, want 1.1020", bars[0].Close)
	}
	if bars[0].Volume != 100 {
		t.Errorf("Volume = %f, want 100", bars[0].Volume)
	}
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
		{"", 60_000},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			if got := periodMs(tt.period); got != tt.want {
				t.Errorf("periodMs(%q) = %d, want %d", tt.period, got, tt.want)
			}
		})
	}
}

func TestGetPriceHistory_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.GetPriceHistory(context.Background(), "acct-1", "EURUSD", "1h", 0, 3600_000)
	if err == nil {
		t.Error("GetPriceHistory should fail when not connected")
	}
}

func TestDisconnect_NilConn(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if err := gw.Disconnect(context.Background()); err != nil {
		t.Errorf("Disconnect on nil conn should not error, got %v", err)
	}
}

func TestDisconnect_FullState(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{Login: "123", Password: "p", BrokerHost: "h"}, zap.NewNop())
	gw.client = &mockMT4Client{}
	gw.connCli = &mockConnCli{}
	gw.streamCli = &mockStreamsClient{}
	gw.subCli = &mockSubCli{}
	gw.sessionID = "sid"
	ctx1, c1 := context.WithCancel(context.Background())
	ctx2, c2 := context.WithCancel(context.Background())
	ctx3, c3 := context.WithCancel(context.Background())
	gw.cancelSub = c1
	gw.cancelProfitSub = c2
	gw.cancelOrderUpdateSub = c3
	if err := gw.Disconnect(context.Background()); err != nil {
		t.Errorf("Disconnect should not error: %v", err)
	}
	checkCancelled := func(name string, ctx context.Context) {
		select {
		case <-ctx.Done():
		default:
			t.Errorf("%s should be cancelled", name)
		}
	}
	checkCancelled("cancelSub", ctx1)
	checkCancelled("cancelProfitSub", ctx2)
	checkCancelled("cancelOrderUpdateSub", ctx3)
	if gw.client != nil {
		t.Error("client should be nil after Disconnect")
	}
	if gw.sessionID != "" {
		t.Error("sessionID should be empty after Disconnect")
	}
}

func TestEnsureConnected_AlreadySet(t *testing.T) {
	t.Parallel()
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	gw.conn = &cc
	bo := 100 * time.Millisecond
	if err := gw.ensureConnected(context.Background(), &bo, time.Second); err != nil {
		t.Errorf("ensureConnected should succeed when conn is set: %v", err)
	}
}

func TestGetPriceHistory_UnsupportedPeriod(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = &mockMT4Client{}
	_, err := gw.GetPriceHistory(context.Background(), "acct-1", "EURUSD", "2w", 0, 3600_000)
	if err == nil {
		t.Error("GetPriceHistory should fail for unsupported period")
	}
}

func TestGetPriceHistory_WithMockData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	mock := &mockMT4Client{
		quoteHistoryRes: &pb.QuoteHistoryReply{
			Result: []*pb.Bar{
				{Time: ts, Open: 1.1000, High: 1.1050, Low: 1.0990, Close: 1.1020, Volume: 100},
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	bars, err := gw.GetPriceHistory(context.Background(), "acct-1", "EURUSD", "1h", 0, 3600_000)
	if err != nil {
		t.Fatalf("GetPriceHistory: %v", err)
	}
	if len(bars) != 1 {
		t.Fatalf("expected 1 bar, got %d", len(bars))
	}
	if bars[0].AccountID != "acct-1" {
		t.Errorf("AccountID = %q, want acct-1", bars[0].AccountID)
	}
}

func TestGetPriceHistory_MockError(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{quoteHistoryErr: fmt.Errorf("mtapi error")}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	_, err := gw.GetPriceHistory(context.Background(), "acct-1", "EURUSD", "1h", 0, 3600_000)
	if err == nil {
		t.Error("GetPriceHistory should propagate mock error")
	}
}

func TestGetPriceHistory_ErrorCode(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{
		quoteHistoryRes: &pb.QuoteHistoryReply{
			Error: &pb.Error{Code: 1, Message: "bad request"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	_, err := gw.GetPriceHistory(context.Background(), "acct-1", "EURUSD", "1h", 0, 3600_000)
	if err == nil {
		t.Error("GetPriceHistory should fail when mtapi returns error code")
	}
}

func TestFetchOpenedOrders_WithMockData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	mock := &mockMT4Client{
		openedOrdersRes: &pb.OpenedOrdersReply{
			Result: []*pb.Order{
				{
					Ticket:     1001,
					Symbol:     "EURUSD",
					Type:       pb.Op_Op_Buy,
					Lots:       0.1,
					OpenPrice:  1.1000,
					ClosePrice: 1.1020,
					OpenTime:   ts,
					CloseTime:  ts,
					Profit:     20.0,
					Swap:       -1.0,
					Commission: -0.5,
					Comment:    "test",
					MagicNumber: 42,
				},
				{
					Ticket:     1002,
					Symbol:     "GBPUSD",
					Type:       pb.Op_Op_SellLimit,
					Lots:       0.2,
					OpenPrice:  1.3050,
					ClosePrice: 1.3030,
					OpenTime:   ts,
					CloseTime:  ts,
					Profit:     -10.0,
					Swap:       0.5,
					Commission: -1.0,
					Comment:    "limit",
					MagicNumber: 99,
				},
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	orders, err := gw.FetchOpenedOrders(context.Background())
	if err != nil {
		t.Fatalf("FetchOpenedOrders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	if orders[0].Ticket != 1001 {
		t.Errorf("Ticket = %d, want 1001", orders[0].Ticket)
	}
	if orders[0].Side != mthub.SideBuy {
		t.Errorf("Side = %v, want buy", orders[0].Side)
	}
	if orders[0].SymbolRaw != "EURUSD" {
		t.Errorf("SymbolRaw = %q, want EURUSD", orders[0].SymbolRaw)
	}
	if orders[1].Side != mthub.SideSell {
		t.Errorf("Side = %v, want sell for sell-limit", orders[1].Side)
	}
	if orders[1].OrderType != mthub.OrderLimit {
		t.Errorf("OrderType = %v, want limit", orders[1].OrderType)
	}
}

func TestFetchOrderHistory_WithMockData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	closeTs := timestamppb.New(time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC))
	mock := &mockMT4Client{
		orderHistoryRes: &pb.OrderHistoryReply{
			Result: []*pb.Order{
				{
					Ticket:     2001, Symbol: "XAUUSD",
					Type: pb.Op_Op_BuyStop, Lots: 0.5,
					OpenPrice: 1950.0, ClosePrice: 1960.0,
					OpenTime: ts, CloseTime: closeTs,
					Profit: 500.0, Swap: -5.0, Commission: -2.5,
					Comment: "history", MagicNumber: 7,
				},
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	orders, err := gw.FetchOrderHistory(context.Background(), time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("FetchOrderHistory: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].Ticket != 2001 {
		t.Errorf("Ticket = %d, want 2001", orders[0].Ticket)
	}
	if orders[0].Side != mthub.SideBuy {
		t.Errorf("Side = %v, want buy (buy-stop)", orders[0].Side)
	}
	if orders[0].OrderType != mthub.OrderStop {
		t.Errorf("OrderType = %v, want stop", orders[0].OrderType)
	}
	if orders[0].State != mthub.OrderStateClosed {
		t.Errorf("State = %v, want closed", orders[0].State)
	}
}

func TestFetchOrderHistory_OpenState(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	openTs := timestamppb.New(time.Unix(0, 0))
	mock := &mockMT4Client{
		orderHistoryRes: &pb.OrderHistoryReply{
			Result: []*pb.Order{
				{Ticket: 3001, Symbol: "BTCUSD", Type: pb.Op_Op_Buy, Lots: 1.0,
					OpenPrice: 50000.0, ClosePrice: 0, OpenTime: ts, CloseTime: openTs,
					Profit: 0, Swap: 0, Commission: 0, Comment: "open"},
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	orders, err := gw.FetchOrderHistory(context.Background(), time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("FetchOrderHistory: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].State != mthub.OrderStateOpen {
		t.Errorf("State = %v, want open (close_time=0)", orders[0].State)
	}
}

func TestFetchOrderHistory_ErrorResponse(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{
		orderHistoryRes: &pb.OrderHistoryReply{
			Error: &pb.Error{Code: 5, Message: "server error"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	_, err := gw.FetchOrderHistory(context.Background(), time.Now().Add(-24*time.Hour), time.Now())
	if err == nil {
		t.Error("FetchOrderHistory should fail when mtapi returns error")
	}
}

func TestSleep_CtxCancelled(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	gw.sleep(ctx, time.Second)
	if time.Since(start) > 100*time.Millisecond {
		t.Error("sleep should return immediately when ctx is cancelled")
	}
}

func TestRecvLoop_ReceivesTicks(t *testing.T) {
	t.Parallel()
	ts := timestamppb.Now()
	stream := &mockQuoteStream{
		quotes: []*pb.OnQuoteReply{
			{Result: &pb.QuoteEventArgs{Symbol: "EURUSD", Bid: 1.1000, Ask: 1.1001, Time: ts}},
			{Result: &pb.QuoteEventArgs{Symbol: "GBPUSD", Bid: 1.3050, Ask: 1.3051, Time: ts}},
			{Result: &pb.QuoteEventArgs{Symbol: "BTCUSD", Bid: 50000.0, Ask: 50001.0, Time: ts}},
		},
	}
	var cc grpc.ClientConn
	sc := &mockStreamsClient{quoteStream: stream}
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = sc

	ticks := make(chan *mdtick.Tick, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go gw.recvLoop(ctx, func(tk *mdtick.Tick) {
		ticks <- tk
	})

	var received []*mdtick.Tick
	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case tk := <-ticks:
			received = append(received, tk)
		case <-timeout:
			t.Fatalf("timeout waiting for ticks, got %d", len(received))
		}
	}
	cancel() // stop the loop

	if len(received) < 3 {
		t.Errorf("expected at least 3 ticks, got %d", len(received))
	}
	if len(received) > 0 && received[0].SymbolRaw != "EURUSD" {
		t.Errorf("first tick SymbolRaw = %q, want EURUSD", received[0].SymbolRaw)
	}
}

func TestProfitRecvLoop_ReceivesUpdates(t *testing.T) {
	t.Parallel()
	stream := &mockProfitStream{
		updates: []*pb.OnOrderProfitReply{
			{Result: &pb.ProfitUpdate{Balance: 10000, Equity: 10100, Margin: 500}},
			{Result: &pb.ProfitUpdate{Balance: 10000, Equity: 10200, Margin: 500}},
		},
	}
	var cc grpc.ClientConn
	sc := &mockStreamsClient{profitStream: stream}
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = sc

	updates := make(chan *mdtick.ProfitUpdate, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go gw.profitRecvLoop(ctx, func(p *mdtick.ProfitUpdate) {
		updates <- p
	})

	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case u := <-updates:
			if u.Balance != 10000 {
				t.Errorf("Balance = %f, want 10000", u.Balance)
			}
		case <-timeout:
			t.Fatalf("timeout waiting for profit updates, got %d", i)
		}
	}
	cancel()
}

func TestOrderUpdateRecvLoop_ReceivesUpdates(t *testing.T) {
	t.Parallel()
	ts := timestamppb.Now()
	stream := &mockOrderUpdateStream{
		updates: []*pb.OnOrderUpdateReply{
			{
				Result: &pb.OrderUpdateSummary{
					Balance: 10000, Equity: 10100, Margin: 500,
					Update: &pb.OrderUpdateEventArgs{
						Action: pb.UpdateAction_UpdateAction_PositionOpen,
						Order:  &pb.Order{Ticket: 1001, Symbol: "EURUSD", Lots: 0.1, OpenPrice: 1.1000},
					},
					OpenedOrders: []*pb.Order{
						{Ticket: 1001, Symbol: "EURUSD", Type: pb.Op_Op_Buy, Lots: 0.1, OpenPrice: 1.1000, ClosePrice: 1.1020, OpenTime: ts, CloseTime: ts},
					},
				},
			},
		},
	}
	var cc grpc.ClientConn
	sc := &mockStreamsClient{orderUpdateStream: stream}
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = sc

	updates := make(chan *mdtick.OrderUpdate, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go gw.orderUpdateRecvLoop(ctx, func(o *mdtick.OrderUpdate) {
		updates <- o
	})

	timeout := time.After(2 * time.Second)
	select {
	case u := <-updates:
		if u.UpdateTicket != 1001 {
			t.Errorf("UpdateTicket = %d, want 1001", u.UpdateTicket)
		}
		if u.UpdateType != "open" {
			t.Errorf("UpdateType = %q, want open", u.UpdateType)
		}
		if len(u.Positions) != 1 {
			t.Errorf("Positions = %d, want 1", len(u.Positions))
		}
	case <-timeout:
		t.Fatal("timeout waiting for order update")
	}
	cancel()
}
