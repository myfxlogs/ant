package interp

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
	"anttrader/strategy/sdk"
)

// Sentinel errors for control flow.
var (
	errBreak    = errors.New("break")
	errContinue = errors.New("continue")
	errReturn   = errors.New("return")
)

// Interpreter executes a compiled IR against SDK interfaces.
// Pure Go — no tree-sitter dependency — safe for WASM.
type Interpreter struct {
	ir        *IR
	ctx       sdk.Context
	globals   map[string]Value
	scopes    []map[string]Value // block scope stack
	series    *SeriesAccessor
	orderPool *MQL4OrderPool
	posPool   *MQL5PositionPool
	signal    *sdk.Signal
	retVal    Value // return value from user function

	// runtimeBlindSpots tracks unimplemented functions encountered during execution.
	// Map key = function name, value = hit count.
	runtimeBlindSpots map[string]int
}

// RuntimeBlindSpot is a single entry returned by GetRuntimeBlindSpots.
type RuntimeBlindSpot struct {
	Builtin  string
	Count    int
	Severity string
}

// errFatalBlindSpot is returned when a fatal unimplemented function is called,
// aborting the current OnTick/OnBar execution.
var errFatalBlindSpot = errors.New("fatal blind spot: unimplemented function critical to EA logic")

// NewInterpreter creates an Interpreter from a compiled IR.
func NewInterpreter(ir *IR) *Interpreter {
	it := &Interpreter{
		ir:                ir,
		globals:           make(map[string]Value),
		scopes:            []map[string]Value{make(map[string]Value)}, // root scope
		runtimeBlindSpots: make(map[string]int),
	}
	it.series = &SeriesAccessor{}
	it.orderPool = &MQL4OrderPool{}
	it.posPool = &MQL5PositionPool{}

	// Initialize globals
	for _, g := range ir.Globals {
		if g.InitVal != nil {
			it.globals[g.Name] = it.evalExpr(g.InitVal)
		} else if isClassType(g.Type) {
			it.globals[g.Name] = Value{
				Kind: ValClass,
				Class: &ClassInstance{
					Name:   g.Type,
					Fields: make(map[string]Value),
				},
			}
		} else {
			it.globals[g.Name] = zeroValue(g.Type)
		}
	}

	return it
}

// GetGlobal returns a global variable value by name (for test/debug inspection).
func (it *Interpreter) GetGlobal(name string) Value {
	return it.globals[name]
}

// GetRuntimeBlindSpots returns unimplemented functions encountered during execution,
// sorted by severity (fatal first) then by hit count (descending).
func (it *Interpreter) GetRuntimeBlindSpots() []RuntimeBlindSpot {
	result := make([]RuntimeBlindSpot, 0, len(it.runtimeBlindSpots))
	for name, count := range it.runtimeBlindSpots {
		severity := classifyRuntimeSeverity(name)
		result = append(result, RuntimeBlindSpot{
			Builtin:  name,
			Count:    count,
			Severity: severity,
		})
	}
	// Sort: fatal first, then by count descending
	sort.Slice(result, func(i, j int) bool {
		if result[i].Severity != result[j].Severity {
			return result[i].Severity == "致命"
		}
		return result[i].Count > result[j].Count
	})
	return result
}

// ResetRuntimeBlindSpots clears the runtime blind spot log (e.g. between backtest runs).
func (it *Interpreter) ResetRuntimeBlindSpots() {
	it.runtimeBlindSpots = make(map[string]int)
}

// ── Variable management ─────────────────────────────────────────────

// curScope returns the innermost scope.
func (it *Interpreter) curScope() map[string]Value {
	return it.scopes[len(it.scopes)-1]
}

func (it *Interpreter) getVar(name string) Value {
	// Search scopes from innermost to outermost
	for i := len(it.scopes) - 1; i >= 0; i-- {
		if v, ok := it.scopes[i][name]; ok {
			return v
		}
	}
	if v, ok := it.globals[name]; ok {
		return v
	}
	// MQL predefined variables (Digits, Ask, Bid, Point, Symbol, etc.)
	// are compiled as ExprVar but need to be resolved via market data dispatch.
	if it.ctx != nil {
		if v, ok := it.callMarketData(name, nil); ok {
			return v
		}
	}
	return NoneVal()
}

