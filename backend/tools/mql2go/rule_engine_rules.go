// rule_engine_rules.go — Rules R06-R09 extracted from rule_engine.go.
package mql2go

import (
	"fmt"
	"strings"

	"alphaforge/tools/mql2go/interp"
)

func (ruleOrderSelectHistory) ID() string { return "R06_orderselect_history" }

func (ruleOrderSelectHistory) Match(input RuleInput) *DiagnosticFinding {
	sourceUpper := strings.ToUpper(input.Source)
	if !strings.Contains(sourceUpper, "ORDERSELECT") {
		return nil
	}
	if !strings.Contains(sourceUpper, "MODE_HISTORY") {
		return nil
	}
	return &DiagnosticFinding{
		RuleID:   "R06_orderselect_history",
		Severity: sevWarningEn,
		Title:    "OrderSelect with MODE_HISTORY pool",
		Detail:   "EA iterates the history pool (MODE_HISTORY). History pool support may be limited — closed orders may not be fully accessible.",
		Suggest:  "Verify that OrdersHistoryTotal() and OrderSelect(idx, SELECT_BY_POS, MODE_HISTORY) return correct closed-order data.",
	}
}

// ── Rule 7: OrderProfit() on open positions ──────────────────────────

type ruleOrderProfitOpenPos struct{}

func (ruleOrderProfitOpenPos) ID() string { return "R07_orderprofit_open" }

func (ruleOrderProfitOpenPos) Match(input RuleInput) *DiagnosticFinding {
	sourceUpper := strings.ToUpper(input.Source)
	if !strings.Contains(sourceUpper, "ORDERPROFIT") {
		return nil
	}
	// Check if EA uses OrderProfit() in conditions (not just display)
	// Heuristic: OrderProfit appears in an if/while/return context
	if !strings.Contains(input.Source, "if") && !strings.Contains(input.Source, "while") {
		return nil
	}
	return &DiagnosticFinding{
		RuleID:   "R07_orderprofit_open",
		Severity: sevInfoEn,
		Title:    "OrderProfit() used in conditional logic",
		Detail:   "EA uses OrderProfit() in conditions. For open positions, OrderProfit() returns floating P&L based on current market price.",
		Suggest:  "Verify that OrderProfit() returns floating P&L for open positions, not 0.",
	}
}

// ── Rule 8: Indicator mode parameter + missing constant ──────────────

// ── Rule 9: Parameter name matches MQL primitive type ─────────────────

// mqlPrimitiveTypes are MQL4/MQL5 built-in type keywords.
var mqlPrimitiveTypes = map[string]bool{
	"int": true, "double": true, "float": true, "bool": true, "string": true,
	"color": true, "datetime": true, "long": true, "short": true,
	"uint": true, "ulong": true, "char": true, "uchar": true, "void": true,
}

type ruleParamNameIsType struct{}

func (ruleParamNameIsType) ID() string { return "R09_param_name_is_type" }

func (ruleParamNameIsType) Match(input RuleInput) *DiagnosticFinding {
	if input.Coverage == nil || len(input.Coverage.BlindSpots) == 0 {
		return nil
	}
	// Scan coverage blind spots for "unknown constant" entries that look like
	// a parameter name matched a type keyword (tree-sitter parser bug).
	// Pattern: "unknown constant: <type>" where <type> is an MQL primitive type
	// and the source code has extern/input declarations using that type.
	for _, bs := range input.Coverage.BlindSpots {
		if !strings.Contains(bs, "unknown constant: ") {
			continue
		}
		name := strings.TrimPrefix(bs, "unknown constant: ")
		if mqlPrimitiveTypes[name] {
			return &DiagnosticFinding{
				RuleID:   "R09_param_name_is_type",
				Severity: sevFatalEn,
				Title:    fmt.Sprintf("Parameter name '%s' is an MQL type keyword — likely parser bug", name),
				Detail:   fmt.Sprintf("The compiler resolved a parameter name as '%s' (an MQL primitive type). This usually means tree-sitter's findIdent captured the type instead of the variable name in an extern/input declaration. The actual variable has no value → defaults to 0.", name),
				Suggest:  fmt.Sprintf("Check extern/input declarations in the EA source. If using 'input %s VarName = value;', the parser may need a fix for that declaration style.", name),
			}
		}
	}
	return nil
}

// ── Rule 8: Indicator mode parameter + missing constant ──────────────

type ruleIndicatorModeMissing struct{}

func (ruleIndicatorModeMissing) ID() string { return "R08_indicator_mode" }

func (ruleIndicatorModeMissing) Match(input RuleInput) *DiagnosticFinding {
	// Check for MODE_ constants used in indicator calls
	modeConsts := []string{"MODE_MAIN", "MODE_SIGNAL", "MODE_PLUSDI", "MODE_MINUSDI",
		"MODE_UPPER", "MODE_LOWER", "MODE_TENKAN", "MODE_KIJUN",
		"MODE_SENKOUA", "MODE_SENKOUB", "MODE_CHIKOU"}
	sourceUpper := strings.ToUpper(input.Source)
	var missing []string
	for _, c := range modeConsts {
		if strings.Contains(sourceUpper, c) {
			if _, ok := interp.LookupMQLConstant(c); !ok {
				missing = append(missing, c)
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return &DiagnosticFinding{
		RuleID:   "R08_indicator_mode",
		Severity: sevFatalEn,
		Title:    fmt.Sprintf("Missing indicator mode constants: %s", strings.Join(missing, ", ")),
		Detail:   fmt.Sprintf("EA uses indicator mode constants that are not defined: %s. These will resolve to 0, causing indicators to return the wrong line.", strings.Join(missing, ", ")),
		Suggest:  fmt.Sprintf("Add %s to constants.go with correct MQL values.", strings.Join(missing, ", ")),
	}
}
