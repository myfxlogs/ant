package mt5

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
