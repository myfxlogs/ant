package mql2go

import (
	"strings"

	"anttrader/tools/mql2go/interp"
)

// ── IR Statement → Go source emission ──────────────────────────────

func (g *irGenerator) emitStmt(s *interp.Statement) {
	switch s.Kind {
	case interp.StmtExpr:
		g.emitExprStmt(s.Expr)

	case interp.StmtIf:
		g.emitIf(s)

	case interp.StmtFor:
		g.emitFor(s)

	case interp.StmtWhile:
		g.emitWhile(s)

	case interp.StmtDoWhile:
		g.emitDoWhile(s)

	case interp.StmtReturn:
		if s.Expr != nil {
			val := g.expr(s.Expr)
			// OnInit/OnDeinit return error — convert integer 0 to nil
			if val == "0" && (g.curFunc == "OnInit" || g.curFunc == "OnDeinit") {
				g.emit("return nil")
			} else {
				g.emitf("return %s", val)
			}
		} else {
			g.emit("return nil")
		}

	case interp.StmtBreak:
		g.emit("break")

	case interp.StmtContinue:
		g.emit("continue")

	case interp.StmtBlock:
		g.emit("{")
		g.indent++
		g.emitStmts(s.Body)
		g.indent--
		g.emit("}")

	case interp.StmtSwitch:
		g.emitSwitch(s)
	}
}

func (g *irGenerator) emitExprStmt(e *interp.Expr) {
	if e == nil {
		return
	}
	expr := g.expr(e)
	if expr == "" || expr == "nil" {
		return
	}
	// Multi-line expressions (OrderSend closures) need special handling
	if strings.Contains(expr, "\n") {
		// Indent each line properly
		lines := strings.Split(expr, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}
		g.emit(strings.Join(lines, "\n"+strings.Repeat("\t", g.indent)))
	} else {
		g.emit(expr)
	}
}

func (g *irGenerator) emitIf(s *interp.Statement) {
	cond := g.expr(s.Cond)
	// Eliminate dead code: if true { ... } → just emit body
	if cond == "true" {
		g.emitStmts(s.Body)
		return
	}
	// Eliminate dead code: if false { ... } else { ... } → just emit else body
	if cond == "false" {
		if len(s.ElseBody) > 0 {
			g.emitStmts(s.ElseBody)
		}
		return
	}
	g.emitf("if %s {", cond)
	g.indent++
	g.emitStmts(s.Body)
	g.indent--
	if len(s.ElseBody) > 0 {
		// Check if else body is a single if statement (else-if chain)
		if len(s.ElseBody) == 1 && s.ElseBody[0].Kind == interp.StmtIf {
			g.b.WriteString(strings.Repeat("\t", g.indent))
			g.b.WriteString("} else ")
			g.emitStmt(&s.ElseBody[0])
		} else {
			g.emit("} else {")
			g.indent++
			g.emitStmts(s.ElseBody)
			g.indent--
			g.emit("}")
		}
	} else {
		g.emit("}")
	}
}

func (g *irGenerator) emitFor(s *interp.Statement) {
	// Detect MQL4 OrdersTotal() / MQL5 PositionsTotal() loop pattern
	// and transform to a Go range loop over ctx.Broker().Positions().
	if loopVar := g.detectPositionLoop(s); loopVar != "" {
		g.emitf("for _, %s := range ctx.Broker().Positions(0) {", loopVar)
		g.indent++
		saved := g.posLoopVar
		g.posLoopVar = loopVar
		g.emitStmts(s.Body)
		g.posLoopVar = saved
		g.indent--
		g.emit("}")
		return
	}

	var init, cond, update string
	if s.Init != nil {
		var b strings.Builder
		old := g.b
		g.b = b
		g.emitStmt(s.Init)
		init = strings.TrimSpace(g.b.String())
		g.b = old
	}
	if s.Cond != nil {
		cond = g.expr(s.Cond)
	}
	if s.Update != nil {
		var b strings.Builder
		old := g.b
		g.b = b
		g.emitStmt(s.Update)
		update = strings.TrimSpace(g.b.String())
		g.b = old
	}
	g.emitf("for %s; %s; %s {", init, cond, update)
	g.indent++
	g.emitStmts(s.Body)
	g.indent--
	g.emit("}")
}

// detectPositionLoop checks if a for-loop iterates over OrdersTotal()/PositionsTotal()
// and returns the loop variable name if so.
func (g *irGenerator) detectPositionLoop(s *interp.Statement) string {
	if s.Init == nil || s.Init.Expr == nil {
		return ""
	}
	// Init must be a declaration: int i = OrdersTotal() - 1  or  int i = 0
	if s.Init.Expr.Kind != interp.ExprDecl && s.Init.Expr.Kind != interp.ExprAssignment {
		return ""
	}
	loopVar := s.Init.Expr.Name
	if len(s.Init.Expr.Args) == 0 {
		return ""
	}
	initExpr := &s.Init.Expr.Args[0]

	// Check if init involves OrdersTotal() or PositionsTotal()
	usesTotal := g.exprUsesBuiltin(initExpr, "OrdersTotal") ||
		g.exprUsesBuiltin(initExpr, "PositionsTotal")

	if !usesTotal {
		// Also accept: for (int i = 0; i < OrdersTotal(); i++)
		if s.Cond != nil && g.exprUsesBuiltin(s.Cond, "OrdersTotal") {
			usesTotal = true
		}
		if s.Cond != nil && g.exprUsesBuiltin(s.Cond, "PositionsTotal") {
			usesTotal = true
		}
	}

	if !usesTotal {
		return ""
	}

	// Check body for OrderSelect/PositionSelectByTicket pattern
	for i := range s.Body {
		if s.Body[i].Expr != nil && g.exprUsesBuiltin(s.Body[i].Expr, "OrderSelect") {
			return loopVar
		}
		if s.Body[i].Expr != nil && g.exprUsesBuiltin(s.Body[i].Expr, "PositionSelectByTicket") {
			return loopVar
		}
	}
	// Even without OrderSelect, if it uses OrdersTotal, transform it
	return loopVar
}

// exprUsesBuiltin checks if an expression tree contains a call to the given builtin.
func (g *irGenerator) exprUsesBuiltin(e *interp.Expr, name string) bool {
	if e == nil {
		return false
	}
	if e.Kind == interp.ExprCall && e.Name == name {
		return true
	}
	for i := range e.Args {
		if g.exprUsesBuiltin(&e.Args[i], name) {
			return true
		}
	}
	if e.Cond != nil && g.exprUsesBuiltin(e.Cond, name) {
		return true
	}
	if e.Index != nil && g.exprUsesBuiltin(e.Index, name) {
		return true
	}
	return false
}

func (g *irGenerator) emitWhile(s *interp.Statement) {
	cond := g.expr(s.Cond)
	g.emitf("for %s {", cond)
	g.indent++
	g.emitStmts(s.Body)
	g.indent--
	g.emit("}")
}

func (g *irGenerator) emitDoWhile(s *interp.Statement) {
	g.emit("for {")
	g.indent++
	g.emitStmts(s.Body)
	cond := g.expr(s.Cond)
	g.emitf("if !%s { break }", cond)
	g.indent--
	g.emit("}")
}

func (g *irGenerator) emitSwitch(s *interp.Statement) {
	expr := g.expr(s.Expr)
	g.emitf("switch %s {", expr)
	g.indent++
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Expr == nil {
			g.emit("default:")
		} else {
			g.emitf("case %s:", g.expr(c.Expr))
		}
		g.indent++
		g.emitStmts(c.Body)
		g.indent--
	}
	g.indent--
	g.emit("}")
}
