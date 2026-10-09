package mt5

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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
	rec, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1),
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if rec.Ticket != 5001 {
		t.Errorf("ticket = %d, want 5001", rec.Ticket)
	}
}

// PlaceOrder must carry the broker's stated side/orderType into the record —
// a SellStop reply with zero-value fields would inject into the runner's
// live state as a BUY market position. Also covers the BuyStopLimit case
// that mt5OrderTypeToSideAndOrderType was missing (fell to default Buy/Market).

// PlaceOrder must carry the broker's stated side/orderType into the record —
// a SellStop reply with zero-value fields would inject into the runner's
// live state as a BUY market position. Also covers the BuyStopLimit case
// that mt5OrderTypeToSideAndOrderType was missing (fell to default Buy/Market).
func TestPlaceOrder_MapsReplySideAndOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ot   pb.OrderType
		side mthub.Side
		ot2  mthub.OrderType
	}{
		{"sell_stop", pb.OrderType_OrderType_SellStop, mthub.SideSell, mthub.OrderStop},
		{"buy_stop_limit", pb.OrderType_OrderType_BuyStopLimit, mthub.SideBuy, mthub.OrderStopLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
			gw.sessionID = "sid"
			gw.tradingCli = &mockTradingClient{
				orderSendRes: &pb.OrderSendReply{
					Result: &pb.Order{Ticket: 888, OrderType: tc.ot},
				},
			}
			rec, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
				Canonical: "EURUSD", Side: tc.side, OrderType: tc.ot2,
				Volume: decimal.NewFromFloat(0.1),
			})
			if err != nil {
				t.Fatalf("PlaceOrder: %v", err)
			}
			if rec.Side != tc.side {
				t.Errorf("rec.Side = %d, want %d", rec.Side, tc.side)
			}
			if rec.OrderType != tc.ot2 {
				t.Errorf("rec.OrderType = %d, want %d", rec.OrderType, tc.ot2)
			}
		})
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

// T5 (VM-ERR-CODE-COLLAPSE-1): MT5 broker rejections carry the platform
// retcode (100xx namespace) verbatim via *mthub.BrokerRejectError.
// Adversarial: reverting to fmt.Errorf loses errors.As → RED.
func TestPlaceOrder_BrokerRejectError_CarriesCode(t *testing.T) {
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Error: &pb.Error{Code: pb.ErrorCode_INVALID_STOPS, Message: "Invalid stops"},
		},
	}
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		Canonical: "EURUSD", Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1),
	})
	if err == nil {
		t.Fatal("broker rejection must return error")
	}
	if !errors.Is(err, mthub.ErrBrokerRejected) {
		t.Fatal("errors.Is(ErrBrokerRejected) = false — classification would break")
	}
	var bre *mthub.BrokerRejectError
	if !errors.As(err, &bre) || bre == nil {
		t.Fatalf("err type = %T, want *mthub.BrokerRejectError", err)
	}
	if bre.Code != 10016 {
		t.Fatalf("BrokerRejectError.Code = %d, want 10016 (INVALID_STOPS)", bre.Code)
	}
}
