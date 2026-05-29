package mt5

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNew(t *testing.T) {
	t.Parallel()
	cfg := mdtick.AccountConfig{AccountID: "acct-5", Platform: "mt5"}
	gw := New(cfg, zap.NewNop())
	if gw == nil {
		t.Fatal("New returned nil")
	}
	if gw.Platform() != "mt5" {
		t.Errorf("Platform() = %q, want %q", gw.Platform(), "mt5")
	}
	if gw.AccountID() != "acct-5" {
		t.Errorf("AccountID() = %q, want %q", gw.AccountID(), "acct-5")
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

func TestMT5Client_Nil(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if gw.MT5Client() != nil {
		t.Error("MT5Client() should be nil when not connected")
	}
}

func TestStrToUint64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  uint64
	}{
		{"digits only", "12345", 12345},
		{"with letters", "abc999def", 999},
		{"empty", "", 0},
		{"no digits", "abc", 0},
		{"mixed", "user42", 42},
		{"zero", "0", 0},
		{"max uint64", "18446744073709551615", 18446744073709551615},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strToUint64(tt.input)
			if got != tt.want {
				t.Errorf("strToUint64(%q) = %d, want %d", tt.input, got, tt.want)
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
}

func TestPfloat64(t *testing.T) {
	t.Parallel()
	zero := pfloat64(decimal.Decimal{})
	if zero != nil {
		t.Error("pfloat64(zero) should be nil")
	}
	val := decimal.NewFromFloat(1.2345)
	p := pfloat64(val)
	if p == nil {
		t.Error("pfloat64(non-zero) should not be nil")
		return
	}
	if *p != 1.2345 {
		t.Errorf("pfloat64(non-zero) = %f, want 1.2345", *p)
	}
}

func TestPInt64(t *testing.T) {
	t.Parallel()
	nilPtr := pInt64(0)
	if nilPtr != nil {
		t.Error("pInt64(0) should be nil")
	}
	p := pInt64(42)
	if p == nil {
		t.Error("pInt64(42) should not be nil")
		return
	}
	if *p != 42 {
		t.Errorf("pInt64(42) = %d, want 42", *p)
	}
}

func TestMT5OrderType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		side mthub.Side
		ot   mthub.OrderType
		want pb.OrderType
	}{
		{"buy market", mthub.SideBuy, mthub.OrderMarket, pb.OrderType_OrderType_Buy},
		{"sell market", mthub.SideSell, mthub.OrderMarket, pb.OrderType_OrderType_Sell},
		{"buy limit", mthub.SideBuy, mthub.OrderLimit, pb.OrderType_OrderType_BuyLimit},
		{"sell limit", mthub.SideSell, mthub.OrderLimit, pb.OrderType_OrderType_SellLimit},
		{"buy stop", mthub.SideBuy, mthub.OrderStop, pb.OrderType_OrderType_BuyStop},
		{"sell stop", mthub.SideSell, mthub.OrderStop, pb.OrderType_OrderType_SellStop},
		{"buy stop limit", mthub.SideBuy, mthub.OrderStopLimit, pb.OrderType_OrderType_BuyStopLimit},
		{"sell stop limit", mthub.SideSell, mthub.OrderStopLimit, pb.OrderType_OrderType_SellStopLimit},
		{"default (unknown)", mthub.SideBuy, mthub.OrderType(99), pb.OrderType_OrderType_Buy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt5OrderType(tt.side, tt.ot); got != tt.want {
				t.Errorf("mt5OrderType(%v, %v) = %v, want %v", tt.side, tt.ot, got, tt.want)
			}
		})
	}
}

func TestPlaceOrder_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{})
	if err == nil {
		t.Error("PlaceOrder should fail when not connected")
	}
}

func TestCloseOrder_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.CloseOrder(context.Background(), 0, decimal.Decimal{})
	if err == nil {
		t.Error("CloseOrder should fail when not connected")
	}
}

