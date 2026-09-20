package mt4

// VM-LIVE-PARITY-F1/F2 adapter tests (T1/T2/T3 of builder-handoff-vm-live-parity-f1f3).
//
// T1 pins the OrderSend receipt contract: request Comment/Slippage reach
// mtapi, and the broker's full Order reply maps into OrderRecord verbatim —
// with State derived explicitly (market→Open, pending→Pending).
// T2/T3 pin the per-field flat→Ex fallback and the TradeMode semantic source.

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// T1: broker receipt → OrderRecord verbatim; fill facts ≠ request values.
// Mutations: delete `Comment: req.Comment` → request-assert RED (M3);
// echo req.Price/req.Volume into the record → fill-assert RED.
func TestPlaceOrder_Parity_ReceiptMappedVerbatim(t *testing.T) {
	openTime := time.Unix(1700000000, 0).UTC()
	mock := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{
			Result: &pb.Order{
				Ticket:      12345,
				OpenPrice:   81262.24, // broker fill — request price is 0 (market)
				Lots:        0.02,     // broker normalized — request volume 0.015
				StopLoss:    81000.5,
				TakeProfit:  83000.75,
				Comment:     "PARITY-X",
				MagicNumber: 42,
				OpenTime:    timestamppb.New(openTime),
			},
		},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = mock

	rec, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		AccountID: "acct-1", Canonical: "BTCUSDm",
		Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.015), Price: decimal.NewFromFloat(0),
		Comment: "PARITY-X", Magic: 42, Deviation: 3,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	// Request side: comment and deviation must reach mtapi.
	if got := mock.lastOrderSend.GetComment(); got != "PARITY-X" {
		t.Errorf("OrderSendRequest.Comment = %q, want PARITY-X", got)
	}
	if got := mock.lastOrderSend.GetSlippage(); got != 3 {
		t.Errorf("OrderSendRequest.Slippage = %d, want 3 (Deviation)", got)
	}

	// Receipt side: broker facts, not request echoes.
	if rec.Ticket != 12345 {
		t.Errorf("Ticket = %d, want 12345", rec.Ticket)
	}
	if !rec.OpenPrice.Equal(decimal.NewFromFloat(81262.24)) {
		t.Errorf("OpenPrice = %s, want broker fill 81262.24 (request was 0)", rec.OpenPrice)
	}
	if !rec.Volume.Equal(decimal.NewFromFloat(0.02)) {
		t.Errorf("Volume = %s, want broker normalized 0.02 (request was 0.015)", rec.Volume)
	}
	if !rec.StopLoss.Equal(decimal.NewFromFloat(81000.5)) {
		t.Errorf("StopLoss = %s, want 81000.5", rec.StopLoss)
	}
	if !rec.TakeProfit.Equal(decimal.NewFromFloat(83000.75)) {
		t.Errorf("TakeProfit = %s, want 83000.75", rec.TakeProfit)
	}
	if rec.Comment != "PARITY-X" {
		t.Errorf("Comment = %q, want broker receipt PARITY-X", rec.Comment)
	}
	if rec.Magic != 42 {
		t.Errorf("Magic = %d, want 42", rec.Magic)
	}
	if !rec.OpenTime.Equal(openTime) {
		t.Errorf("OpenTime = %v, want %v", rec.OpenTime, openTime)
	}
	if rec.AccountID != "acct-1" || rec.Canonical != "BTCUSDm" {
		t.Errorf("AccountID/Canonical = %s/%s, want acct-1/BTCUSDm", rec.AccountID, rec.Canonical)
	}

	// State: market fill must be Open — explicitly assigned, never the zero
	// value masquerading as Pending.
	if rec.State != mthub.OrderStateOpen {
		t.Errorf("market order State = %d, want Open", rec.State)
	}
}

