package mql2go

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// hashSource is defined in failure_signature.go and reused for cache integrity
// (VM-CACHE-INTEGRITY-1). It computes SHA256 of the trimmed source.

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

// marshalHook is used only by tests to inject marshal failures into
// CompileMQLCached/CompilePythonCached (VM-CACHE-INTEGRITY-1/2 S5 rework).
// Production code leaves this nil so MarshalBytecode is called directly.
var marshalHook func(*Bytecode) ([]byte, error)

// marshalBytecode resolves the marshal function to use: the test-injected
// marshalHook when set, otherwise the real MarshalBytecode. This keeps the
// production path unchanged while allowing adversarial tests to force a
// marshal error and verify it is propagated (not swallowed).
func marshalBytecode(bc *Bytecode) ([]byte, error) {
	if marshalHook != nil {
		return marshalHook(bc)
	}
	return MarshalBytecode(bc)
}

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
// VM-CACHE-INTEGRITY-1: on cache hit, verifies SourceHash matches current
// source; mismatch forces recompile to prevent stale bytecode execution.
// VM-CACHE-INTEGRITY-5: also verifies Version is MQL (mql4/mql5) to prevent
// Python bytecode from being used for MQL source.
func CompileMQLCached(source string, cachedBytecode []byte) (runner *VMRunner, bytecode []byte, err error) {
	if len(cachedBytecode) > 0 {
		r, e := CompileMQLFromBytecode(cachedBytecode)
		if e == nil && r.Bytecode().SourceHash == hashSource(source) && isMQLVersion(r.Bytecode().Version) {
			return r, cachedBytecode, nil
		}
		// Cache corrupted, source changed, or language mismatch — fall through
	}
	r, err := CompileMQL(source)
	if err != nil {
		return nil, nil, err
	}
	bc := r.Bytecode()
	data, mErr := marshalBytecode(bc)
	if mErr != nil {
		return nil, nil, fmt.Errorf("marshal freshly compiled bytecode: %w", mErr)
	}
	return r, data, nil
}

// CompilePythonCached mirrors CompileMQLCached for the Python subset path.
// It verifies the cached bytecode's SourceHash against the current source,
// falls back to full compilation on mismatch/corruption, and returns the
// serialized bytecode for caller-side caching.
// VM-CACHE-INTEGRITY-2: SourceHash verification for Python cache path.
// VM-CACHE-INTEGRITY-5: (1) restores CoverageResult on cache hit by
// recompiling from source (bytecode cache omits coverage); coverage restore
// failure returns error (no silent degradation). (2) verifies Version is
// "python" to prevent MQL bytecode from being used for Python source.
func CompilePythonCached(source string, cachedBytecode []byte) (runner *VMRunner, bytecode []byte, err error) {
	if len(cachedBytecode) > 0 {
		r, e := CompileMQLFromBytecode(cachedBytecode)
		if e == nil && r.Bytecode().SourceHash == hashSource(source) && r.Bytecode().Version == "python" {
			// VM-CACHE-INTEGRITY-5: restore CoverageResult by recompiling
			// from source. Bytecode cache omits coverage analysis.
			var cov *CoverageResult
			var covErr error
			if coverageRestoreHook != nil {
				cov, covErr = coverageRestoreHook(source)
			} else {
				_, cov, covErr = CompilePythonWithCoverage(source)
			}
			if covErr != nil {
				return nil, nil, fmt.Errorf("restore coverage on cache hit: %w", covErr)
			}
			if cov == nil {
				return nil, nil, fmt.Errorf("restore coverage on cache hit: nil coverage result")
			}
			r.InjectCoverageResult(cov)
			return r, cachedBytecode, nil
		}
		// Cache corrupted, source changed, or language mismatch — fall through
	}
	r, err := CompilePython(source)
	if err != nil {
		return nil, nil, err
	}
	bc := r.Bytecode()
	data, mErr := marshalBytecode(bc)
	if mErr != nil {
		return nil, nil, fmt.Errorf("marshal freshly compiled Python bytecode: %w", mErr)
	}
	return r, data, nil
}

// isMQLVersion returns true if the bytecode version indicates an MQL strategy
// (mql4 or mql5). VM-CACHE-INTEGRITY-5: prevents Python bytecode from being
// accepted for MQL source in CompileMQLCached.
func isMQLVersion(version string) bool {
	return version == "mql4" || version == "mql5"
}

