package interp

import (
	"anttrader/strategy/sdk"
)

// StrategyFactory wraps the compile+create pipeline.
// Host-side code calls this to get an sdk.Strategy from MQL source.
//
// Usage:
//
//	ir, err := mql2go.CompileToIR(source)
//	factory := interp.NewStrategyFactory(ir)
//	strategy := factory.Create() // returns sdk.Strategy
//	runner.Run(strategy, config)
//
// For WASM: host serializes IR via SerializeIR, WASM module
// deserializes via DeserializeIR, then creates interpreter.
type StrategyFactory struct {
	IR *IR
}

// NewStrategyFactory creates a factory from a pre-compiled IR.
func NewStrategyFactory(ir *IR) *StrategyFactory {
	return &StrategyFactory{IR: ir}
}

// Create returns a new Interpreter instance for backtesting.
// Each call creates a fresh interpreter with its own state.
func (f *StrategyFactory) Create() sdk.Strategy {
	return NewInterpreter(f.IR)
}
