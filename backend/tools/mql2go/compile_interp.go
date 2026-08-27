package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
	sitter "github.com/smacker/go-tree-sitter"
)

// CompileToIR parses MQL source and compiles it to a pure Go IR
// suitable for interpretation. This is the host-side compile step.
//
// Safety: enforces MaxSourceSize limit and recovers from panics
// (tree-sitter cgo panics, deep recursion). ADR-0023 §5.4.
func CompileToIR(source string) (ir *interp.IR, err error) {
	if len(source) > MaxSourceSize {
		return nil, fmt.Errorf("MQL source too large: %d bytes (max %d)", len(source), MaxSourceSize)
	}
	defer func() {
		if r := recover(); r != nil {
			ir = nil
			err = fmt.Errorf("compile MQL panic: %v", r)
		}
	}()

	// Run preprocessor first (#define, #property stripping)
	source = PreprocessMQL(source)

	root, err := ParseMQL(source)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	version := detectMQLVersion(source)
	c := &compiler{source: source, version: version}

	result := c.compile(root)
	if c.err != nil {
		return nil, c.err
	}
	return result, nil
}

type compiler struct {
	source  string
	version string
	err     error // VM-COMPILER-SEMANTICS-1: compile-time errors (local arrays, etc.)
}

func (c *compiler) compile(root *sitter.Node) *interp.IR {
	ir := &interp.IR{
		Version:    c.version,
		Funcs:      make(map[string]*interp.FuncDef),
		Enums:      make(map[string]int32),
		ClassTypes: make(map[string]bool), // VM-COMPILER-SEMANTICS-1
	}

	// First pass: collect known class/struct types and enums
	knownClasses := make(map[string]bool)
	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		switch n.Type() {
		case "class_specifier", "struct_specifier":
			name := c.findTypeName(n)
			if name != "" {
				knownClasses[name] = true
			}
		case "enum_specifier":
			c.collectEnum(ir, n)
		}
	}

	// VM-COMPILER-SEMANTICS-1: populate ClassTypes (user-defined + builtin).
	for name := range knownClasses {
		ir.ClassTypes[name] = true
	}
	for _, builtin := range []string{"CTrade", "MqlTradeRequest", "MqlTradeResult",
		"MqlDateTime", "MqlRates", "MqlTick"} {
		ir.ClassTypes[builtin] = true
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		// VM-COMPILER-SEMANTICS-4: reject `input`/`extern` used as an identifier
		// in any node type (not just default branch). `int x = input ;` is parsed
		// as a declaration, so the check must run before the switch.
		if err := checkReservedKeywordUsage(n, c); err != nil {
			c.err = err
			return ir
		}
		switch n.Type() {
		case "declaration":
			// VM-COMPILER-SEMANTICS-4: missing-initializer guard — reject
			// syntactically invalid MQL declarations (e.g. `int x = ;`) instead
			// of silently returning empty IR. We check specifically for a MISSING
			// node inside an init_declarator (missing initializer expression),
			// not just any MISSING node in the declaration — Python source
			// parsed as MQL produces declarations with a missing `;` (not a
			// missing initializer), and the MQL compiler must stay lenient on
			// non-MQL input so compileForLive(isPython=false) doesn't error.
			if hasMissingInitializer(n) {
				c.err = fmt.Errorf("syntax error in declaration at line %d (missing initializer)", n.StartPoint().Row+1)
				return ir
			}
			c.collectGlobal(ir, n)
			c.collectClassInstance(ir, n, knownClasses)
		case "class_specifier", "struct_specifier":
			c.collectClassDecl(ir, n)
		case "enum_specifier":
			// already processed in first pass
		case "function_definition":
			c.collectFunction(ir, n)
		default:
			// VM-COMPILER-SEMANTICS-4: structured detection replaces
			// `int x = input ;`. checkReservedKeywordUsage already ran
			if isInputDeclaration(n, c) || isExternDeclaration(n, c) {
				if !isValidInputDeclaration(n, c) {
					c.err = fmt.Errorf("invalid input/extern declaration: missing initializer at line %d", n.StartPoint().Row+1)
					return ir
				}
				c.collectParam(ir, n)
			}
		}
	}

	return ir
}

