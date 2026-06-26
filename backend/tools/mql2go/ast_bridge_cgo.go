//go:build cgo

package mql2go

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// ParseMQL parses MQL source code into a SourceFile AST using tree-sitter via cgo.
func ParseMQL(source string) (*SourceFile, error) {
	lang, err := Language()
	if err != nil {
		return nil, fmt.Errorf("load MQL grammar: %w", err)
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil || root.IsError() {
		return nil, fmt.Errorf("parse error")
	}

	ast := &SourceFile{}
	n := int(root.ChildCount())
	for i := 0; i < n; i++ {
		decl := translateCST(source, root.Child(i))
		if decl != nil {
			ast.Declarations = append(ast.Declarations, decl)
		}
	}
	return ast, nil
}

// translateCST converts a tree-sitter CST node to our AST.
func translateCST(source string, node *sitter.Node) Node {
	if node == nil { return nil }
	switch node.Type() {
	case "function_definition": return translateFunc(source, node)
	case "declaration", "field_declaration": return translateVar(source, node)
	case "compound_statement": return translateBlock(source, node)
	case "expression_statement": return translateExpr(source, node)
	case "if_statement": return translateIf(source, node)
	case "for_statement": return translateFor(source, node)
	case "return_statement": return translateRet(source, node)
	case "call_expression": return translateCall(source, node)
	case "binary_expression": return translateBin(source, node)
	case "unary_expression", "not_expression": return translateUn(source, node)
	case "identifier", "field_identifier", "statement_identifier": return &Identifier{Name: cstText(source, node)}
	case "number_literal": return &NumberLiteral{Value: cstText(source, node)}
	case "string_literal", "system_lib_string":
		v := cstText(source, node)
		if len(v) >= 2 { v = v[1:len(v)-1] }
		return &StringLiteral{Value: v}
	case "subscript_expression": return translateSub(source, node)
	case "assignment_expression": return translateAssign(source, node)
	case "parenthesized_expression":
		for i := 0; i < int(node.ChildCount()); i++ {
			if n := translateCST(source, node.Child(i)); n != nil { return n }
		}
	}
	return nil
}

func translateFunc(source string, n *sitter.Node) *FuncDef {
	fn := &FuncDef{ReturnType: "void"}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "identifier", "field_identifier", "statement_identifier": fn.Name = cstText(source, c)
		case "function_declarator":
			for j := 0; j < int(c.ChildCount()); j++ {
				gc := c.Child(j)
				switch gc.Type() {
				case "identifier", "field_identifier", "statement_identifier": fn.Name = cstText(source, gc)
				case "parameter_list": fn.Params = translateParamsCST(source, gc)
				}
			}
		case "parameter_list": fn.Params = translateParamsCST(source, c)
		case "compound_statement": fn.Body = translateBlock(source, c)
		}
	}
	return fn
}

func translateVar(source string, n *sitter.Node) *VarDecl {
	vd := &VarDecl{VarType: "int"}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "storage_class_specifier":
			if cstText(source, c) == "extern" { vd.IsExtern = true }
			if cstText(source, c) == "input" { vd.IsInput = true }
		case "primitive_type", "type_identifier", "sized_type_specifier", "macro_type_specifier": vd.VarType = cstText(source, c)
		case "identifier", "field_identifier", "statement_identifier": vd.Name = cstText(source, c)
		case "number_literal": vd.Value = &NumberLiteral{Value: cstText(source, c)}
		case "string_literal", "system_lib_string":
			v := cstText(source, c)
			if len(v) >= 2 { v = v[1:len(v)-1] }
			vd.Value = &StringLiteral{Value: v}
		case "call_expression": vd.Value = translateCall(source, c)
		case "binary_expression": vd.Value = translateBin(source, c)
		}
	}
	return vd
}

func translateBlock(source string, n *sitter.Node) *CompoundStmt {
	cs := &CompoundStmt{}
	for i := 0; i < int(n.ChildCount()); i++ {
		if x := translateCST(source, n.Child(i)); x != nil { cs.Statements = append(cs.Statements, x) }
	}
	return cs
}

func translateExpr(source string, n *sitter.Node) *ExpressionStmt {
	for i := 0; i < int(n.ChildCount()); i++ {
		if x := translateCST(source, n.Child(i)); x != nil { return &ExpressionStmt{Expr: x} }
	}
	return &ExpressionStmt{}
}

