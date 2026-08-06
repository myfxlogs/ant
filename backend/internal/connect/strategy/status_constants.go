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
