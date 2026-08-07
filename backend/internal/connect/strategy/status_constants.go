package strategy

// Backtest run and experiment status constants.
// These correspond to proto enum BacktestRunStatus and experiment statuses.
// Use these constants instead of raw string literals to ensure type consistency
// across the codebase and prevent silent typos.
const (
	StatusPending         = "PENDING"
	StatusRunning         = "RUNNING"
	StatusSucceeded       = "SUCCEEDED"
	StatusFailed          = "FAILED"
	StatusCancelRequested = "CANCEL_REQUESTED"
	StatusCanceled        = "CANCELED"
	StatusCompleted       = "COMPLETED" // experiment terminal status
	StatusDegraded        = "DEGRADED"  // execution succeeded but defense-line-B verdict is unreliable
)

// isTerminalBacktestStatus reports whether the given status is a terminal
// state (no further transitions possible). DEGRADED is terminal — the run
// completed but results are flagged unreliable by invariant checks (ADR-0028).
func isTerminalBacktestStatus(status string) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusCanceled, StatusDegraded:
		return true
	}
	return false
}