func translateIf(source string, n *sitter.Node) *IfStmt {
	s := &IfStmt{}; var inElse bool
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "condition_clause":
			for j := 0; j < int(c.ChildCount()); j++ {
				gc := c.Child(j)
				switch gc.Type() {
				case "parenthesized_expression":
					for k := 0; k < int(gc.ChildCount()); k++ {
						if x := translateCST(source, gc.Child(k)); x != nil { s.Condition = x }
					}
				case "compound_statement": s.ThenBranch = translateBlock(source, gc)
				case "return_statement": s.ThenBranch = &CompoundStmt{Statements: []Node{translateRet(source, gc)}}
				}
			}
		case "else_clause": inElse = true
		case "compound_statement":
			if inElse { s.ElseBranch = translateBlock(source, c) } else { s.ThenBranch = translateBlock(source, c) }
		case "if_statement":
			if inElse { s.ElseBranch = translateIf(source, c) }
		}
	}
	return s
}

func translateFor(source string, n *sitter.Node) *ForStmt {
	fs := &ForStmt{}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "compound_statement": fs.Body = translateBlock(source, c)
		case "for_header":
			for j := 0; j < int(c.ChildCount()); j++ {
				if x := translateCST(source, c.Child(j)); x != nil { fs.Body = translateBlock(source, c); break }
			}
		}
	}
	return fs
}

func translateRet(source string, n *sitter.Node) *ReturnStmt {
	rs := &ReturnStmt{}
	for i := 0; i < int(n.ChildCount()); i++ {
		if x := translateCST(source, n.Child(i)); x != nil { rs.Value = x }
	}
	return rs
}

func translateCall(source string, n *sitter.Node) *CallExpr {
	c := &CallExpr{}
	for i := 0; i < int(n.ChildCount()); i++ {
		ch := n.Child(i)
		switch ch.Type() {
		case "identifier", "field_identifier", "statement_identifier": c.Name = cstText(source, ch)
		case "argument_list":
			for j := 0; j < int(ch.ChildCount()); j++ {
				if x := translateCST(source, ch.Child(j)); x != nil { c.Args = append(c.Args, x) }
			}
		}
	}
	return c
}

func translateBin(source string, n *sitter.Node) *BinaryOp {
	op := &BinaryOp{}
	for i := 0; i < int(n.ChildCount()); i++ {
		ch := n.Child(i)
		if x := translateCST(source, ch); x != nil {
			if op.Left == nil { op.Left = x } else { op.Right = x }
		} else {
			t, v := ch.Type(), cstText(source, ch)
			if v == "+" || v == "-" || v == "*" || v == "/" || v == "%" ||
				v == "==" || v == "!=" || v == "<" || v == ">" || v == "<=" || v == ">=" ||
				v == "&&" || v == "||" { op.Op = v }
			_ = t
		}
	}
	return op
}

func translateUn(source string, n *sitter.Node) *UnaryOp {
	op := &UnaryOp{}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "!": op.Op = "!"
		case "-": op.Op = "-"
		default:
			if x := translateCST(source, c); x != nil { op.Operand = x }
		}
	}
	return op
}

func translateSub(source string, n *sitter.Node) *SubscriptExpr {
	ss := &SubscriptExpr{}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "identifier", "field_identifier": ss.Name = cstText(source, c)
		default:
			if x := translateCST(source, c); x != nil && ss.Index == nil { ss.Index = x }
		}
	}
	return ss
}

func translateAssign(source string, n *sitter.Node) *AssignmentExpr {
	ae := &AssignmentExpr{}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch c.Type() {
		case "identifier", "field_identifier": ae.LHS = cstText(source, c)
		default:
			if x := translateCST(source, c); x != nil && ae.RHS == nil { ae.RHS = x }
		}
	}
	return ae
}

func translateParamsCST(source string, n *sitter.Node) []string {
	var p []string
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == "parameter_declaration" || c.Type() == "optional_parameter_declaration" {
			for j := 0; j < int(c.ChildCount()); j++ {
				gc := c.Child(j)
				if gc.Type() == "identifier" || gc.Type() == "field_identifier" { p = append(p, cstText(source, gc)) }
			}
		}
	}
	return p
}

func cstText(source string, n *sitter.Node) string { return source[n.StartByte():n.EndByte()] }
