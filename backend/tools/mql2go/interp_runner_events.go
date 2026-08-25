package mql2go

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

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

// InjectCoverageResult restores the full static analysis result when bytecode
// was loaded from cache and coverage had to be recomputed from source.
func (r *VMRunner) InjectCoverageResult(cov *CoverageResult) {
	r.coverageResult = cov
	if cov != nil {
		r.defenseAViolations = cov.DefenseAViolations
	}
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
