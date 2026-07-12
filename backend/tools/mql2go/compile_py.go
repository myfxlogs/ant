package mql2go

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"alphaforge/tools/mql2go/interp"
)

// CompilePythonToIR parses Python subset source and compiles it to interp.IR.
// Safety: enforces MaxSourceSize + subset validation + panic recovery.
func CompilePythonToIR(source string) (ir *interp.IR, err error) {
	if len(source) > MaxSourceSize {
		return nil, fmt.Errorf("Python source too large: %d bytes (max %d)", len(source), MaxSourceSize)
	}
	defer func() {
		if r := recover(); r != nil {
			ir = nil
			err = fmt.Errorf("compile Python panic: %v", r)
		}
	}()

	if verr := validatePythonSubset(source); verr != nil {
		return nil, verr
	}

	root, err := ParsePython(source)
	if err != nil {
		return nil, fmt.Errorf("parse Python: %w", err)
	}

	if root.HasError() {
		return nil, fmt.Errorf("parse Python: syntax error in source (tree-sitter HasError=true)")
	}

	if verr := validatePythonCST(root, source); verr != nil {
		return nil, verr
	}

	c := &pyCompiler{source: source}
	ir = c.compile(root)
	if len(c.errors) > 0 {
		return nil, c.errors[0]
	}
	return ir, nil
}

type pyCompiler struct {
	source      string
	selfVars    map[string]bool
	posLoopVars map[string]bool
	errors      []error
}

// errorf appends a compilation error with line number from the tree-sitter node.
func (c *pyCompiler) errorf(n *sitter.Node, format string, args ...any) {
	line := 0
	if n != nil {
		line = int(n.StartPoint().Row) + 1
	}
	c.errors = append(c.errors, fmt.Errorf("line %d: "+format, append([]any{line}, args...)...))
}

func (c *pyCompiler) text(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return c.source[n.StartByte():n.EndByte()]
}

func (c *pyCompiler) compile(root *sitter.Node) *interp.IR {
	ir := &interp.IR{
		Version:   "python",
		Funcs:     make(map[string]*interp.FuncDef),
		Enums:     make(map[string]int32),
		EnumTypes: make(map[string]bool),
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		switch n.Type() {
		case "class_definition":
			c.compileClass(ir, n)
		case "function_definition":
			c.compileFunction(ir, n)
		case "import_statement", "import_from_statement":
			// Only "from decimal import Decimal" is allowed; already validated.
			// No IR output needed — Decimal is a known type.
		case "expression_statement":
			// Top-level expression (rare in subset, but handle assignments)
			c.compileTopLevelExpr(ir, n)
		}
	}

	// Collect self.field references as global variables
	for name := range c.selfVars {
		ir.Globals = append(ir.Globals, interp.GlobalVar{
			Name: name,
			Type: "auto",
		})
	}

	return ir
}

// compileClass handles `class MyStrategy(StrategyBase): ...`
func (c *pyCompiler) compileClass(ir *interp.IR, n *sitter.Node) {
	// Find class name
	name := c.findClassName(n)
	if name == "" {
		return
	}

	// Multiple inheritance is prohibited (ADR-0024 D3)
	if baseCount := c.countBases(n); baseCount > 1 {
		c.errorf(n, "multiple inheritance not supported (%d bases), extend StrategyBase only", baseCount)
		return
	}

	// Find class body (block)
	body := c.findBlock(n)
	if body == nil {
		return
	}

	// Compile methods inside the class
	for i := 0; i < int(body.NamedChildCount()); i++ {
		child := body.NamedChild(i)
		if child.Type() == "function_definition" {
			c.compileMethod(ir, child)
		}
	}
}

