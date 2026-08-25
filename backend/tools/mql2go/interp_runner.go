package mql2go

import (
	"fmt"
	"sync"

	"alphaforge/tools/mql2go/interp"
)

// compilePythonWithCoverageFn is the coverage compiler function used by
// CompilePythonCached. It defaults to CompilePythonWithCoverage but can be
// overridden in tests via setCompilePythonWithCoverageFn to simulate failures.
// VM-CACHE-INTEGRITY-5 round 4: test-injectable coverage failure for
// adversarial proof. Mutex-protected to avoid race-prone global state.
var (
	compilePythonWithCoverageMu sync.Mutex
	compilePythonWithCoverageFn = CompilePythonWithCoverage
)

// callCompilePythonWithCoverage calls the current coverage compiler function.
func callCompilePythonWithCoverage(source string) (*VMRunner, *CoverageResult, error) {
	compilePythonWithCoverageMu.Lock()
	fn := compilePythonWithCoverageFn
	compilePythonWithCoverageMu.Unlock()
	return fn(source)
}

// VMRunner wraps a Bytecode VM to implement sdk.Strategy.
// This is the in-process execution entrypoint that replaces the WASM harness.
//
// Usage:
//
//	ir, _ := mql2go.CompileToIR(mqlSource)
//	bc, _ := mql2go.CompileAST(ir)
//	runner := mql2go.NewVMRunner(bc)
//	engine := backtest.NewEngine(runner, ...)
//	result := engine.Run(ctx)
type VMRunner struct {
	vm                 *VM
	bc                 *Bytecode
	defenseAViolations []interp.DefenseAViolation
	coverageResult     *CoverageResult
}

// NewVMRunner creates a sdk.Strategy runner from compiled Bytecode.
func NewVMRunner(bc *Bytecode) *VMRunner {
	return &VMRunner{
		vm: NewVM(bc),
		bc: bc,
	}
}

// MaxSourceSize is the maximum accepted MQL source size (500 KB).
// Strategies exceeding this are rejected before parsing to prevent
// resource exhaustion. ADR-0023 §5.4.
const MaxSourceSize = 500_000

// CompileMQLFromBytecode creates a VMRunner from cached bytecode data.
// Returns error if the bytecode is invalid or corrupted.
func CompileMQLFromBytecode(data []byte) (*VMRunner, error) {
	bc, err := UnmarshalBytecode(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal cached bytecode: %w", err)
	}
	return NewVMRunner(bc), nil
}

// CompileMQLCached tries to load cached bytecode first; on failure or nil cache,
// falls back to full compilation from source. Returns the runner and the
// serialized bytecode (for caching by the caller).
func CompileMQLCached(source string, cachedBytecode []byte) (runner *VMRunner, bytecode []byte, err error) {
	if len(cachedBytecode) > 0 {
		r, e := CompileMQLFromBytecode(cachedBytecode)
		// VM-CACHE-INTEGRITY-5: verify SourceHash AND language (Version must
		// be "mql4" or "mql5", not "python") before accepting cached bytecode.
		if e == nil && r.Bytecode().SourceHash == hashSource(source) && isMQLVersion(r.Bytecode().Version) {
			return r, cachedBytecode, nil
		}
		// Cache corrupted, compiled from different source, or wrong language — recompile.
	}
	r, err := CompileMQL(source)
	if err != nil {
		return nil, nil, err
	}
	bc := r.Bytecode()
	data, mErr := MarshalBytecode(bc)
	if mErr != nil {
		return nil, nil, fmt.Errorf("marshal freshly compiled bytecode: %w", mErr)
	}
	return r, data, nil
}

// isMQLVersion returns true if the version indicates MQL (not Python).
// VM-CACHE-INTEGRITY-5: prevents Python bytecode from being used as MQL cache.
func isMQLVersion(v string) bool {
	return v == "mql4" || v == "mql5"
}

// CompilePythonCached mirrors CompileMQLCached for the Python subset path.
// It verifies the cached bytecode's SourceHash AND language (Version == "python")
// against the current source, falls back to full compilation on mismatch/corruption,
// and returns the serialized bytecode for caller-side caching.
// VM-CACHE-INTEGRITY-2: SourceHash verification.
// VM-CACHE-INTEGRITY-5: language (Version) verification + CoverageResult restore.
func CompilePythonCached(source string, cachedBytecode []byte) (runner *VMRunner, bytecode []byte, err error) {
	if len(cachedBytecode) > 0 {
		r, e := CompileMQLFromBytecode(cachedBytecode)
		// VM-CACHE-INTEGRITY-5: verify SourceHash AND language (Version == "python").
		if e == nil && r.Bytecode().SourceHash == hashSource(source) && r.Bytecode().Version == "python" {
			// VM-CACHE-INTEGRITY-5: restore CoverageResult on cache hit.
			// Bytecode cache omits coverage data; recompile from source
			// to restore severity-aware blind spots and Defense A data.
			// If recompilation fails, return error — do NOT return a cache
			// runner without coverage (would be a silent degradation).
			if r.GetCoverageResult() == nil && source != "" {
				covRunner, cov, covErr := callCompilePythonWithCoverage(source)
				if covErr != nil {
					return nil, nil, fmt.Errorf("restore coverage from source: %w", covErr)
				}
				if cov == nil {
					return nil, nil, fmt.Errorf("restore coverage from source: CoverageResult is nil")
				}
				r.InjectCoverage(covRunner.GetCoverage())
				r.InjectCoverageResult(cov)
			}
			return r, cachedBytecode, nil
		}
		// Cache corrupted, compiled from different source, or wrong language — recompile.
	}
	r, err := CompilePython(source)
	if err != nil {
		return nil, nil, err
	}
	bc := r.Bytecode()
	data, mErr := MarshalBytecode(bc)
	if mErr != nil {
		return nil, nil, fmt.Errorf("marshal freshly compiled Python bytecode: %w", mErr)
	}
	return r, data, nil
}

