package mql2go

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func (c *compiler) text(n *sitter.Node) string {
	return nodeText(c.source, n)
}

func (c *compiler) findIdent(n *sitter.Node) string {
	// First pass: a direct identifier/field_identifier child that is NOT a
	// primitive type. The MQL tree-sitter grammar can surface the type as the
	// first identifier child for float-literal defaults — e.g. "input double
	// Lots=0.1" parses with "double" as a direct identifier inside
	// init_declarator — so primitive type names must be skipped to reach the
	// real variable name. (No legal MQL identifier is a primitive type keyword.)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeIdentifier || child.Type() == "field_identifier" {
			name := c.text(child)
			if !isMQLPrimitiveType(name) {
				return name
			}
		}
	}
	// Second pass: the same grammar quirk buries the real identifier inside an
	// ERROR recovery node. Descend into ERROR nodes to recover it.
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "ERROR" {
			if name := c.findIdent(child); name != "" {
				return name
			}
		}
	}
	// Direct identifier (n itself).
	if n.Type() == nodeIdentifier || n.Type() == "field_identifier" {
		name := c.text(n)
		if !isMQLPrimitiveType(name) {
			return name
		}
	}
	return ""
}

// findArraySize detects array dimension in declarations like double Gd_720[30].
// Returns (size, true) if an array dimension is found, (0, false) otherwise.
func (c *compiler) findArraySize(n *sitter.Node) (int, bool) {
	if n == nil {
		return 0, false
	}
	if n.Type() == "array_declarator" {
		return c.parseArrayDimension(n)
	}
	if n.Type() != "init_declarator" && n.Type() != "declarator" {
		return 0, false
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "array_declarator" {
			return c.parseArrayDimension(child)
		}
	}
	return 0, false
}

func (c *compiler) parseArrayDimension(n *sitter.Node) (int, bool) {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() != nodeNumberLiteral {
			continue
		}
		var size int
		if _, err := fmt.Sscanf(c.text(child), "%d", &size); err == nil && size > 0 {
			return size, true
		}
	}
	return 0, true
}

func (c *compiler) findType(n *sitter.Node) string {
	// First check ERROR nodes for primitive types (MQL5 'input int' pattern)
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "ERROR" {
			txt := c.text(child)
			if isMQLPrimitiveType(txt) {
				return txt
			}
		}
	}
	if pt := childByType(n, "primitive_type"); pt != nil {
		return c.text(pt)
	}
	// Find type_identifier, skipping 'input'/'extern' keywords
	// (tree-sitter parses 'input BuyOrSell0 x' with 'input' as type_identifier)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeTypeIdentifier {
			name := c.text(child)
			if name != "input" && name != "extern" {
				return name
			}
		}
	}
	// Fallback for the float-default grammar quirk: "input double Lots=0.1"
	// parses with "double" as an identifier inside init_declarator (not as
	// primitive_type). Scan descendants for a primitive-type identifier so the
	// param's Type is populated (else injectParams skips it → volume=0).
	if t := c.findPrimitiveTypeIdent(n); t != "" {
		return t
	}
	return ""
}

// findPrimitiveTypeIdent scans descendants of n for an identifier whose text is a
// MQL primitive type (int/double/string/...). Needed for the float-default quirk
// where the type sits as an identifier inside init_declarator.
func (c *compiler) findPrimitiveTypeIdent(n *sitter.Node) string {
	if n.Type() == nodeIdentifier {
		if isMQLPrimitiveType(c.text(n)) {
			return c.text(n)
		}
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		if r := c.findPrimitiveTypeIdent(n.NamedChild(i)); r != "" {
			return r
		}
	}
	return ""
}

func (c *compiler) findExprChild(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case nodeNumberLiteral, "string_literal", nodeIdentifier,
			"call_expression", "binary_expression", "unary_expression",
			"subscript_expression", "conditional_expression",
			nodeParenExpr, "field_expression",
			"assignment_expression", nodeTrue, nodeFalse,
			"cast_expression", "comma_expression":
			return child
		}
	}
	return nil
}

func (c *compiler) findInitValue(n *sitter.Node, declName string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case nodeNumberLiteral, "string_literal",
			"call_expression", "binary_expression", "unary_expression",
			"subscript_expression", "conditional_expression",
			nodeParenExpr, "field_expression",
			"assignment_expression", nodeTrue, nodeFalse,
			"cast_expression", "comma_expression":
			return child
		case nodeIdentifier:
			// Skip the param name itself AND primitive-type identifiers.
			// Float-default quirk: "input double Lots=0.1" parses with "double"
			// as an identifier in init_declarator — it's the type, not the value.
			txt := c.text(child)
			if txt != declName && !isMQLPrimitiveType(txt) {
				return child
			}
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func isBinaryOp(t string) bool {
	switch t {
	case "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=",
		"&&", "||", "&", "|", "^", "<<", ">>":
		return true
	}
	return false
}

func isUnaryOp(s string) bool {
	switch s {
	case "-", "!", "~", "+":
		return true
	}
	return false
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
