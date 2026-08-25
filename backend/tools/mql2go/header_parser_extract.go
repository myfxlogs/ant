// header_parser_extract.go — Enum/class/method extraction and registry generation extracted from header_parser.go.
package mql2go

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"alphaforge/tools/mql2go/interp"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractEnumValues(source string, n *sitter.Node, sourceFile string) []HeaderSymbol {
	var symbols []HeaderSymbol

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "enumerator_list" {
			counter := int32(0)
			for j := 0; j < int(child.NamedChildCount()); j++ {
				ec := child.NamedChild(j)
				if ec.Type() != "enumerator" {
					continue
				}
				name := ""
				for k := 0; k < int(ec.NamedChildCount()); k++ {
					ecChild := ec.NamedChild(k)
					if ecChild.Type() == nodeIdentifier {
						name = nodeText(source, ecChild)
					} else if ecChild.Type() == nodeNumberLiteral {
						nVal := interp.ParseNumberLiteral(nodeText(source, ecChild))
						counter = nVal.ToInt()
					}
				}
				if name != "" {
					symbols = append(symbols, HeaderSymbol{
						Name:   name,
						Kind:   "enum_value",
						Value:  fmt.Sprintf("%d", counter),
						Source: sourceFile,
					})
					counter++
				}
			}
		}
	}

	return symbols
}

// extractClassMethods extracts method declarations from a class/struct body.
// tree-sitter parses MQL classes as function_definition with compound_statement
// body (not class_specifier with field_declaration_list), so we walk the
// compound_statement for declaration nodes.
func extractClassMethods(source string, n *sitter.Node, sourceFile string) []HeaderSymbol {
	var symbols []HeaderSymbol

	// Get class name from the identifier child (type_identifier is "class"/"struct")
	className := ""
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == nodeIdentifier {
			name := nodeText(source, child)
			// Skip "class"/"struct" keywords which may appear as identifiers
			if name != "class" && name != "struct" {
				className = name
				break
			}
		}
	}
	if className == "" {
		return nil
	}

	// Walk compound_statement body for method declarations
	body := childByType(n, nodeCompoundStatement)
	if body == nil {
		return nil
	}

	for i := 0; i < int(body.NamedChildCount()); i++ {
		child := body.NamedChild(i)
		// Methods appear as labeled_statement (public:/private:) containing declarations
		if child.Type() == "labeled_statement" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				inner := child.NamedChild(j)
				symbols = appendMethodIfValid(symbols, source, className, sourceFile, inner)
			}
		}
		// Direct declaration (no access label)
		symbols = appendMethodIfValid(symbols, source, className, sourceFile, child)
	}

	return symbols
}

func appendMethodIfValid(symbols []HeaderSymbol, source, className, sourceFile string, n *sitter.Node) []HeaderSymbol {
	if n.Type() != "declaration" {
		return symbols
	}
	decl := childByType(n, "function_declarator")
	if decl == nil {
		return symbols
	}
	name := funcName(source, n)
	if name == "" {
		return symbols
	}
	return append(symbols, HeaderSymbol{
		Name:      className + "." + name,
		Kind:      "class_method",
		Signature: buildSignature(source, n),
		Source:    sourceFile,
	})
}

// buildSignature extracts a function signature from a function node.
func buildSignature(source string, n *sitter.Node) string {
	// Get return type
	retType := ""
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		t := child.Type()
		if t == "primitive_type" || t == nodeTypeIdentifier || t == "sized_type_specifier" {
			retType = nodeText(source, child)
			break
		}
	}

	// Get parameter list
	decl := childByType(n, "function_declarator")
	if decl == nil {
		decl = n
	}
	params := childByType(decl, "parameter_list")
	paramText := "()"
	if params != nil {
		paramText = nodeText(source, params)
	}

	if retType != "" {
		return retType + " " + funcName(source, n) + paramText
	}
	return funcName(source, n) + paramText
}

// GenerateRegistryEntries converts extracted header symbols into Go source code
// for the api_registry.go unsupportedSymbols list.
// Only symbols NOT already in the registry are included.
func GenerateRegistryEntries(symbols []HeaderSymbol) string {
	// Deduplicate by name
	seen := make(map[string]bool)
	var unique []HeaderSymbol
	for _, s := range symbols {
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		unique = append(unique, s)
	}

	// Sort by name
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Name < unique[j].Name
	})

	var b strings.Builder
	b.WriteString("// Auto-generated from .mqh header files by tools/mql2go/header_parser.go\n")
	b.WriteString("// Run: go run ./tools/mql2go/cmd/parse_headers <mqh_dir>\n\n")

	for _, s := range unique {
		// Skip if already implemented
		if interp.IsAPIImplemented(s.Name) {
			continue
		}
		// Skip if already in unsupported list
		if interp.IsAPIUnsupported(s.Name) {
			continue
		}

		kind := "function"
		switch s.Kind {
		case "class_method":
			kind = "method"
		case "constant", "enum_value":
			kind = "constant"
		}

		fmt.Fprintf(&b, "// %s — %s (from %s)\n", s.Name, kind, filepath.Base(s.Source))
		if s.Signature != "" {
			fmt.Fprintf(&b, "//   signature: %s\n", s.Signature)
		}
		if s.Value != "" {
			fmt.Fprintf(&b, "//   value: %s\n", s.Value)
		}
		b.WriteString("//   status: needs classification (implemented/unsupported)\n\n")
	}

	return b.String()
}
