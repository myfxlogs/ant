package mt5

import (
	"context"
	"fmt"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

// VM-TRADE-CONTEXT-7: verify Deviation is passed to mtapi OrderSend as Slippage.
// Adversarial: remove `Slippage: pUint64(...)` in orders.go → in.Slippage == nil (RED).
func TestPlaceOrder_PassesDeviationAsSlippage(t *testing.T) {
	t.Parallel()
	tc := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Result: &pb.Order{Ticket: 5001, Symbol: "EURUSD"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = tc
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Magic: 42, Deviation: 30,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if tc.lastOrderSend == nil {
		t.Fatal("OrderSend not called")
	}
	if tc.lastOrderSend.Slippage == nil || *tc.lastOrderSend.Slippage != 30 {
		t.Fatalf("OrderSend.Slippage = %v, want 30 (adversarial: remove `Slippage: pUint64(...)` → RED)", tc.lastOrderSend.Slippage)
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
					Ticket: 6001, Symbol: "EURUSD",
					OrderType: pb.OrderType_OrderType_Buy,
					Lots:      0.1, OpenPrice: 1.1000, ClosePrice: 1.1020,
					OpenTime: ts, CloseTime: ts,
					Profit: 20.0, Swap: -1.0, Commission: -0.5,
					Comment: "test", ExpertId: 42,
				},
				{
					Ticket: 6002, Symbol: "GBPUSD",
					OrderType: pb.OrderType_OrderType_SellLimit,
					Lots:      0.2, OpenPrice: 1.3050, ClosePrice: 1.3030,
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
			if !u.Balance.Equal(decimal.NewFromInt(10000)) {
				t.Errorf("Balance = %s, want 10000", u.Balance.String())
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
