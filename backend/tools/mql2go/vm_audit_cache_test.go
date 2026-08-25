package mql2go

import (
	"context"
	"encoding/binary"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

func TestVM_Audit_BytecodeSerializationDeterministic(t *testing.T) {
	const source = `
extern int B = 2;
extern int A = 1;
int g_two = 2;
int g_one = 1;
int OnInit() { return 0; }
void OnTick() { first(); second(); }
void first() { g_one = A; }
void second() { g_two = B; }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	first, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}
	for i := 0; i < 100; i++ {
		data, err := MarshalBytecode(runner.Bytecode())
		if err != nil {
			t.Fatalf("MarshalBytecode iteration %d: %v", i, err)
		}
		if string(data) != string(first) {
			t.Fatalf("bytecode serialization changed at iteration %d", i)
		}
	}
}

func TestVM_Audit_CacheRejectsDifferentSource(t *testing.T) {
	const sourceA = `
int g_value = 1;
int OnInit() { return 0; }
void OnTick() {}
`
	const sourceB = `
int g_value = 2;
int OnInit() { return 0; }
void OnTick() {}
`
	runnerA, err := CompileMQL(sourceA)
	if err != nil {
		t.Fatalf("CompileMQL source A: %v", err)
	}
	data, err := MarshalBytecode(runnerA.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}
	runnerB, _, err := CompileMQLCached(sourceB, data)
	if err != nil {
		t.Fatalf("CompileMQLCached: %v", err)
	}
	if err := runnerB.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	value, ok := runnerB.GetGlobal("g_value")
	if !ok || value.ToInt() != 2 {
		t.Fatalf("cached source value = %v, want 2", value)
	}
}

func TestVM_Audit_CorruptBytecodeRejected(t *testing.T) {
	runner, err := CompileMQL(`int OnInit() { return 0; } void OnTick() {}`)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	data, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}
	if _, err := CompileMQLFromBytecode(append(data, 0xFF)); err == nil {
		t.Fatal("bytecode with trailing data was accepted")
	}
	bad := *runner.Bytecode()
	bad.Code = append(append([]Instruction(nil), bad.Code...), Instruction{Op: Opcode(255)})
	if _, err := MarshalBytecode(&bad); err == nil {
		t.Fatal("bytecode with invalid opcode was marshaled")
	}
}

// ── VM-CACHE-INTEGRITY-1 behavior tests ──────────────────────────────

// TestVM_Audit_BytecodeRoundTripEqual verifies that marshal→unmarshal→marshal
// produces byte-identical output, proving the serialization is deterministic
// and lossless. This catches map iteration non-determinism and field omission.
func TestVM_Audit_BytecodeRoundTripEqual(t *testing.T) {
	const source = `
extern int B = 2;
extern int A = 1;
int g_two = 2;
int g_one = 1;
int OnInit() { return 0; }
void OnTick() { first(); second(); }
void first() { g_one = A; }
void second() { g_two = B; }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	original, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode (original): %v", err)
	}
	bc, err := UnmarshalBytecode(original)
	if err != nil {
		t.Fatalf("UnmarshalBytecode: %v", err)
	}
	// Run 50 iterations to catch map iteration non-determinism.
	for i := 0; i < 50; i++ {
		roundTrip, err := MarshalBytecode(bc)
		if err != nil {
			t.Fatalf("MarshalBytecode (round-trip iteration %d): %v", i, err)
		}
		if string(original) != string(roundTrip) {
			t.Fatalf("round-trip serialization differs at iteration %d: original=%d bytes, round-trip=%d bytes", i, len(original), len(roundTrip))
		}
	}
}