func TestModifyOrder_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.ModifyOrder(context.Background(), 0, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{})
	if err == nil {
		t.Error("ModifyOrder should fail when not connected")
	}
}

func TestFetchSymbolParams_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchSymbolParams(context.Background(), nil)
	if err == nil {
		t.Error("FetchSymbolParams should fail when not connected")
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

func TestFetchOrderHistory_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	// FetchOrderHistory now checks connection state; unconnected gateway returns error.
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

func TestMT5UpdateTypeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tp   pb.UpdateType
		want string
	}{
		{"market open", pb.UpdateType_UpdateType_MarketOpen, "open"},
		{"market close", pb.UpdateType_UpdateType_MarketClose, "close"},
		{"partial close", pb.UpdateType_UpdateType_PartialClose, "close"},
		{"pending open", pb.UpdateType_UpdateType_PendingOpen, "pending_open"},
		{"pending close", pb.UpdateType_UpdateType_PendingClose, "pending_close"},
		{"market modify", pb.UpdateType_UpdateType_MarketModify, "modify"},
		{"pending modify", pb.UpdateType_UpdateType_PendingModify, "modify"},
		{"unknown (default)", pb.UpdateType_UpdateType_Unknown, "unknown"},
		{"started (default)", pb.UpdateType_UpdateType_Started, "unknown"},
		{"expired (default)", pb.UpdateType_UpdateType_Expired, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt5UpdateTypeLabel(tt.tp); got != tt.want {
				t.Errorf("mt5UpdateTypeLabel(%v) = %q, want %q", tt.tp, got, tt.want)
			}
		})
	}
}

func TestMT5OrderTypeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ot   pb.OrderType
		want string
	}{
		{"buy", pb.OrderType_OrderType_Buy, "buy"},
		{"sell", pb.OrderType_OrderType_Sell, "sell"},
		{"buy limit", pb.OrderType_OrderType_BuyLimit, "buy_limit"},
		{"sell limit", pb.OrderType_OrderType_SellLimit, "sell_limit"},
		{"buy stop", pb.OrderType_OrderType_BuyStop, "buy_stop"},
		{"sell stop", pb.OrderType_OrderType_SellStop, "sell_stop"},
		{"buy stop limit", pb.OrderType_OrderType_BuyStopLimit, "buy_stop_limit"},
		{"sell stop limit", pb.OrderType_OrderType_SellStopLimit, "sell_stop_limit"},
		{"balance (defaults to buy)", pb.OrderType_OrderType_Balance, "buy"},
		{"credit (defaults to buy)", pb.OrderType_OrderType_Credit, "buy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt5OrderTypeLabel(tt.ot); got != tt.want {
				t.Errorf("mt5OrderTypeLabel(%v) = %q, want %q", tt.ot, got, tt.want)
			}
		})
	}
}

func TestMT5PeriodToTimeframe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   int32
	}{
		{"1m", 1},
		{"5m", 5},
		{"15m", 15},
		{"30m", 30},
		{"1h", 60},
		{"4h", 240},
		{"1d", 1440},
		{"unknown", 60},
		{"", 60},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got := mt5PeriodToTimeframe(tt.period)
			if got != tt.want {
				t.Errorf("mt5PeriodToTimeframe(%q) = %d, want %d", tt.period, got, tt.want)
			}
		})
	}
}

func TestConvertMT5Bars_Empty(t *testing.T) {
	t.Parallel()
	bars := convertMT5Bars(nil, "acct-5", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from nil, got %d", len(bars))
	}
	bars = convertMT5Bars([]*pb.Bar{}, "acct-5", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from empty, got %d", len(bars))
	}
}

