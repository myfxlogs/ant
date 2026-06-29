package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── CST helpers (shared by compile_interp.go and compile_interp_expr.go) ──

func nodeType(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Type()
}

func nodeText(source string, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	if source == "" {
		source = parseSource
	}
	return source[n.StartByte():n.EndByte()]
}

func childByType(source string, n *sitter.Node, kind string) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == kind {
			return c
		}
	}
	return nil
}

func childrenByType(n *sitter.Node, kind string) []*sitter.Node {
	var out []*sitter.Node
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == kind {
			out = append(out, c)
		}
	}
	return out
}

func findChild(source string, n *sitter.Node, kinds ...string) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		for _, k := range kinds {
			if c.Type() == k {
				return c
			}
		}
	}
	return nil
}

func findNamedChild(n *sitter.Node, kinds ...string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		for _, k := range kinds {
			if c.Type() == k {
				return c
			}
		}
	}
	return nil
}

func walkCST(n *sitter.Node, visitor func(*sitter.Node) bool) {
	if n == nil {
		return
	}
	if !visitor(n) {
		return
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		walkCST(n.Child(i), visitor)
	}
}

func findCall(n *sitter.Node, name string) *sitter.Node {
	var found *sitter.Node
	walkCST(n, func(n *sitter.Node) bool {
		if found != nil {
			return false
		}
		if n.Type() == "call_expression" {
			if id := childByType("", n, "identifier"); id != nil && nodeText("", id) == name {
				found = n
				return false
			}
			if id := childByType("", n, "field_identifier"); id != nil && nodeText("", id) == name {
				found = n
				return false
			}
		}
		return true
	})
	return found
}

func callFuncName(n *sitter.Node) string {
	if id := childByType("", n, "identifier"); id != nil {
		return nodeText("", id)
	}
	if fe := childByType("", n, "field_expression"); fe != nil {
		if id := childByType("", fe, "field_identifier"); id != nil {
			return nodeText("", id)
		}
		if id := childByType("", fe, "identifier"); id != nil {
			return nodeText("", id)
		}
	}
	if id := childByType("", n, "field_identifier"); id != nil {
		return nodeText("", id)
	}
	if id := childByType("", n, "statement_identifier"); id != nil {
		return nodeText("", id)
	}
	return ""
}

func callArg(n *sitter.Node, idx int) string {
	args := childByType("", n, "argument_list")
	if args == nil {
		return ""
	}
	named := getNamedChildren(args)
	if idx < len(named) {
		return nodeText("", named[idx])
	}
	return ""
}

func callArgID(n *sitter.Node, idx int) string {
	args := childByType("", n, "argument_list")
	if args == nil {
		return ""
	}
	named := getNamedChildren(args)
	if idx < len(named) {
		c := named[idx]
		if id := childByType("", c, "identifier"); id != nil {
			return nodeText("", id)
		}
		if id := childByType("", c, "field_identifier"); id != nil {
			return nodeText("", id)
		}
		return nodeText("", c)
	}
	return ""
}

// ── Function extraction ─────────────────────────────────────────────

func findFunctions(root *sitter.Node) []*sitter.Node {
	var fns []*sitter.Node
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() == "function_definition" {
			fns = append(fns, n)
		}
		return true
	})
	return fns
}

func funcName(n *sitter.Node) string {
	decl := childByType("", n, "function_declarator")
	if decl != nil {
		id := childByType("", decl, "identifier")
		if id == nil {
			id = childByType("", decl, "field_identifier")
		}
		if id == nil {
			id = childByType("", decl, "statement_identifier")
		}
		return nodeText("", id)
	}
	id := childByType("", n, "identifier")
	if id == nil {
		id = childByType("", n, "field_identifier")
	}
	if id == nil {
		id = childByType("", n, "statement_identifier")
	}
	return nodeText("", id)
}

func funcBody(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == "compound_statement" {
			return c
		}
	}
	return nil
}

// ── Utility functions ───────────────────────────────────────────────

func getNamedChildren(n *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	if n == nil {
		return out
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		out = append(out, n.NamedChild(i))
	}
	return out
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func isMQLPrimitiveType(t string) bool {
	switch t {
	case "int", "long", "uint", "ulong", "double", "float", "string", "bool", "char", "short", "uchar", "ushort":
		return true
	}
	return false
}

// nonEmpty filters out empty strings.
func nonEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// replaceWord replaces whole-word occurrences of old with new in s.
func replaceWord(s, old, new string) string {
	if s == "" || old == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(new))
	i := 0
	for i < len(s) {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			if isWordBoundary(s, i-1) && isWordBoundary(s, i+len(old)) {
				b.WriteString(new)
				i += len(old)
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func isWordBoundary(s string, pos int) bool {
	if pos < 0 || pos >= len(s) {
		return true
	}
	c := s[pos]
	return !isWordChar(c)
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
