package mql2go

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// allowedDunders are dunder names permitted in the Python subset.
var allowedDunders = map[string]bool{
	"__init__": true,
}

// validatePythonSubset checks that the Python source conforms to the allowed subset.
// It performs two passes:
//  1. Text-based pre-parse check for imports, forbidden builtins, walrus operator
//  2. CST-based post-parse check for forbidden node types and dunder access
func validatePythonSubset(source string) error {
	lines := strings.Split(source, "\n")

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "import ") {
			return fmt.Errorf("line %d: 'import x' not allowed, only 'from decimal import Decimal' is permitted", lineNum+1)
		}
		if strings.HasPrefix(trimmed, "from ") {
			if trimmed != "from decimal import Decimal" {
				return fmt.Errorf("line %d: only 'from decimal import Decimal' is allowed, got: %s", lineNum+1, trimmed)
			}
			continue
		}

		for _, forbidden := range forbiddenBuiltins {
			if containsWordBuiltin(trimmed, forbidden) {
				return fmt.Errorf("line %d: use of '%s' is not allowed in Python subset", lineNum+1, forbidden)
			}
		}

		if strings.Contains(trimmed, ":=") {
			return fmt.Errorf("line %d: walrus operator ':=' is not allowed in Python subset", lineNum+1)
		}

		// Complex number literals: 3j, 3.14j, 3J — tree-sitter parses as integer/float
		if isComplexLiteral(trimmed) {
			return fmt.Errorf("line %d: complex number literals are not allowed in Python subset", lineNum+1)
		}
	}

	return nil
}

// validatePythonCST performs CST-based validation after parsing.
// This catches forbidden syntax that text-based scanning would miss or misidentify.
func validatePythonCST(root *sitter.Node, source string) error {
	if err := walkPyCST(root, source, checkForbiddenNodes); err != nil {
		return err
	}
	return validateTypeAnnotations(root, source)
}

func checkForbiddenNodes(n *sitter.Node, src string) error {
	nodeType := n.Type()

	if forbiddenNodeTypes[nodeType] {
		return fmt.Errorf("line %d: '%s' is not allowed in Python subset", n.StartPoint().Row+1, nodeType)
	}

	if nodeType == nodeIdentifier {
		name := src[n.StartByte():n.EndByte()]
		if strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") && len(name) > 4 {
			if !allowedDunders[name] {
				return fmt.Errorf("line %d: dunder access '%s' is not allowed in Python subset", n.StartPoint().Row+1, name)
			}
		}
	}

	if nodeType == nodeString {
		raw := src[n.StartByte():n.EndByte()]
		if len(raw) > 1 && (raw[0] == 'f' || raw[0] == 'F') && (raw[1] == '"' || raw[1] == '\'') {
			return fmt.Errorf("line %d: f-strings are not allowed in Python subset", n.StartPoint().Row+1)
		}
	}

	return nil
}

