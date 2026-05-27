package strategy

import (
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
)

func TestRunBacktest_MockFallback(t *testing.T) {
	srv := &StrategyServer{log: zap.NewNop()} // svc and client are nil
	req := connect.NewRequest(&antv1.RunBacktestRequest{
		TemplateId:     "00000000-0000-0000-0000-000000000001",
		Symbol:         "EURUSD",
		Timeframe:      "1h",
		InitialCapital: 10000,
	})
	resp, err := srv.RunBacktest(t.Context(), req)
	if err != nil {
		t.Fatalf("RunBacktest: %v", err)
	}
	if !resp.Msg.Success {
		t.Error("expected success=true in mock fallback")
	}
	if resp.Msg.Metrics != nil {
		t.Log("metrics populated from python (expected nil in mock fallback)")
	}
}

func TestRunBacktest_NoTemplateId(t *testing.T) {
	srv := &StrategyServer{log: zap.NewNop()}
	req := connect.NewRequest(&antv1.RunBacktestRequest{
		Symbol:    "EURUSD",
		Timeframe: "1h",
	})
	resp, err := srv.RunBacktest(t.Context(), req)
	if err != nil {
		t.Fatalf("RunBacktest: %v", err)
	}
	if !resp.Msg.Success {
		t.Error("expected success=true with empty template (mock fallback)")
	}
}

func TestSetClient_StrategyServer(t *testing.T) {
	srv := NewStrategyServer(nil, zap.NewNop())
	if srv.client != nil {
		t.Error("expected nil client initially")
	}
	// SetClient with nil: no panic.
	srv.SetClient(nil)
	if srv.client != nil {
		t.Error("expected nil client after SetClient(nil)")
	}
}
