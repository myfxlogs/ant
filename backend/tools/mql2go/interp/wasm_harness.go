//go:build wasm && wasip1

package interp

// WASM harness primitives for wasip1/wasm target (wazero runtime).
//
// The host compiles MQL → IR, serializes via SerializeIR, and passes bytes
// to the WASM module via stdin. The generated harness (backtest_harness.go)
// produces a main() that calls WASMRunSetup to get an interpreter, then
// handles proto request/response I/O itself.
//
// This file is only compiled under GOOS=wasip1 GOARCH=wasm.
// Uses WASI stdio — no syscall/js dependency.

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WASMRunSetup reads a serialized IR from stdin and returns a ready-to-use
// Interpreter. The generated harness calls this once at startup, then
// handles the bar event loop with proto I/O.
//
// Protocol: host writes u32 irLen + irBytes as the first stdin message.
func WASMRunSetup() *Interpreter {
	irData, err := readLengthPrefixed(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wasm: read IR: %v\n", err)
		os.Exit(1)
	}

	ir := DeserializeIR(irData)
	if ir == nil {
		fmt.Fprintf(os.Stderr, "wasm: deserialize IR failed\n")
		os.Exit(1)
	}

	return NewInterpreter(ir)
}

// readLengthPrefixed reads a u32 length prefix followed by that many bytes.
func readLengthPrefixed(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(lenBuf[:])
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}
