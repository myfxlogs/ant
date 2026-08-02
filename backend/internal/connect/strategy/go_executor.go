// Package strategy — go_executor.go
//
// DEPRECATED: GoExecutor removed per Gap 3 (pre-launch).
// Go strategy execution via subprocess 'go run' has been retired.
// All strategies now route through the in-process Bytecode VM (mql2go).
// This file is retained for reference and emergency rollback only.
//
// To restore: revert this commit, re-add the SetGoExecutor injection
// in cmd/server/handlers_strategy.go, and rebuild.
package strategy

// GoExecutor is retained as a no-op stub. All call sites (Execute, ExecuteLive,
// RunBacktest) now return CodeUnimplemented directing users to convert to MQL.
type GoExecutor struct{}

// NewGoExecutor returns a no-op stub.
func NewGoExecutor(workDir string, log interface{}) *GoExecutor { return &GoExecutor{} }