// CompilePython is a convenience function that compiles Python subset source to a VMRunner.
// Pipeline: Python source → CST → IR → Bytecode → VMRunner
// Safety: CompilePythonToIR enforces MaxSourceSize + subset validation + panic recovery.
func CompilePython(source string) (runner *VMRunner, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("compile Python panic: %v", r)
			runner = nil
		}
	}()
	ir, err := CompilePythonToIR(source)
	if err != nil {
		return nil, fmt.Errorf("compile Python to IR: %w", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		return nil, fmt.Errorf("compile IR to bytecode: %w", err)
	}
	bc.SourceHash = hashSource(source)
	runner = NewVMRunner(bc)
	runner.coverageResult = AnalyzeCoverage(ir, bc)
	runner.defenseAViolations = runner.coverageResult.DefenseAViolations
	return runner, nil
}

// CompilePythonWithCoverage compiles Python subset source and returns both the runner
// and the coverage analysis result. Mirrors CompileMQLWithCoverage.
func CompilePythonWithCoverage(source string) (r *VMRunner, cov *CoverageResult, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("compile Python panic: %v", p)
			r = nil
			cov = nil
		}
	}()
	ir, err := CompilePythonToIR(source)
	if err != nil {
		return nil, nil, fmt.Errorf("compile Python to IR: %w", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		return nil, nil, fmt.Errorf("compile IR to bytecode: %w", err)
	}
	coverage := AnalyzeCoverage(ir, bc)
	bc.SourceHash = hashSource(source)
	runner := NewVMRunner(bc)
	runner.defenseAViolations = coverage.DefenseAViolations
	runner.coverageResult = coverage
	return runner, coverage, nil
}

// CompileMQL is a convenience function that compiles MQL source to a VMRunner.
// This is the single entrypoint for the in-process execution path:
// MQL source → CST → AST (IR) → Bytecode → VMRunner
//
// Safety: CompileToIR enforces MaxSourceSize + panic recovery (ADR-0023 §5.4).
// CompileAST panic recovery is handled here.
func CompileMQL(source string) (runner *VMRunner, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("compile MQL panic: %v", r)
			runner = nil
		}
	}()
	ir, err := CompileToIR(source)
	if err != nil {
		return nil, fmt.Errorf("compile MQL to IR: %w", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		return nil, fmt.Errorf("compile IR to bytecode: %w", err)
	}
	bc.SourceHash = hashSource(source)
	runner = NewVMRunner(bc)
	runner.coverageResult = AnalyzeCoverage(ir, bc)
	runner.defenseAViolations = runner.coverageResult.DefenseAViolations
	return runner, nil
}

// CompileMQLWithCoverage compiles MQL source and returns both the runner
// and the coverage analysis result.
//
// Safety: same as CompileMQL — CompileToIR handles size limit + panic recovery.
func CompileMQLWithCoverage(source string) (r *VMRunner, cov *CoverageResult, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("compile MQL panic: %v", p)
			r = nil
			cov = nil
		}
	}()
	ir, err := CompileToIR(source)
	if err != nil {
		return nil, nil, fmt.Errorf("compile MQL to IR: %w", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		return nil, nil, fmt.Errorf("compile IR to bytecode: %w", err)
	}
	coverage := AnalyzeCoverage(ir, bc)
	bc.SourceHash = hashSource(source)
	runner := NewVMRunner(bc)
	runner.defenseAViolations = coverage.DefenseAViolations
	runner.coverageResult = coverage
	return runner, coverage, nil
}

// DetectLookaheadFromSource compiles source to IR and runs lookahead detection.
// Returns nil if compilation fails (caller should handle compile errors separately).
// This is a lightweight alternative to CompileMQLWithCoverage when only lookahead
// detection is needed (e.g. gate pipeline re-evaluation).
func DetectLookaheadFromSource(source string) []interp.LookaheadViolation {
	defer func() {
		// Suppress panics — lookahead detection is best-effort.
		// If compilation fails, there are no violations to report.
		_ = recover()
	}()
	ir, err := CompileToIR(source)
	if err != nil {
		return nil
	}
	return interp.DetectLookahead(ir)
}

// safeRun executes a VM function with panic recovery.
// Panics from cgo (tree-sitter) or deep recursion are converted to errors
// instead of crashing the process. ADR-0023 §5.4.
func safeRun(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("VM panic: %v", r)
		}
	}()
	return fn()
}

// OnInit implements sdk.Strategy.
