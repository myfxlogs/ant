package mql2go

import (
	"context"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// MaxTicks is the maximum number of instructions the VM will execute
// before aborting (prevents infinite loops).
const MaxTicks = 10_000_000

// MaxCallDepth limits recursive user function calls to prevent Go stack overflow.
const MaxCallDepth = 256

// MaxStackDepth limits the VM operand stack to prevent OOM from buggy EA.
const MaxStackDepth = 4096

// VM executes compiled MQL Bytecode against SDK interfaces.
type VM struct {
	bc              *Bytecode
	ctx             sdk.Context
	stack           []interp.Value
	globals         []interp.Value
	locals          []interp.Value // flat local variable space (frames are contiguous)
	pc              int32
	ticks           int64
	signal          *sdk.Signal
	currentPos      *sdk.Position      // current position being iterated (for Order* builtins)
	currentOrder    *sdk.PendingOrder  // current pending order being iterated (for Order* builtins)
	cachedPositions []sdk.Position     // cached list for OrderSelect(i, SELECT_BY_POS, MODE_TRADES)
	cachedOrders    []sdk.PendingOrder // cached pending orders for MODE_TRADES indexing
	cachedHistory   []sdk.Position     // cached list for OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)
	runCtx          context.Context    // context for cancellation checks
	callDepth       int                // current user function call depth
	fatalError      string             // set when a critical builtin is missing (ADR §5.4)

	// signalMode is true for live trading: Order* builtins build a pending
	// sdk.Signal instead of executing through the broker. The runner returns
	// this signal for server-side dispatch (paper / live OMS).
	signalMode bool

	// Pre-built lookup: EntryPC → FuncEntry (avoids O(n) scan per call)
	funcByEntryPC map[int32]FuncEntry

	// Runtime blind spots (function name → hit count)
	runtimeBlindSpots map[string]int
}

// NewVM creates a VM for the given Bytecode.
func NewVM(bc *Bytecode) *VM {
	vm := &VM{
		bc:                bc,
		runtimeBlindSpots: make(map[string]int),
		funcByEntryPC:     make(map[int32]FuncEntry),
	}
	for _, fn := range bc.Funcs {
		vm.funcByEntryPC[fn.EntryPC] = fn
	}
	return vm
}

// SetContext sets the SDK context for the VM (provides market data, broker, etc.)
func (vm *VM) SetContext(ctx sdk.Context) {
	vm.ctx = ctx
}

// Signal returns the last signal set during execution (nil if none).
func (vm *VM) Signal() *sdk.Signal {
	return vm.signal
}

// SetSignal sets the current signal.
func (vm *VM) SetSignal(s *sdk.Signal) {
	vm.signal = s
}

// SetSignalMode enables or disables signal-only mode for live execution.
// In signal mode, trade builtins produce a pending sdk.Signal instead of
// calling the broker. The caller (live runner) then dispatches it.
func (vm *VM) SetSignalMode(enabled bool) {
	vm.signalMode = enabled
}

// getSeriesHelper returns a bar series value by name and shift (int).
// Used by builtin series functions (Close(), Open(), etc.).
func (vm *VM) getSeriesHelper(name string, shift int) interp.Value {
	return vm.getSeries(name, int32(shift))
}

// RunOnInit executes the OnInit event handler.
func (vm *VM) RunOnInit(ctx context.Context) error {
	if vm.bc.OnInit < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnInit)
}

// RunOnBar executes the OnBar event handler.
func (vm *VM) RunOnBar(ctx context.Context) error {
	if vm.bc.OnBar < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnBar)
}

// RunOnTick executes the OnTick event handler.
func (vm *VM) RunOnTick(ctx context.Context) error {
	if vm.bc.OnTick < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnTick)
}

// RunOnTrade executes the OnTrade event handler.
func (vm *VM) RunOnTrade(ctx context.Context) error {
	if vm.bc.OnTrade < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnTrade)
}

// RunOnTimer executes the OnTimer event handler.
func (vm *VM) RunOnTimer(ctx context.Context) error {
	if vm.bc.OnTimer < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnTimer)
}

// RunOnDeinit executes the OnDeinit event handler.
func (vm *VM) RunOnDeinit(ctx context.Context) error {
	if vm.bc.OnDeinit < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnDeinit)
}

// RunOnTradeTransaction executes the OnTradeTransaction event handler (MQL5).
func (vm *VM) RunOnTradeTransaction(ctx context.Context) error {
	if vm.bc.OnTradeTransaction < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnTradeTransaction)
}

// RunOnBookEvent executes the OnBookEvent event handler (MQL5).
func (vm *VM) RunOnBookEvent(ctx context.Context) error {
	if vm.bc.OnBookEvent < 0 {
		return nil
	}
	return vm.runEvent(ctx, vm.bc.OnBookEvent)
}

// GetRuntimeBlindSpots returns the blind spots encountered during execution.
func (vm *VM) GetRuntimeBlindSpots() []interp.RuntimeBlindSpot {
	var out []interp.RuntimeBlindSpot
	for name, count := range vm.runtimeBlindSpots {
		out = append(out, interp.RuntimeBlindSpot{
			Builtin:  name,
			Count:    count,
			Severity: interp.SeverityForBuiltin(name),
		})
	}
	return out
}

// runEvent executes the VM starting at the given entry point.
func (vm *VM) runEvent(ctx context.Context, entryPC int32) error {
	// Reset state for this event invocation
	vm.stack = vm.stack[:0]
	vm.currentPos = nil
	vm.currentOrder = nil
	vm.cachedPositions = nil
	vm.cachedOrders = nil
	vm.cachedHistory = nil
	vm.callDepth = 0
	vm.signal = nil
	vm.pc = entryPC

	// Allocate local variable space for this event handler.
	// Event handlers don't go through OP_CALL_USER, so we must set up locals here.
	if n, ok := vm.bc.EventLocals[entryPC]; ok && n > 0 {
		vm.locals = make([]interp.Value, n)
	} else {
		vm.locals = nil
	}
	vm.ticks = 0
	vm.runCtx = ctx

	// Initialize globals if not yet done
	if vm.globals == nil {
		vm.initGlobals()
	}

	// Run the VM loop
	return vm.runLoop(ctx)
}

// initGlobals initializes global variables from the Bytecode's variable slots.
func (vm *VM) initGlobals() {
	vm.globals = make([]interp.Value, len(vm.bc.GlobalSlots))
	// Initialize array globals with proper size
	for _, decl := range vm.bc.GlobalDecls {
		if decl.IsArray && decl.ArraySize > 0 {
			slot, ok := vm.bc.GlobalSlots[decl.Name]
			if !ok {
				continue
			}
			arr := make([]interp.Value, decl.ArraySize)
			zeroVal := zeroValueForType(decl.Type)
			for i := range arr {
				arr[i] = zeroVal
			}
			vm.globals[slot] = interp.Value{Kind: interp.ValArray, Array: arr}
		}
	}
}

// zeroValueForType returns a zero Value for the given MQL type.
func zeroValueForType(typeName string) interp.Value {
	switch typeName {
	case "int", "long", "datetime", "bool":
		return interp.IntVal(0)
	case "double", "float":
		return interp.DecimalVal(decimal.Zero)
	case "string":
		return interp.StringVal("")
	default:
		return interp.DecimalVal(decimal.Zero)
	}
}
