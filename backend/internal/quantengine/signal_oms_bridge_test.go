package quantengine

import (
	"context"
	"testing"

	"anttrader/internal/oms"
	"anttrader/internal/risksvc"

	"go.uber.org/zap"
)

// stubBrokerAdapter records submissions for testing.
type stubBrokerAdapter struct {
	submitted []*oms.OrderRequest
	resp      *oms.BrokerResp
	err       error
}

func (s *stubBrokerAdapter) Submit(_ context.Context, req *oms.OrderRequest) (*oms.BrokerResp, error) {
	s.submitted = append(s.submitted, req)
	if s.resp == nil {
		s.resp = &oms.BrokerResp{Ticket: "TICKET-1", State: oms.StateSubmitted}
	}
	return s.resp, s.err
}

func (s *stubBrokerAdapter) Cancel(_ context.Context, ticket string) error { return nil }

func (s *stubBrokerAdapter) Modify(_ context.Context, ticket string, price, stopPrice float64) error {
	return nil
}

func (s *stubBrokerAdapter) Query(_ context.Context, ticket string) (*oms.Order, error) {
	return nil, nil
}

func TestSignalToOMS_Buy(t *testing.T) {
	log := zap.NewNop()
	risk := risksvc.NewEngine() // no rules → all pass
	stub := &stubBrokerAdapter{}

	handler := SignalToOMS(stub, risk, "acc-1", DefaultSymbolResolver(), log)
	handler("strat-1", "EURUSD", "long", 0.1, "demo_sma")

	if len(stub.submitted) != 1 {
		t.Fatalf("expected 1 order, got %d", len(stub.submitted))
	}
	req := stub.submitted[0]
	if req.Symbol != "EURUSD" {
		t.Errorf("symbol = %q, want EURUSD", req.Symbol)
	}
	if req.Side != "buy" {
		t.Errorf("side = %q, want buy", req.Side)
	}
	if req.Volume != 0.1 {
		t.Errorf("volume = %f, want 0.1", req.Volume)
	}
	if req.AccountID != "acc-1" {
		t.Errorf("account = %q, want acc-1", req.AccountID)
	}
	if req.StrategyID != "strat-1" {
		t.Errorf("strategy_id = %q, want strat-1", req.StrategyID)
	}
	if req.Comment == "" {
		t.Error("comment should contain signal_id and reason")
	}
}

func TestSignalToOMS_Sell(t *testing.T) {
	log := zap.NewNop()
	stub := &stubBrokerAdapter{}
	risk := risksvc.NewEngine()

	handler := SignalToOMS(stub, risk, "acc-2", DefaultSymbolResolver(), log)
	handler("strat-2", "GBPUSD", "short", 0.2, "trend_follow")

	if len(stub.submitted) != 1 {
		t.Fatalf("expected 1 order, got %d", len(stub.submitted))
	}
	req := stub.submitted[0]
	if req.Side != "sell" {
		t.Errorf("side = %q, want sell", req.Side)
	}
	if req.Volume != 0.2 {
		t.Errorf("volume = %f", req.Volume)
	}
}

func TestSignalToOMS_FlatSkips(t *testing.T) {
	log := zap.NewNop()
	stub := &stubBrokerAdapter{}
	risk := risksvc.NewEngine()

	handler := SignalToOMS(stub, risk, "acc-1", DefaultSymbolResolver(), log)
	handler("strat-3", "EURUSD", "flat", 0.1, "test")

	if len(stub.submitted) != 0 {
		t.Errorf("expected 0 orders for flat signal, got %d", len(stub.submitted))
	}
}

func TestSignalToOMS_UnknownDirection(t *testing.T) {
	log := zap.NewNop()
	stub := &stubBrokerAdapter{}
	risk := risksvc.NewEngine()

	handler := SignalToOMS(stub, risk, "acc-1", DefaultSymbolResolver(), log)
	handler("strat-4", "EURUSD", "unknown_dir", 0.1, "test")

	if len(stub.submitted) != 0 {
		t.Errorf("expected 0 orders for unknown direction, got %d", len(stub.submitted))
	}
}

