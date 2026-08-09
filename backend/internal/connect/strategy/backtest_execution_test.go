package strategy

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// mockBarSource returns fixed K-line bars for testing.
type mockBarSource struct {
	bars []*antv1.ExecuteKlineBar
}

func (m *mockBarSource) Name() string { return "mock" }
func (m *mockBarSource) Fetch(_ context.Context, _, _ string, _, _ *time.Time) ([]*antv1.ExecuteKlineBar, error) {
	return m.bars, nil
}

// TestExecuteBacktestDirect_NoDBRecord is an adversarial test (EXP-2):
// It proves that ExecuteBacktestDirect does NOT touch backtestRepo.
// The server is constructed with backtestRepo=nil; if ExecuteBacktestDirect
// or any function in its call chain dereferences backtestRepo, the test
// panics with a nil pointer dereference — the adversarial proof.
//
// The test runs a minimal MQL strategy through the full in-process path:
//   ExecuteBacktestDirect → fetchBars → executeGoBacktest → executeVMBacktest → runVMEngine
//
// runVMEngine guards failureSigRepo with nil-check and guards run.ID!=uuid.Nil
// before using runID — both are nil/uuid.Nil in this path, so no DB access occurs.
func TestExecuteBacktestDirect_NoDBRecord(t *testing.T) {
	// 10 synthetic bars with rising prices to trigger a buy in the simple strategy.
	bars := make([]*antv1.ExecuteKlineBar, 10)
	basePrice := 1.1000
	for i := range bars {
		bars[i] = &antv1.ExecuteKlineBar{
			OpenTimeMs: int64(i) * 3600_000,
			Open:       "1.1000",
			High:       "1.1100",
			Low:        "1.0900",
			Close:      "1.1001",
			Volume:     "100",
		}
		_ = basePrice
	}

	// Minimal MQL strategy: OnTick with no trades (just returns).
	// This exercises the full compile → VM → engine path without needing
	// complex indicator logic.
	mqlCode := `
int OnInit() { return 0; }
void OnTick() { return; }
`

	srv := &StrategyExecutionServer{
		// backtestRepo is intentionally nil — adversarial proof.
		// If ExecuteBacktestDirect touches it, this test panics.
		log:       zap.NewNop(),
		barSource: &mockBarSource{bars: bars},
		// mtHub=nil, failureSigRepo=nil, importedRepo=nil — all nil-guarded.
	}

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	resp, err := srv.ExecuteBacktestDirect(context.Background(), mqlCode, nil, "EURUSD", "1h", from, to)
	if err != nil {
		t.Fatalf("ExecuteBacktestDirect returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("ExecuteBacktestDirect returned nil response")
	}
	if !resp.GetSuccess() {
		t.Fatalf("ExecuteBacktestDirect returned unsuccessful response: %s", resp.GetError())
	}

	// Adversarial assertion: if we reached here, backtestRepo was never accessed.
	// The nil backtestRepo did not cause a panic — proving no DB record path is hit.
	t.Logf("adversarial proof passed: backtestRepo=nil, no panic, success=%v trades=%d",
		resp.GetSuccess(), len(resp.GetTrades()))
}

// TestExecuteBacktestDirect_EmptyCodeReturnsError verifies the guard clause.
func TestExecuteBacktestDirect_EmptyCodeReturnsError(t *testing.T) {
	srv := &StrategyExecutionServer{
		log: zap.NewNop(),
	}
	_, err := srv.ExecuteBacktestDirect(context.Background(), "", nil, "EURUSD", "1h", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for empty code, got nil")
	}
}
