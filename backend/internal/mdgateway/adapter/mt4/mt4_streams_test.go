package mt4

import (
	"context"
	"fmt"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	gw.client = &mockMT4Client{accountSummaryRes: &pb.AccountSummaryReply{
		Result: &pb.AccountSummary{Balance: 10000, Equity: 10100, Profit: 100, Margin: 500, FreeMargin: 9600, MarginLevel: 2020, Leverage: 100},
	}}

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
			if !u.Balance.Equal(decimal.NewFromInt(10000)) {
				t.Errorf("Balance = %s, want 10000", u.Balance.String())
			}
		case <-timeout:
			t.Fatalf("timeout waiting for profit updates, got %d", i)
		}
	}
	cancel()
}

// TestDATATRUTH2_MarginFromAccountSummary verifies that MT4 margin values come
// from AccountSummary RPC, not from the OnOrderProfit stream frame (which always
// returns margin=0). Adversarial proof: if fetchAndPublish is removed and margin
// is taken from the stream frame, this test fails because stream margin=0 while
// AccountSummary margin=500.
func TestDATATRUTH2_MarginFromAccountSummary(t *testing.T) {
	t.Parallel()
	stream := &mockProfitStream{
		updates: []*pb.OnOrderProfitReply{
			{Result: &pb.ProfitUpdate{Balance: 10000, Equity: 10100, Profit: 0, Margin: 0, FreeMargin: 10100, MarginLevel: 0}},
		},
	}
	var cc grpc.ClientConn
	sc := &mockStreamsClient{profitStream: stream}
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = sc
	gw.client = &mockMT4Client{
		accountSummaryRes: &pb.AccountSummaryReply{
			Result: &pb.AccountSummary{
				Balance: 10000, Equity: 10100, Profit: 100, Margin: 500, FreeMargin: 9600, MarginLevel: 2020, Credit: 0,
			},
		},
	}

	updates := make(chan *mdtick.ProfitUpdate, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go gw.profitRecvLoop(ctx, func(p *mdtick.ProfitUpdate) {
		updates <- p
	})

	select {
	case u := <-updates:
		if !u.Margin.Equal(decimal.NewFromInt(500)) {
			t.Errorf("Margin = %s, want 500 (from AccountSummary, not stream frame's 0)", u.Margin.String())
		}
		if !u.FreeMargin.Equal(decimal.NewFromInt(9600)) {
			t.Errorf("FreeMargin = %s, want 9600 (from AccountSummary, not stream frame's equity)", u.FreeMargin.String())
		}
		if !u.MarginLevel.Equal(decimal.NewFromInt(2020)) {
			t.Errorf("MarginLevel = %s, want 2020 (from AccountSummary, not stream frame's 0)", u.MarginLevel.String())
		}
		if !u.Profit.Equal(decimal.NewFromInt(100)) {
			t.Errorf("Profit = %s, want 100 (from AccountSummary, not local equity.Sub(balance))", u.Profit.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for profit update")
	}
	cancel()
}

func TestDATATRUTH2_AccountSummaryFailureRejectsFinancialSnapshot(t *testing.T) {
	t.Parallel()
	stream := &mockProfitStream{updates: []*pb.OnOrderProfitReply{
		{Result: &pb.ProfitUpdate{Balance: 10000, Equity: 10100, Margin: 0, FreeMargin: 10100}},
	}}
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = &mockStreamsClient{profitStream: stream}
	gw.client = &mockMT4Client{accountSummaryErr: fmt.Errorf("account summary unavailable")}

	updates := make(chan *mdtick.ProfitUpdate, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go gw.profitRecvLoop(ctx, func(p *mdtick.ProfitUpdate) { updates <- p })
	select {
	case u := <-updates:
		t.Fatalf("published financial snapshot after AccountSummary failure: %+v", u)
	case <-ctx.Done():
	}
}

func TestAccountSummaryRefreshContinuesWithoutProfitFrames(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "a1"}, zap.NewNop())
	gw.client = &mockMT4Client{
		accountSummaryRes: &pb.AccountSummaryReply{Result: &pb.AccountSummary{
			Balance: 10000, Equity: 10000, FreeMargin: 10000, Leverage: 100,
		}},
		// OpenedOrders returns empty reply → positions are authoritative (0 positions).
		// This is the fix for stale snapshot: refreshAccountSummary now fetches
		// OpenedOrders so positionsReceivedAt stays fresh even for empty accounts.
		openedOrdersRes: &pb.OpenedOrdersReply{Result: []*pb.Order{}},
	}
	updates := make(chan *mdtick.ProfitUpdate, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	go gw.refreshAccountSummary(ctx, "sid", 10*time.Millisecond, func(p *mdtick.ProfitUpdate) { updates <- p })
	for i := 0; i < 2; i++ {
		select {
		case got := <-updates:
			if !got.FreeMargin.Equal(decimal.NewFromInt(10000)) {
				t.Fatalf("unexpected FreeMargin: %s", got.FreeMargin)
			}
			// PositionsAuthoritative must be true — refreshAccountSummary fetches
			// OpenedOrders to keep positionsReceivedAt fresh.
			if !got.PositionsAuthoritative {
				t.Fatalf("expected PositionsAuthoritative=true (OpenedOrders fetched), got false: %+v", got)
			}
		case <-ctx.Done():
			t.Fatal("periodic AccountSummary refresh stopped without profit frames")
		}
	}
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
