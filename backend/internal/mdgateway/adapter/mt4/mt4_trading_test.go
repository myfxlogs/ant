package mt4

import (
	"context"
	"fmt"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// --- mt4Op pure function tests ---

func TestMt4Op(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		side    mthub.Side
		ot      mthub.OrderType
		want    pb.Op
		wantErr bool
	}{
		{"buy market", mthub.SideBuy, mthub.OrderMarket, pb.Op_Op_Buy, false},
		{"sell market", mthub.SideSell, mthub.OrderMarket, pb.Op_Op_Sell, false},
		{"buy limit", mthub.SideBuy, mthub.OrderLimit, pb.Op_Op_BuyLimit, false},
		{"sell limit", mthub.SideSell, mthub.OrderLimit, pb.Op_Op_SellLimit, false},
		{"buy stop", mthub.SideBuy, mthub.OrderStop, pb.Op_Op_BuyStop, false},
		{"sell stop", mthub.SideSell, mthub.OrderStop, pb.Op_Op_SellStop, false},
		{"buy stop_limit unsupported", mthub.SideBuy, mthub.OrderStopLimit, 0, true},
		{"sell stop_limit unsupported", mthub.SideSell, mthub.OrderStopLimit, 0, true},
		{"unknown type returns error", mthub.SideBuy, mthub.OrderType(99), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mt4Op(tt.side, tt.ot)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("mt4Op(%v, %v) expected error, got %v", tt.side, tt.ot, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("mt4Op(%v, %v) unexpected error: %v", tt.side, tt.ot, err)
			}
			if got != tt.want {
				t.Errorf("mt4Op(%v, %v) = %v, want %v", tt.side, tt.ot, got, tt.want)
			}
		})
	}
}

// --- PlaceOrder with mock TradingClient ---

func TestPlaceOrder_Success(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Result: &pb.Order{Ticket: 12345},
		},
	}
	ticket, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1000),
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if ticket != 12345 {
		t.Errorf("ticket = %d, want 12345", ticket)
	}
}

// ARCH-4-MT4-MAGIC: verify Magic is passed to mtapi OrderSend.
// Adversarial: remove `Magic: req.Magic` in orders.go → in.Magic == 0 != 42 (RED).
func TestPlaceOrder_PassesMagic(t *testing.T) {
	t.Parallel()
	mock := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Result: &pb.Order{Ticket: 12345},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = mock
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1000),
		Magic: 42,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if mock.lastOrderSend == nil {
		t.Fatal("OrderSend not called")
	}
	if mock.lastOrderSend.Magic != 42 {
		t.Fatalf("OrderSend.Magic = %d, want 42 (adversarial: remove `Magic: req.Magic` → RED)", mock.lastOrderSend.Magic)
	}
}

// VM-TRADE-CONTEXT-7: verify Deviation is passed to mtapi OrderSend as Slippage.
// Adversarial: remove `Slippage: req.Deviation` in orders.go → in.Slippage == 0 != 20 (RED).
func TestPlaceOrder_PassesDeviationAsSlippage(t *testing.T) {
	t.Parallel()
	mock := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Result: &pb.Order{Ticket: 12345},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = mock
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1000),
		Magic: 42, Deviation: 20,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if mock.lastOrderSend == nil {
		t.Fatal("OrderSend not called")
	}
	if mock.lastOrderSend.Slippage != 20 {
		t.Fatalf("OrderSend.Slippage = %d, want 20 (adversarial: remove `Slippage: req.Deviation` → RED)", mock.lastOrderSend.Slippage)
	}
}

func TestPlaceOrder_ErrorCode(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Error: &pb.Error{Code: 1, Message: "bad request"},
		},
	}
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1000),
	})
	if err == nil {
		t.Error("PlaceOrder should fail when mtapi returns error code")
	}
}

func TestPlaceOrder_TransportErr(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderSendErr: fmt.Errorf("connection refused"),
	}
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1000),
	})
	if err == nil {
		t.Error("PlaceOrder should fail on transport error")
	}
}

// --- CloseOrder with mock TradingClient ---

func TestCloseOrder_Success(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderCloseRes: &pb.OrderCloseReply{Result: &pb.Order{Ticket: 1001}},
	}
	err := gw.CloseOrder(context.Background(), 1001, decimal.NewFromFloat(0.1))
	if err != nil {
		t.Errorf("CloseOrder: %v", err)
	}
}

func TestCloseOrder_ErrorCode(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderCloseRes: &pb.OrderCloseReply{
			Error: &pb.Error{Code: 5, Message: "server error"},
		},
	}
	err := gw.CloseOrder(context.Background(), 1001, decimal.NewFromFloat(0.1))
	if err == nil {
		t.Error("CloseOrder should fail when mtapi returns error code")
	}
}

// --- ModifyOrder with mock TradingClient ---

func TestModifyOrder_Success(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderModifyRes: &pb.OrderModifyReply{Result: &pb.Order{Ticket: 1001}},
	}
	err := gw.ModifyOrder(context.Background(), 1001,
		decimal.NewFromFloat(1.0900), decimal.NewFromFloat(1.1100), decimal.NewFromFloat(1.1000))
	if err != nil {
		t.Errorf("ModifyOrder: %v", err)
	}
}

