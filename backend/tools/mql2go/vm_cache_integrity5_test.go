package mql2go

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// ── VM-CACHE-INTEGRITY-5 behavior tests (返工第三阶段) ────────────────

// TestCompilePythonCached_RestoresCoverageOnCacheHit verifies that
// CompilePythonCached restores CoverageResult when loading from cache.
// VM-CACHE-INTEGRITY-5: previously cache hit returned nil CoverageResult.
func TestCompilePythonCached_RestoresCoverageOnCacheHit(t *testing.T) {
	source := `def on_bar(ctx, bars):
    ctx.signal("buy")
`
	// First compile to get bytecode.
	r1, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("first compile: %v", err)
	}
	if r1 == nil || bcData == nil {
		t.Fatal("first compile returned nil runner or bytecode")
	}
	// First compile should have CoverageResult.
	if r1.GetCoverageResult() == nil {
		t.Fatal("first compile should have CoverageResult")
	}

	// Second compile with cached bytecode — should restore CoverageResult.
	r2, _, err := CompilePythonCached(source, bcData)
	if err != nil {
		t.Fatalf("second compile (cache hit): %v", err)
	}
	if r2 == nil {
		t.Fatal("second compile returned nil runner")
	}
	if r2.GetCoverageResult() == nil {
		t.Fatal("cache hit should restore CoverageResult (VM-CACHE-INTEGRITY-5), got nil")
	}
}

// TestCompilePythonCached_RejectsMQLBytecode verifies that CompilePythonCached
// rejects cached MQL bytecode when compiling Python source, even if the
// SourceHash is manipulated to match.
// VM-CACHE-INTEGRITY-5: language (Version) must match.
// Adversarial: remove the Version == "python" check → RED (MQL bytecode accepted).
func TestCompilePythonCached_RejectsMQLBytecode(t *testing.T) {
	// Compile MQL source to get MQL bytecode.
	mqlSource := `void OnInit() { }`
	mqlRunner, mqlBc, err := CompileMQLCached(mqlSource, nil)
	if err != nil {
		t.Fatalf("compile MQL: %v", err)
	}
	if mqlRunner == nil || mqlBc == nil {
		t.Fatal("MQL compile returned nil")
	}
	// Verify the MQL bytecode has Version "mql4" or "mql5" (not "python").
	mqlVer := mqlRunner.Bytecode().Version
	if mqlVer == "python" {
		t.Fatalf("MQL bytecode has Version='python' (unexpected)")
	}

	// Construct a Python source whose hash matches the MQL bytecode's SourceHash.
	pySource := `def on_bar(ctx, bars):
    ctx.signal("buy")
`
	targetHash := mqlRunner.Bytecode().SourceHash
	// Re-marshal the MQL bytecode with the Python source's hash to simulate
	// a cross-language cache collision.
	mqlRunner.Bytecode().SourceHash = hashSource(pySource)
	tamperedBc, err := MarshalBytecode(mqlRunner.Bytecode())
	if err != nil {
		t.Fatalf("marshal tampered bytecode: %v", err)
	}
	// Restore the original hash for cleanup.
	mqlRunner.Bytecode().SourceHash = targetHash

	// Try to use tampered MQL bytecode as cache for Python source.
	// VM-CACHE-INTEGRITY-5: should reject because Version != "python".
	r2, _, err := CompilePythonCached(pySource, tamperedBc)
	if err != nil {
		t.Fatalf("CompilePythonCached with tampered MQL bytecode: %v", err)
	}
	if r2 == nil {
		t.Fatal("CompilePythonCached returned nil runner")
	}
	// The runner should be Python-compiled (Version == "python"), not MQL.
	if r2.Bytecode().Version != "python" {
		t.Errorf("Version=%q, want 'python' (MQL bytecode should be rejected for Python source even with matching SourceHash)", r2.Bytecode().Version)
	}
}