func (it *Interpreter) setVar(name string, val Value) {
	// If exists in any scope, update there
	for i := len(it.scopes) - 1; i >= 0; i-- {
		if _, ok := it.scopes[i][name]; ok {
			it.scopes[i][name] = val
			return
		}
	}
	// If exists in globals, update there
	if _, ok := it.globals[name]; ok {
		it.globals[name] = val
		return
	}
	// In MQL4, assigning to an undeclared variable at function scope
	// means it's a global variable (MQL4 has no block-scoped locals
	// for undeclared names). Store in globals to persist across calls.
	it.globals[name] = val
}

// ── Statement execution ─────────────────────────────────────────────

func (it *Interpreter) execStmt(stmt *Statement) error {
	switch stmt.Kind {
	case StmtExpr:
		it.evalExpr(stmt.Expr)

	case StmtIf:
		if it.evalExpr(stmt.Cond).IsTrue() {
			return it.execBlock(stmt.Body)
		} else if len(stmt.ElseBody) > 0 {
			return it.execBlock(stmt.ElseBody)
		}

	case StmtFor:
		// MQL uses C-style scoping: for-loop init variables are accessible
		// in the enclosing block, so no pushScope/popScope here.
		if stmt.Init != nil {
			if err := it.execStmt(stmt.Init); err != nil {
				return err
			}
		}
		for {
			if stmt.Cond != nil && !it.evalExpr(stmt.Cond).IsTrue() {
				break
			}
			if err := it.execBlock(stmt.Body); err != nil {
				if errors.Is(err, errBreak) {
					break
				}
				if errors.Is(err, errContinue) {
					// fall through to update
				} else {
					return err
				}
			}
			if it.signal != nil {
				return nil
			}
			if stmt.Update != nil {
				if err := it.execStmt(stmt.Update); err != nil {
					return err
				}
			}
		}

	case StmtWhile:
		for it.evalExpr(stmt.Cond).IsTrue() {
			if err := it.execBlock(stmt.Body); err != nil {
				if errors.Is(err, errBreak) {
					break
				}
				if errors.Is(err, errContinue) {
					// fall through to signal check
				} else {
					return err
				}
			}
			if it.signal != nil {
				return nil
			}
		}

	case StmtDoWhile:
		for {
			if err := it.execBlock(stmt.Body); err != nil {
				if errors.Is(err, errBreak) {
					break
				}
				if errors.Is(err, errContinue) {
					// fall through to condition check
				} else {
					return err
				}
			}
			if it.signal != nil {
				return nil
			}
			if !it.evalExpr(stmt.Cond).IsTrue() {
				break
			}
		}

	case StmtReturn:
		if stmt.Expr != nil {
			it.retVal = it.evalExpr(stmt.Expr)
		} else {
			it.retVal = NoneVal()
		}
		return errReturn

	case StmtBreak:
		return errBreak

	case StmtContinue:
		return errContinue

	case StmtBlock:
		return it.execBlock(stmt.Body)

	case StmtSwitch:
		switchVal := it.evalExpr(stmt.Expr)
		var defaultCase *SwitchCase
		for i := range stmt.Cases {
			c := &stmt.Cases[i]
			if c.Expr == nil {
				defaultCase = c
				continue
			}
			caseVal := it.evalExpr(c.Expr)
			if switchVal.Equal(caseVal) {
				err := it.execBlock(c.Body)
				if errors.Is(err, errBreak) {
					return nil
				}
				return err
			}
		}
		if defaultCase != nil {
			err := it.execBlock(defaultCase.Body)
			if errors.Is(err, errBreak) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (it *Interpreter) execBlock(stmts []Statement) (err error) {
	// Recover from fatal blind spot panics (triggered by recordBlindSpot)
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok && errors.Is(e, errFatalBlindSpot) {
				err = e
				return
			}
			panic(r) // re-panic non-blindspot panics
		}
	}()
	for i := range stmts {
		if err := it.execStmt(&stmts[i]); err != nil {
			return err
		}
		if it.signal != nil {
			return nil
		}
	}
	return nil
}

// ── SDK.Strategy implementation ─────────────────────────────────────

// OnInit implements sdk.Strategy.
func (it *Interpreter) OnInit(ctx sdk.Context) error {
	it.ctx = ctx
	it.series.bars = ctx.Bars()
	it.scopes = []map[string]Value{make(map[string]Value)}

	// Inject extern/input parameters from ctx into globals
	for _, p := range it.ir.Params {
		if it.ctx != nil {
			switch p.Type {
			case "int", "long":
				var def int
				if p.Default != nil {
					def = int(it.evalExpr(p.Default).ToInt())
				}
				it.globals[p.Name] = IntVal(int32(ctx.ParamInt(p.Name, def)))
			case "double", "float":
				var def decimal.Decimal
				if p.Default != nil {
					def = it.evalExpr(p.Default).ToDecimal()
				}
				it.globals[p.Name] = DecimalVal(ctx.ParamDecimal(p.Name, def))
			case "string":
				var def string
				if p.Default != nil {
					def = it.evalExpr(p.Default).ToString()
				}
				it.globals[p.Name] = StringVal(ctx.ParamString(p.Name, def))
			case "bool":
				var def bool
				if p.Default != nil {
					def = it.evalExpr(p.Default).IsTrue()
				}
				it.globals[p.Name] = BoolVal(ctx.ParamBool(p.Name, def))
			default:
				// Enum types or other custom types — treat as int (MQL enums are int32)
				if it.ir.EnumTypes != nil && it.ir.EnumTypes[p.Type] {
					var def int
					if p.Default != nil {
						def = int(it.evalExpr(p.Default).ToInt())
					}
					it.globals[p.Name] = IntVal(int32(ctx.ParamInt(p.Name, def)))
				} else {
					// Unknown type — try as string param
					var def string
					if p.Default != nil {
						def = it.evalExpr(p.Default).ToString()
					}
					it.globals[p.Name] = StringVal(ctx.ParamString(p.Name, def))
				}
			}
		}
	}

	if len(it.ir.OnInit) > 0 {
		err := it.execBlock(it.ir.OnInit)
		if errors.Is(err, errReturn) {
			return nil
		}
		return err
	}
	return nil
}

// OnBar implements sdk.Strategy.
func (it *Interpreter) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	it.ctx = ctx
	it.series.bars = ctx.Bars()
	it.scopes = []map[string]Value{make(map[string]Value)}
	it.signal = nil
	it.orderPool.Reset(ctx)
	it.posPool.Reset(ctx)

	if len(it.ir.OnBar) > 0 {
		if err := it.execBlock(it.ir.OnBar); err != nil {
			if errors.Is(err, errReturn) {
				return it.signal, nil
			}
			return nil, fmt.Errorf("OnBar: %w", err)
		}
	}
	return it.signal, nil
}