func TestModifyOrder_ErrorCode(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderModifyRes: &pb.OrderModifyReply{
			Error: &pb.Error{Code: 3, Message: "invalid ticket"},
		},
	}
	err := gw.ModifyOrder(context.Background(), 9999,
		decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{})
	if err == nil {
		t.Error("ModifyOrder should fail when mtapi returns error code")
	}
}

// --- FetchSymbolParams with mock ---

func TestFetchSymbolParams_Success(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{
		symbolParamsRes: &pb.SymbolParamsReply{
			Result: &pb.SymbolParams{
				SymbolName: "EURUSD",
				Symbol:     &pb.SymbolInfo{Digits: 5, Point: 0.00001, Spread: 10, StopsLevel: 0},
				GroupParams: &pb.ConGroupSec{
					MinLot: 0.01, MaxLot: 100, LotStep: 0.01, Execution: 0,
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
	if params[0].Digits != 5 {
		t.Errorf("Digits = %d, want 5", params[0].Digits)
	}
	if params[0].Canonical != "EURUSD" {
		t.Errorf("Canonical = %q, want EURUSD", params[0].Canonical)
	}
	if params[0].SpreadFloat != true {
		t.Error("SpreadFloat should be true when spread > 0")
	}
}

func TestFetchSymbolParams_ErrorCode(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{
		symbolParamsRes: &pb.SymbolParamsReply{
			Error: &pb.Error{Code: 2, Message: "symbol not found"},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock

	_, err := gw.FetchSymbolParams(context.Background(), []string{"INVALID"})
	if err == nil {
		t.Error("FetchSymbolParams should fail on mtapi error")
	}
}

func TestFetchSymbolParams_NoSpread(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{
		symbolParamsRes: &pb.SymbolParamsReply{
			Result: &pb.SymbolParams{
				SymbolName: "FIXED",
				Symbol:     &pb.SymbolInfo{Digits: 3, Point: 0.001, Spread: 0, StopsLevel: 10},
				GroupParams: &pb.ConGroupSec{
					MinLot: 1, MaxLot: 50, LotStep: 1, Execution: 1,
				},
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock

	params, err := gw.FetchSymbolParams(context.Background(), []string{"FIXED"})
	if err != nil {
		t.Fatalf("FetchSymbolParams: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].SpreadFloat != false {
		t.Error("SpreadFloat should be false when spread == 0")
	}
	if params[0].StopLevel != 10 {
		t.Errorf("StopLevel = %d, want 10", params[0].StopLevel)
	}
}

func TestFetchSymbolParams_NilResult(t *testing.T) {
	t.Parallel()
	mock := &mockMT4Client{
		symbolParamsRes: &pb.SymbolParamsReply{},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock

	params, err := gw.FetchSymbolParams(context.Background(), []string{"NILRESULT"})
	if err != nil {
		t.Fatalf("FetchSymbolParams: %v", err)
	}
	if len(params) != 0 {
		t.Errorf("expected 0 params for nil result, got %d", len(params))
	}
}

// --- SubscribeOrderEvents with mock stream ---

func TestSubscribeOrderEvents_Success(t *testing.T) {
	t.Parallel()
	stream := &mockOrderUpdateStream{
		updates: []*pb.OnOrderUpdateReply{
			{
				Result: &pb.OrderUpdateSummary{
					Update: &pb.OrderUpdateEventArgs{
						Action: pb.UpdateAction_UpdateAction_PositionOpen,
						Order:  &pb.Order{Ticket: 5001, Symbol: "EURUSD"},
					},
				},
			},
		},
	}
	var cc grpc.ClientConn
	sc := &mockStreamsClient{orderUpdateStream: stream}
	gw := New(mdtick.AccountConfig{UserID: "u1", AccountID: "a1", Broker: "test", MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.conn = &cc
	gw.streamCli = sc

	events := make(chan *mthub.OrderEvent, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gw.SubscribeOrderEvents(ctx, func(ev *mthub.OrderEvent) {
		events <- ev
	})
	if err != nil {
		t.Fatalf("SubscribeOrderEvents: %v", err)
	}

	select {
	case ev := <-events:
		if ev.Ticket != 5001 {
			t.Errorf("Ticket = %d, want 5001", ev.Ticket)
		}
		if ev.AccountID != "a1" {
			t.Errorf("AccountID = %q, want a1", ev.AccountID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for order event")
	}
	cancel()
}

func TestSubscribeOrderEvents_StreamError(t *testing.T) {
	t.Parallel()
	sc := &mockStreamsClient{orderUpdateErr: fmt.Errorf("stream unavailable")}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.streamCli = sc

	// MDGATEWAY-4: stream creation moved into goroutine reconnect loop.
	// SubscribeOrderEvents returns nil (success) — stream errors are
	// handled asynchronously with backoff retry inside the goroutine.
	err := gw.SubscribeOrderEvents(context.Background(), nil)
	if err != nil {
		t.Errorf("SubscribeOrderEvents should not fail on initial stream error (goroutine retries): %v", err)
	}
}