func TestConvertMT5Bars_WithData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, OpenPrice: 1.1000, HighPrice: 1.1050, LowPrice: 1.0990, ClosePrice: 1.1020, Volume: 100, TickVolume: 50},
		{Time: ts, OpenPrice: 1.1020, HighPrice: 1.1080, LowPrice: 1.1010, ClosePrice: 1.1060, Volume: 200, TickVolume: 80},
	}
	bars := convertMT5Bars(pbBars, "acct-5", "1h")
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].AccountID != "acct-5" {
		t.Errorf("AccountID = %q, want acct-5", bars[0].AccountID)
	}
	if bars[0].Period != "1h" {
		t.Errorf("Period = %q, want 1h", bars[0].Period)
	}
	if !bars[0].Open.Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("Open = %s, want 1.1000", bars[0].Open)
	}
	if bars[0].Volume != 100 {
		t.Errorf("Volume = %f, want 100", bars[0].Volume)
	}
	if bars[0].TickCount != 50 {
		t.Errorf("TickCount = %d, want 50", bars[0].TickCount)
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
	_, err := gw.GetPriceHistory(context.Background(), "acct-5", "EURUSD", "1h", 0, 3600_000)
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
	gw.client = &mockMT5Client{}
	gw.connCli = &mockMT5ConnCli{}
	gw.streamCli = &mt5MockStreamsClient{}
	gw.subCli = &mockMT5SubCli{}
	gw.tradingCli = &mockTradingClient{}
	gw.qhCli = &mockQHClient{}
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

func TestPlaceOrder_WithMock(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Result: &pb.Order{Ticket: 5001, Symbol: "EURUSD"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	ticket, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1),
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if ticket != 5001 {
		t.Errorf("ticket = %d, want 5001", ticket)
	}
}

func TestPlaceOrder_MockError(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{orderSendErr: fmt.Errorf("mtapi error")}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1),
	})
	if err == nil {
		t.Error("PlaceOrder should propagate mock error")
	}
}

func TestPlaceOrder_ErrorCode(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Error: &pb.Error{Code: 1, Message: "bad request"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1),
	})
	if err == nil {
		t.Error("PlaceOrder should fail when mtapi returns error code")
	}
}

func TestPlaceOrder_NilResult(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1),
	})
	if err == nil {
		t.Error("PlaceOrder should fail when result is nil")
	}
}

func TestCloseOrder_WithMock(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{orderCloseRes: &pb.OrderCloseReply{}}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	if err := gw.CloseOrder(context.Background(), 5001, decimal.NewFromFloat(0.1)); err != nil {
		t.Errorf("CloseOrder: %v", err)
	}
}

func TestCloseOrder_MockError(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{orderCloseErr: fmt.Errorf("mtapi error")}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	if err := gw.CloseOrder(context.Background(), 5001, decimal.NewFromFloat(0.1)); err == nil {
		t.Error("CloseOrder should propagate mock error")
	}
}

