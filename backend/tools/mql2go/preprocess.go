package mql2go

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"alphaforge/tools/mql2go/interp"
	sitter "github.com/smacker/go-tree-sitter"
)

// ── MQL5 class/struct compilation ───────────────────────────────────

// collectClassDecl processes MQL5 class/struct declarations and registers
// them as ClassInstance templates in the IR globals.
func (c *compiler) collectClassDecl(ir *interp.IR, n *sitter.Node) {
	switch n.Type() {
	case "class_specifier", "struct_specifier":
		// Class/struct declarations define types, not variables.
		// Don't add to ir.Globals — just let knownClasses track them.
	}
}

// findTypeName extracts the type name from a class/struct specifier.
func (c *compiler) findTypeName(n *sitter.Node) string {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeTypeIdentifier {
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
// #include — inject stub class declarations for known MQL5 headers
// #define — macro substitution
// #ifdef/#ifndef/#else/#endif — simple conditional inclusion
// #property — strip (metadata only)
// #import — strip (DLL imports, not supported)
func PreprocessMQL(source string) string {
	lines := strings.Split(source, "\n")
	defines := make(map[string]string)
	var result []string
	var condStack []bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		skipped := false
		for _, active := range condStack {
			if !active {
				skipped = true
				break
			}
		}

		if handled, out := handlePreprocessorDirective(trimmed, line, skipped, defines, &condStack); handled {
			result = append(result, out)
			continue
		}

		if skipped {
			result = append(result, "")
			continue
		}

		if strings.HasPrefix(trimmed, "#property ") || strings.HasPrefix(trimmed, "#import ") {
			result = append(result, "")
			continue
		}

		if strings.HasPrefix(trimmed, "#include ") {
			stub := includeStub(trimmed)
			if stub != "" {
				result = append(result, stub)
			} else {
				result = append(result, line)
			}
			continue
		}

		processed := line
		for key, val := range defines {
			processed = replaceWord(processed, key, val)
		}
		processed = rewriteInputEnum(processed)
		processed = rewriteDatetimeLiteral(processed)
		result = append(result, processed)
	}

	return strings.Join(result, "\n")
}

func handlePreprocessorDirective(trimmed, line string, skipped bool, defines map[string]string, condStack *[]bool) (handled bool, out string) {
	if strings.HasPrefix(trimmed, "#define ") {
		if !skipped {
			parts := strings.SplitN(trimmed[8:], " ", 2)
			if len(parts) >= 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				defines[key] = val
			}
		}
		return true, line
	}
	if strings.HasPrefix(trimmed, "#undef ") {
		if !skipped {
			key := strings.TrimSpace(trimmed[7:])
			delete(defines, key)
		}
		return true, ""
	}
	if strings.HasPrefix(trimmed, "#ifdef ") {
		name := strings.TrimSpace(trimmed[7:])
		*condStack = append(*condStack, !skipped && defines[name] != "")
		return true, ""
	}
	if strings.HasPrefix(trimmed, "#ifndef ") {
		name := strings.TrimSpace(trimmed[8:])
		*condStack = append(*condStack, !skipped && defines[name] == "")
		return true, ""
	}
	if trimmed == "#else" {
		if len(*condStack) > 0 {
			outerSkipped := false
			for i := 0; i < len(*condStack)-1; i++ {
				if !(*condStack)[i] {
					outerSkipped = true
					break
				}
			}
			if outerSkipped {
				(*condStack)[len(*condStack)-1] = false
			} else {
				(*condStack)[len(*condStack)-1] = !(*condStack)[len(*condStack)-1]
			}
		}
		return true, ""
	}
	if trimmed == "#endif" {
		if len(*condStack) > 0 {
			*condStack = (*condStack)[:len(*condStack)-1]
		}
		return true, ""
	}
	return false, ""
}