// T1b: pending op (BuyLimit) → OrderStatePending.
func TestPlaceOrder_Parity_PendingOpStatePending(t *testing.T) {
	mock := &mockTradingClient{
		orderSendRes: &pb.OrderSendReply{Result: &pb.Order{Ticket: 777}},
	}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.tradingCli = mock

	rec, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{
		AccountID: "acct-1", Canonical: "BTCUSDm",
		Side: mthub.SideBuy, OrderType: mthub.OrderLimit,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(80000),
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if rec.State != mthub.OrderStatePending {
		t.Errorf("pending order State = %d, want Pending", rec.State)
	}
}

// parityFetchParams builds a Gateway over a fake SymbolParams reply.
func parityFetchParams(t *testing.T, res *pb.SymbolParamsReply) *Gateway {
	t.Helper()
	mock := &mockMT4Client{symbolParamsRes: res}
	gw := New(mdtick.AccountConfig{MtapiToken: "t"}, zap.NewNop())
	gw.sessionID = "sid"
	gw.client = mock
	return gw
}

// T2: per-field flat→Ex fallback — flat zero consults Ex; flat set wins;
// double zero stays zero.
func TestFetchSymbolParams_Parity_FlatToExFallback(t *testing.T) {
	gw := parityFetchParams(t, &pb.SymbolParamsReply{
		Result: &pb.SymbolParams{
			Symbol: &pb.SymbolInfo{
				Digits: 2,
				Ex: &pb.SymbolInfoEx{
					Point: 0.01, ContractSize: 1, StopsLevel: 30,
					TickValue: 1.5, TickSize: 0.01,
					SwapLong: -7.5, SwapShort: -2.5,
					FreezeLevel: 10, Trade: 4,
				},
			},
			GroupParams: &pb.ConGroupSec{MinLot: 0.01, MaxLot: 100, LotStep: 0.01},
		},
	})
	params, err := gw.FetchSymbolParams(context.Background(), []string{"BTCUSDm"})
	if err != nil || len(params) != 1 {
		t.Fatalf("FetchSymbolParams: %v (%d params)", err, len(params))
	}
	p := params[0]
	if !p.PointValue.Equal(decimal.NewFromFloat(0.01)) {
		t.Errorf("PointValue = %s, want Ex 0.01", p.PointValue)
	}
	if !p.ContractSize.Equal(decimal.NewFromFloat(1)) {
		t.Errorf("ContractSize = %s, want Ex 1", p.ContractSize)
	}
	if p.StopLevel != 30 {
		t.Errorf("StopLevel = %d, want Ex 30", p.StopLevel)
	}
	if !p.TickValue.Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("TickValue = %s, want 1.5", p.TickValue)
	}
	if !p.TickSize.Equal(decimal.NewFromFloat(0.01)) {
		t.Errorf("TickSize = %s, want 0.01", p.TickSize)
	}
	if !p.SwapLong.Equal(decimal.NewFromFloat(-7.5)) {
		t.Errorf("SwapLong = %s, want -7.5", p.SwapLong)
	}
	if !p.SwapShort.Equal(decimal.NewFromFloat(-2.5)) {
		t.Errorf("SwapShort = %s, want -2.5", p.SwapShort)
	}
	if p.FreezeLevel != 10 {
		t.Errorf("FreezeLevel = %d, want 10", p.FreezeLevel)
	}

	// flat wins when non-zero.
	gw2 := parityFetchParams(t, &pb.SymbolParamsReply{
		Result: &pb.SymbolParams{
			Symbol: &pb.SymbolInfo{
				Digits: 2, Point: 0.5, ContractSize: 100, StopsLevel: 20,
				Ex: &pb.SymbolInfoEx{Point: 0.01, ContractSize: 1, StopsLevel: 30},
			},
		},
	})
	params2, err := gw2.FetchSymbolParams(context.Background(), []string{"BTCUSDm"})
	if err != nil || len(params2) != 1 {
		t.Fatalf("FetchSymbolParams(flat): %v", err)
	}
	p2 := params2[0]
	if !p2.PointValue.Equal(decimal.NewFromFloat(0.5)) || !p2.ContractSize.Equal(decimal.NewFromFloat(100)) || p2.StopLevel != 20 {
		t.Errorf("flat values must win: point=%s contract=%s stops=%d", p2.PointValue, p2.ContractSize, p2.StopLevel)
	}

	// double zero stays zero (= unknown).
	gw3 := parityFetchParams(t, &pb.SymbolParamsReply{
		Result: &pb.SymbolParams{Symbol: &pb.SymbolInfo{Digits: 2}},
	})
	params3, err := gw3.FetchSymbolParams(context.Background(), []string{"BTCUSDm"})
	if err != nil || len(params3) != 1 {
		t.Fatalf("FetchSymbolParams(zero): %v", err)
	}
	if !params3[0].PointValue.IsZero() || !params3[0].ContractSize.IsZero() || params3[0].StopLevel != 0 {
		t.Errorf("double zero must stay zero: point=%s contract=%s stops=%d",
			params3[0].PointValue, params3[0].ContractSize, params3[0].StopLevel)
	}
}

