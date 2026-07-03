package mql2go

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"anttrader/tools/mql2go/interp"
)

// ── Statement compilation (Python CST → interp.Statement) ──────────

func (c *pyCompiler) compileBlock(n *sitter.Node) []interp.Statement {
	if n == nil || n.Type() != "block" {
		return nil
	}
	var stmts []interp.Statement
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if s := c.compileStmt(child); s != nil {
			stmts = append(stmts, *s)
		}
	}
	return stmts
}

func (c *pyCompiler) compileStmt(n *sitter.Node) *interp.Statement {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "expression_statement":
		expr := c.compileExprFromStmt(n)
		if expr == nil {
			return nil
		}
		return &interp.Statement{Kind: interp.StmtExpr, Expr: expr}

	case "if_statement":
		return c.compileIf(n)

	case "for_statement":
		return c.compileFor(n)

	case "while_statement":
		return c.compileWhile(n)

	case "return_statement":
		expr := c.findExprChild(n)
		var e *interp.Expr
		if expr != nil {
			e = c.compileExpr(expr)
		}
		return &interp.Statement{Kind: interp.StmtReturn, Expr: e}

	case "block":
		body := c.compileBlock(n)
		return &interp.Statement{Kind: interp.StmtBlock, Body: body}

	case "break_statement":
		return &interp.Statement{Kind: interp.StmtBreak}

	case "continue_statement":
		return &interp.Statement{Kind: interp.StmtContinue}

	case "pass_statement":
		return nil
	case "class_definition":
		panic(fmt.Sprintf("line %d: nested class definitions are not supported in Python subset", n.StartPoint().Row+1))
	}
	return nil
}

func (c *pyCompiler) compileIf(n *sitter.Node) *interp.Statement {
	cond := c.findExprChild(n)
	if cond == nil {
		return nil
	}
	stmt := &interp.Statement{
		Kind: interp.StmtIf,
		Cond: c.compileExpr(cond),
	}

	// Tail tracking for elif chaining: each elif nests inside the previous one's ElseBody
	elseTail := stmt

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "block":
			if stmt.Body == nil {
				stmt.Body = c.compileBlock(child)
			} else {
				elseTail.ElseBody = c.compileBlock(child)
			}
		case "elif_clause":
			elifCond := c.findExprChild(child)
			elifBody := findNamedChild(child, "block")
			elifStmt := &interp.Statement{
				Kind: interp.StmtIf,
				Cond: c.compileExpr(elifCond),
			}
			if elifBody != nil {
				elifStmt.Body = c.compileBlock(elifBody)
			}
			elseTail.ElseBody = []interp.Statement{*elifStmt}
			elseTail = &elseTail.ElseBody[0]
		case "else_clause":
			for j := 0; j < int(child.NamedChildCount()); j++ {
				ec := child.NamedChild(j)
				if ec.Type() == "block" {
					elseTail.ElseBody = c.compileBlock(ec)
				}
			}
		}
	}
	return stmt
}

func (c *pyCompiler) compileFor(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtFor}

	var targetNode *sitter.Node
	var iterNode *sitter.Node
	var bodyNode *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "identifier":
			if targetNode == nil {
				targetNode = child
			} else if iterNode == nil {
				iterNode = child
			}
		case "call":
			iterNode = child
		case "attribute":
			iterNode = child
		case "block":
			bodyNode = child
		case "else_clause":
			panic(fmt.Sprintf("line %d: for...else is not supported in Python subset", n.StartPoint().Row+1))
		}
	}

	if targetNode == nil {
		return stmt
	}
	if iterNode == nil {
		panic(fmt.Sprintf("line %d: Python subset only supports 'for ... in range(...)' or 'for ... in ctx.positions'", n.StartPoint().Row+1))
	}

	targetName := c.text(targetNode)

	// Check for ctx.positions iterator (for pos in ctx.positions:)
	if isCtxPositions(c, iterNode) {
		return c.compileForPositions(stmt, targetName, bodyNode)
	}

	funcName := c.callName(iterNode)
	if funcName != "range" {
		panic(fmt.Sprintf("line %d: Python subset only supports 'for ... in range(...)' or 'for ... in ctx.positions', got: for %s in %s",
			n.StartPoint().Row+1, targetName, c.text(iterNode)))
	}

	args := c.compileArgs(iterNode)
	if len(args) == 0 {
		return stmt
	}

	stmt.Init = &interp.Statement{
		Kind: interp.StmtExpr,
		Expr: &interp.Expr{
			Kind: interp.ExprDecl,
			Name: targetName,
			Args: []interp.Expr{rangeInit(args)},
		},
	}
	if len(args) == 1 {
		stmt.Cond = &interp.Expr{
			Kind: interp.ExprBinary,
			Op:   "<",
			Args: []interp.Expr{
				{Kind: interp.ExprVar, Name: targetName},
				args[0],
			},
		}
	} else {
		stmt.Cond = &interp.Expr{
			Kind: interp.ExprBinary,
			Op:   "<",
			Args: []interp.Expr{
				{Kind: interp.ExprVar, Name: targetName},
				args[1],
			},
		}
	}
	if len(args) >= 3 {
		stmt.Update = &interp.Statement{
			Kind: interp.StmtExpr,
			Expr: &interp.Expr{
				Kind: interp.ExprCompoundAssign,
				Name: targetName,
				Op:   "+=",
				Args: []interp.Expr{args[2]},
			},
		}
	} else {
		stmt.Update = &interp.Statement{
			Kind: interp.StmtExpr,
			Expr: &interp.Expr{
				Kind: interp.ExprUpdate,
				Name: targetName,
				Op:   "++",
			},
		}
	}

	if bodyNode != nil {
		stmt.Body = c.compileBlock(bodyNode)
	}

	return stmt
}

