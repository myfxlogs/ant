package strategy

// VM-LIVE-PARITY-F1/F3 link tests (T6/T7/T8 of builder-handoff-vm-live-parity-f1f3).
//
// T6  pointOrDerived: the sole allowed derivation (10^-digits when the broker
//     reports no point) — labeled definitional, adapter output stays raw.
// T7  vmSignalToProto: comment + deviation reach the proto signal; magic and
//     opposite_ticket stay unmapped by design (system magic / no CloseBy).
// T8  submitOrder: signal comment + deviation reach the mthub executor request.
//     Mutation: delete req.Comment/req.Deviation assignments → RED.

import (
	"context"
	"testing"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/strategy/sdk"

	"github.com/shopspring/decimal"
)

// ── T6: pointOrDerived ──

func TestPointOrDerived(t *testing.T) {
	cases := []struct {
		name  string
		param *mthub.SymbolParam
		want  string
	}{
		{
			name:  "broker point wins",
			param: &mthub.SymbolParam{PointValue: decimal.NewFromFloat(0.25), Digits: 2},
			want:  "0.25",
		},
		{
			name:  "zero point + digits → definitional 10^-digits",
			param: &mthub.SymbolParam{Digits: 2},
			want:  "0.01",
		},
		{
			name:  "zero point + zero digits → zero (unknown)",
			param: &mthub.SymbolParam{},
			want:  "0",
		},
	}
	for _, tc := range cases {
		if got := pointOrDerived(tc.param); got != tc.want {
			t.Errorf("%s: pointOrDerived = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// ── T7: vmSignalToProto comment/deviation mapping ──

func TestVMSignalToProto_CommentAndDeviationMapped(t *testing.T) {
	sig := &sdk.Signal{
		Action:         sdk.ActionBuy,
		Volume:         decimal.NewFromFloat(0.1),
		Comment:        "PARITY-X",
		Deviation:      3,
		Magic:          999, // deliberately NOT mapped (system magic override)
		OppositeTicket: 555, // deliberately NOT mapped (CloseBy unimplemented, R4)
	}
	pb := vmSignalToProto(sig, "BTCUSDm")
	if pb.GetComment() != "PARITY-X" {
		t.Errorf("Comment = %q, want PARITY-X", pb.GetComment())
	}
	if pb.GetDeviation() != 3 {
		t.Errorf("Deviation = %d, want 3", pb.GetDeviation())
	}
	if pb.GetMagic() != 0 {
		t.Errorf("Magic must stay unmapped (system override), got %d", pb.GetMagic())
	}
	if pb.GetOppositeTicket() != 0 {
		t.Errorf("OppositeTicket must stay unmapped (R4), got %d", pb.GetOppositeTicket())
	}
}

// ── T8: submitOrder → executor request carries comment + deviation ──

func TestSubmitOrder_CommentAndDeviationReachExecutor(t *testing.T) {
	captured := make(chan *mthub.OrderRequest, 1)
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			captured <- req
			return &mthub.OrderRecord{Ticket: 77, AccountID: req.AccountID, Canonical: req.Canonical, State: mthub.OrderStateOpen}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// Confirm the barrier once the mutation is in flight, so the synchronous
	// submitOrder completes cleanly (same pattern as the reentry tests).
	go func() {
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer waitCancel()
		sess.barrier.WaitState(waitCtx, barrierSubmitting)
		publishOrderUpdate(broker, cfg.AccountID, 77, strategyMagic(cfg.ScheduleID), "open")
	}()

	sig := &antv1.StrategySignal{
		SignalType: "buy",
		Volume:     "0.1",
		Comment:    "PARITY-X",
		Deviation:  3,
	}
	srv.submitOrder(context.Background(), cfg, mthub.SideBuy, mthub.OrderMarket, time.Now().Unix(), sig, sess)

	req := <-captured
	if req.Comment != "PARITY-X" {
		t.Errorf("executor req.Comment = %q, want PARITY-X", req.Comment)
	}
	if req.Deviation != 3 {
		t.Errorf("executor req.Deviation = %d, want 3", req.Deviation)
	}
	if state := sess.barrier.State(); state != barrierIdle {
		t.Errorf("barrier state = %s, want idle (confirmed+released)", state)
	}
}
