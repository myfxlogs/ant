package mt5

import (
	"context"
	"fmt"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
