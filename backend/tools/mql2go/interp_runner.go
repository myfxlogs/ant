package mql2go

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

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
	vm *VM
	bc *Bytecode
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
		if e == nil {
			return r, cachedBytecode, nil
		}
		// Cache corrupted — fall through to recompile
	}
	r, err := CompileMQL(source)
	if err != nil {
		return nil, nil, err
	}
	bc := r.Bytecode()
	data, mErr := MarshalBytecode(bc)
	if mErr != nil {
		return r, nil, nil
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
	coverage := AnalyzeCoverage(ir, bc)
	return NewVMRunner(bc), coverage, nil
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
	coverage := AnalyzeCoverage(ir, bc)
	return NewVMRunner(bc), coverage, nil
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
// The VM trades directly through ctx.Broker() (MQL semantics),
// so the returned signal is always nil — the engine must not double-dispatch.
func (r *VMRunner) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnBar(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnBar: %w", err)
	}

	return nil, nil
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

	return nil, nil
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

	return nil, nil
}

// OnTradeTransaction implements sdk.TradeTransactionStrategy (optional, MQL5).
func (r *VMRunner) OnTradeTransaction(ctx sdk.Context) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnTradeTransaction(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnTradeTransaction: %w", err)
	}

	return nil, nil
}

// OnBookEvent implements sdk.BookEventStrategy (optional, MQL5).
func (r *VMRunner) OnBookEvent(ctx sdk.Context) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnBookEvent(ctx.GoContext()) }); err != nil {
		return nil, fmt.Errorf("VM OnBookEvent: %w", err)
	}

	return nil, nil
}

// HasOnTradeTransaction returns true if the EA has OnTradeTransaction bytecode.
func (r *VMRunner) HasOnTradeTransaction() bool {
	return r.vm.bc.OnTradeTransaction >= 0
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

	return nil, nil
}

// GetRuntimeBlindSpots returns blind spots encountered during VM execution.
func (r *VMRunner) GetRuntimeBlindSpots() []interp.RuntimeBlindSpot {
	return r.vm.GetRuntimeBlindSpots()
}

// GetCoverage returns the compile-time coverage report.
func (r *VMRunner) GetCoverage() *CoverageReport {
	return r.bc.Coverage
}

// InjectCoverage sets the compile-time coverage report on the runner.
// Used when bytecode was loaded from cache (which omits coverage) and
// coverage is recovered by recompiling from source.
func (r *VMRunner) InjectCoverage(cov *CoverageReport) {
	r.bc.Coverage = cov
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