// OnDeinit implements sdk.Strategy.
func (it *Interpreter) OnDeinit(ctx sdk.Context, reason string) error {
	it.ctx = ctx
	it.scopes = []map[string]Value{make(map[string]Value)}
	if len(it.ir.OnDeinit) > 0 {
		err := it.execBlock(it.ir.OnDeinit)
		if errors.Is(err, errReturn) {
			return nil
		}
		return err
	}
	return nil
}

// OnTick implements sdk.TickStrategy (optional).
// Called on every price update (Bid/Ask) for scalping/market-making EAs.
func (it *Interpreter) OnTick(ctx sdk.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	if len(it.ir.OnTick) == 0 {
		return nil, nil
	}
	it.ctx = ctx
	it.series.bars = ctx.Bars()
	it.scopes = []map[string]Value{make(map[string]Value)}
	it.signal = nil
	it.orderPool.Reset(ctx)
	it.posPool.Reset(ctx)

	if err := it.execBlock(it.ir.OnTick); err != nil {
		if errors.Is(err, errReturn) {
			return it.signal, nil
		}
		return nil, fmt.Errorf("OnTick: %w", err)
	}
	return it.signal, nil
}

// OnTimer implements sdk.TimerStrategy (optional).
// Called periodically after EventSetTimer/EventSetTimerMillisecond in OnInit.
func (it *Interpreter) OnTimer(ctx sdk.Context) (*sdk.Signal, error) {
	if len(it.ir.OnTimer) == 0 {
		return nil, nil
	}
	it.ctx = ctx
	it.series.bars = ctx.Bars()
	it.scopes = []map[string]Value{make(map[string]Value)}
	it.signal = nil
	it.orderPool.Reset(ctx)
	it.posPool.Reset(ctx)

	if err := it.execBlock(it.ir.OnTimer); err != nil {
		if errors.Is(err, errReturn) {
			return it.signal, nil
		}
		return nil, fmt.Errorf("OnTimer: %w", err)
	}
	return it.signal, nil
}

