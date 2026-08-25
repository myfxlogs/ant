package mt4

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

func TestFetchOpenedOrders_WithMockData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	mock := &mockMT4Client{
		openedOrdersRes: &pb.OpenedOrdersReply{
			Result: []*pb.Order{
				{
					Ticket:      1001,
					Symbol:      "EURUSD",
					Type:        pb.Op_Op_Buy,
					Lots:        0.1,
					OpenPrice:   1.1000,
					ClosePrice:  1.1020,
					OpenTime:    ts,
					CloseTime:   ts,
					Profit:      20.0,
					Swap:        -1.0,
					Commission:  -0.5,
					Comment:     "test",
					MagicNumber: 42,
				},
				{
					Ticket:      1002,
					Symbol:      "GBPUSD",
					Type:        pb.Op_Op_SellLimit,
					Lots:        0.2,
					OpenPrice:   1.3050,
					ClosePrice:  1.3030,
					OpenTime:    ts,
					CloseTime:   ts,
					Profit:      -10.0,
					Swap:        0.5,
					Commission:  -1.0,
					Comment:     "limit",
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
					Ticket: 2001, Symbol: "XAUUSD",
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