// coverageRestoreHook is a test-only hook that overrides the coverage
// restore path in CompilePythonCached. When non-nil, it replaces the
// CompilePythonWithCoverage call. Production code leaves this nil.
var coverageRestoreHook func(src string) (*CoverageResult, error)

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
	return NewVMRunner(bc), nil
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
	bc.SourceHash = hashSource(source)
	coverage := AnalyzeCoverage(ir, bc)
	runner := NewVMRunner(bc)
	runner.defenseAViolations = coverage.DefenseAViolations
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
	return NewVMRunner(bc), nil
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
	bc.SourceHash = hashSource(source)
	coverage := AnalyzeCoverage(ir, bc)
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
func (r *VMRunner) OnInit(ctx sdk.Context) error {
	r.vm.SetContext(ctx)

	// Inject extern/input parameters from SDK context into VM globals
	r.injectParams(ctx)

	// Run OnInit bytecode
	return safeRun(func() error { return r.vm.RunOnInit(ctx.GoContext()) })
}

// OnBar implements sdk.Strategy.
// In live (signalMode=true) the VM builds a pending signal from Order* builtins;
// the runner returns it for server-side dispatch. In backtest (signalMode=false)
// the VM executes through ctx.Broker() and returns nil so the engine does not
// double-dispatch.
func (r *VMRunner) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnBar(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnBar: %w", err)
	}

	return r.vm.Signal(), nil
}

// OnTick implements sdk.TickStrategy (optional).
// Only executes OnTick bytecode — does NOT fallback to OnBar.
// The engine checks for TickStrategy and calls OnTick; if the EA
// only has OnBar, the engine's else-branch calls OnBar directly.
func (r *VMRunner) OnTick(ctx sdk.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if r.vm.bc.OnTick >= 0 {
		if err := safeRun(func() error { return r.vm.RunOnTick(ctx.GoContext()) }); err != nil {
			return nil, fmt.Errorf("VM OnTick: %w", err)
		}
	}

	return r.vm.Signal(), nil
}

// HasOnTick implements sdk.TickCapable — returns true if the EA has OnTick bytecode.
func (r *VMRunner) HasOnTick() bool {
	return r.vm.bc.OnTick >= 0
}

// OnTrade implements sdk.TradeStrategy (optional).
func (r *VMRunner) OnTrade(ctx sdk.Context, event sdk.TradeEvent) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnTrade(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnTrade: %w", err)
	}

	return r.vm.Signal(), nil
}

// OnTradeTransaction implements sdk.TradeTransactionStrategy (optional, MQL5).
func (r *VMRunner) OnTradeTransaction(ctx sdk.Context) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnTradeTransaction(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnTradeTransaction: %w", err)
	}

	return r.vm.Signal(), nil
}

// OnBookEvent implements sdk.BookEventStrategy (optional, MQL5).
func (r *VMRunner) OnBookEvent(ctx sdk.Context) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnBookEvent(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnBookEvent: %w", err)
	}

	return r.vm.Signal(), nil
}

// HasOnTradeTransaction returns true if the EA has OnTradeTransaction bytecode.
func (r *VMRunner) HasOnTradeTransaction() bool {
	return r.vm.bc.OnTradeTransaction >= 0
}

// SetSignalMode enables signal-only mode for live execution.
func (r *VMRunner) SetSignalMode(enabled bool) {
	r.vm.SetSignalMode(enabled)
}

// HasOnBookEvent returns true if the EA has OnBookEvent bytecode.
func (r *VMRunner) HasOnBookEvent() bool {
	return r.vm.bc.OnBookEvent >= 0
}

// OnDeinit implements sdk.Strategy.
func (r *VMRunner) OnDeinit(ctx sdk.Context, reason string) error {
	r.vm.SetContext(ctx)
	return safeRun(func() error { return r.vm.RunOnDeinit(ctx.GoContext()) })
}

// OnTimer implements sdk.TimerStrategy (optional).
func (r *VMRunner) OnTimer(ctx sdk.Context) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnTimer(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnTimer: %w", err)
	}

	return r.vm.Signal(), nil
}

// GetRuntimeBlindSpots returns blind spots encountered during VM execution.
func (r *VMRunner) GetRuntimeBlindSpots() []interp.RuntimeBlindSpot {
	return r.vm.GetRuntimeBlindSpots()
}

// GetCoverage returns the compile-time coverage report.
func (r *VMRunner) GetCoverage() *CoverageReport {
	return r.bc.Coverage
}

