package mql2go

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// HeaderSymbol is a symbol extracted from an MQL header file (.mqh).
type HeaderSymbol struct {
	Name      string
	Kind      string // "function", "constant", "enum_value", "class_method"
	Signature string // function signature (e.g. "double iMA(string,int,int,int,int,int,int)")
	Value     string // constant value (for #define and enum)
	Source    string // source file path
}

// ParseHeaderFile parses a single .mqh file with tree-sitter and extracts
// all function declarations, #define constants, and enum values.
func ParseHeaderFile(path string) ([]HeaderSymbol, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	source := string(data)

	// Preprocess: handle #define macros by extracting them before tree-sitter parsing.
	var symbols []HeaderSymbol
	symbols = append(symbols, extractDefines(source, path)...)

	// Parse with tree-sitter
	root, err := ParseMQL(source)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Walk the CST and extract function declarations, enums, class methods
	symbols = append(symbols, extractSymbolsFromCST(source, root, path)...)

	return symbols, nil
}

// ParseHeaderDir parses all .mqh files in a directory (recursively).
func ParseHeaderDir(dir string) ([]HeaderSymbol, error) {
	var allSymbols []HeaderSymbol
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".mqh") && !strings.HasSuffix(path, ".h") {
			return nil
		}
		symbols, err := ParseHeaderFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", path, err)
			return nil
		}
		allSymbols = append(allSymbols, symbols...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return allSymbols, nil
}

// extractDefines extracts #define constants from source text.
func extractDefines(source, sourceFile string) []HeaderSymbol {
	var symbols []HeaderSymbol
	lines := strings.Split(source, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#define ") {
			continue
		}
		// #define NAME VALUE
		parts := strings.Fields(trimmed)
		if len(parts) < 3 {
			continue
		}
		name := parts[1]
		value := strings.Join(parts[2:], " ")
		symbols = append(symbols, HeaderSymbol{
			Name:   name,
			Kind:   "constant",
			Value:  value,
			Source: sourceFile,
		})
	}
	return symbols
}

// isClassNode checks if a function_definition node is actually a class/struct
// declaration (tree-sitter parses MQL classes this way).
func isClassNode(source string, n *sitter.Node) bool {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "type_identifier" {
			txt := nodeText(source, child)
			if txt == "class" || txt == "struct" {
				return true
			}
		}
	}
	return false
}

// extractSymbolsFromCST walks the tree-sitter CST and extracts function declarations,
// enum values, and class method declarations.
func extractSymbolsFromCST(source string, root *sitter.Node, sourceFile string) []HeaderSymbol {
	var symbols []HeaderSymbol

	for i := 0; i < int(root.NamedChildCount()); i++ {
		n := root.NamedChild(i)
		switch n.Type() {
		case "function_definition":
			// tree-sitter parses MQL class declarations as function_definition
			// with type_identifier "class"/"struct". Detect and handle as class.
			if isClassNode(source, n) {
				symbols = append(symbols, extractClassMethods(source, n, sourceFile)...)
				break
			}
			name := funcName(source, n)
			if name != "" {
				sig := buildSignature(source, n)
				symbols = append(symbols, HeaderSymbol{
					Name:      name,
					Kind:      "function",
					Signature: sig,
					Source:    sourceFile,
				})
			}
		case "declaration":
			// Could be a function prototype (declaration without body)
			decl := childByType(n, "function_declarator")
			if decl != nil {
				name := funcName(source, n)
				if name != "" {
					sig := buildSignature(source, n)
					symbols = append(symbols, HeaderSymbol{
						Name:      name,
						Kind:      "function",
						Signature: sig,
						Source:    sourceFile,
					})
				}
			}
		case "enum_specifier":
			symbols = append(symbols, extractEnumValues(source, n, sourceFile)...)
		case "class_specifier", "struct_specifier":
			symbols = append(symbols, extractClassMethods(source, n, sourceFile)...)
		case "preproc_function_def":
			// #define MACRO(x) ... — macro function
			id := childByType(n, nodeIdentifier)
			if id != nil {
				name := nodeText(source, id)
				symbols = append(symbols, HeaderSymbol{
					Name:   name,
					Kind:   "constant",
					Value:  "macro",
					Source: sourceFile,
				})
			}
		}
	}

	return symbols
}

// extractEnumValues extracts individual enum values from an enum_specifier node.