// TestVM_Audit_CacheHitCoverageRestored verifies that a cache hit (which
// produces a runner with nil CoverageResult) can be restored by recompiling
// from source, and the restored CoverageResult has the same fatal blind spots
// as a cold compile. This proves cache hits don't silently lose reliability
// conclusions.
func TestVM_Audit_CacheHitCoverageRestored(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() { double v = iUnknownIndicator(Symbol(), 0, 14, 0); }
`
	// Cold compile — has CoverageResult with fatal blind spot.
	coldRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL (cold): %v", err)
	}
	coldCov := coldRunner.GetCoverageResult()
	if coldCov == nil {
		t.Fatal("cold compile lost CoverageResult")
	}
	coldFatalCount := 0
	for _, bs := range coldCov.BlindSpots {
		if bs.Severity == interp.SeverityFatal {
			coldFatalCount++
		}
	}
	if coldFatalCount == 0 {
		t.Fatalf("cold compile should have fatal blind spots for iUnknownIndicator, got: %+v", coldCov.BlindSpots)
	}

	// Cache the bytecode and reload — CoverageResult is nil on cache hit.
	cachedData, err := MarshalBytecode(coldRunner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}
	cachedRunner, _, err := CompileMQLCached(source, cachedData)
	if err != nil {
		t.Fatalf("CompileMQLCached: %v", err)
	}
	if cachedRunner.GetCoverageResult() != nil {
		t.Fatal("cache hit should have nil CoverageResult (bytecode cache omits coverage)")
	}

	// Restore coverage by recompiling from source (mirrors backtest_worker_vm.go:42-50).
	restoreRunner, restoreCov, err := CompileMQLWithCoverage(source)
	if err != nil {
		t.Fatalf("CompileMQLWithCoverage (restore): %v", err)
	}
	cachedRunner.InjectCoverage(restoreRunner.GetCoverage())
	cachedRunner.InjectCoverageResult(restoreCov)

	// The restored CoverageResult must have the same fatal blind spots.
	restoreCovResult := cachedRunner.GetCoverageResult()
	if restoreCovResult == nil {
		t.Fatal("restored CoverageResult is nil after InjectCoverageResult")
	}
	restoreFatalCount := 0
	for _, bs := range restoreCovResult.BlindSpots {
		if bs.Severity == interp.SeverityFatal {
			restoreFatalCount++
		}
	}
	if restoreFatalCount != coldFatalCount {
		t.Fatalf("restored fatal blind spot count = %d, want %d (same as cold compile)", restoreFatalCount, coldFatalCount)
	}
}

// TestVM_Audit_CacheHitSameSourcePreservesBytecode verifies that a cache hit
// with the same source returns the cached bytecode unchanged (no recompile).
func TestVM_Audit_CacheHitSameSourcePreservesBytecode(t *testing.T) {
	const source = `int OnInit() { return 0; } void OnTick() {}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	cached, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}
	hitRunner, hitBytecode, err := CompileMQLCached(source, cached)
	if err != nil {
		t.Fatalf("CompileMQLCached: %v", err)
	}
	if string(hitBytecode) != string(cached) {
		t.Fatal("cache hit returned different bytecode than cached — should return cached bytes unchanged")
	}
	if hitRunner.Bytecode().SourceHash != runner.Bytecode().SourceHash {
		t.Fatal("cache hit SourceHash differs from cold compile")
	}
}

