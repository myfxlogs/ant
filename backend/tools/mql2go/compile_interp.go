package mql2go

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"anttrader/tools/mql2go/interp"
)

// CompileToIR parses MQL source and compiles it to a pure Go IR
// suitable for interpretation. This is the host-side compile step;
// the resulting IR has no tree-sitter dependency and can run in WASM.
func CompileToIR(source string) (*interp.IR, error) {
	// Run preprocessor first (#define, #property stripping)
	source = PreprocessMQL(source)

	analyzeMu.Lock()
	parseSource = source
	defer analyzeMu.Unlock()

	root, err := ParseMQL(source)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	version := detectMQLVersion(source)
	c := &compiler{source: source, version: version}

	return c.compile(root), nil
}

type compiler struct {
	source  string
	version string
}

func (c *compiler) compile(root *sitter.Node) *interp.IR {
	ir := &interp.IR{Version: c.version}

	// First pass: collect known class/struct types
	knownClasses := make(map[string]bool)
	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		switch n.Type() {
		case "class_specifier", "struct_specifier":
			name := c.findTypeName(n)
			if name != "" {
				knownClasses[name] = true
			}
		}
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		switch n.Type() {
		case "declaration":
			c.collectGlobal(ir, n)
			c.collectClassInstance(ir, n, knownClasses)
		case "class_specifier", "struct_specifier":
			c.collectClassDecl(ir, n)
		case "function_definition":
			c.collectFunction(ir, n)
		}
	}

	return ir
}

// collectGlobal processes top-level declarations (globals + params).
func (c *compiler) collectGlobal(ir *interp.IR, n *sitter.Node) {
	text := c.text(n)
	if strings.Contains(text, "extern ") || strings.Contains(text, "input ") {
		c.collectParam(ir, n)
		return
	}
	// Skip function declarations
	if childByType(c.source, n, "function_declarator") != nil {
		return
	}
	c.collectGlobalVar(ir, n)
}

func (c *compiler) collectParam(ir *interp.IR, n *sitter.Node) {
	decl := n
	// Walk for init_declarator or parameter_declaration
	for i := 0; i < int(decl.NamedChildCount()); i++ {
		child := decl.NamedChild(i)
		if child.Type() == "init_declarator" || child.Type() == "declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			pd := interp.ParamDecl{
				Name: name,
				Type: c.findType(n),
			}
			// Look for default value
			if init := childByType(c.source, child, "init_declarator"); init != nil {
				if valExpr := c.findExprChild(init); valExpr != nil {
					pd.Default = c.compileExpr(valExpr)
				}
			}
			ir.Params = append(ir.Params, pd)
		}
	}
}

func (c *compiler) collectGlobalVar(ir *interp.IR, n *sitter.Node) {
	typeName := c.findType(n)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "init_declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			gv := interp.GlobalVar{
				Name: name,
				Type: typeName,
			}
			if valExpr := c.findExprChild(child); valExpr != nil {
				gv.InitVal = c.compileExpr(valExpr)
			}
			ir.Globals = append(ir.Globals, gv)
		} else if child.Type() == "declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			ir.Globals = append(ir.Globals, interp.GlobalVar{
				Name: name,
				Type: typeName,
			})
		} else if child.Type() == "identifier" && typeName != "" {
			// Direct declaration: CTrade trade; (no init_declarator wrapper)
			// Avoid double-adding if already handled by init_declarator above
			name := c.text(child)
			// Skip if this is the type_identifier itself
			if name != typeName {
				ir.Globals = append(ir.Globals, interp.GlobalVar{
					Name: name,
					Type: typeName,
				})
			}
		}
	}
}

// collectFunction maps MQL event functions to IR slots.
func (c *compiler) collectFunction(ir *interp.IR, n *sitter.Node) {
	name := funcName(n)
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
	case "OnBar":
		ir.OnBar = stmts
	case "OnTimer":
		ir.OnTimer = stmts
	case "OnDeinit":
		ir.OnDeinit = stmts
	default:
		// User-defined functions — not yet supported in IR
		// (future: add to a function table)
	}
}

// ── Statement compilation ───────────────────────────────────────────