// collectGlobal processes top-level declarations (globals + params).
func (c *compiler) collectGlobal(ir *interp.IR, n *sitter.Node) {
	// VM-COMPILER-SEMANTICS-4: structured detection replaces strings.Contains.
	if isInputDeclaration(n, c) || isExternDeclaration(n, c) {
		c.collectParam(ir, n)
		return
	}
	// Skip function declarations
	if childByType(n, "function_declarator") != nil {
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
			if child.Type() == "init_declarator" {
				if valExpr := c.findInitValue(child, name); valExpr != nil {
					pd.Default = c.compileExpr(valExpr)
				}
			} else if init := childByType(decl, "init_declarator"); init != nil {
				if valExpr := c.findInitValue(init, name); valExpr != nil {
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
			// Check for array declaration: init_declarator may contain subscript_expression
			if arrSize, isArr := c.findArraySize(child); isArr {
				gv.IsArray = true
				gv.ArraySize = arrSize
			}
			if valExpr := c.findInitValue(child, name); valExpr != nil {
				gv.InitVal = c.compileExpr(valExpr)
			}
			ir.Globals = append(ir.Globals, gv)
		} else if child.Type() == "declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			gv := interp.GlobalVar{
				Name: name,
				Type: typeName,
			}
			if arrSize, isArr := c.findArraySize(child); isArr {
				gv.IsArray = true
				gv.ArraySize = arrSize
			}
			ir.Globals = append(ir.Globals, gv)
		} else if child.Type() == nodeIdentifier && typeName != "" {
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

func (c *compiler) compileFor(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtFor}
	lastIdx := int(n.NamedChildCount()) - 1
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		// VM-COMPILER-SEMANTICS-1: the body is always the last named child.
		// Handle it as body before the switch, so expression_statement bodies
		// don't get consumed as init/cond.
		if i == lastIdx && stmt.Body == nil {
			if child.Type() == nodeCompoundStatement {
				stmt.Body = c.compileBlock(child)
				continue
			}
			if s := c.compileStmt(child); s != nil {
				stmt.Body = []interp.Statement{*s}
			}
			continue
		}
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
		case "binary_expression", "call_expression", nodeIdentifier, "number_literal":
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
		case nodeCompoundStatement:
			stmt.Body = c.compileBlock(child)
		}
	}
	return stmt
}

func (c *compiler) compileWhile(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtWhile}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeParenExpr {
			stmt.Cond = c.compileExpr(child)
		} else if child.Type() == nodeCompoundStatement {
			stmt.Body = c.compileBlock(child)
		} else {
			// VM-COMPILER-SEMANTICS-1: single-statement body (no braces).
			if s := c.compileStmt(child); s != nil {
				if stmt.Body == nil {
					stmt.Body = []interp.Statement{*s}
				}
			}
		}
	}
	return stmt
}

func (c *compiler) compileSwitch(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtSwitch}
	// VM-COMPILER-SEMANTICS-1: tree-sitter wraps case_statement children inside
	// a compound_statement. Also, case values are direct expressions (number_literal,
	// identifier, etc.), not case_label nodes. Default is a case_statement with no
	// value expression child.
	var caseNodes []*sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeParenExpr {
			stmt.Expr = c.compileExpr(child)
		} else if child.Type() == "case_statement" {
			caseNodes = append(caseNodes, child)
		} else if child.Type() == nodeCompoundStatement {
			// Cases are inside the compound_statement body.
			for j := 0; j < int(child.NamedChildCount()); j++ {
				cc := child.NamedChild(j)
				if cc.Type() == "case_statement" {
					caseNodes = append(caseNodes, cc)
				}
			}
		}
	}
	for _, child := range caseNodes {
		sc := interp.SwitchCase{}
		for j := 0; j < int(child.NamedChildCount()); j++ {
			cc := child.NamedChild(j)
			switch cc.Type() {
			case "break_statement":
				// VM-COMPILER-SEMANTICS-1: track break for fallthrough detection.
				sc.HasBreak = true
			case nodeCompoundStatement:
				sc.Body = c.compileBlock(cc)
			case "expression_statement", "declaration", "if_statement", "for_statement",
				"while_statement", "do_statement", "switch_statement", "return_statement",
				"continue_statement":
				if s := c.compileStmt(cc); s != nil {
					sc.Body = append(sc.Body, *s)
				}
			default:
				// The first non-statement child is the case value expression
				// (number_literal, identifier, binary_expression, etc.).
				// If it's a "default" keyword, this is the default case.
				if sc.Expr == nil && !isStmtType(cc.Type()) {
					txt := c.text(cc)
					if txt == "default" {
						sc.Expr = nil
					} else {
						sc.Expr = c.compileExpr(cc)
					}
				}
			}
		}
		stmt.Cases = append(stmt.Cases, sc)
	}
	return stmt
}