// T3: TradeMode semantic source is SymbolInfoEx.Trade — the group Execution
// mode is a different enum and must never land in TradeMode.
// Mutation: restore `param.TradeMode = gp.GetExecution()` → RED.
func TestFetchSymbolParams_Parity_TradeModeFromExTrade(t *testing.T) {
	gw := parityFetchParams(t, &pb.SymbolParamsReply{
		Result: &pb.SymbolParams{
			Symbol: &pb.SymbolInfo{
				Digits: 2,
				Ex:     &pb.SymbolInfoEx{Trade: 4},
			},
			GroupParams: &pb.ConGroupSec{MinLot: 0.01, MaxLot: 100, LotStep: 0.01, Execution: 9},
		},
	})
	params, err := gw.FetchSymbolParams(context.Background(), []string{"BTCUSDm"})
	if err != nil || len(params) != 1 {
		t.Fatalf("FetchSymbolParams: %v", err)
	}
	if params[0].TradeMode != 4 {
		t.Errorf("TradeMode = %d, want ex.Trade=4 (not gp.Execution=9)", params[0].TradeMode)
	}
}

// VM-LIVE-VENUE-R2 T1: venue trade enums come from SymbolInfoEx verbatim;
// Ex absent → -1 unknown sentinel ×3 — the Go zero value 0 would collide
// with real enum 0 (disabled / no-freeze / instant).
// Mutation M1: revert the Ex==nil branch to zero values → RED.
func TestFetchSymbolParams_Parity_TradeEnumsExOrSentinel(t *testing.T) {
	// Ex present → Trade/FreezeLevel/Exemode pass through verbatim.
	gw := parityFetchParams(t, &pb.SymbolParamsReply{
		Result: &pb.SymbolParams{
			Symbol: &pb.SymbolInfo{
				Digits: 2,
				Ex:     &pb.SymbolInfoEx{Trade: 1, FreezeLevel: 20, Exemode: 3},
			},
		},
	})
	params, err := gw.FetchSymbolParams(context.Background(), []string{"BTCUSDm"})
	if err != nil || len(params) != 1 {
		t.Fatalf("FetchSymbolParams(Ex): %v (%d params)", err, len(params))
	}
	p := params[0]
	if p.TradeMode != 1 {
		t.Errorf("TradeMode = %d, want ex.Trade=1 (long_only)", p.TradeMode)
	}
	if p.FreezeLevel != 20 {
		t.Errorf("FreezeLevel = %d, want ex.FreezeLevel=20", p.FreezeLevel)
	}
	if p.TradeExemode != 3 {
		t.Errorf("TradeExemode = %d, want ex.Exemode=3 (exchange)", p.TradeExemode)
	}

	// Ex absent → -1 unknown ×3, never 0.
	gwNil := parityFetchParams(t, &pb.SymbolParamsReply{
		Result: &pb.SymbolParams{Symbol: &pb.SymbolInfo{Digits: 2}},
	})
	paramsNil, err := gwNil.FetchSymbolParams(context.Background(), []string{"BTCUSDm"})
	if err != nil || len(paramsNil) != 1 {
		t.Fatalf("FetchSymbolParams(no Ex): %v (%d params)", err, len(paramsNil))
	}
	pn := paramsNil[0]
	if pn.TradeMode != -1 || pn.FreezeLevel != -1 || pn.TradeExemode != -1 {
		t.Errorf("Ex==nil must yield -1 sentinels, got TradeMode=%d FreezeLevel=%d TradeExemode=%d",
			pn.TradeMode, pn.FreezeLevel, pn.TradeExemode)
	}
}