// TestVM_Audit_CorruptBytecodeAttackSamples verifies that various corrupted
// bytecode payloads are all rejected by UnmarshalBytecode. This covers
// truncation, corrupt counts, bad opcodes, bad jump targets, bad builtin IDs,
// and bad constant IDs.
func TestVM_Audit_CorruptBytecodeAttackSamples(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() { g = 1; }
int g = 0;
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	valid, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(data []byte) []byte
	}{
		{
			name:   "truncated by 1 byte",
			mutate: func(d []byte) []byte { return d[:len(d)-1] },
		},
		{
			name:   "truncated by 10 bytes",
			mutate: func(d []byte) []byte { return d[:len(d)-10] },
		},
		{
			name:   "truncated to just magic",
			mutate: func(d []byte) []byte { return d[:4] },
		},
		{
			name:   "trailing garbage byte",
			mutate: func(d []byte) []byte { return append(append([]byte(nil), d...), 0xFF) },
		},
		{
			name:   "corrupt magic",
			mutate: func(d []byte) []byte { bad := append([]byte(nil), d...); bad[0] = 'X'; return bad },
		},
		{
			name:   "empty data",
			mutate: func(_ []byte) []byte { return []byte{} },
		},
		{
			name:   "single byte",
			mutate: func(_ []byte) []byte { return []byte{0x00} },
		},
		{
			name: "huge consts count (overflow remaining data)",
			mutate: func(d []byte) []byte {
				bad := append([]byte(nil), d...)
				// Overwrite consts count (after magic+version+sourceHash) with 0xFFFFFFFF.
				// Find the consts count offset: magic(4+2 len) + version(2+len) + sourceHash(2+len).
				// We just corrupt a count field to be huge — any count field works.
				// The first count after the header is consts count at a fixed offset.
				// Instead of computing offset, corrupt ALL u32 counts to max.
				// But that's complex; just corrupt the first count field.
				// magic = "BC01" (4 bytes), then string len (2 bytes) + "2026-08-24-v2" (12 bytes),
				// then string len (2) + sourceHash (64 hex chars).
				// consts count starts at 4 + 2 + 12 + 2 + 64 = 84.
				offset := 4 + 2 + len(CompilerVersion) + 2 + len(runner.Bytecode().SourceHash)
				if offset+4 <= len(bad) {
					bad[offset] = 0xFF
					bad[offset+1] = 0xFF
					bad[offset+2] = 0xFF
					bad[offset+3] = 0x7F
				}
				return bad
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			corrupt := tc.mutate(valid)
			if _, err := UnmarshalBytecode(corrupt); err == nil {
				t.Fatalf("corrupt bytecode %q was accepted (should be rejected)", tc.name)
			}
		})
	}

	// Bad opcode: marshal a bytecode with an invalid opcode in the code array.
	badBC := *runner.Bytecode()
	badBC.Code = append(append([]Instruction(nil), badBC.Code...), Instruction{Op: Opcode(255)})
	if _, err := MarshalBytecode(&badBC); err == nil {
		t.Fatal("bytecode with invalid opcode 255 was marshaled (validateBytecode should reject)")
	}

	// Bad jump target: marshal a bytecode with an out-of-range jump target.
	badJump := *runner.Bytecode()
	badJump.Code = append(append([]Instruction(nil), badJump.Code...), Instruction{Op: OP_JMP, A: 999999})
	if _, err := MarshalBytecode(&badJump); err == nil {
		t.Fatal("bytecode with out-of-range jump target was marshaled (validateBytecode should reject)")
	}

	// Bad builtin ID: marshal a bytecode with an out-of-range builtin ID.
	badBuiltin := *runner.Bytecode()
	badBuiltin.Builtins = map[string]BuiltinID{"fake": BuiltinID(60000)}
	if _, err := MarshalBytecode(&badBuiltin); err == nil {
		t.Fatal("bytecode with invalid builtin ID was marshaled (validateBytecode should reject)")
	}

	// Bad constant ID in PUSH_CONST: marshal a bytecode with out-of-range const ref.
	badConst := *runner.Bytecode()
	badConst.Code = append(append([]Instruction(nil), badConst.Code...), Instruction{Op: OP_PUSH_CONST, A: 999999})
	if _, err := MarshalBytecode(&badConst); err == nil {
		t.Fatal("bytecode with invalid constant ID was marshaled (validateBytecode should reject)")
	}
}

// ── VM-CACHE-INTEGRITY-2 behavior tests ──────────────────────────────

// TestVM_Audit_PythonCacheSourceHashVerified verifies that CompilePythonCached
// rejects cached bytecode whose SourceHash doesn't match the current source
// (stale cache from a different source version).
func TestVM_Audit_PythonCacheSourceHashVerified(t *testing.T) {
	const source1 = `
def on_bar(ctx):
    pass
`
	const source2 = `
def on_bar(ctx):
    x = 1
    return None
`
	// Compile source1 and cache it.
	r1, bcData, err := CompilePythonCached(source1, nil)
	if err != nil {
		t.Fatalf("CompilePythonCached (cold): %v", err)
	}
	if bcData == nil {
		t.Fatal("cold compile should return bytecode data for caching")
	}
	if r1.Bytecode().SourceHash == "" {
		t.Fatal("SourceHash should be set for Python bytecode")
	}

	// Use cached bytecode with a DIFFERENT source — must NOT accept the cache.
	r2, bcData2, err := CompilePythonCached(source2, bcData)
	if err != nil {
		t.Fatalf("CompilePythonCached (stale cache): %v", err)
	}
	if r2.Bytecode().SourceHash == r1.Bytecode().SourceHash {
		t.Fatal("stale cache from different source should not be accepted — SourceHash matches old source")
	}
	if bcData2 == nil {
		t.Fatal("recompiled bytecode should return new bytecode data")
	}
}