// validateTypeAnnotations checks that __init__ parameters have type annotations
// and all class methods have return type annotations.
func validateTypeAnnotations(root *sitter.Node, source string) error {
	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		if n.Type() != "class_definition" {
			continue
		}
		body := findNamedChild(n, "block")
		if body == nil {
			continue
		}
		for j := 0; j < int(body.NamedChildCount()); j++ {
			child := body.NamedChild(j)
			if child.Type() != "function_definition" {
				continue
			}
			funcName := findFuncNameFromNode(child, source)

			retType := findNamedChild(child, "type")
			if retType == nil {
				return fmt.Errorf("line %d: method '%s' missing return type annotation (use 'def %s(...) -> Type:')",
					child.StartPoint().Row+1, funcName, funcName)
			}

			if funcName != "__init__" {
				continue
			}
			if err := validateInitParams(child, source); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateInitParams(child *sitter.Node, source string) error {
	params := findNamedChild(child, "parameters")
	if params == nil {
		return nil
	}
	for k := 0; k < int(params.NamedChildCount()); k++ {
		p := params.NamedChild(k)
		switch p.Type() {
		case nodeIdentifier:
			name := source[p.StartByte():p.EndByte()]
			if name == "self" {
				continue
			}
			return fmt.Errorf("line %d: parameter '%s' in __init__ missing type annotation", p.StartPoint().Row+1, name)
		case "default_parameter":
			name := source[p.NamedChild(0).StartByte():p.NamedChild(0).EndByte()]
			if name == "self" {
				continue
			}
			return fmt.Errorf("line %d: parameter '%s' in __init__ missing type annotation (use 'name: type = default')", p.StartPoint().Row+1, name)
		}
	}
	return nil
}

func walkPyCST(n *sitter.Node, source string, fn func(*sitter.Node, string) error) error {
	if err := fn(n, source); err != nil {
		return err
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		if err := walkPyCST(n.NamedChild(i), source, fn); err != nil {
			return err
		}
	}
	return nil
}

func findFuncNameFromNode(n *sitter.Node, source string) string {
	id := findNamedChild(n, nodeIdentifier)
	if id != nil {
		return source[id.StartByte():id.EndByte()]
	}
	return ""
}

var forbiddenBuiltins = []string{
	"exec", "eval", "compile", "open", "__import__",
	"globals", "locals", "vars", "dir", "getattr", "setattr",
	"delattr", "hasattr", "type", "isinstance", "issubclass",
	"classmethod", "staticmethod", "property",
	"input", "print", "help", "exit", "quit",
	// Python builtins not in the subset and not mapped to VM builtins
	"len", "sorted", "sum", "enumerate", "zip",
	"reversed", "any", "all",
}

var forbiddenNodeTypes = map[string]bool{
	// Comprehensions
	"list_comprehension":     true,
	"set_comprehension":      true,
	"dictionary_comprehension": true,
	"generator_expression":   true,
	// Decorators
	"decorator": true,
	// Lambda
	"lambda": true,
	// Yield
	"yield": true,
	// Try/except/finally
	"try_statement":  true,
	// With
	"with_statement": true,
	// Global/nonlocal
	"global_statement":   true,
	"nonlocal_statement": true,
	// Del
	"delete_statement": true,
	// Assert
	"assert_statement": true,
	// Async/await
	"async_statement": true,
	// Raise
	"raise_statement": true,
	// Slicing (subscript with slice)
	"slice": true,
	// Walrus operator (named in tree-sitter as 'named_expression')
	"named_expression": true,
	// *args / **kwargs (parameter level)
	"list_splat_pattern":      true,
	"dict_splat_pattern":      true,
	// *args / **kwargs (call argument level)
	"list_splat":       true,
	"dictionary_splat": true,
	// Collection literals — not in the trading strategy subset
	"list":             true,
	"tuple":            true,
	"dictionary":       true,
	"set":              true,
	"expression_list":  true, // return a, b / x = a, b
	// Tuple unpacking assignment: a, b = ...
	"pattern_list":     true,
	// Pattern matching (Python 3.10+)
	"match_statement":  true,
	// F-string interpolation (catches rf"..." / fr"..." bypass)
	"interpolation":    true,
	// Tree-sitter error recovery
	"ERROR":            true,
	// Ellipsis (... literal) — not meaningful in trading subset
	"ellipsis":             true,
	// Python 3.12+ type alias — not supported
	"type_alias_statement": true,
	// Python 2 print statement — not supported (print is also in forbiddenBuiltins)
	"print_statement":      true,
	// Multiple inheritance is rejected at compile time (ADR-0024 D3) — see compileClass
}

// isComplexLiteral checks if a line contains a complex number literal (e.g. 3j, 3.14j, 3J).
// Tree-sitter parses these as integer/float nodes, so they bypass node-type checking.
func isComplexLiteral(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 'j' || s[i] == 'J' {
			if i > 0 && isWordChar(s[i-1]) {
				// Check it's a number before the j, not a variable name like "myj"
				j := i - 1
				for j >= 0 && (isWordChar(s[j]) || s[j] == '.') {
					j--
				}
				// If preceded by a digit, it's a complex literal
				if j+1 < i && (s[j+1] >= '0' && s[j+1] <= '9') {
					return true
				}
			}
		}
	}
	return false
}

// containsWordBuiltin checks for a forbidden builtin name but skips matches
// preceded by '.' (method calls like ctx.bars().open(1) are not builtin open()).
func containsWordBuiltin(s, word string) bool {
	idx := 0
	for {
		pos := strings.Index(s[idx:], word)
		if pos < 0 {
			return false
		}
		pos += idx
		leftOK := pos == 0 || !isWordChar(s[pos-1])
		rightOK := pos+len(word) >= len(s) || !isWordChar(s[pos+len(word)])
		if leftOK && rightOK {
			// Skip if preceded by '.' (method call, not builtin)
			if pos > 0 && s[pos-1] == '.' {
				idx = pos + 1
				continue
			}
			return true
		}
		idx = pos + 1
	}
}