func (c *pyCompiler) compileMethod(ir *interp.IR, n *sitter.Node) {
	name := c.findFuncName(n)
	if name == "" {
		return
	}

	body := c.findBlock(n)
	if body == nil {
		return
	}
	stmts := c.compileBlock(body)

	// Map Python event methods to IR event slots
	switch name {
	case "__init__":
		// Extract __init__ params as strategy parameters (validation ensures they're typed)
		ir.Params = c.collectParams(n)
		// Reuse already-compiled stmts (compiled at line 121)
		ir.OnInit = append(stmts, ir.OnInit...)
	case "on_init":
		ir.OnInit = append(ir.OnInit, stmts...)
	case "on_bar":
		ir.OnBar = stmts
	case "on_tick":
		ir.OnTick = stmts
	case "on_timer":
		ir.OnTimer = stmts
	case "on_trade":
		ir.OnTrade = stmts
	case "on_trade_transaction":
		ir.OnTradeTransaction = stmts
	case "on_book_event":
		ir.OnBookEvent = stmts
	case "on_deinit":
		ir.OnDeinit = stmts
	default:
		// User-defined method
		params := c.collectParams(n)
		ir.Funcs[name] = &interp.FuncDef{
			Name:   name,
			Params: params,
			Body:   stmts,
		}
	}
}

func (c *pyCompiler) compileFunction(ir *interp.IR, n *sitter.Node) {
	name := c.findFuncName(n)
	if name == "" {
		return
	}
	body := c.findBlock(n)
	if body == nil {
		return
	}
	stmts := c.compileBlock(body)
	params := c.collectParams(n)
	ir.Funcs[name] = &interp.FuncDef{
		Name:   name,
		Params: params,
		Body:   stmts,
	}
}

func (c *pyCompiler) compileTopLevelExpr(ir *interp.IR, n *sitter.Node) {
	// Handle simple top-level assignments (rare in subset)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "assignment" {
			expr := c.compileExpr(child)
			if expr != nil {
				ir.Globals = append(ir.Globals, interp.GlobalVar{
					Name:    expr.Name,
					Type:    "auto",
					InitVal: &expr.Args[0],
				})
			}
		}
	}
}

// collectParams extracts function parameters from a function_definition node.
func (c *pyCompiler) collectParams(n *sitter.Node) []interp.ParamDecl {
	params := findNamedChild(n, "parameters")
	if params == nil {
		return nil
	}
	var result []interp.ParamDecl
	for i := 0; i < int(params.NamedChildCount()); i++ {
		p := params.NamedChild(i)
		if p.Type() != "identifier" && p.Type() != "default_parameter" && p.Type() != "typed_parameter" &&
			p.Type() != "typed_default_parameter" {
			continue
		}
		pd := interp.ParamDecl{}
		switch p.Type() {
		case "identifier":
			pd.Name = c.text(p)
		case "default_parameter":
			pd.Name = c.text(p.NamedChild(0)) // identifier
			if defVal := c.findDefaultVal(p); defVal != nil {
				pd.Default = c.compileExpr(defVal)
			}
		case "typed_parameter":
			pd.Name = c.findParamName(p)
			pd.Type = c.findParamType(p)
		case "typed_default_parameter":
			pd.Name = c.findParamName(p)
			pd.Type = c.findParamType(p)
			if defVal := c.findDefaultVal(p); defVal != nil {
				pd.Default = c.compileExpr(defVal)
			}
		}
		if pd.Name != "" && pd.Name != "self" {
			result = append(result, pd)
		}
	}
	return result
}

// findParamName extracts the parameter name from a typed_parameter or typed_default_parameter.
func (c *pyCompiler) findParamName(p *sitter.Node) string {
	for i := 0; i < int(p.NamedChildCount()); i++ {
		child := p.NamedChild(i)
		if child.Type() == "identifier" {
			return c.text(child)
		}
	}
	return ""
}

// findParamType extracts the type annotation from a typed_parameter or typed_default_parameter.
func (c *pyCompiler) findParamType(p *sitter.Node) string {
	for i := 0; i < int(p.NamedChildCount()); i++ {
		child := p.NamedChild(i)
		if child.Type() == "type" {
			return c.text(child)
		}
	}
	return ""
}