// TestVM_Audit_PythonCacheSameSourceAccepted verifies that CompilePythonCached
// accepts cached bytecode when the source hash matches.
func TestVM_Audit_PythonCacheSameSourceAccepted(t *testing.T) {
	const source = `
def on_bar(ctx):
    pass
`
	r1, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("CompilePythonCached (cold): %v", err)
	}

	// Cache hit with same source — should return the cached bytecode.
	r2, bcData2, err := CompilePythonCached(source, bcData)
	if err != nil {
		t.Fatalf("CompilePythonCached (cache hit): %v", err)
	}
	if r2.Bytecode().SourceHash != r1.Bytecode().SourceHash {
		t.Fatal("cache hit SourceHash should match cold compile")
	}
	if bcData2 == nil {
		t.Fatal("cache hit should return cached bytecode data")
	}
}

// TestVM_Audit_MarshalErrorNotSwallowed verifies that MarshalBytecode returns
// an error (not silently nil) when given invalid bytecode that fails validation.
// VM-TEST-EVIDENCE-2: the previous version of this test only checked the success
// path (normal source compiles and marshals fine), which is a false-green —
// removing the error propagation in CompileMQLCached would still pass.
func TestVM_Audit_MarshalErrorNotSwallowed(t *testing.T) {
	// Construct a bytecode with an invalid opcode to trigger validation failure.
	bc := &Bytecode{
		Code: []Instruction{{Op: Opcode(255)}}, // invalid opcode
	}
	_, err := MarshalBytecode(bc)
	if err == nil {
		t.Fatal("MarshalBytecode with invalid opcode should return error, got nil (marshal error swallowed)")
	}
}

