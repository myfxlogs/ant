package mql2go

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file contains structured input/extern declaration detection helpers.
// VM-COMPILER-SEMANTICS-4 round 5: replaces strings.Contains-based detection
// with structural tree-sitter node-type checks.

// isInputDeclaration checks if a top-level node is an "input" declaration.
// VM-COMPILER-SEMANTICS-4 round 5: tree-sitter mis-parses "input" as a
// type_identifier (the first named child). This is a structural check,
// not a strings.Contains check.
func isInputDeclaration(n *sitter.Node, source string) bool {
	if n.Type() != "declaration" {
		return false
	}
	if n.NamedChildCount() == 0 {
		return false
	}
	first := n.NamedChild(0)
	if first.Type() != nodeTypeIdentifier {
		return false
	}
	return source[first.StartByte():first.EndByte()] == "input"
}

// isValidInputDeclaration checks if an input declaration has a valid
// initializer. VM-COMPILER-SEMANTICS-4 round 5: tree-sitter mis-parses
// "input" declarations in two different ways depending on variable name
// length, but in both cases the init_declarator's last named child is
// the initializer value. A valid input declaration has a non-empty
// initializer value; an invalid one (e.g. "input int X = ;") has an
// empty identifier as the last named child.
//
// Pattern A (short name): declaration[type_identifier "input", init_declarator "int X = 5"]
//
//	init_declarator has named children: identifier "int", ERROR "X", number_literal "5"
//
// Pattern B (long name): declaration[type_identifier "input", ERROR "int", init_declarator "MagicNumber = 12345"]
//
//	init_declarator has named children: identifier "MagicNumber", number_literal "12345"
//
// In both valid patterns, the init_declarator's last named child is a
// non-empty value node. In invalid patterns (missing initializer), the
// last named child is an empty identifier.
func isValidInputDeclaration(n *sitter.Node, source string) bool {
	// Find the init_declarator child (may be at index 1 or 2 depending
	// on whether tree-sitter emitted an ERROR node for the type).
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() != "init_declarator" {
			continue
		}
		if c.NamedChildCount() == 0 {
			return false // no initializer at all
		}
		last := c.NamedChild(int(c.NamedChildCount()) - 1)
		lastText := source[last.StartByte():last.EndByte()]
		// If the last named child is an empty identifier, the initializer
		// is missing (e.g. "input int X = ;" → init_declarator "X ="
		// with last named child identifier "").
		if lastText == "" {
			return false
		}
		// Non-empty last named child = valid initializer value.
		return true
	}
	// No init_declarator — "input int X;" (no default value) is valid in MQL5.
	return true
}

// isExternDeclaration checks if a top-level node is an "extern" declaration.
// VM-COMPILER-SEMANTICS-4 round 5: tree-sitter correctly parses "extern"
// as a storage_class_specifier. Valid extern declarations have hasError=false.
// This function is only called when n.HasError() is true, so if it matches,
// it's a real syntax error (e.g. "extern int X = ;").
func isExternDeclaration(n *sitter.Node, source string) bool {
	if n.Type() != "declaration" {
		return false
	}
	if n.NamedChildCount() == 0 {
		return false
	}
	first := n.NamedChild(0)
	if first.Type() != "storage_class_specifier" {
		return false
	}
	return source[first.StartByte():first.EndByte()] == "extern"
}

// mqlReservedKeywords are MQL5 keywords that must not be used as identifiers
// (variable names, function names, or initializer values). tree-sitter's MQL
// grammar may accept them as identifiers, but they are reserved in the
// language spec. VM-COMPILER-SEMANTICS-4 round 5.
var mqlReservedKeywords = map[string]bool{
	"input":  true,
	"extern": true,
}

// checkReservedKeywordUsage walks a declaration node and rejects any
// identifier whose text is a reserved MQL keyword (input/extern).
// VM-COMPILER-SEMANTICS-4 round 5: "int x = input ;" passes tree-sitter
// parsing (hasError=false) because tree-sitter treats "input" as an
// identifier, but "input" is a reserved keyword and must not be used
// as a value. This check catches that case.
func checkReservedKeywordUsage(n *sitter.Node, source string) error {
	var found error
	walkNode(n, func(node *sitter.Node) {
		if found != nil {
			return
		}
		if node.Type() == "identifier" {
			text := source[node.StartByte():node.EndByte()]
			if mqlReservedKeywords[text] {
				found = fmt.Errorf("compile: reserved keyword %q used as identifier at declaration: %s", text, truncate(source[n.StartByte():n.EndByte()], 80))
			}
		}
	})
	return found
}

// walkNode recursively walks a tree-sitter node and calls fn for each node.
func walkNode(n *sitter.Node, fn func(*sitter.Node)) {
	fn(n)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		walkNode(n.NamedChild(i), fn)
	}
}
