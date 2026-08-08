package interp

import (
	"fmt"
	"strings"
)

// DefenseAViolation describes a post-parse validation failure (ADR-0028 §4.1 Defense A).
// Violations are fatal — they indicate structural issues that make the strategy
// unreliable (e.g. parameter name collides with a type name, no entry point mapped).
type DefenseAViolation struct {
	Rule       string // "param_name_keyword", "param_name_duplicate", "undefined_reference", "no_entry_point"
	Identifier string // the offending identifier (empty for "no_entry_point")
	Severity   string // always SeverityFatal
	Message    string // human-readable description
}

// mqlKeywordsAndTypes are MQL reserved words and type names that cannot be used
// as parameter names. Using them shadows the type/keyword and causes silent
// semantic errors (e.g. xianhua: Lots→double collision).
var mqlKeywordsAndTypes = map[string]bool{
	// Type names
	"int": true, "double": true, "string": true, "bool": true,
	"datetime": true, "void": true, "char": true, "uchar": true,
	"short": true, "ushort": true, "long": true, "ulong": true,
	"uint": true, "float": true, "color": true, "font": true,
	// Control flow keywords
	"if": true, "else": true, "for": true, "while": true,
	"do": true, "switch": true, "case": true, "default": true,
	"break": true, "continue": true, "return": true,
	"const": true, "static": true, "extern": true,
	"input": true, "sinput": true, "class": true, "struct": true,
	"enum": true, "typedef": true, "sizeof": true, "new": true,
	"delete": true, "true": true, "false": true, "NULL": true,
	"public": true, "private": true, "protected": true,
	"virtual": true, "override": true, "operator": true,
	"template": true, "typename": true, "namespace": true,
	"using": true, "this": true, "ref": true,
	// Common MQL built-in variable names that should not be parameter names
	"Point": true, "Digits": true, "Bars": true, "Period": true,
	"Ask": true, "Bid": true, "Close": true, "Open": true,
	"High": true, "Low": true, "Time": true, "Volume": true,
}

// ValidateDefenseA performs post-parse validation on the IR (ADR-0028 §4.1).
// Returns a list of violations; empty list means the IR passes all Defense A checks.
//
// Rules:
//  1. Parameter name legality — param names must not be MQL keywords/type names
//  2. Parameter name uniqueness — no duplicate param names
//  3. Entry point exists — at least one of OnTick/OnBar/OnTimer/start must be mapped
//  4. Undefined references — collected during bytecode compilation (resolveVar),
//     not here; this function only checks IR-level structural issues.
//
// Rule 2 (undefined references) is enforced during CompileAST via resolveVar,
// which records blind spots for MQL4 implicit variables and errors for MQL5.
// This function focuses on rules that can be checked purely from the IR.
func ValidateDefenseA(ir *IR) []DefenseAViolation {
	var violations []DefenseAViolation

	// Rule 1: Parameter name legality
	for _, p := range ir.Params {
		if mqlKeywordsAndTypes[p.Name] {
			violations = append(violations, DefenseAViolation{
				Rule:       "param_name_keyword",
				Identifier: p.Name,
				Severity:   SeverityFatal,
				Message:    fmt.Sprintf("parameter name '%s' is a reserved MQL keyword or type name", p.Name),
			})
		}
	}

	// Rule 2: Parameter name uniqueness
	seen := map[string]bool{}
	for _, p := range ir.Params {
		if seen[p.Name] {
			violations = append(violations, DefenseAViolation{
				Rule:       "param_name_duplicate",
				Identifier: p.Name,
				Severity:   SeverityFatal,
				Message:    fmt.Sprintf("duplicate parameter name '%s'", p.Name),
			})
		}
		seen[p.Name] = true
	}

	// Rule 3: Entry point exists
	hasEntry := len(ir.OnTick) > 0 || len(ir.OnBar) > 0 || len(ir.OnTimer) > 0
	if !hasEntry {
		// Check for 'start' function (MQL4 legacy entry point)
		if _, ok := ir.Funcs["start"]; ok {
			hasEntry = true
		}
	}
	if !hasEntry {
		violations = append(violations, DefenseAViolation{
			Rule:     "no_entry_point",
			Severity: SeverityFatal,
			Message:  "no entry point found (OnTick, OnBar, OnTimer, or start function required)",
		})
	}

	return violations
}

// FormatDefenseAViolations returns a human-readable summary of Defense A violations.
func FormatDefenseAViolations(violations []DefenseAViolation) string {
	if len(violations) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Defense A violations:\n")
	for _, v := range violations {
		fmt.Fprintf(&sb, "  [%s] %s\n", v.Rule, v.Message)
	}
	return sb.String()
}