// isStmtType returns true if the node type is a statement type (not a case value).
func isStmtType(t string) bool {
	switch t {
	case "expression_statement", "declaration", "if_statement", "for_statement",
		"while_statement", "do_statement", "switch_statement", "return_statement",
		"break_statement", "continue_statement", nodeCompoundStatement:
		return true
	}
	return false
}

func (c *compiler) compileDeclaration(n *sitter.Node) *interp.Statement {
	// VM-COMPILER-SEMANTICS-1: handle all declarators (multi-variable + no-init).
	typeName := c.findType(n)
	var decls []interp.Expr
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "init_declarator":
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			valExpr := c.findInitValue(child, name)
			var expr interp.Expr
			if valExpr != nil {
				expr = interp.Expr{
					Kind: interp.ExprDecl,
					Name: name,
					Args: []interp.Expr{*c.compileExpr(valExpr)},
				}
			} else {
				// No initializer — zero value.
				expr = interp.Expr{
					Kind: interp.ExprDecl,
					Name: name,
					Args: []interp.Expr{zeroValueExpr(typeName)},
				}
			}
			decls = append(decls, expr)
		case "declarator":
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			decls = append(decls, interp.Expr{
				Kind: interp.ExprDecl,
				Name: name,
				Args: []interp.Expr{zeroValueExpr(typeName)},
			})
		case "array_declarator":
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			// Local arrays not supported — compile error.
			if c.err == nil {
				c.err = fmt.Errorf("local arrays not supported: %s", name)
			}
			return nil
		}
	}
	if len(decls) == 0 {
		return nil
	}
	if len(decls) == 1 {
		return &interp.Statement{Kind: interp.StmtExpr, Expr: &decls[0]}
	}
	// Multiple declarators — wrap in ExprSeq.
	return &interp.Statement{
		Kind: interp.StmtExpr,
		Expr: &interp.Expr{Kind: interp.ExprSeq, Args: decls},
	}
}

// zeroValueExpr returns a compile-time Expr representing the zero value for a type.
// VM-COMPILER-SEMANTICS-1: used by compileDeclaration for no-initializer declarators.
func zeroValueExpr(typeName string) interp.Expr {
	switch typeName {
	case "int", "long", "datetime", "bool":
		return interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
	case "string":
		return interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal("")}
	default:
		// double, float, unknown — decimal zero.
		return interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(decimal.Zero)}
	}
}

// collectFuncParams extracts parameter declarations from a function_definition node.
func (c *compiler) collectFuncParams(n *sitter.Node) []interp.ParamDecl {
	var params []interp.ParamDecl
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "function_declarator" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				fc := child.NamedChild(j)
				if fc.Type() == "parameter_list" {
					for k := 0; k < int(fc.NamedChildCount()); k++ {
						pd := fc.NamedChild(k)
						if pd.Type() == "parameter_declaration" {
							pName := c.findIdent(pd)
							pType := c.findType(pd)
							if pName != "" {
								params = append(params, interp.ParamDecl{
									Name: pName,
									Type: pType,
								})
							}
						}
					}
				}
			}
		}
	}
	return params
}

// collectEnum processes enum declarations and registers constants.
func (c *compiler) collectEnum(ir *interp.IR, n *sitter.Node) {
	var enumName string
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "type_identifier" {
			enumName = c.text(child)
			if ir.EnumTypes == nil {
				ir.EnumTypes = make(map[string]bool)
			}
			ir.EnumTypes[enumName] = true
		}
		if child.Type() == "enumerator_list" {
			c.processEnumeratorList(ir, child, enumName)
		}
	}
}

func (c *compiler) processEnumeratorList(ir *interp.IR, list *sitter.Node, enumName string) {
	counter := int32(0)
	for j := 0; j < int(list.NamedChildCount()); j++ {
		ec := list.NamedChild(j)
		if ec.Type() != "enumerator" {
			continue
		}
		name, val := c.parseEnumerator(ec)
		if name == "" {
			continue
		}
		if val != nil {
			counter = *val
		}
		ir.Enums[name] = counter
		if enumName != "" {
			ir.Enums[enumName+"::"+name] = counter
		}
		counter++
	}
}

func (c *compiler) parseEnumerator(ec *sitter.Node) (name string, val *int32) {
	for k := 0; k < int(ec.NamedChildCount()); k++ {
		ecChild := ec.NamedChild(k)
		if ecChild.Type() == nodeIdentifier {
			name = c.text(ecChild)
		} else if ecChild.Type() == "number_literal" {
			nVal := interp.ParseNumberLiteral(c.text(ecChild))
			v := nVal.ToInt()
			val = &v
		}
	}
	return
}

