package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── CST helpers (shared by compile_interp.go and compile_interp_expr.go) ──

func nodeText(source string, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return source[n.StartByte():n.EndByte()]
}

func childByType(n *sitter.Node, kind string) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == kind {
			return c
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

func callFuncName(source string, n *sitter.Node) string {
	if id := childByType(n, "identifier"); id != nil {
		return nodeText(source, id)
	}
	if fe := childByType(n, "field_expression"); fe != nil {
		if id := childByType(fe, "field_identifier"); id != nil {
			return nodeText(source, id)
		}
		if id := childByType(fe, "identifier"); id != nil {
			return nodeText(source, id)
		}
	}
	if id := childByType(n, "field_identifier"); id != nil {
		return nodeText(source, id)
	}
	if id := childByType(n, "statement_identifier"); id != nil {
		return nodeText(source, id)
	}
	return ""
}

// ── Function extraction ─────────────────────────────────────────────

func funcName(source string, n *sitter.Node) string {
	decl := childByType(n, "function_declarator")
	if decl != nil {
		id := childByType(decl, "identifier")
		if id == nil {
			id = childByType(decl, "field_identifier")
		}
		if id == nil {
			id = childByType(decl, "statement_identifier")
		}
		return nodeText(source, id)
	}
	id := childByType(n, "identifier")
	if id == nil {
		id = childByType(n, "field_identifier")
	}
	if id == nil {
		id = childByType(n, "statement_identifier")
	}
	return nodeText(source, id)
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

func isMQLPrimitiveType(t string) bool {
	switch t {
	case "int", "long", "uint", "ulong", "double", "float", "string", "bool", "char", "short", "uchar", "ushort":
		return true
	}
	return false
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
