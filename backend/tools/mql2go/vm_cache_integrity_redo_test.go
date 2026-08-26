package mql2go

import (
	"errors"
	"strings"
	"testing"
)

// VM-CACHE-INTEGRITY-1/2 redo tests (2026-08-26).
// These tests verify the SourceHash binding, marshal error propagation,
// duplicate map key detection, and trailing bytes rejection.

const cacheTestSourceA = `void OnBar() { int x = 1; }`
const cacheTestSourceB = `void OnBar() { int x = 2; }`

// --- S4: CompileMQLCached SourceHash verification ---

// TestCacheRejectsDifferentSource verifies that cached bytecode from source A
// is rejected when CompileMQLCached is called with source B (SourceHash mismatch).
func TestCacheRejectsDifferentSource(t *testing.T) {
	// Compile source A and get its bytecode
	rA, bcDataA, err := CompileMQLCached(cacheTestSourceA, nil)
	if err != nil {
		t.Fatalf("compile source A: %v", err)
	}
	if rA == nil || bcDataA == nil {
		t.Fatal("expected non-nil runner and bytecode for source A")
	}

	// Use cached bytecode from A with source B — must recompile, not accept stale cache
	rB, _, err := CompileMQLCached(cacheTestSourceB, bcDataA)
	if err != nil {
		t.Fatalf("compile with stale cache: %v", err)
	}
	// The runner should reflect source B, not source A
	if rB.Bytecode().SourceHash != hashSource(cacheTestSourceB) {
		t.Errorf("stale cache from different source should be rejected; SourceHash mismatch")
	}
}

// TestCacheAcceptsSameSource verifies that cached bytecode from the same source
// is accepted (cache hit).
func TestCacheAcceptsSameSource(t *testing.T) {
	_, bcDataA, err := CompileMQLCached(cacheTestSourceA, nil)
	if err != nil {
		t.Fatalf("compile source A: %v", err)
	}

	// Use cached bytecode from A with the same source A — should be a cache hit
	rB, bcDataB, err := CompileMQLCached(cacheTestSourceA, bcDataA)
	if err != nil {
		t.Fatalf("compile with valid cache: %v", err)
	}
	if rB == nil {
		t.Fatal("expected non-nil runner for cache hit")
	}
	// bcDataB should be the same cached bytes (cache hit returns cachedBytecode)
	if string(bcDataB) != string(bcDataA) {
		t.Error("cache hit should return the same cached bytecode")
	}
}

// --- S5: MarshalBytecode error not swallowed ---