// TestVM_Audit_DuplicateMapKeyRejected verifies that unmarshal functions
// reject duplicate keys in map sections (corrupted/tampered bytecode).
func TestVM_Audit_DuplicateMapKeyRejected(t *testing.T) {
	// Build a valid bytecode, marshal it, then corrupt the enums section
	// by duplicating a key. The unmarshal should fail.
	const source = `int OnInit() { return 0; } void OnTick() {}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	bcData, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}
	_ = bcData // we test at the unmarshal level directly

	// The bytecode format has a deterministic layout. Rather than binary
	// surgery (fragile), we test duplicate key detection at the unmarshal
	// function level directly.
	// Build a corrupted enums section with a duplicate key.
	// All integers are little-endian (matching bytecodeReader).
	var buf []byte
	// count = 2 (U32 LE)
	buf = append(buf, 2, 0, 0, 0)
	// key1 = "DUPE" (U16 LE length=4, then 4 bytes)
	buf = append(buf, 4, 0)
	buf = append(buf, []byte("DUPE")...)
	// val1 = 1 (I32 LE)
	buf = append(buf, 1, 0, 0, 0)
	// key2 = "DUPE" (duplicate!) (U16 LE length=4, then 4 bytes)
	buf = append(buf, 4, 0)
	buf = append(buf, []byte("DUPE")...)
	// val2 = 2 (I32 LE)
	buf = append(buf, 2, 0, 0, 0)

	r := &bytecodeReader{data: buf}
	enums, err := unmarshalEnums(r)
	if err == nil {
		if len(enums) != 1 {
			t.Fatalf("unmarshalEnums should reject duplicate keys (err nil but map has %d entries, expected error)", len(enums))
		}
		t.Fatal("unmarshalEnums should reject duplicate keys, got nil error and map silently lost a duplicate entry")
	}
}

// ── VM-CACHE-INTEGRITY-3 behavior tests ──────────────────────────────

// TestVM_Audit_EventLocalsDuplicateKeyRejected verifies that unmarshalEventLocals
// rejects duplicate PC keys instead of silently overwriting.
func TestVM_Audit_EventLocalsDuplicateKeyRejected(t *testing.T) {
	// Construct bytecode with duplicate EventLocals PC entries:
	// count=2, pc=0 count=1, pc=0 count=2 (duplicate PC)
	var buf []byte
	buf = binary.LittleEndian.AppendUint32(buf, 2) // count
	buf = binary.LittleEndian.AppendUint32(buf, 0) // pc=0
	buf = binary.LittleEndian.AppendUint32(buf, 1) // count=1
	buf = binary.LittleEndian.AppendUint32(buf, 0) // pc=0 (duplicate!)
	buf = binary.LittleEndian.AppendUint32(buf, 2) // count=2
	r := &bytecodeReader{data: buf}
	bc := &Bytecode{EventLocals: make(map[int32]int)}
	err := unmarshalEventLocals(r, bc)
	if err == nil {
		t.Fatal("unmarshalEventLocals should reject duplicate PC keys, got nil error")
	}
}

// TestVM_Audit_CacheCountExceedsMaxRejected verifies that readCount rejects
// absurdly large counts that exceed the maxBytecodeCount limit.
func TestVM_Audit_CacheCountExceedsMaxRejected(t *testing.T) {
	// Construct a count that exceeds maxBytecodeCount (1<<20).
	var buf []byte
	buf = binary.LittleEndian.AppendUint32(buf, maxBytecodeCount+1)
	r := &bytecodeReader{data: buf}
	_, err := r.readCount(0, "test")
	if err == nil {
		t.Fatal("readCount should reject count exceeding maxBytecodeCount, got nil error")
	}
}

// TestVM_Audit_CacheRejectsOldCompilerVersion verifies that bytecode cached
// with an older compiler version is rejected. VM-CACHE-INTEGRITY-4: after
// semantic changes (switch layout, compound assign, timeframe handling),
// the CompilerVersion must be bumped so stale v2 caches are not accepted.
func TestVM_Audit_CacheRejectsOldCompilerVersion(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() { int x = 1; }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	cached, err := MarshalBytecode(runner.Bytecode())
	if err != nil {
		t.Fatalf("MarshalBytecode: %v", err)
	}

	// Rebuild cached bytecode with an old compiler version string.
	// Format: writeString(magic) + writeString(version) + writeString(sourceHash) + ...
	// writeString = u16(len) + bytes.
	// magic = u16(4) + "BC01" = 6 bytes
	// version = u16(len) + version string
	oldVersion := "2026-08-24-v2"
	magicEnd := 2 + 4 // u16(4) + "BC01"
	origVerLen := int(binary.LittleEndian.Uint16(cached[magicEnd:]))
	origVerEnd := magicEnd + 2 + origVerLen
	rest := cached[origVerEnd:]

	var corrupted []byte
	corrupted = append(corrupted, cached[:magicEnd]...) // magic
	corrupted = binary.LittleEndian.AppendUint16(corrupted, uint16(len(oldVersion)))
	corrupted = append(corrupted, oldVersion...) // old version
	corrupted = append(corrupted, rest...)       // rest of bytecode

	_, _, err = CompileMQLCached(source, corrupted)
	if err != nil {
		t.Fatalf("CompileMQLCached should fall back to recompile (not error): %v", err)
	}

	// The old compiler version must be rejected by UnmarshalBytecode directly.
	_, err = CompileMQLFromBytecode(corrupted)
	if err == nil {
		t.Fatal("CompileMQLFromBytecode should reject old compiler version, got nil error")
	}

	// Verify CompileMQLCached returned freshly compiled bytecode (new version),
	// not the stale cached bytecode with the old version.
	freshRunner, freshBC, err := CompileMQLCached(source, corrupted)
	if err != nil {
		t.Fatalf("CompileMQLCached: %v", err)
	}
	if freshRunner == nil {
		t.Fatal("CompileMQLCached returned nil runner")
	}
	// The returned bytecode should have the current compiler version, not the old one.
	freshVerLen := int(binary.LittleEndian.Uint16(freshBC[magicEnd:]))
	freshVer := string(freshBC[magicEnd+2 : magicEnd+2+freshVerLen])
	if freshVer != CompilerVersion {
		t.Fatalf("CompileMQLCached returned bytecode with version %q, want current %q (stale cache accepted)", freshVer, CompilerVersion)
	}
}
