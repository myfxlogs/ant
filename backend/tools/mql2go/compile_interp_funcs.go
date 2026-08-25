package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file contains function collection and statement compilation
// extracted from compile_interp.go for file-size compliance.

// collectFunction maps MQL event functions to IR slots.
func (c *compiler) collectFunction(ir *interp.IR, n *sitter.Node) {
	name := funcName(c.source, n)
	if name == "" {
		return
	}
	// Skip class declarations that tree-sitter mis-parses as function_definition
	if isBuiltinClass(name) {
		return
	}
	body := funcBody(n)
	if body == nil {
		return
	}
	stmts := c.compileBlock(body)

	switch name {
	case "OnInit":
		ir.OnInit = stmts
	case "OnTick":
		ir.OnTick = stmts
	case "start":
		// MQL4 legacy: start() is equivalent to OnTick()
		ir.OnTick = stmts
	case "OnBar":
		ir.OnBar = stmts
	case "OnTimer":
		ir.OnTimer = stmts
	case "OnTrade":
		ir.OnTrade = stmts
	case "OnTradeTransaction":
		ir.OnTradeTransaction = stmts
	case "OnBookEvent":
		ir.OnBookEvent = stmts
	case "OnDeinit":
		ir.OnDeinit = stmts
	default:
		// User-defined function
		params := c.collectFuncParams(n)
		ir.Funcs[name] = &interp.FuncDef{
			Name:   name,
			Params: params,
			Body:   stmts,
		}
		return
	}
	// Event handlers: also register as callable user functions so they
	// can be invoked from other code (e.g. OnTick calling OnTimer()).
	ir.Funcs[name] = &interp.FuncDef{
		Name:   name,
		Params: c.collectFuncParams(n),
		Body:   stmts,
	}
}

// ── Statement compilation ───────────────────────────────────────────

func (c *compiler) compileBlock(n *sitter.Node) []interp.Statement {
	if n == nil || n.Type() != nodeCompoundStatement {
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

func (c *compiler) compileStmt(n *sitter.Node) *interp.Statement {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "expression_statement":
		expr := c.compileExprFromStmt(n)
		if expr == nil {
			if c.err == nil {
				c.err = fmt.Errorf("unsupported MQL expression statement: %s", c.text(n))
			}
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
			// Handle 'return(val)' which tree-sitter may parse as call_expression
			// with function name 'return' — unwrap to the argument
			if expr.Type() == "call_expression" && callFuncName(c.source, expr) == "return" {
				args := c.compileArgs(expr)
				if len(args) > 0 {
					e = &args[0]
				}
			} else {
				e = c.compileExpr(expr)
			}
		}
		return &interp.Statement{Kind: interp.StmtReturn, Expr: e}

	case nodeCompoundStatement:
		body := c.compileBlock(n)
		return &interp.Statement{Kind: interp.StmtBlock, Body: body}

	case "switch_statement":
		return c.compileSwitch(n)

	case "break_statement":
		return &interp.Statement{Kind: interp.StmtBreak}

	case "continue_statement":
		return &interp.Statement{Kind: interp.StmtContinue}

	case "do_statement":
		return c.compileDoWhile(n)

	case "declaration":
		return c.compileDeclaration(n)

	case "update_expression":
		expr := c.compileExpr(n)
		if expr == nil {
			return nil
		}
		return &interp.Statement{Kind: interp.StmtExpr, Expr: expr}

	case "empty_statement", "comment":
		return nil
	default:
		if isMQLExpressionNodeType(n.Type()) {
			return nil
		}
		if c.err == nil {
			c.err = fmt.Errorf("unsupported MQL statement node: %s", n.Type())
		}
		return nil
	}
}

func (c *compiler) compileIf(n *sitter.Node) *interp.Statement {
	cond := c.findExprChild(n)
	if cond == nil {
		if c.err == nil {
			c.err = fmt.Errorf("if statement has no condition")
		}
		return nil
	}
	stmt := &interp.Statement{
		Kind: interp.StmtIf,
		Cond: c.compileExpr(cond),
	}
	// Find body (compound_statement or single statement) and else clause
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeCompoundStatement {
			stmt.Body = c.compileBlock(child)
		} else if child.Type() == "if_statement" {
			// else if → nested if in ElseBody
			stmt.ElseBody = []interp.Statement{*c.compileIf(child)}
		} else if child.Type() == "else_clause" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				ec := child.NamedChild(j)
				if ec.Type() == nodeCompoundStatement {
					stmt.ElseBody = c.compileBlock(ec)
				} else if ec.Type() == "if_statement" {
					stmt.ElseBody = []interp.Statement{*c.compileIf(ec)}
				} else if s := c.compileStmt(ec); s != nil {
					// single-statement else body
					stmt.ElseBody = append(stmt.ElseBody, *s)
				}
			}
		} else if s := c.compileStmt(child); s != nil {
			// single-statement if body (no braces)
			if stmt.Body == nil {
				stmt.Body = []interp.Statement{*s}
			}
		}
	}
	return stmt
}
