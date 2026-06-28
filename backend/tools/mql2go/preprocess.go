package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"anttrader/tools/mql2go/interp"
)

// ── MQL5 class/struct compilation ───────────────────────────────────

// collectClassDecl processes MQL5 class/struct declarations and registers
// them as ClassInstance templates in the IR globals.
func (c *compiler) collectClassDecl(ir *interp.IR, n *sitter.Node) {
	switch n.Type() {
	case "class_specifier", "struct_specifier":
		name := c.findTypeName(n)
		if name == "" {
			return
		}
		// Register as a global variable of type class
		ir.Globals = append(ir.Globals, interp.GlobalVar{
			Name: name,
			Type: "class",
		})
	}
}

// findTypeName extracts the type name from a class/struct specifier.
func (c *compiler) findTypeName(n *sitter.Node) string {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "type_identifier" {
			return c.text(child)
		}
	}
	return ""
}

// collectClassInstance processes declarations like `CTrade trade;`
// where CTrade is a known class type. Creates a ClassInstance global.
func (c *compiler) collectClassInstance(ir *interp.IR, n *sitter.Node, knownClasses map[string]bool) {
	// Look for declarations with class type
	typeName := c.findType(n)
	if typeName == "" {
		return
	}
	if !knownClasses[typeName] && !isBuiltinClass(typeName) {
		return
	}

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "init_declarator" || child.Type() == "declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			// Register as a class instance global
			ir.Globals = append(ir.Globals, interp.GlobalVar{
				Name: name,
				Type: typeName, // Store the class type name
			})
		}
	}
}

// isBuiltinClass returns true for MQL5 built-in class/struct types.
func isBuiltinClass(name string) bool {
	switch name {
	case "CTrade", "MqlTradeRequest", "MqlTradeResult",
		"MqlDateTime", "MqlRates", "MqlTick":
		return true
	}
	return false
}

// ── Preprocessor ────────────────────────────────────────────────────

// PreprocessMQL handles MQL preprocessor directives before parsing.
// #include — file inclusion (stub: skip, tree-sitter handles gracefully)
// #define — macro substitution
// #property — strip (metadata only)
// #import — strip (DLL imports, not supported)
func PreprocessMQL(source string) string {
	lines := strings.Split(source, "\n")
	defines := make(map[string]string)
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// #define NAME value
		if strings.HasPrefix(trimmed, "#define ") {
			parts := strings.SplitN(trimmed[8:], " ", 2)
			if len(parts) >= 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				defines[key] = val
			}
			result = append(result, line) // Keep for tree-sitter
			continue
		}

		// #property, #import — strip but keep empty line for line numbers
		if strings.HasPrefix(trimmed, "#property ") || strings.HasPrefix(trimmed, "#import ") {
			result = append(result, "")
			continue
		}

		// #include — keep for tree-sitter (it handles or ignores)
		if strings.HasPrefix(trimmed, "#include ") {
			result = append(result, line)
			continue
		}

		// Apply #define substitutions
		processed := line
		for key, val := range defines {
			processed = replaceWord(processed, key, val)
		}

		result = append(result, processed)
	}

	return strings.Join(result, "\n")
}