// TestUnmarshalBytecode_PayloadLimitExceedsMax verifies that UnmarshalBytecode
// rejects payloads exceeding the max size limit with a specific error message.
// VM-CACHE-INTEGRITY-5 返工: must assert the specific rejection contract
// ("exceeds max"), not just err != nil.
func TestUnmarshalBytecode_PayloadLimitExceedsMax(t *testing.T) {
	// Construct a payload larger than maxBytecodePayload.
	huge := make([]byte, maxBytecodePayload+1)
	_, err := UnmarshalBytecode(huge)
	if err == nil {
		t.Fatal("UnmarshalBytecode should reject payload exceeding maxBytecodePayload")
	}
	// VM-CACHE-INTEGRITY-5 返工: assert specific rejection contract.
	if !strings.Contains(err.Error(), "exceeds max") {
		t.Fatalf("error should contain 'exceeds max', got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "payload size") {
		t.Fatalf("error should contain 'payload size', got: %s", err.Error())
	}
}

// TestUnmarshalBytecode_PayloadExactlyAtLimit verifies that a payload exactly
// at the limit is NOT rejected (boundary check: > not >=).
func TestUnmarshalBytecode_PayloadExactlyAtLimit(t *testing.T) {
	// Construct a payload exactly at maxBytecodePayload.
	// It will fail for other reasons (invalid magic), but should NOT fail
	// with the payload-size error.
	exact := make([]byte, maxBytecodePayload)
	_, err := UnmarshalBytecode(exact)
	if err != nil && strings.Contains(err.Error(), "exceeds max") {
		t.Fatalf("payload at exact limit should not be rejected for size, got: %s", err.Error())
	}
}

// TestUnmarshalBytecode_TruncatedPayload verifies that a truncated payload
// is rejected (structure attack sample).
func TestUnmarshalBytecode_TruncatedPayload(t *testing.T) {
	// Compile valid bytecode then truncate it.
	source := `int OnInit() { return 0; }`
	_, bcData, err := CompileMQLCached(source, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(bcData) < 10 {
		t.Fatal("bytecode too short for truncation test")
	}
	truncated := bcData[:len(bcData)/2]
	_, err = UnmarshalBytecode(truncated)
	if err == nil {
		t.Fatal("truncated bytecode should be rejected")
	}
}

// TestUnmarshalBytecode_TrailingGarbage verifies that bytecode with trailing
// garbage after the valid payload is rejected (structure attack sample).
// VM-CACHE-INTEGRITY-5 round 4: must assert specific trailing-data error,
// not t.Log and pass. The production code at bytecode_cache.go:229-231
// rejects trailing data — this test must fail if that check is removed.
func TestUnmarshalBytecode_TrailingGarbage(t *testing.T) {
	source := `int OnInit() { return 0; }`
	_, bcData, err := CompileMQLCached(source, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// Append garbage.
	tampered := append(bcData, []byte{0xFF, 0xFF, 0xFF, 0xFF}...)
	_, err = UnmarshalBytecode(tampered)
	if err == nil {
		t.Fatal("trailing garbage should be rejected — bytecode has trailing data check at bytecode_cache.go:229-231")
	}
	// VM-CACHE-INTEGRITY-5 round 4: assert specific trailing-data error.
	if !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("error should mention trailing data, got: %s", err.Error())
	}
}

// TestUnmarshalBytecode_InvalidMagic verifies that an invalid magic string
// is rejected (structure attack sample).
func TestUnmarshalBytecode_InvalidMagic(t *testing.T) {
	// Start with a valid payload then corrupt the magic.
	source := `int OnInit() { return 0; }`
	_, bcData, err := CompileMQLCached(source, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(bcData) < 4 {
		t.Fatal("bytecode too short for magic test")
	}
	// Corrupt the first few bytes (magic).
	corrupted := make([]byte, len(bcData))
	copy(corrupted, bcData)
	corrupted[0] = 'X'
	corrupted[1] = 'X'
	_, err = UnmarshalBytecode(corrupted)
	if err == nil {
		t.Fatal("invalid magic should be rejected")
	}
	if !strings.Contains(err.Error(), "magic") {
		t.Fatalf("error should mention magic, got: %s", err.Error())
	}
}

// TestCompilePythonCached_CoverageRestoreFailureReturnsError verifies that
// when coverage restoration fails, the error is returned, not silently ignored.
// VM-CACHE-INTEGRITY-5 round 5: the injected function returns NON-NIL runner
// and NON-NIL coverage plus error, so that deleting the covErr check does NOT
// fall through to the cov==nil check (which would also return error → false
// GREEN). Instead, deleting covErr → cov != nil → InjectCoverage succeeds →
// cache hit returns nil error → test expects error → RED.
func TestCompilePythonCached_CoverageRestoreFailureReturnsError(t *testing.T) {
	// First compile valid Python to get cached bytecode.
	validSource := `def on_bar(ctx, bars):
    ctx.signal("buy")
`
	_, bcData, err := CompilePythonCached(validSource, nil)
	if err != nil {
		t.Fatalf("first compile: %v", err)
	}

	// Inject a coverage compiler that returns non-nil runner, non-nil
	// coverage, AND an error. This ensures the covErr branch is the ONLY
	// thing preventing silent degradation — if covErr is deleted, the
	// non-nil cov skips the cov==nil check and InjectCoverage succeeds,
	// making the cache hit return nil error (test expects error → RED).
	// VM-CACHE-INTEGRITY-5 round 5: must use a distinct error sentinel.
	injectedErr := fmt.Errorf("simulated coverage compiler failure (sentinel: COVERAGE_RESTORE_FAIL_5F3A)")
	dummyCov := &CoverageResult{BlindSpots: []CoverageBlindSpot{{Builtin: "test", Severity: "high"}}}
	restore := setCompilePythonWithCoverageFn(func(s string) (*VMRunner, *CoverageResult, error) {
		// Return non-nil runner (compile a fresh one), non-nil coverage, and error.
		r, _, e := CompilePythonWithCoverage(s)
		if e != nil {
			return nil, nil, fmt.Errorf("inner compile: %w", e)
		}
		return r, dummyCov, injectedErr
	})
	defer restore()

	// Now try cache hit — coverage restoration should fail and return error.
	_, _, err = CompilePythonCached(validSource, bcData)
	if err == nil {
		t.Fatal("CompilePythonCached should return error when coverage restoration fails, not silently degrade")
	}
	if !strings.Contains(err.Error(), "restore coverage") {
		t.Fatalf("error should mention 'restore coverage', got: %s", err.Error())
	}
	// VM-CACHE-INTEGRITY-5 round 5: assert the error wraps our sentinel,
	// proving the covErr branch (not the cov==nil branch) caught it.
	if !strings.Contains(err.Error(), "COVERAGE_RESTORE_FAIL_5F3A") {
		t.Fatalf("error should contain sentinel COVERAGE_RESTORE_FAIL_5F3A, got: %s", err.Error())
	}
}

// TestCompilePythonCached_CoverageRestoreNilCoverageReturnsError verifies
// that when coverage restoration returns nil CoverageResult (without error),
// the error is returned, not silently ignored.
func TestCompilePythonCached_CoverageRestoreNilCoverageReturnsError(t *testing.T) {
	validSource := `def on_bar(ctx, bars):
    ctx.signal("buy")
`
	_, bcData, err := CompilePythonCached(validSource, nil)
	if err != nil {
		t.Fatalf("first compile: %v", err)
	}

	// Inject a coverage compiler that returns nil coverage without error.
	restore := setCompilePythonWithCoverageFn(func(s string) (*VMRunner, *CoverageResult, error) {
		return nil, nil, nil // nil coverage, nil error
	})
	defer restore()

	_, _, err = CompilePythonCached(validSource, bcData)
	if err == nil {
		t.Fatal("CompilePythonCached should return error when coverage restoration returns nil CoverageResult")
	}
	if !strings.Contains(err.Error(), "CoverageResult is nil") {
		t.Fatalf("error should mention 'CoverageResult is nil', got: %s", err.Error())
	}
}

// TestCompilePythonCached_ColdCompileHasCoverage verifies that a cold compile
// (no cache) produces a runner with CoverageResult.
func TestCompilePythonCached_ColdCompileHasCoverage(t *testing.T) {
	source := `def on_bar(ctx, bars):
    ctx.signal("buy")
`
	r, _, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("cold compile: %v", err)
	}
	if r.GetCoverageResult() == nil {
		t.Fatal("cold compile should have CoverageResult")
	}
}

// TestCompilePythonCached_CacheHitVsColdCompileCoverageEqual verifies that
// the CoverageResult from a cache hit matches the one from a cold compile
// (same fatal severity, same blind spot identity, same Defense A results).
// VM-CACHE-INTEGRITY-5 round 4: compare severity, blind spot identity and
// Defense A results, not just counts.
func TestCompilePythonCached_CacheHitVsColdCompileCoverageEqual(t *testing.T) {
	source := `def on_bar(ctx, bars):
    ctx.signal("buy")
`
	coldRunner, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("cold compile: %v", err)
	}
	coldCov := coldRunner.GetCoverageResult()
	if coldCov == nil {
		t.Fatal("cold compile should have CoverageResult")
	}

	cacheRunner, _, err := CompilePythonCached(source, bcData)
	if err != nil {
		t.Fatalf("cache hit: %v", err)
	}
	cacheCov := cacheRunner.GetCoverageResult()
	if cacheCov == nil {
		t.Fatal("cache hit should restore CoverageResult")
	}

	// Compare blind spots count.
	if len(coldCov.BlindSpots) != len(cacheCov.BlindSpots) {
		t.Errorf("BlindSpots count: cold=%d, cache=%d (should match)", len(coldCov.BlindSpots), len(cacheCov.BlindSpots))
	}
	// VM-CACHE-INTEGRITY-5 round 4: compare blind spot identity (not just count).
	for i, coldBS := range coldCov.BlindSpots {
		if i >= len(cacheCov.BlindSpots) {
			break
		}
		cacheBS := cacheCov.BlindSpots[i]
		if coldBS.Builtin != cacheBS.Builtin {
			t.Errorf("BlindSpot[%d].Builtin: cold=%q, cache=%q (should match)", i, coldBS.Builtin, cacheBS.Builtin)
		}
		if coldBS.Severity != cacheBS.Severity {
			t.Errorf("BlindSpot[%d].Severity: cold=%q, cache=%q (should match)", i, coldBS.Severity, cacheBS.Severity)
		}
	}
	// Compare Defense A violations count.
	if len(coldCov.DefenseAViolations) != len(cacheCov.DefenseAViolations) {
		t.Errorf("DefenseAViolations count: cold=%d, cache=%d (should match)", len(coldCov.DefenseAViolations), len(cacheCov.DefenseAViolations))
	}
	// VM-CACHE-INTEGRITY-5 round 4: compare Defense A violation identity.
	for i, coldDA := range coldCov.DefenseAViolations {
		if i >= len(cacheCov.DefenseAViolations) {
			break
		}
		cacheDA := cacheCov.DefenseAViolations[i]
		if coldDA.Rule != cacheDA.Rule {
			t.Errorf("DefenseAViolation[%d].Rule: cold=%q, cache=%q (should match)", i, coldDA.Rule, cacheDA.Rule)
		}
	}
}

// TestBytecode_NoLanguageField verifies that the Bytecode struct does NOT
// have a Language field (VM-CACHE-INTEGRITY-5 返工: dead field removed).
// Version ("mql4"/"mql5"/"python") serves as the language discriminator.
// VM-CACHE-INTEGRITY-5 round 4: uses reflection to check the field does NOT
// exist. If someone re-adds `Language string` to Bytecode, this test must RED.
func TestBytecode_NoLanguageField(t *testing.T) {
	// Use reflection to check that the Bytecode struct has no field named "Language".
	bcType := reflect.TypeOf(Bytecode{})
	_, exists := bcType.FieldByName("Language")
	if exists {
		t.Fatal("Bytecode struct should NOT have a Language field — Version is the language discriminator. " +
			"Re-adding Language would revive the dead field that VM-CACHE-INTEGRITY-5 removed.")
	}
	// Also verify Version is the language discriminator.
	bc := Bytecode{Version: "python"}
	if bc.Version != "python" {
		t.Errorf("Version=%q, want 'python'", bc.Version)
	}
}