// compileDoWhile compiles a do { } while(cond) statement.
func (c *compiler) compileDoWhile(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtDoWhile}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeCompoundStatement {
			stmt.Body = c.compileBlock(child)
		} else if child.Type() == nodeParenExpr {
			stmt.Cond = c.compileExpr(child)
		} else {
			// VM-COMPILER-SEMANTICS-1: single-statement body (no braces).
			if s := c.compileStmt(child); s != nil {
				if stmt.Body == nil {
					stmt.Body = []interp.Statement{*s}
				}
			}
		}
	}
	return stmt
}

// HasError recursively checks if a tree-sitter node or any of its named
// descendants is an ERROR node (parse error). VM-COMPILER-SEMANTICS-4:
// tree-sitter's MQL grammar produces ERROR nodes for `input` declarations
// (which are valid), so HasError is NOT used as a blanket guard. It is
// retained for diagnostic purposes and T8 (proves tree-sitter can produce
// ERROR nodes, just not at root level).
func HasError(n *sitter.Node) bool {
	if n == nil {
		return false
	}
	if n.Type() == "ERROR" {
		return true
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		if HasError(n.NamedChild(i)) {
			return true
		}
	}
	return false
}

// hasMissingNode checks if a tree-sitter node or any of its descendants
// is a MISSING node (tree-sitter inserted it to recover from a syntax error).
// VM-COMPILER-SEMANTICS-4: this catches `int x = ;` (missing initializer)
// and `input int X = ;` (missing initializer) without false-positiving on
// valid `input int X = 5;` (which has ERROR but no MISSING children).
func hasMissingNode(n *sitter.Node) bool {
	if n == nil {
		return false
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.IsMissing() {
			return true
		}
		if hasMissingNode(c) {
			return true
		}
	}
	return false
}

// hasMissingInitializer checks if a declaration node has an init_declarator
// with a MISSING child (indicating a missing initializer expression, e.g.
// `int x = ;`). VM-COMPILER-SEMANTICS-4: this is more precise than
// hasMissingNode — Python source parsed as MQL produces declarations with
// a missing `;` (not a missing initializer), and the MQL compiler must stay
// lenient on non-MQL input.
func hasMissingInitializer(n *sitter.Node) bool {
	if n == nil {
		return false
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() == "init_declarator" && hasMissingNode(c) {
			return true
		}
	}
	return false
}

// isInputDeclaration checks if a declaration node starts with the `input`
// keyword (MQL input parameter). VM-COMPILER-SEMANTICS-4: replaces
// strings.Contains(txt, "input ") which false-matched `int x = input ;`.
func isInputDeclaration(n *sitter.Node, c *compiler) bool {
	if n == nil || n.NamedChildCount() == 0 {
		return false
	}
	first := n.NamedChild(0)
	return first.Type() == "type_identifier" && c.text(first) == "input"
}

// isExternDeclaration checks if a declaration node starts with `extern`.
func isExternDeclaration(n *sitter.Node, c *compiler) bool {
	if n == nil || n.NamedChildCount() == 0 {
		return false
	}
	first := n.NamedChild(0)
	return first.Type() == "storage_class_specifier" && c.text(first) == "extern"
}

// isValidInputDeclaration checks that an input/extern declaration has a
// non-empty initializer (catches `input int X = ;`). VM-COMPILER-SEMANTICS-4.
func isValidInputDeclaration(n *sitter.Node, c *compiler) bool {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "init_declarator" {
			// The last named child of init_declarator is the initializer
			// expression. If it's missing or ERROR, the declaration is invalid.
			last := child.NamedChild(int(child.NamedChildCount()) - 1)
			if last == nil || last.Type() == "ERROR" {
				return false
			}
		}
	}
	return true
}

// checkReservedKeywordUsage rejects `input`/`extern` used as an identifier
// (e.g. `int x = input ;`). VM-COMPILER-SEMANTICS-4.
func checkReservedKeywordUsage(n *sitter.Node, c *compiler) error {
	return checkReservedInSubtree(n, c)
}

func checkReservedInSubtree(n *sitter.Node, c *compiler) error {
	if n == nil {
		return nil
	}
	if n.Type() == "identifier" {
		txt := c.text(n)
		if txt == "input" || txt == "extern" {
			return fmt.Errorf("reserved keyword %q used as identifier at line %d", txt, n.StartPoint().Row+1)
		}
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		if err := checkReservedInSubtree(n.NamedChild(i), c); err != nil {
			return err
		}
	}
	return nil
}
