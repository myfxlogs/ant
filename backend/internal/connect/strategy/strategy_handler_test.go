package strategy

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
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

func TestCancelTemplateDraft_InvalidUUID(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	req := connect.NewRequest(&antv1.CancelTemplateDraftRequest{Id: "not-a-uuid"})
	_, err := srv.CancelTemplateDraft(t.Context(), req)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestCancelTemplateDraft_NilSvc(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	req := connect.NewRequest(&antv1.CancelTemplateDraftRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
	})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when svc is nil")
		}
	}()
	srv.CancelTemplateDraft(t.Context(), req)
}

func TestStrategyServer_UserID_EmptyContext(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	uid := srv.userID(context.Background())
	if uid.String() != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected Nil UUID, got %s", uid)
	}
}

func TestStrategyServer_UserID_ValidContext(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "550e8400-e29b-41d4-a716-446655440000")
	uid := srv.userID(ctx)
	if uid.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("expected 550e8400-e29b-41d4-a716-446655440000, got %s", uid)
	}
}

func TestStrategyServer_UserID_InvalidUUID(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "not-a-uuid")
	uid := srv.userID(ctx)
	if uid.String() != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected Nil UUID for invalid input, got %s", uid)
	}
}

func TestStrategyServer_SetCodeAccessChecker(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	if srv.codeAccess != nil {
		t.Fatal("expected nil codeAccess before set")
	}
	srv.SetCodeAccessChecker(nil)
}

func TestStrategyServer_SetEngine(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	if srv.engine != nil {
		t.Fatal("expected nil engine before set")
	}
	srv.SetEngine(nil)
}

func TestStrategyServer_SetPgListen(t *testing.T) {
	t.Parallel()
	srv := &StrategyServer{log: zap.NewNop()}
	if srv.pgListen != nil {
		t.Fatal("expected nil pgListen before set")
	}
	srv.SetPgListen(nil)
}
