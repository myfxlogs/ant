package interp

import (
	"anttrader/strategy/sdk"
)

// CompileAndCreate is a convenience function that compiles MQL source
// (via the mql2go package) and returns a ready-to-use Interpreter.
// This is the adapter from MQL source → sdk.Strategy.
//
// Usage on host side:
//
//	strategy, err := interp.CompileAndCreate(mqlSource)
//	if err != nil { ... }
//	runner.Run(strategy, config)
//
// On WASM side, the IR is pre-compiled and deserialized:
//
//	ir := interp.DeserializeIR(data)
//	strategy := interp.NewInterpreter(ir)
//
// CompileAndCreate is only available on host (requires tree-sitter).
// To avoid import cycles, the actual CompileToIR call is done externally
// and the IR is passed to NewInterpreter.

// StrategyFactory wraps the compile+create pipeline.
// Host-side code calls this to get an sdk.Strategy from MQL source.
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

// SerializeIR returns the IR as a byte slice for WASM transfer.
// Uses a simple length-prefixed encoding (no JSON per project rules).
// This is a placeholder — full serialization will use proto.
func SerializeIR(ir *IR) []byte {
	// TODO: implement proto serialization for WASM transfer
	return nil
}

// DeserializeIR reconstructs an IR from a byte slice.
// This is a placeholder — full deserialization will use proto.
func DeserializeIR(data []byte) *IR {
	// TODO: implement proto deserialization for WASM transfer
	return nil
}