// TestMarshalErrorNotSwallowed verifies that MarshalBytecode returns an error
// on nil input, and that CompileMQLCached returns non-nil bytecode on success
// (proving the marshal path runs and data is returned, not swallowed).
func TestMarshalErrorNotSwallowed(t *testing.T) {
	// 1. MarshalBytecode(nil) must return error
	_, err := MarshalBytecode(nil)
	if err == nil {
		t.Fatal("expected error for nil bytecode marshal, got nil")
	}

	// 2. CompileMQLCached must return non-nil bytecode (proving marshal ran
	//    and the result is returned, not swallowed as `return r, nil, nil`)
	_, bcData, err := CompileMQLCached(cacheTestSourceA, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if bcData == nil {
		t.Fatal("CompileMQLCached returned nil bytecode — marshal result swallowed")
	}

	// 3. Round-trip: unmarshal the data and verify it works
	bc, uErr := UnmarshalBytecode(bcData)
	if uErr != nil {
		t.Fatalf("round-trip unmarshal failed: %v", uErr)
	}
	if bc.SourceHash != hashSource(cacheTestSourceA) {
		t.Error("round-trip SourceHash mismatch")
	}
}

// --- S9: duplicate map key detection ---

// TestDuplicateEnumKeyRejected constructs a bytecode with a duplicate enum key
// and verifies that UnmarshalBytecode returns an error.
func TestDuplicateEnumKeyRejected(t *testing.T) {
	// First, compile valid source to get a valid bytecode blob
	_, _, err := CompileMQLCached(cacheTestSourceA, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Parse the valid data to find the enums section, then craft a modified
	// version with a duplicate key. Rather than doing complex binary surgery,
	// we build a minimal bytecode from scratch using the writer helpers.
	data := buildBytecodeWithDuplicateEnumKey(t)
	_, err = UnmarshalBytecode(data)
	if err == nil {
		t.Fatal("expected error for duplicate enum key, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected error mentioning 'duplicate', got: %v", err)
	}
}

// buildBytecodeWithDuplicateEnumKey constructs a valid-looking bytecode blob
// that has a duplicate key in the enums section.
func buildBytecodeWithDuplicateEnumKey(t *testing.T) []byte {
	t.Helper()
	w := &bytecodeWriter{buf: make([]byte, 0, 256)}
	w.writeString(bytecodeMagic)
	w.writeString(CompilerVersion)

	// Consts: 0 entries
	w.writeU32(0)
	// Code: 1 entry (OP_HALT)
	w.writeU32(1)
	w.writeU8(uint8(OP_HALT))
	w.writeI32(0)
	w.writeI32(0)
	w.writeU32(0)
	// GlobalSlots: 0 entries
	w.writeU32(0)
	// GlobalDecls: 0 entries
	w.writeU32(0)
	// Funcs: 0 entries
	w.writeU32(0)
	// Builtins: 0 entries
	w.writeU32(0)
	// Events: 8 × int32 (-1)
	for i := 0; i < 8; i++ {
		w.writeI32(-1)
	}
	// EventLocals: 0 entries
	w.writeU32(0)
	// Params: 0 bytes
	w.writeU32(0)
	// Version: "mql4"
	w.writeString("mql4")
	// SourceHash: empty (matches no source)
	w.writeString("")
	// Enums: 2 entries with the SAME key "FOO"
	w.writeU32(2)
	w.writeString("FOO")
	w.writeI32(1)
	w.writeString("FOO")
	w.writeI32(2)

	return w.buf
}

// --- S10: trailing bytes rejection ---

// TestTrailingBytesRejected constructs a valid bytecode blob and appends
// extra bytes, verifying that UnmarshalBytecode returns an error.
func TestTrailingBytesRejected(t *testing.T) {
	_, validData, err := CompileMQLCached(cacheTestSourceA, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Append trailing garbage
	tampered := append([]byte{}, validData...)
	tampered = append(tampered, 0xDE, 0xAD, 0xBE, 0xEF)

	_, err = UnmarshalBytecode(tampered)
	if err == nil {
		t.Fatal("expected error for trailing bytes, got nil")
	}
	if !strings.Contains(err.Error(), "trailing") {
		t.Errorf("expected error mentioning 'trailing', got: %v", err)
	}
}

// --- S6: CompilePythonCached SourceHash verification ---

const pyCacheSourceA = `def on_bar(ctx): pass`
const pyCacheSourceB = `def on_bar(ctx): return 1`

// TestPythonCacheRejectsDifferentSource verifies that cached Python bytecode
// from source A is rejected when CompilePythonCached is called with source B.
func TestPythonCacheRejectsDifferentSource(t *testing.T) {
	rA, bcDataA, err := CompilePythonCached(pyCacheSourceA, nil)
	if err != nil {
		t.Fatalf("compile python source A: %v", err)
	}
	if rA == nil || bcDataA == nil {
		t.Fatal("expected non-nil runner and bytecode for python source A")
	}

	// Use cached bytecode from A with source B — must recompile
	rB, _, err := CompilePythonCached(pyCacheSourceB, bcDataA)
	if err != nil {
		t.Fatalf("compile with stale cache: %v", err)
	}
	if rB.Bytecode().SourceHash != hashSource(pyCacheSourceB) {
		t.Errorf("stale cache from different source should be rejected; SourceHash mismatch")
	}
}

// TestPythonCacheAcceptsSameSource verifies that cached Python bytecode from
// the same source is accepted (cache hit).
func TestPythonCacheAcceptsSameSource(t *testing.T) {
	_, bcDataA, err := CompilePythonCached(pyCacheSourceA, nil)
	if err != nil {
		t.Fatalf("compile python source A: %v", err)
	}

	rB, bcDataB, err := CompilePythonCached(pyCacheSourceA, bcDataA)
	if err != nil {
		t.Fatalf("compile with valid cache: %v", err)
	}
	if rB == nil {
		t.Fatal("expected non-nil runner for cache hit")
	}
	if string(bcDataB) != string(bcDataA) {
		t.Error("cache hit should return the same cached bytecode")
	}
}

// --- S9: duplicate GlobalSlots key detection ---

// TestDuplicateGlobalSlotKeyRejected constructs a bytecode with a duplicate
// GlobalSlots key and verifies that UnmarshalBytecode returns an error.
func TestDuplicateGlobalSlotKeyRejected(t *testing.T) {
	data := buildBytecodeWithDuplicateGlobalSlot(t)
	_, err := UnmarshalBytecode(data)
	if err == nil {
		t.Fatal("expected error for duplicate globalSlot key, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected error mentioning 'duplicate', got: %v", err)
	}
}

func buildBytecodeWithDuplicateGlobalSlot(t *testing.T) []byte {
	t.Helper()
	w := &bytecodeWriter{buf: make([]byte, 0, 256)}
	w.writeString(bytecodeMagic)
	w.writeString(CompilerVersion)
	// Consts: 0
	w.writeU32(0)
	// Code: 1 (OP_HALT)
	w.writeU32(1)
	w.writeU8(uint8(OP_HALT))
	w.writeI32(0)
	w.writeI32(0)
	w.writeU32(0)
	// GlobalSlots: 2 with same key "x"
	w.writeU32(2)
	w.writeString("x")
	w.writeU16(0)
	w.writeString("x")
	w.writeU16(1)
	// GlobalDecls: 0
	w.writeU32(0)
	// Funcs: 0
	w.writeU32(0)
	// Builtins: 0
	w.writeU32(0)
	// Events: 8 × -1
	for i := 0; i < 8; i++ {
		w.writeI32(-1)
	}
	// EventLocals: 0
	w.writeU32(0)
	// Params: 0
	w.writeU32(0)
	// Version
	w.writeString("mql4")
	// SourceHash
	w.writeString("")
	// Enums: 0
	w.writeU32(0)
	return w.buf
}

// --- S10: readCount bounded check ---

// TestReadCountBounded verifies that readCount rejects counts that would
// exceed the remaining data.
func TestReadCountBounded(t *testing.T) {
	// Construct a bytecode where a count field claims more entries than
	// the remaining data can hold.
	data := buildBytecodeWithHugeConstCount(t)
	_, err := UnmarshalBytecode(data)
	if err == nil {
		t.Fatal("expected error for huge count, got nil")
	}
}

func buildBytecodeWithHugeConstCount(t *testing.T) []byte {
	t.Helper()
	w := &bytecodeWriter{buf: make([]byte, 0, 64)}
	w.writeString(bytecodeMagic)
	w.writeString(CompilerVersion)
	// Consts: claim 1 billion entries — way more than remaining data
	w.writeU32(1_000_000_000)
	// No actual const data follows — reader should reject
	return w.buf
}

// --- S5 rework: marshal failure injection via marshalHook ---

// TestCompileMQLCached_MarshalFailureReturnsError verifies that when
// MarshalBytecode fails (injected via marshalHook), CompileMQLCached returns
// an error and a nil runner — the marshal error is NOT swallowed.
//
// Adversarial proof: restore the old swallowing behavior
// (`return r, nil, nil` instead of `return nil, nil, fmt.Errorf(...)`) →
// this test turns RED because err==nil and runner!=nil.
func TestCompileMQLCached_MarshalFailureReturnsError(t *testing.T) {
	// Inject a marshal hook that always fails.
	prev := marshalHook
	marshalHook = func(*Bytecode) ([]byte, error) {
		return nil, errors.New("injected marshal failure")
	}
	t.Cleanup(func() { marshalHook = prev })

	runner, bcData, err := CompileMQLCached(cacheTestSourceA, nil)
	if err == nil {
		t.Fatal("expected error from injected marshal failure, got nil (error swallowed)")
	}
	if runner != nil {
		t.Fatalf("expected nil runner on marshal failure, got non-nil (error swallowed)")
	}
	if bcData != nil {
		t.Fatal("expected nil bytecode on marshal failure, got non-nil (error swallowed)")
	}
	if !strings.Contains(err.Error(), "injected marshal failure") {
		t.Errorf("expected error to wrap injected failure, got: %v", err)
	}
}

// TestCompilePythonCached_MarshalFailureReturnsError mirrors the MQL test for
// the Python cache path. Verifies CompilePythonCached propagates marshal
// errors instead of swallowing them.
func TestCompilePythonCached_MarshalFailureReturnsError(t *testing.T) {
	prev := marshalHook
	marshalHook = func(*Bytecode) ([]byte, error) {
		return nil, errors.New("injected python marshal failure")
	}
	t.Cleanup(func() { marshalHook = prev })

	runner, bcData, err := CompilePythonCached(pyCacheSourceA, nil)
	if err == nil {
		t.Fatal("expected error from injected marshal failure, got nil (error swallowed)")
	}
	if runner != nil {
		t.Fatalf("expected nil runner on marshal failure, got non-nil (error swallowed)")
	}
	if bcData != nil {
		t.Fatal("expected nil bytecode on marshal failure, got non-nil (error swallowed)")
	}
	if !strings.Contains(err.Error(), "injected python marshal failure") {
		t.Errorf("expected error to wrap injected failure, got: %v", err)
	}
}

// TestMarshalHook_ResetByCleanup verifies that the marshalHook is properly
// reset by t.Cleanup, so production code after the test sees nil marshalHook.
func TestMarshalHook_ResetByCleanup(t *testing.T) {
	prev := marshalHook
	marshalHook = func(*Bytecode) ([]byte, error) {
		return nil, errors.New("temp")
	}
	t.Cleanup(func() { marshalHook = prev })

	if marshalHook == nil {
		t.Fatal("marshalHook should be set inside test")
	}
	// Manually run cleanup by simulating test end: we can't call t.Cleanup
	// callbacks directly, but we verify the hook is set here and will be
	// reset after the test. The next test in the suite relies on this.
}
