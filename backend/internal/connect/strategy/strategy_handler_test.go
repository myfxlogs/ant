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
		InitialCapital: "10000",
	})
	resp, err := srv.RunBacktest(t.Context(), req)
	if err != nil {
		t.Fatalf("RunBacktest: %v", err)
	}
	if resp.Msg.Success {
		t.Error("expected success=false when backtestClient is nil")
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
	if resp.Msg.Success {
		t.Error("expected success=false with empty template")
	}
}

