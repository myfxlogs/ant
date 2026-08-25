package mql2go

import (
	"fmt"
	"strings"

	"alphaforge/tools/mql2go/interp"

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

	// VM-COMPILER-SEMANTICS-4 round 4: reject source with syntax errors.
	// tree-sitter performs error recovery and always returns a "translation_unit"
	// root, but internal ERROR/MISSING nodes indicate syntax errors.
	// Without this check, invalid declarations like "int x = ;" or "int x = 1 + ;"
	// are silently accepted (the compiler skips ERROR nodes in declarations).
	//
	// We check each top-level named child for HasError(), but allow known
	// false-positive patterns that the compiler handles via special fallbacks:
	//   - "input"/"extern" declarations (tree-sitter MQL grammar doesn't fully
	//     parse the input keyword; compiler falls back to collectParam)
	// This avoids false rejection of valid MQL5 strategies with #include stubs.
	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		if !n.HasError() {
			continue
		}
		txt := source[n.StartByte():n.EndByte()]
		// VM-COMPILER-SEMANTICS-4 round 5: structured input/extern exception.
		// Instead of strings.Contains (which lets "int x = input ;" through),
		// we check the node structure:
		//   - "extern" declaration: first named child is storage_class_specifier
		//     with text "extern". Valid extern declarations have hasError=false
		//     (tree-sitter parses them correctly). If hasError=true → reject.
		//   - "input" declaration: first named child is type_identifier with
		//     text "input" (tree-sitter mis-parses "input" as a type). The
		//     init_declarator child has hasError=true due to this structural
		//     mis-parse, but a valid input declaration has a non-empty
		//     initializer value. We check the init_declarator's last named
		//     child is not an empty ERROR/identifier (which indicates a
		//     missing initializer).
		if isInputDeclaration(n, source) {
			if isValidInputDeclaration(n, source) {
				continue
			}
			return nil, fmt.Errorf("parse MQL: syntax error in input declaration at node %q: %s", n.Type(), truncate(txt, 80))
		}
		if isExternDeclaration(n, source) {
			// Valid extern declarations have hasError=false. If hasError=true,
			// it's a real syntax error (e.g. "extern int X = ;").
			return nil, fmt.Errorf("parse MQL: syntax error in extern declaration at node %q: %s", n.Type(), truncate(txt, 80))
		}
		return nil, fmt.Errorf("parse MQL: syntax error in source at node %q: %s", n.Type(), truncate(txt, 80))
	}

	version := detectMQLVersion(source)
	c := &compiler{source: source, version: version}
	ir = c.compile(root)
	if c.err != nil {
		return nil, c.err
	}
	return ir, nil
}

type compiler struct {
	source  string
	version string
	err     error
}

func (c *compiler) compile(root *sitter.Node) *interp.IR {
	ir := &interp.IR{
		Version:    c.version,
		Funcs:      make(map[string]*interp.FuncDef),
		Enums:      make(map[string]int32),
		EnumTypes:  make(map[string]bool),
		ClassTypes: make(map[string]bool),
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
				ir.ClassTypes[name] = true
			}
		case "enum_specifier":
			c.collectEnum(ir, n)
		}
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		txt := c.text(n)
		switch n.Type() {
		case "declaration":
			c.collectGlobal(ir, n)
			c.collectClassInstance(ir, n, knownClasses)
		case "class_specifier", "struct_specifier":
			c.collectClassDecl(ir, n)
		case "enum_specifier":
			// already processed in first pass
		case "function_definition":
			c.collectFunction(ir, n)
		default:
			// Fallback: handle nodes that tree-sitter doesn't parse as 'declaration'
			// (e.g. 'input BuyOrSell0 x = 2;' with enum type may parse as ERROR)
			if strings.Contains(txt, "input ") || strings.Contains(txt, "extern ") {
				c.collectParam(ir, n)
			} else if n.Type() != "preproc_def" && n.Type() != "preproc_function_def" &&
				n.Type() != "preproc_include" && n.Type() != "preproc_ifdef" &&
				n.Type() != "preproc_call" && n.Type() != "comment" &&
				n.Type() != "expression_statement" && n.Type() != "linkage_specification" {
				// VM-COMPILER-SEMANTICS-2/4: unknown root nodes must not be silently
				// skipped — report a compile error so malformed source is caught.
				// Note: expression_statement at top level is allowed (e.g. empty
				// statement ";" or CTrade instance declarations that tree-sitter
				// parses as expression_statement) — these have no event handler
				// to execute in, so they are silently ignored.
				if c.err == nil {
					c.err = fmt.Errorf("compile: unrecognized top-level node %q: %s", n.Type(), truncate(txt, 80))
				}
			}
		}
	}

	return ir
}

// collectGlobal processes top-level declarations (globals + params).
func (c *compiler) collectGlobal(ir *interp.IR, n *sitter.Node) {
	// VM-COMPILER-SEMANTICS-4 round 5: use structured detection instead of
	// strings.Contains (which matched "int x = input ;" as an input declaration).
	if isInputDeclaration(n, c.source) || isExternDeclaration(n, c.source) {
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
	// VM-COMPILER-SEMANTICS-4 round 5: check that no identifier in this
	// declaration uses a reserved keyword (input/extern) as a variable name
	// or initializer value. tree-sitter accepts "int x = input ;" because
	// it treats "input" as an identifier, but "input" is a reserved MQL5
	// keyword and must not be used as a value.
	if c.err == nil {
		if err := checkReservedKeywordUsage(n, c.source); err != nil {
			c.err = err
			return
		}
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "init_declarator" || child.Type() == "declarator" || child.Type() == "array_declarator" {
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
			if !gv.IsArray {
				if valExpr := c.findInitValue(child, name); valExpr != nil {
					gv.InitVal = c.compileExpr(valExpr)
				}
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
