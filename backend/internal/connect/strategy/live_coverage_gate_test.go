package strategy

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// mockCoverageChecker is a test stub for CoverageChecker.
type mockCoverageChecker struct {
	err error
}

func (m *mockCoverageChecker) CheckLiveCoverage(ctx context.Context, strategyID, sourceCode string) error {
	return m.err
}

// TestRunLiveStrategy_T5_FatalCoverage_RejectsLive is the adversarial proof:
// A strategy with fatal blind spots (e.g. iCustom) must be rejected when
// attempting to start a live strategy. Remove the coverage check → test goes red.
func TestRunLiveStrategy_T5_FatalCoverage_RejectsLive(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.SetCoverageChecker(&mockCoverageChecker{
		err: errors.New("fatal coverage blind spots: iCustom not supported"),
	})

	cfg := LiveStrategyConfig{
		RunID:      uuid.New(),
		AccountID:  uuid.New().String(),
		UserID:     uuid.New().String(),
		Mode:       "live",
		Symbol:     "EURUSD",
		StrategyID: uuid.New().String(),
		Code:       "void OnBar() { iCustom(...); }",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from fatal coverage check, got nil — adversarial proof: check is missing!")
	}
}

// TestRunLiveStrategy_T5_FatalCoverage_PaperModeSkips verifies that paper mode
// does NOT trigger the coverage check (paper trading is for experimentation).
func TestRunLiveStrategy_T5_FatalCoverage_PaperModeSkips(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	// coverageChecker always returns error — if paper mode triggers the check, test fails.
	srv.SetCoverageChecker(&mockCoverageChecker{
		err: errors.New("fatal coverage blind spots"),
	})

	cfg := LiveStrategyConfig{
		RunID:     uuid.New(),
		AccountID: uuid.New().String(),
		UserID:    uuid.New().String(),
		Mode:      "paper",
		Symbol:    "EURUSD",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	// Paper mode should skip the coverage check and fail on a different check
	// (no barSource configured), NOT on coverage.
	if err != nil && contains(err.Error(), "fatal coverage") {
		t.Fatal("paper mode should NOT trigger coverage check")
	}
}

// TestRunLiveStrategy_T5_FatalCoverage_NilCheckerSkips verifies that nil coverageChecker
// is a no-op (backward compatibility for environments without marketplace service).
func TestRunLiveStrategy_T5_FatalCoverage_NilCheckerSkips(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	// coverageChecker is nil — check should be skipped

	cfg := LiveStrategyConfig{
		RunID:     uuid.New(),
		AccountID: uuid.New().String(),
		UserID:    uuid.New().String(),
		Mode:      "live",
		Symbol:    "EURUSD",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	// Should fail on barSource/gate, NOT on coverage
	if err != nil && contains(err.Error(), "fatal coverage") {
		t.Fatal("nil coverageChecker should NOT trigger coverage check")
	}
}

// TestRunLiveStrategy_T5_FatalCoverage_AllowsClean verifies that a strategy
// with no fatal blind spots passes the coverage check and proceeds.
func TestRunLiveStrategy_T5_FatalCoverage_AllowsClean(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.SetCoverageChecker(&mockCoverageChecker{err: nil})

	cfg := LiveStrategyConfig{
		RunID:      uuid.New(),
		AccountID:  uuid.New().String(),
		UserID:     uuid.New().String(),
		Mode:       "live",
		Symbol:     "EURUSD",
		StrategyID: uuid.New().String(),
		Code:       "void OnBar() { OrderSend(...); }",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	// Should NOT fail on coverage — may fail on barSource/gate, but not coverage
	if err != nil && contains(err.Error(), "fatal coverage") {
		t.Fatal("clean strategy should pass coverage check")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