// rewriteInputEnum transforms 'input EnumType name = value;' → 'extern int name = value;'
// and 'input EnumType name;' → 'extern int name;'
// tree-sitter can't parse 'input CustomType' as a declaration because 'input'
// is not a C keyword and custom enum types confuse the parser.
func rewriteInputEnum(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "input ") && !strings.HasPrefix(trimmed, "extern ") {
		return line
	}
	// Remove 'input ' or 'extern ' prefix (different lengths!)
	var rest string
	if strings.HasPrefix(trimmed, "input ") {
		rest = trimmed[6:] // len("input ") == 6
	} else {
		rest = trimmed[7:] // len("extern ") == 7
	}
	// Check if the next token is a primitive type — if so, no rewrite needed
	primitives := map[string]bool{"int": true, "long": true, "double": true, "float": true, "string": true, "bool": true, "datetime": true, "char": true, "void": true, "color": true}
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return line
	}
	typeName := strings.TrimSpace(parts[0])
	if primitives[typeName] {
		return line
	}
	// Non-primitive type (enum/class) — rewrite to 'extern int name = value;'
	// Preserve leading indentation
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	suffix := parts[1]
	return indent + "extern int " + suffix
}

var datetimeLiteralRe = regexp.MustCompile(`D'(\d{4})\.(\d{2})\.(\d{2})(?:\s+(\d{2}):(\d{2}):(\d{2}))?'`)

// rewriteDatetimeLiteral converts MQL datetime literals D'YYYY.MM.DD' or
// D'YYYY.MM.DD HH:MM:SS' to Unix millisecond timestamps so tree-sitter can
// parse them as number literals.
func rewriteDatetimeLiteral(line string) string {
	return datetimeLiteralRe.ReplaceAllStringFunc(line, func(match string) string {
		sub := datetimeLiteralRe.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		year := sub[1]
		month := sub[2]
		day := sub[3]
		hour := "00"
		min := "00"
		sec := "00"
		if len(sub) >= 7 && sub[4] != "" {
			hour = sub[4]
			min = sub[5]
			sec = sub[6]
		}
		t, err := time.Parse("2006.01.02 15:04:05", year+"."+month+"."+day+" "+hour+":"+min+":"+sec)
		if err != nil {
			return match
		}
		return fmt.Sprintf("%d", t.UnixMilli())
	})
}

// includeStub returns a stub class/struct declaration for known MQL5 headers
// so tree-sitter can parse the types. Returns empty string for unknown headers.
func includeStub(includeLine string) string {
	// Extract header path from <...> or "..."
	hdr := ""
	if strings.Contains(includeLine, "<") && strings.Contains(includeLine, ">") {
		start := strings.Index(includeLine, "<")
		end := strings.LastIndex(includeLine, ">")
		if start+1 < end {
			hdr = includeLine[start+1 : end]
		}
	} else if strings.Contains(includeLine, "\"") {
		start := strings.Index(includeLine, "\"")
		end := strings.LastIndex(includeLine, "\"")
		if start+1 < end {
			hdr = includeLine[start+1 : end]
		}
	}

	switch hdr {
	case "Trade/Trade.mqh":
		return "class CTrade { public: void SetExpertMagicNumber(int); bool Buy(double,string,double,double,double,string); bool Sell(double,string,double,double,double,string); bool BuyLimit(double,string,double,double,double,string); bool SellLimit(double,string,double,double,double,string); bool BuyStop(double,string,double,double,double,string); bool SellStop(double,string,double,double,double,string); bool PositionClose(int64); bool PositionClosePartial(int64,double); bool PositionCloseBy(int64,int64); bool PositionModify(int64,double,double); bool OrderDelete(int64); void SetDeviationInPoints(int); };"
	case "Trade/PositionInfo.mqh":
		return "class CPositionInfo { public: int64 Ticket(); string Symbol(); double Volume(); double PriceCurrent(); double PriceOpen(); double StopLoss(); double TakeProfit(); };"
	case "Trade/OrderInfo.mqh":
		return "class COrderInfo { public: int64 Ticket(); string Symbol(); double Volume(); double PriceOpen(); double PriceCurrent(); double StopLoss(); double TakeProfit(); };"
	case "Trade/SymbolInfo.mqh":
		return "class CSymbolInfo { public: string Name(); int Digits(); double Point(); double Bid(); double Ask(); };"
	case "Arrays/ArrayDouble.mqh":
		return "class CArrayDouble { public: int Add(double); int Size(); double At(int); };"
	default:
		// For unknown headers, return empty line to preserve line numbers
		return ""
	}
}