// OnTrade implements sdk.TradeStrategy (optional).
// Called after an order is filled, closed, or modified.
// Maps MQL5 OnTrade() and OnTradeTransaction() callbacks.
func (it *Interpreter) OnTrade(ctx sdk.Context, event sdk.TradeEvent) (*sdk.Signal, error) {
	if len(it.ir.OnTrade) == 0 && len(it.ir.OnTradeTransaction) == 0 {
		return nil, nil
	}
	it.ctx = ctx
	it.series.bars = ctx.Bars()
	it.scopes = []map[string]Value{make(map[string]Value)}
	it.signal = nil
	it.orderPool.Reset(ctx)
	it.posPool.Reset(ctx)

	// OnTradeTransaction receives a MqlTradeTransaction struct.
	// We expose the trade event as global variables that the transaction
	// handler can read, since the IR has no struct parameter passing.
	if len(it.ir.OnTradeTransaction) > 0 {
		it.globals["_TransactionTicket"] = IntVal(int32(event.Ticket))
		it.globals["_TransactionSymbol"] = StringVal(event.Symbol)
		it.globals["_TransactionVolume"] = DecimalVal(event.Volume)
		it.globals["_TransactionPrice"] = DecimalVal(event.Price)
		it.globals["_TransactionProfit"] = DecimalVal(event.Profit)
		switch event.EventType {
		case sdk.TradeFilled:
			it.globals["_TransactionType"] = IntVal(0)
		case sdk.TradeClosed:
			it.globals["_TransactionType"] = IntVal(1)
		case sdk.TradeModified:
			it.globals["_TransactionType"] = IntVal(2)
		case sdk.TradeCancelled:
			it.globals["_TransactionType"] = IntVal(3)
		}
		if err := it.execBlock(it.ir.OnTradeTransaction); err != nil {
			if !errors.Is(err, errReturn) {
				return nil, fmt.Errorf("OnTradeTransaction: %w", err)
			}
		}
	}

	// OnTrade has no arguments — just executes the body.
	if len(it.ir.OnTrade) > 0 {
		it.scopes = []map[string]Value{make(map[string]Value)}
		it.signal = nil
		if err := it.execBlock(it.ir.OnTrade); err != nil {
			if errors.Is(err, errReturn) {
				return it.signal, nil
			}
			return nil, fmt.Errorf("OnTrade: %w", err)
		}
	}
	return it.signal, nil
}

// ── Helpers ─────────────────────────────────────────────────────────

func zeroValue(typeName string) Value {
	switch typeName {
	case "int", "long":
		return IntVal(0)
	case "double", "float":
		return DecimalVal(decimal.Zero)
	case "string":
		return StringVal("")
	case "bool":
		return BoolVal(false)
	case "datetime":
		return Value{Kind: ValDatetime, Datetime: 0}
	default:
		return NoneVal()
	}
}

// isClassType returns true for MQL5 class/struct type names.
func isClassType(typeName string) bool {
	if typeName == "class" {
		return true
	}
	switch typeName {
	case "CTrade", "MqlTradeRequest", "MqlTradeResult",
		"MqlDateTime", "MqlRates", "MqlTick":
		return true
	}
	// User-defined struct types: capitalized names that aren't primitives
	if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
		switch typeName {
		case "Int", "Double", "String", "Bool", "Long", "Float":
			return false
		}
		return true
	}
	return false
}

// classifyRuntimeSeverity determines the severity of a runtime blind spot.
// Fatal blind spots abort the current tick/bar; warnings allow continued execution.
func classifyRuntimeSeverity(name string) string {
	if isPermanentBlindSpot(name) {
		return "永久盲区"
	}
	// Trade / indicator functions are fatal — returning NoneVal corrupts EA logic
	if isTradeName(name) || isIndicatorName(name) {
		return "致命"
	}
	// iXxx pattern (custom indicators etc.)
	if len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z' {
		return "致命"
	}
	// Order*/Position*/Account* pattern
	if strings.HasPrefix(name, "Order") || strings.HasPrefix(name, "Position") || strings.HasPrefix(name, "Account") {
		return "致命"
	}
	return "警告"
}
