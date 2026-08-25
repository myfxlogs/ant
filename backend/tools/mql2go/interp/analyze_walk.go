package interp

import (
	"sort"
)

func walkIR(ir *IR, visit func(*Expr)) {
	walkStmts(ir.OnInit, visit)
	walkStmts(ir.OnBar, visit)
	walkStmts(ir.OnTick, visit)
	walkStmts(ir.OnTimer, visit)
	walkStmts(ir.OnTrade, visit)
	walkStmts(ir.OnTradeTransaction, visit)
	walkStmts(ir.OnDeinit, visit)
	funcNames := make([]string, 0, len(ir.Funcs))
	for name := range ir.Funcs {
		funcNames = append(funcNames, name)
	}
	sort.Strings(funcNames)
	for _, name := range funcNames {
		if fn := ir.Funcs[name]; fn != nil {
			walkStmts(fn.Body, visit)
		}
	}
}

func walkStmts(stmts []Statement, visit func(*Expr)) {
	for i := range stmts {
		walkStmt(&stmts[i], visit)
	}
}

func walkStmt(s *Statement, visit func(*Expr)) {
	if s.Expr != nil {
		walkExpr(s.Expr, visit)
	}
	if s.Cond != nil {
		walkExpr(s.Cond, visit)
	}
	if s.Init != nil {
		walkStmt(s.Init, visit)
	}
	if s.Update != nil {
		walkStmt(s.Update, visit)
	}
	walkStmts(s.Body, visit)
	walkStmts(s.ElseBody, visit)
	for i := range s.Cases {
		if s.Cases[i].Expr != nil {
			walkExpr(s.Cases[i].Expr, visit)
		}
		walkStmts(s.Cases[i].Body, visit)
	}
}

func walkExpr(e *Expr, visit func(*Expr)) {
	visit(e)
	for i := range e.Args {
		walkExpr(&e.Args[i], visit)
	}
	if e.Index != nil {
		walkExpr(e.Index, visit)
	}
	if e.Cond != nil {
		walkExpr(e.Cond, visit)
	}
	if e.ThenExpr != nil {
		walkExpr(e.ThenExpr, visit)
	}
	if e.ElseExpr != nil {
		walkExpr(e.ElseExpr, visit)
	}
}

// ── utilities ───────────────────────────────────────────────────────

func determineExecKind(ir *IR) string {
	if len(ir.OnTick) > 0 {
		return "on_tick"
	}
	if len(ir.OnBar) > 0 {
		return "on_bar"
	}
	if len(ir.OnTimer) > 0 {
		return "on_timer"
	}
	return ""
}

func severityRank(severity string) int {
	switch severity {
	case SeverityFatal:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

func finalizeBlindSpots(m map[string]*IRBlindSpot) []IRBlindSpot {
	result := make([]IRBlindSpot, 0, len(m))
	for _, bs := range m {
		result = append(result, *bs)
	}
	sort.Slice(result, func(i, j int) bool {
		rankI := severityRank(result[i].Severity)
		rankJ := severityRank(result[j].Severity)
		if rankI != rankJ {
			return rankI < rankJ
		}
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Builtin < result[j].Builtin
	})
	return result
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// countEntryExitRules counts entry-order and exit-order calls in the IR.
// Entry: OrderSend, CTrade.Buy/Sell/BuyLimit/SellLimit/BuyStop/SellStop
// Exit:  OrderClose, OrderCloseBy, OrderDelete, CTrade.PositionClose/PositionClosePartial/PositionCloseBy
func countEntryExitRules(ir *IR) (entry, exit int) {
	entryNames := map[string]bool{
		"OrderSend": true,
	}
	exitNames := map[string]bool{
		"OrderClose": true, "OrderCloseBy": true, "OrderDelete": true,
	}
	ctradeEntry := map[string]bool{
		"Buy": true, "Sell": true, "BuyLimit": true, "SellLimit": true,
		"BuyStop": true, "SellStop": true,
	}
	ctradeExit := map[string]bool{
		"PositionClose": true, "PositionClosePartial": true, "PositionCloseBy": true,
		"OrderDelete": true,
	}
	globalTypes := buildGlobalTypes(ir)
	visit := func(e *Expr) {
		switch e.Kind {
		case ExprCall:
			if entryNames[e.Name] {
				entry++
			}
			if exitNames[e.Name] {
				exit++
			}
		case ExprField:
			if !e.IsAssign && len(e.Args) > 1 {
				clsType := resolveClassType(&e.Args[0], globalTypes)
				if clsType == "CTrade" {
					if ctradeEntry[e.Name] {
						entry++
					}
					if ctradeExit[e.Name] {
						exit++
					}
				}
			}
		}
	}
	walkIR(ir, visit)
	return
}

// EvalExprLiteral evaluates a simple literal/var Expr to its string form.
// Used for extracting parameter default values without a full evaluation pass.
func EvalExprLiteral(e *Expr) string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case ExprLiteral:
		return e.Val.ToString()
	case ExprVar, ExprConst:
		return e.Name
	case ExprUnary:
		if e.Op == "-" {
			return "-" + EvalExprLiteral(&e.Args[0])
		}
		return EvalExprLiteral(&e.Args[0])
	}
	return ""
}
