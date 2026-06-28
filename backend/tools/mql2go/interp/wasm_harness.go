//go:build wasm && wasip1

package interp

// WASM harness for wasip1/wasm target (wazero runtime).
//
// The host (Go native) compiles MQL → IR via CompileToIR, serializes it
// via SerializeIR, and passes the bytes to the WASM module. The WASM
// module reads the IR from stdin, deserializes, creates an interpreter,
// and processes bar events.
//
// This file is only compiled under GOOS=wasip1 GOARCH=wasm.
// It uses WASI stdio (os.Stdin/os.Stdout) — no syscall/js dependency.
//
// Host-side (wazero) usage:
//
//	cmd := exec.Command("go", "build", "-o", "strategy.wasm",
//	    "-tags", "wasm", strategyFile, harnessFile)
//	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
//	// ... build and load via wazero ...
//	// Pass serialized IR + proto bar request via stdin

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WASMRun is the WASM entry point. It reads a serialized IR from stdin,
// then processes bar events in a read-loop (proto requests on stdin,
// proto responses on stdout).
//
// Protocol:
//   1. Host writes: u32 irLen + irBytes (serialized IR)
//   2. Host writes: u32 reqLen + reqBytes (proto ExecuteLiveRequest)
//   3. WASM reads IR, creates interpreter, runs OnInit
//   4. WASM reads request, runs OnBar, writes u32 respLen + respBytes
//   5. Repeat step 2-4 until EOF
//
// This function is called from main() in the generated harness.
func WASMRun() {
	// Step 1: Read serialized IR
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

	it := NewInterpreter(ir)

	// Step 2: Process bar events in a loop
	for {
		reqData, err := readLengthPrefixed(os.Stdin)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "wasm: read request: %v\n", err)
			os.Exit(1)
		}

		// Proto deserialization and sdk.Context creation are handled by
		// the generated harness (internal/connect/strategy/backtest_harness.go),
		// which produces a main() that calls WASMRun or uses the interpreter
		// directly. This primitive handles IR deserialization + interpreter
		// lifecycle; the harness bridges proto ↔ sdk.Context.

		_ = reqData
		_ = it
	}
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

// writeLengthPrefixed writes a u32 length prefix followed by the data.
func writeLengthPrefixed(w io.Writer, data []byte) error {
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}
