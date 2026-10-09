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
)

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