func (c *compiler) compileBlock(n *sitter.Node) []interp.Statement {
	if n == nil || n.Type() != "compound_statement" {
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

	case "compound_statement":
		body := c.compileBlock(n)
		return &interp.Statement{Kind: interp.StmtBlock, Body: body}

	case "switch_statement":
		return c.compileSwitch(n)

	case "declaration":
		return c.compileDeclaration(n)

	case "update_expression":
		expr := c.compileExpr(n)
		if expr == nil {
			return nil
		}
		return &interp.Statement{Kind: interp.StmtExpr, Expr: expr}
	}
	return nil
}

func (c *compiler) compileIf(n *sitter.Node) *interp.Statement {
	cond := c.findExprChild(n)
	if cond == nil {
		return nil
	}
	stmt := &interp.Statement{
		Kind: interp.StmtIf,
		Cond: c.compileExpr(cond),
	}
	// Find body (compound_statement) and else clause
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "compound_statement" {
			stmt.Body = c.compileBlock(child)
		} else if child.Type() == "if_statement" {
			// else if → nested if in ElseBody
			stmt.ElseBody = []interp.Statement{*c.compileIf(child)}
		} else if child.Type() == "else_clause" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				ec := child.NamedChild(j)
				if ec.Type() == "compound_statement" {
					stmt.ElseBody = c.compileBlock(ec)
				} else if ec.Type() == "if_statement" {
					stmt.ElseBody = []interp.Statement{*c.compileIf(ec)}
				}
			}
		}
	}
	return stmt
}

func (c *compiler) compileFor(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtFor}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "declaration":
			if s := c.compileDeclaration(child); s != nil {
				stmt.Init = s
			}
		case "expression_statement":
			// Could be init or condition
			expr := c.compileExprFromStmt(child)
			if expr != nil {
				if stmt.Init == nil {
					stmt.Init = &interp.Statement{Kind: interp.StmtExpr, Expr: expr}
				} else if stmt.Cond == nil {
					stmt.Cond = expr
				}
			}
		case "binary_expression", "call_expression", "identifier", "number_literal":
			if stmt.Cond == nil {
				stmt.Cond = c.compileExpr(child)
			}
		case "update_expression":
			if s := c.compileStmt(child); s != nil {
				stmt.Update = s
			} else {
				expr := c.compileExpr(child)
				if expr != nil {
					stmt.Update = &interp.Statement{Kind: interp.StmtExpr, Expr: expr}
				}
			}
		case "compound_statement":
			stmt.Body = c.compileBlock(child)
		}
	}
	return stmt
}

func (c *compiler) compileWhile(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtWhile}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "parenthesized_expression" {
			stmt.Cond = c.compileExpr(child)
		} else if child.Type() == "compound_statement" {
			stmt.Body = c.compileBlock(child)
		}
	}
	return stmt
}

func (c *compiler) compileSwitch(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtSwitch}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "parenthesized_expression" {
			stmt.Expr = c.compileExpr(child)
		} else if child.Type() == "case_statement" {
			sc := interp.SwitchCase{}
			for j := 0; j < int(child.NamedChildCount()); j++ {
				cc := child.NamedChild(j)
				if cc.Type() == "default_label" {
					sc.Expr = nil
				} else if cc.Type() == "case_label" {
					sc.Expr = c.compileExpr(cc)
				} else if cc.Type() == "expression_statement" || cc.Type() == "compound_statement" || cc.Type() == "break_statement" {
					if cc.Type() == "compound_statement" {
						sc.Body = c.compileBlock(cc)
					} else if cc.Type() != "break_statement" {
						if s := c.compileStmt(cc); s != nil {
							sc.Body = append(sc.Body, *s)
						}
					}
				}
			}
			stmt.Cases = append(stmt.Cases, sc)
		}
	}
	return stmt
}

func (c *compiler) compileDeclaration(n *sitter.Node) *interp.Statement {
	// Variable declaration as a statement: int x = 5;
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "init_declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			valExpr := c.findExprChild(child)
			if valExpr != nil {
				return &interp.Statement{
					Kind: interp.StmtExpr,
					Expr: &interp.Expr{
						Kind: interp.ExprAssignment,
						Name: name,
						Args: []interp.Expr{*c.compileExpr(valExpr)},
					},
				}
			}
		}
	}
	return nil
}
