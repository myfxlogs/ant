package mt5

// VM-LIVE-PARITY-F2 mt5 adapter test (T4): PointValue comes from
// SymbolInfo.points (point SIZE), not tick_value (monetary) — the old
// mapping mislabeled every symbol. Mutation: restore GetTickValue → RED.

import (
	"context"
	"testing"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestFetchSymbolParams_Parity_PointValueFromPoints(t *testing.T) {
	mock := &mockMT5Client{
		symbolParamsRes: &pb.SymbolParamsReply{
			Result: &pb.SymbolParams{
				Symbol: "EURUSD",
				SymbolInfo: &pb.SymbolInfo{
					Digits:       5,
					Points:       0.01,
					TickValue:    1.5,
					TickSize:     0.01,
					ContractSize: 100000,
					Spread:       1,
				},
				SymbolGroup: &pb.SymGroup{
					TradeMode: 4,
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
	if err != nil || len(params) != 1 {
		t.Fatalf("FetchSymbolParams: %v (%d params)", err, len(params))
	}
	p := params[0]
	if !p.PointValue.Equal(decimal.NewFromFloat(0.01)) {
		t.Errorf("PointValue = %s, want si.points 0.01 (not tick_value 1.5)", p.PointValue)
	}
	if !p.TickValue.Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("TickValue = %s, want si.tick_value 1.5", p.TickValue)
	}
	if !p.TickSize.Equal(decimal.NewFromFloat(0.01)) {
		t.Errorf("TickSize = %s, want si.tick_size 0.01", p.TickSize)
	}
}
