//go:build js && wasm

package interp

import (
	"anttrader/strategy/sdk"
)

// WASMHarness provides the entry point for running an MQL interpreter
// inside a WASM sandbox. The host (Go native) compiles MQL → IR,
// serializes it, and passes it to the WASM module which deserializes
// and executes it.
//
// This file is only compiled under GOOS=js GOARCH=wasm.

// WASMRun is the entry point called from the host via syscall/js.
// It deserializes the IR, creates an interpreter, and runs OnInit/OnBar.
//
// Host-side usage:
//
//	js.Global().Set("mqlRun", js.FuncOf(WASMRun))
//
// WASM-side: receives IR bytes + context proxy, returns signal.
func WASMRun(irData []byte, ctx sdk.Context) (*sdk.Signal, error) {
	ir := DeserializeIR(irData)
	if ir == nil {
		return nil, errWASMNoIR
	}
	it := NewInterpreter(ir)
	if err := it.OnInit(ctx); err != nil {
		return nil, err
	}
	return it.OnBar(ctx, "")
}

var errWASMNoIR = &wasmError{msg: "failed to deserialize IR"}

type wasmError struct{ msg string }

func (e *wasmError) Error() string { return "wasm: " + e.msg }
