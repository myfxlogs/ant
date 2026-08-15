package strategy

import (
	"context"
	"testing"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	"google.golang.org/protobuf/proto"
)

// TestSRD_Wiring_VMDispatchToDiag verifies the FULL capture chain:
// VM dispatch (real MQL OnTick with iMA) → sessionDiag populated →
// diagToProto produces populated StrategyDiagnostics.
// Remove the RecordEval/RecordIndicators calls in vm_live_session.go → RED.
func TestSRD_Wiring_VMDispatchToDiag(t *testing.T) {
	const code = `
		double myMA(double p) { return iMA(NULL, 0, 26, 0, 1, 0, 0); }
		void OnTick() { double v = myMA(1); }
	`
	vmSess, err := NewVMLiveSession(code)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	d := newSessionDiag()
	vmSess.SetDiag(d)

	lctx := &antv1.LiveStrategyContext{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper",
		Close: []string{"1", "2", "3", "4", "5"}, Open: []string{"1", "1", "1", "1", "1"},
		High: []string{"1", "2", "3", "4", "5"}, Low: []string{"1", "1", "1", "1", "1"},
		Volume: []string{"10", "10", "10", "10", "10"}, BarTimesMs: []int64{1, 2, 3, 4, 5},
		CurrentPrice: "5",
	}
	req := &antv1.ExecuteLiveRequest{StrategyCode: code, RequestType: antv1.RequestType_REQUEST_TYPE_BAR, BarContext: lctx}
	b, _ := proto.Marshal(req)
	if _, err := vmSess.Start(context.Background(), b); err != nil {
		t.Fatalf("start: %v", err)
	}
	// bar event → eval recorded
	tctx := &antv1.TickContext{Bid: "5", Ask: "5.1", Symbol: "EURUSD", Timeframe: "M5", Mode: "paper"}
	req2 := &antv1.ExecuteLiveRequest{StrategyCode: code, RequestType: antv1.RequestType_REQUEST_TYPE_TICK, TickContext: tctx}
	b2, _ := proto.Marshal(req2)
	resp2, err := vmSess.SendEvent(context.Background(), b2)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	t.Logf("tick resp success=%v err=%s indicators=%v", protoSuccess(resp2), protoErrField(resp2), vmSess.strategy.LastIndicators())
	time.Sleep(50 * time.Millisecond)

	snap := d.SnapshotDiag()
	if snap.EvalCount == 0 {
		t.Fatal("WIRING: evalCount==0 — dispatch does not reach sessionDiag")
	}
	p := diagToProto(snap)
	if p.GetEvalCount() == 0 {
		t.Fatal("WIRING: proto evalCount==0 — diagToProto broken")
	}
	found := false
	for _, ind := range p.GetIndicators() {
		if len(ind.GetKey()) > 3 && (ind.Key[:3] == "iMA" || ind.Key[:4] == "iMAC") {
			found = true
		}
	}
	if !found {
		t.Fatalf("WIRING: no iMA/iMACD indicator captured (got %d indicators)", len(p.GetIndicators()))
	}
}

func protoSuccess(b []byte) bool { var r antv1.ExecuteLiveResponse; _ = proto.Unmarshal(b, &r); return r.GetSuccess() }
func protoErrField(b []byte) string { var r antv1.ExecuteLiveResponse; _ = proto.Unmarshal(b, &r); return r.GetError() }
