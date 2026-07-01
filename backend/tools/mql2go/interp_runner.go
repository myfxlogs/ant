package mql2go

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go/interp"
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
	return safeRun(func() error { return r.vm.RunOnInit(context.Background()) })
}

// OnBar implements sdk.Strategy.
// The VM trades directly through ctx.Broker() (MQL semantics),
// so the returned signal is always nil — the engine must not double-dispatch.
func (r *VMRunner) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnBar(context.Background()) }); err != nil {
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
		if err := safeRun(func() error { return r.vm.RunOnTick(context.Background()) }); err != nil {
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

	if err := safeRun(func() error { return r.vm.RunOnTrade(context.Background()) }); err != nil {
		return nil, fmt.Errorf("VM OnTrade: %w", err)
	}

	return nil, nil
}

// OnDeinit implements sdk.Strategy.
func (r *VMRunner) OnDeinit(ctx sdk.Context, reason string) error {
	r.vm.SetContext(ctx)
	return safeRun(func() error { return r.vm.RunOnDeinit(context.Background()) })
}

// OnTimer implements sdk.TimerStrategy (optional).
func (r *VMRunner) OnTimer(ctx sdk.Context) (*sdk.Signal, error) {
	r.vm.SetContext(ctx)

	if err := safeRun(func() error { return r.vm.RunOnTimer(context.Background()) }); err != nil {
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

// Bytecode returns the compiled bytecode. Callers can use this for
// parameter extraction without recompiling.
func (r *VMRunner) Bytecode() *Bytecode {
	return r.bc
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

		case "double", "float":
			var def decimal.Decimal
			if p.Default != nil {
				if d, err := decimal.NewFromString(interp.EvalExprLiteral(p.Default)); err == nil {
					def = d
				}
			}
			val = interp.DecimalVal(ctx.ParamDecimal(p.Name, def))

		case "string":
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