// GetCoverageResult returns the full coverage analysis result with
// pre-classified blind spots (severity-tagged). Used by buildBacktestResponse
// to check for fatal blind spots (MQL-HONESTY-3).
func (r *VMRunner) GetCoverageResult() *CoverageResult {
	return r.coverageResult
}

// InjectCoverage sets the compile-time coverage report on the runner.
// Used when bytecode was loaded from cache (which omits coverage) and
// coverage is recovered by recompiling from source.
func (r *VMRunner) InjectCoverage(cov *CoverageReport) {
	r.bc.Coverage = cov
}

// InjectCoverageResult sets the full coverage analysis result on the runner.
// VM-CACHE-INTEGRITY-5: used by CompilePythonCached to restore CoverageResult
// on cache hit (bytecode cache omits coverage, so it must be recovered by
// recompiling from source).
func (r *VMRunner) InjectCoverageResult(cov *CoverageResult) {
	r.coverageResult = cov
}

// GetDefenseAViolations returns post-parse validation failures (ADR-0028 §4.1).
func (r *VMRunner) GetDefenseAViolations() []interp.DefenseAViolation {
	return r.defenseAViolations
}

// InjectDefenseAViolations sets Defense A violations on the runner.
// Used when bytecode was loaded from cache and violations are recovered by recompiling.
func (r *VMRunner) InjectDefenseAViolations(violations []interp.DefenseAViolation) {
	r.defenseAViolations = violations
}

// Bytecode returns the compiled bytecode. Callers can use this for
// parameter extraction without recompiling.
func (r *VMRunner) Bytecode() *Bytecode {
	return r.bc
}

// GetGlobal returns the current value of a global variable by name.
// Used by D3 differential tests to extract indicator values computed by MQL EAs.
func (r *VMRunner) GetGlobal(name string) (interp.Value, bool) {
	slot, ok := r.bc.GlobalSlots[name]
	if !ok || int(slot) >= len(r.vm.globals) {
		return interp.Value{}, false
	}
	return r.vm.globals[slot], true
}

// LastIndicators returns the indicator values captured during the last event execution.
// The map is populated by recordDiag calls in indicator builtins (shift==0 only).
// Returns nil if no indicators were recorded. The caller must not modify the returned map.
func (r *VMRunner) LastIndicators() map[string]decimal.Decimal {
	return r.vm.lastIndicators
}

// OrdersTotal returns the last OrdersTotal value seen by the VM's builtinOrdersTotal.
// This is the VM internal cached value (R3: not from the event loop), which reflects
// the actual position+order count at the time of the last OnBar/OnTick execution.
func (r *VMRunner) OrdersTotal() int {
	if r.vm.cachedPositions == nil && r.vm.cachedOrders == nil {
		return 0
	}
	return len(r.vm.cachedPositions) + len(r.vm.cachedOrders)
}

// injectParams reads extern/input parameters from the SDK context and
// writes them into the VM's global variable slots.
func (r *VMRunner) injectParams(ctx sdk.Context) {
	if r.vm.globals == nil {
		r.vm.initGlobals()
	}

	for _, p := range r.bc.Params {
		slot, ok := r.bc.GlobalSlots[p.Name]
		if !ok {
			continue
		}

		var val interp.Value
		switch p.Type {
		case "int", "long":
			var def int
			if p.Default != nil {
				def, _ = strconv.Atoi(interp.EvalExprLiteral(p.Default))
			}
			val = interp.IntVal(int32(ctx.ParamInt(p.Name, def)))

		case "double", nodeFloat:
			var def decimal.Decimal
			if p.Default != nil {
				if d, err := decimal.NewFromString(interp.EvalExprLiteral(p.Default)); err == nil {
					def = d
				}
			}
			val = interp.DecimalVal(ctx.ParamDecimal(p.Name, def))

		case nodeString:
			var def string
			if p.Default != nil {
				def = interp.EvalExprLiteral(p.Default)
			}
			val = interp.StringVal(ctx.ParamString(p.Name, def))

		case "bool":
			var def bool
			if p.Default != nil {
				def = interp.EvalExprLiteral(p.Default) == "true"
			}
			val = interp.BoolVal(ctx.ParamBool(p.Name, def))

		default:
			// Enum types — treat as int
			var def int
			if p.Default != nil {
				def, _ = strconv.Atoi(interp.EvalExprLiteral(p.Default))
			}
			val = interp.IntVal(int32(ctx.ParamInt(p.Name, def)))
		}

		if int(slot) < len(r.vm.globals) {
			r.vm.globals[slot] = val
		}
	}
}