// isCtxPositions checks if the iterator node is `ctx.positions`.
func isCtxPositions(c *pyCompiler, iterNode *sitter.Node) bool {
	if iterNode == nil {
		return false
	}
	text := c.text(iterNode)
	return text == "ctx.positions"
}

// positionFieldMap maps pos.field → PositionGetXxx(propConstant).
var positionFieldMap = map[string]struct {
	builtin string
	prop    int32
}{
	"ticket":      {"PositionGetInteger", 0},  // POSITION_TICKET
	"magic":       {"PositionGetInteger", 1},  // POSITION_MAGIC
	"side":        {"PositionGetInteger", 2},  // POSITION_TYPE
	"open_time":   {"PositionGetInteger", 3},  // POSITION_TIME
	"volume":      {"PositionGetDouble", 0},   // POSITION_VOLUME
	"open_price":  {"PositionGetDouble", 1},   // POSITION_PRICE_OPEN
	"sl":          {"PositionGetDouble", 2},   // POSITION_SL
	"stop_loss":   {"PositionGetDouble", 2},   // POSITION_SL
	"tp":          {"PositionGetDouble", 3},   // POSITION_TP
	"take_profit": {"PositionGetDouble", 3},   // POSITION_TP
	"swap":        {"PositionGetDouble", 5},   // POSITION_SWAP
	"commission":  {"PositionGetDouble", 6},   // POSITION_COMMISSION
	"profit":      {"PositionGetDouble", 7},   // POSITION_PROFIT
	"symbol":      {"PositionGetSymbol", 0},   // POSITION_SYMBOL
	"comment":     {"PositionGetString", 1},   // POSITION_COMMENT
}

// compileForPositions desugars `for pos in ctx.positions:` into:
//
//	for i in range(0, PositionsTotal()):
//	    ticket = PositionGetTicket(i)
//	    PositionSelectByTicket(ticket)
//	    <body with pos.field → PositionGetXxx(prop)>
func (c *pyCompiler) compileForPositions(stmt *interp.Statement, targetName string, bodyNode *sitter.Node) *interp.Statement {
	indexVar := targetName + "__i"
	ticketVar := targetName + "__ticket"

	// Register position loop variable for attribute mapping
	if c.posLoopVars == nil {
		c.posLoopVars = make(map[string]bool)
	}
	c.posLoopVars[targetName] = true

	// Init: i = 0
	stmt.Init = &interp.Statement{
		Kind: interp.StmtExpr,
		Expr: &interp.Expr{
			Kind: interp.ExprDecl,
			Name: indexVar,
			Args: []interp.Expr{{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}},
		},
	}

	// Cond: i < PositionsTotal()
	stmt.Cond = &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   "<",
		Args: []interp.Expr{
			{Kind: interp.ExprVar, Name: indexVar},
			{Kind: interp.ExprCall, Name: "PositionsTotal"},
		},
	}

	// Update: i++
	stmt.Update = &interp.Statement{
		Kind: interp.StmtExpr,
		Expr: &interp.Expr{
			Kind: interp.ExprUpdate,
			Name: indexVar,
			Op:   "++",
		},
	}

	// Compile body AFTER posLoopVars registration so position field detection works
	var compiledBody []interp.Statement
	if bodyNode != nil {
		compiledBody = c.compileBlock(bodyNode)
	}

	// Prepend to body: ticket = PositionGetTicket(i); PositionSelectByTicket(ticket)
	preBody := []interp.Statement{
		{
			Kind: interp.StmtExpr,
			Expr: &interp.Expr{
				Kind: interp.ExprDecl,
				Name: ticketVar,
				Args: []interp.Expr{
					{Kind: interp.ExprCall, Name: "PositionGetTicket", Args: []interp.Expr{{Kind: interp.ExprVar, Name: indexVar}}},
				},
			},
		},
		{
			Kind: interp.StmtExpr,
			Expr: &interp.Expr{
				Kind: interp.ExprCall,
				Name: "PositionSelectByTicket",
				Args: []interp.Expr{{Kind: interp.ExprVar, Name: ticketVar}},
			},
		},
	}
	stmt.Body = append(preBody, compiledBody...)

	return stmt
}

func (c *pyCompiler) compileWhile(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtWhile}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "block":
			stmt.Body = c.compileBlock(child)
		case "else_clause":
			panic(fmt.Sprintf("line %d: while...else is not supported in Python subset", n.StartPoint().Row+1))
		default:
			if stmt.Cond == nil {
				if e := c.compileExpr(child); e != nil {
					stmt.Cond = e
				}
			}
		}
	}
	return stmt
}
