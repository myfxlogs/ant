package mql2go

import (
	"regexp"
	"strings"
)

// ── Expression helpers ───────────────────────────────────────────

func nonEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func mqlToGoType(t string) string {
	switch t {
	case "int", "long", "uint":
		return "int"
	case "double", "float":
		return "decimal.Decimal"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

func goSignalAction(a OrderAction) string {
	switch a {
	case ActionMarketBuy:
		return "sdk.ActionBuy"
	case ActionMarketSell:
		return "sdk.ActionSell"
	case ActionBuyLimit:
		return "sdk.ActionBuyLimit"
	case ActionSellLimit:
		return "sdk.ActionSellLimit"
	case ActionBuyStop:
		return "sdk.ActionBuyStop"
	case ActionSellStop:
		return "sdk.ActionSellStop"
	}
	return "sdk.ActionNone"
}

func goFillPolicy(a OrderAction) string {
	switch a {
	case ActionMarketBuy, ActionMarketSell:
		return "sdk.FillIOC"
	}
	return "sdk.FillReturn"
}

func prefixRef(expr string) string {
	// If a bare identifier (no prefix), add "s."
	if expr == "" || strings.HasPrefix(expr, "s.") || strings.HasPrefix(expr, "ctx.") {
		return expr
	}
	if strings.Contains(expr, ".") || strings.Contains(expr, "(") || strings.Contains(expr, "\"") {
		return expr // already qualified or literal
	}
	return "s." + expr
}

func mqlToGoExpr(expr string) string {
	if expr == "" {
		return expr
	}
	// self. → s.
	expr = strings.ReplaceAll(expr, "self.", "s.")
	// Python logical operators → Go
	expr = strings.ReplaceAll(expr, " and ", " && ")
	expr = strings.ReplaceAll(expr, " or ", " || ")
	expr = strings.ReplaceAll(expr, " not ", " ! ")
	// Remove Decimal(str(...)) wrapper
	expr = strings.ReplaceAll(expr, "Decimal(str(", "")
	expr = strings.ReplaceAll(expr, "'", "")
	// MQL builtins → ctx methods
	// Point first (replaceWord won't match _Point since _ is a word char),
	// then _Point as plain string replacement
	expr = replaceWord(expr, "Ask", "ctx.Ask()")
	expr = replaceWord(expr, "Bid", "ctx.Bid()")
	expr = replaceWord(expr, "Point", "ctx.Point()")
	expr = strings.ReplaceAll(expr, "_Point", "ctx.Point()")
	// Symbol: replace both MQL4 Symbol() and MQL5 _Symbol in one pass
	expr = symbolRe.ReplaceAllString(expr, "ctx.Symbol()")
	// MQL time series access: Close[1], Open[0], High[2], Low[1] → ctx.Bars().*(n)
	expr = convertTimeSeries(expr)
	return expr
}

// timeSeriesRe matches MQL time series access: Close[1], Open[0], High[2], etc.
var timeSeriesRe = regexp.MustCompile(`\b(Close|Open|High|Low|Volume|Time)\[(\w+)\]`)

// symbolRe matches both MQL4 Symbol() and MQL5 _Symbol in one pass.
var symbolRe = regexp.MustCompile(`\b_?Symbol(?:\(\))?\b`)

// convertTimeSeries converts MQL time series access patterns to Go SDK calls.
// Close[1] → ctx.Bars().Close(1).InexactFloat64()
// Volume[0] → ctx.Bars().Volume(0)
// Time[1] → ctx.Bars().Time(1)
func convertTimeSeries(expr string) string {
	return timeSeriesRe.ReplaceAllStringFunc(expr, func(match string) string {
		sub := timeSeriesRe.FindStringSubmatch(match)
		name, index := sub[1], sub[2]
		switch name {
		case "Volume":
			return "ctx.Bars().Volume(" + index + ")"
		case "Time":
			return "ctx.Bars().Time(" + index + ")"
		default:
			// Close, Open, High, Low return decimal.Decimal
			return "ctx.Bars()." + name + "(" + index + ")"
		}
	})
}

// intCast wraps a non-numeric expression in int() cast for SDK methods that expect int.
func intCast(expr string) string {
	if expr == "" || isNumeric(expr) {
		return expr
	}
	return "int(" + expr + ")"
}

func paramGoType(t ParamType) string {
	switch t {
	case ParamInt:
		return "int32"
	case ParamDouble:
		return "decimal.Decimal"
	case ParamString:
		return "string"
	case ParamBool:
		return "bool"
	default:
		return "string"
	}
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && c != '.' && c != '-' {
			return false
		}
	}
	return true
}

func replaceWord(s, old, new string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			before := i == 0 || !isWordChar(s[i-1])
			after := i+len(old) == len(s) || !isWordChar(s[i+len(old)])
			if before && after {
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

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func (g *generator) convertParams(expr string) string {
	for _, p := range g.intent.Params {
		if p.Type == ParamInt {
			expr = replaceWord(expr, p.Name, "decimal.NewFromInt(int64(s."+p.Name+"))")
		} else {
			expr = replaceWord(expr, p.Name, "s."+p.Name)
		}
	}
	return expr
}

// wrapDecimal wraps a numeric literal in decimal.NewFromFloat() if needed.
// Expressions that are already decimal.Decimal (e.g. ctx.Bars().Close(1)) are passed through.
func wrapDecimal(expr string) string {
	if expr == "" {
		return expr
	}
	if isNumeric(expr) {
		return "decimal.NewFromFloat(" + expr + ")"
	}
	return expr
}

// compareRe matches comparison operators between two operands.
var compareRe = regexp.MustCompile(`(.+?)\s*(>=|<=|==|!=|>|<)\s*(.+)`)

// convertComparisons converts comparison operators to decimal.Decimal method calls.
// e.g. "a > b" → "a.GreaterThan(b)", "a >= b" → "a.GreaterThanOrEqual(b)"
// Numeric literals on the right side are wrapped in decimal.NewFromFloat().
func convertComparisons(expr string) string {
	parts := strings.Split(expr, " && ")
	for i, part := range parts {
		parts[i] = convertSingleComparison(strings.TrimSpace(part))
	}
	return strings.Join(parts, " && ")
}

func convertSingleComparison(expr string) string {
	m := compareRe.FindStringSubmatch(expr)
	if m == nil {
		return expr
	}
	left := strings.TrimSpace(m[1])
	op := m[2]
	right := strings.TrimSpace(m[3])

	// Wrap numeric literal right-hand side
	if isNumeric(right) {
		right = "decimal.NewFromFloat(" + right + ")"
	}

	switch op {
	case ">":
		return left + ".GreaterThan(" + right + ")"
	case ">=":
		return left + ".GreaterThanOrEqual(" + right + ")"
	case "<":
		return left + ".LessThan(" + right + ")"
	case "<=":
		return left + ".LessThanOrEqual(" + right + ")"
	case "==":
		return left + ".Equal(" + right + ")"
	case "!=":
		return left + ".NotEqual(" + right + ")"
	default:
		return expr
	}
}