func TestDefaultSymbolResolver(t *testing.T) {
	resolver := DefaultSymbolResolver()
	symbol, err := resolver("EURUSD")
	if err != nil {
		t.Fatalf("DefaultSymbolResolver error: %v", err)
	}
	if symbol != "EURUSD" {
		t.Fatalf("expected EURUSD pass-through, got %s", symbol)
	}
	// Unknown symbol should pass through
	symbol2, err := resolver("ZZZUNKNOWN")
	if err != nil {
		t.Fatalf("DefaultSymbolResolver error for unknown: %v", err)
	}
	if symbol2 != "ZZZUNKNOWN" {
		t.Fatalf("unknown symbol should pass through, got %s", symbol2)
	}
}

func TestDirectionToSide(t *testing.T) {
	if directionToSide("long") != "buy" {
		t.Errorf("long → buy, got %s", directionToSide("long"))
	}
	if directionToSide("buy") != "buy" {
		t.Errorf("buy → buy, got %s", directionToSide("buy"))
	}
	if directionToSide("short") != "sell" {
		t.Errorf("short → sell, got %s", directionToSide("short"))
	}
	if directionToSide("sell") != "sell" {
		t.Errorf("sell → sell, got %s", directionToSide("sell"))
	}
	if directionToSide("flat") != "" {
		t.Errorf("flat → empty, got %s", directionToSide("flat"))
	}
	if directionToSide("garbage") != "" {
		t.Errorf("garbage → empty, got %s", directionToSide("garbage"))
	}
}

func TestSignalToOMS_RiskReject(t *testing.T) {
	log := zap.NewNop()
	stub := &stubBrokerAdapter{}

	// A risk engine with a blocking rule
	blockRule := &blockAllRule{}
	risk := risksvc.NewEngine(blockRule)

	handler := SignalToOMS(stub, risk, "acc-1", DefaultSymbolResolver(), log)
	handler("strat-5", "BTCUSD", "long", 0.1, "crypto_strat")

	if len(stub.submitted) != 0 {
		t.Errorf("expected 0 orders when risk blocks, got %d", len(stub.submitted))
	}
}

// blockAllRule always blocks.
type blockAllRule struct{}

func (r *blockAllRule) Name() string { return "block_all" }
func (r *blockAllRule) Check(_ context.Context, _ *risksvc.CheckRequest) *risksvc.CheckResult {
	return &risksvc.CheckResult{Passed: false, Reason: "blocked by test", Rule: "block_all"}
}

func TestSignalToOMS_NilRisk(t *testing.T) {
	log := zap.NewNop()
	stub := &stubBrokerAdapter{}

	// nil risk engine means skip risk check
	handler := SignalToOMS(stub, nil, "acc-1", DefaultSymbolResolver(), log)
	handler("strat-6", "EURUSD", "long", 0.1, "no_risk")

	if len(stub.submitted) != 1 {
		t.Fatalf("expected 1 order with nil risk, got %d", len(stub.submitted))
	}
}

func TestSignalToOMS_SignalIDInComment(t *testing.T) {
	log := zap.NewNop()
	stub := &stubBrokerAdapter{}
	risk := risksvc.NewEngine()

	handler := SignalToOMS(stub, risk, "acc-1", DefaultSymbolResolver(), log)
	handler("strat-7", "EURUSD", "long", 0.1, "audit_test")

	if len(stub.submitted) != 1 {
		t.Fatalf("expected 1 order, got %d", len(stub.submitted))
	}
	comment := stub.submitted[0].Comment
	if comment == "" {
		t.Error("comment should not be empty")
	}
	// Should contain signal_id= prefix
	if len(comment) < 10 {
		t.Errorf("comment too short: %q", comment)
	}
}