func TestCloseOrder_ErrorCode(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{
		orderCloseRes: &pb.OrderCloseReply{
			Error: &pb.Error{Code: 3, Message: "invalid"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	if err := gw.CloseOrder(context.Background(), 5001, decimal.NewFromFloat(0.1)); err == nil {
		t.Error("CloseOrder should fail when mtapi returns error code")
	}
}

func TestModifyOrder_WithMock(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{orderModifyRes: &pb.OrderModifyReply{}}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	if err := gw.ModifyOrder(context.Background(), 5001, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{}); err != nil {
		t.Errorf("ModifyOrder: %v", err)
	}
}

func TestModifyOrder_MockError(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{orderModifyErr: fmt.Errorf("mtapi error")}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	if err := gw.ModifyOrder(context.Background(), 5001, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{}); err == nil {
		t.Error("ModifyOrder should propagate mock error")
	}
}

func TestModifyOrder_ErrorCode(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{
		orderModifyRes: &pb.OrderModifyReply{
			Error: &pb.Error{Code: 5, Message: "server error"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	if err := gw.ModifyOrder(context.Background(), 5001, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{}); err == nil {
		t.Error("ModifyOrder should fail when mtapi returns error code")
	}
}

func TestFetchSymbolParams_WithMock(t *testing.T) {
	t.Parallel()
	mock := &mockMT5Client{
		symbolParamsRes: &pb.SymbolParamsReply{
			Result: &pb.SymbolParams{
				Symbol: "EURUSD",
				SymbolInfo: &pb.SymbolInfo{
					Digits:       5,
					TickValue:    10.0,
					ContractSize: 100000,
					Spread:       1,
				},
				SymbolGroup: &pb.SymGroup{
					TradeMode: 0,
					SL:        10,
					LotsStep:  0.01,
					MinLots:   0.01,
					MaxLots:   100,
				},
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	params, err := gw.FetchSymbolParams(context.Background(), []string{"EURUSD"})
	if err != nil {
		t.Fatalf("FetchSymbolParams: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].Canonical != "EURUSD" {
		t.Errorf("Canonical = %q, want EURUSD", params[0].Canonical)
	}
	if params[0].Digits != 5 {
		t.Errorf("Digits = %d, want 5", params[0].Digits)
	}
	if params[0].SpreadFloat != true {
		t.Error("SpreadFloat should be true when spread > 0")
	}
}

func TestFetchSymbolParams_MockError(t *testing.T) {
	t.Parallel()
	mock := &mockMT5Client{symbolParamsErr: fmt.Errorf("mtapi error")}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	params, err := gw.FetchSymbolParams(context.Background(), []string{"EURUSD"})
	if err == nil {
		t.Error("FetchSymbolParams should propagate mock error")
	}
	// Returns partial results collected before the error.
	_ = params
}

func TestFetchSymbolParams_ErrorCode(t *testing.T) {
	t.Parallel()
	mock := &mockMT5Client{
		symbolParamsRes: &pb.SymbolParamsReply{
			Error: &pb.Error{Code: 2, Message: "not found"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	_, err := gw.FetchSymbolParams(context.Background(), []string{"XXX"})
	if err == nil {
		t.Error("FetchSymbolParams should fail when mtapi returns error code")
	}
}

func TestFetchOpenedOrders_WithMock(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	mock := &mockMT5Client{
		openedOrdersRes: &pb.OpenedOrdersReply{
			Result: []*pb.Order{
				{
					Ticket:     6001, Symbol: "EURUSD",
					OrderType: pb.OrderType_OrderType_Buy,
					Lots: 0.1, OpenPrice: 1.1000, ClosePrice: 1.1020,
					OpenTime: ts, CloseTime: ts,
					Profit: 20.0, Swap: -1.0, Commission: -0.5,
					Comment: "test", ExpertId: 42,
				},
				{
					Ticket:     6002, Symbol: "GBPUSD",
					OrderType: pb.OrderType_OrderType_SellLimit,
					Lots: 0.2, OpenPrice: 1.3050, ClosePrice: 1.3030,
					OpenTime: ts, CloseTime: ts,
					Profit: -10.0, Swap: 0.5, Commission: -1.0,
					Comment: "limit", ExpertId: 99,
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
	if orders[0].Ticket != 6001 {
		t.Errorf("Ticket = %d, want 6001", orders[0].Ticket)
	}
	if orders[0].Side != mthub.SideBuy {
		t.Errorf("Side = %v, want buy", orders[0].Side)
	}
	if orders[1].Side != mthub.SideSell {
		t.Errorf("Side = %v, want sell", orders[1].Side)
	}
	if orders[1].OrderType != mthub.OrderLimit {
		t.Errorf("OrderType = %v, want limit", orders[1].OrderType)
	}
}

func TestGetPriceHistory_UnsupportedPeriod(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.qhCli = &mockQHClient{}
	_, err := gw.GetPriceHistory(context.Background(), "acct-5", "EURUSD", "2w", 0, 3600_000)
	if err == nil {
		t.Error("GetPriceHistory should fail for unsupported period")
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
	stream := &mt5MockQuoteStream{
		quotes: []*pb.OnQuoteReply{
			{Result: &pb.Quote{Symbol: "EURUSD", Bid: 1.1000, Ask: 1.1001, Time: ts}},
			{Result: &pb.Quote{Symbol: "GBPUSD", Bid: 1.3050, Ask: 1.3051, Time: ts}},
			{Result: &pb.Quote{Symbol: "BTCUSD", Bid: 50000.0, Ask: 50001.0, Time: ts}},
		},
	}
	var cc grpc.ClientConn
	sc := &mt5MockStreamsClient{quoteStream: stream}
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
	cancel()

	if len(received) < 3 {
		t.Errorf("expected at least 3 ticks, got %d", len(received))
	}
	if len(received) > 0 && received[0].SymbolRaw != "EURUSD" {
		t.Errorf("first tick SymbolRaw = %q, want EURUSD", received[0].SymbolRaw)
	}
	if len(received) > 0 && received[0].Platform != "mt5" {
		t.Errorf("platform = %q, want mt5", received[0].Platform)
	}
}

func TestProfitRecvLoop_ReceivesUpdates(t *testing.T) {
	t.Parallel()
	stream := &mt5MockProfitStream{
		updates: []*pb.OnOrderProfitReply{
			{Result: &pb.ProfitUpdate{Balance: 10000, Equity: 10100, Margin: 500}},
			{Result: &pb.ProfitUpdate{Balance: 10000, Equity: 10200, Margin: 500}},
		},
	}
	mockClient := &mockMT5Client{
		accountSummaryRes: &pb.AccountSummaryReply{
			Result: &pb.AccountSummary{Balance: 10000, Equity: 10100, Margin: 500},
		},
	}
	var cc grpc.ClientConn
	sc := &mt5MockStreamsClient{profitStream: stream}
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test", MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = sc
	gw.client = mockClient

	updates := make(chan *mdtick.ProfitUpdate, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go gw.profitRecvLoop(ctx, func(p *mdtick.ProfitUpdate) {
		updates <- p
	})

	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ { // initial fetch + 2 stream frames
		select {
		case u := <-updates:
			if u.Balance != 10000 {
				t.Errorf("Balance = %f, want 10000", u.Balance)
			}
		case <-timeout:
			if i < 2 {
				t.Fatalf("timeout waiting for profit updates, got %d", i)
			}
			return
		}
	}
	cancel()
}

func TestOrderUpdateRecvLoop_ReceivesUpdates(t *testing.T) {
	t.Parallel()
	ts := timestamppb.Now()
	stream := &mt5MockOrderUpdateStream{
		updates: []*pb.OnOrderUpdateReply{
			{
				Result: &pb.OrderUpdateSummary{
					Balance: 10000, Equity: 10100, Margin: 500,
					Update: &pb.OrderUpdate{
						Type:  pb.UpdateType_UpdateType_MarketOpen,
						Order: &pb.Order{Ticket: 1001, Symbol: "EURUSD", Lots: 0.1, OpenPrice: 1.1000, ClosePrice: 1.1020, OpenTime: ts, CloseTime: ts},
					},
					OpenedOrders: []*pb.Order{
						{Ticket: 1001, Symbol: "EURUSD", OrderType: pb.OrderType_OrderType_Buy, Lots: 0.1, OpenPrice: 1.1000, ClosePrice: 1.1020, OpenTime: ts, CloseTime: ts},
					},
				},
			},
		},
	}
	var cc grpc.ClientConn
	sc := &mt5MockStreamsClient{orderUpdateStream: stream}
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

func FuzzStrToUint64(f *testing.F) {
	f.Add("12345")
	f.Add("abc999def")
	f.Add("")
	f.Add("user42")
	f.Add("18446744073709551615")
	f.Fuzz(func(t *testing.T, s string) {
		v := strToUint64(s)
		_ = v
	})
}

func BenchmarkStrToUint64(b *testing.B) {
	const s = "mt5_login_123456789_abc"
	b.ReportAllocs()
	for b.Loop() {
		strToUint64(s)
	}
}
