package interp

import (
	"fmt"

	"github.com/shopspring/decimal"
	"anttrader/strategy/sdk"
)

// Interpreter executes a compiled IR against SDK interfaces.
// Pure Go — no tree-sitter dependency — safe for WASM.
type Interpreter struct {
	ir        *IR
	ctx       sdk.Context
	globals   map[string]Value
	locals    map[string]Value
	series    *SeriesAccessor
	orderPool *MQL4OrderPool
	posPool   *MQL5PositionPool
	signal    *sdk.Signal
	lastErr   int
	errSet    bool
}

// NewInterpreter creates an Interpreter from a compiled IR.
func NewInterpreter(ir *IR) *Interpreter {
	it := &Interpreter{
		ir:      ir,
		globals: make(map[string]Value),
		locals:  make(map[string]Value),
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

// ── Variable management ─────────────────────────────────────────────

func (it *Interpreter) getVar(name string) Value {
	if v, ok := it.locals[name]; ok {
		return v
	}
	if v, ok := it.globals[name]; ok {
		return v
	}
	return NoneVal()
}

func (it *Interpreter) setVar(name string, val Value) {
	if _, isLocal := it.locals[name]; isLocal {
		it.locals[name] = val
		return
	}
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
				return err
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
				return err
			}
			if it.signal != nil {
				return nil
			}
		}

	case StmtReturn:
		if stmt.Expr != nil {
			it.evalExpr(stmt.Expr)
		}
		return nil

	case StmtBlock:
		return it.execBlock(stmt.Body)

	case StmtSwitch:
		switchVal := it.evalExpr(stmt.Expr)
		for _, c := range stmt.Cases {
			if c.Expr == nil {
				// default
				return it.execBlock(c.Body)
			}
			caseVal := it.evalExpr(c.Expr)
			if switchVal.Equal(caseVal) {
				return it.execBlock(c.Body)
			}
		}
	}
	return nil
}

func (it *Interpreter) execBlock(stmts []Statement) error {
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
	it.locals = make(map[string]Value)
	if len(it.ir.OnInit) > 0 {
		return it.execBlock(it.ir.OnInit)
	}
	return nil
}

// OnBar implements sdk.Strategy.
func (it *Interpreter) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	it.ctx = ctx
	it.series.bars = ctx.Bars()
	it.locals = make(map[string]Value)
	it.signal = nil
	it.orderPool.Reset(ctx)
	it.posPool.Reset(ctx)

	if len(it.ir.OnBar) > 0 {
		if err := it.execBlock(it.ir.OnBar); err != nil {
			return nil, fmt.Errorf("OnBar: %w", err)
		}
	}
	return it.signal, nil
}

// OnDeinit implements sdk.Strategy.
func (it *Interpreter) OnDeinit(ctx sdk.Context, reason string) error {
	it.ctx = ctx
	it.locals = make(map[string]Value)
	if len(it.ir.OnDeinit) > 0 {
		return it.execBlock(it.ir.OnDeinit)
	}
	return nil
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

// SetSignal stores a signal to be returned from OnBar.
func (it *Interpreter) SetSignal(sig *sdk.Signal) {
	it.signal = sig
}

// Context returns the current SDK context.
func (it *Interpreter) Context() sdk.Context {
	return it.ctx
}
