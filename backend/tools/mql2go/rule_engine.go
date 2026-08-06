package mql2go

import (
	"fmt"
	"strings"

	"alphaforge/tools/mql2go/interp"
)

// DiagnosticRule is a rule in the rule engine (§12.1 D2).
// Each rule examines the EA source, coverage report, and backtest result
// to produce a diagnostic finding.
type DiagnosticRule interface {
	ID() string
	Match(input RuleInput) *DiagnosticFinding
}

// RuleInput is the data available to all rules.
type RuleInput struct {
	Source        string              // raw MQL source
	HasOnTick     bool                // true if OnTick/start() was compiled to bytecode
	Coverage      *CoverageReport     // compile-time coverage (may be nil)
	BlindSpots    []CoverageBlindSpot // merged blind spots from AnalyzeCoverage
	TotalTrades   int                 // backtest result trade count
	RuntimeBlinds []RuntimeBlindSpot  // runtime blind spots from VM
}

// RuntimeBlindSpot is a blind spot recorded during VM execution.
type RuntimeBlindSpot struct {
	Builtin  string
	Severity string
	Count    int
}

// DiagnosticFinding is the output of a matched rule.
type DiagnosticFinding struct {
	RuleID   string
	Severity string // "fatal" | "warning" | "info"
	Title    string
	Detail   string
	Suggest  string
}

// RuleEngine runs all diagnostic rules against the input.
type RuleEngine struct {
	rules []DiagnosticRule
}

// NewRuleEngine creates a rule engine with the default rule set.
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: []DiagnosticRule{
			ruleZeroTradesOrderSend{},
			ruleStartEntryNotMapped{},
			ruleMACDModeSignal{},
			ruleOrderTypeMapping{},
			ruleICustomBlindSpot{},
			ruleOrderSelectHistory{},
			ruleOrderProfitOpenPos{},
			ruleIndicatorModeMissing{},
			ruleParamNameIsType{},
		},
	}
}

// Run executes all rules and returns matched findings.
func (e *RuleEngine) Run(input RuleInput) []DiagnosticFinding {
	var findings []DiagnosticFinding
	for _, r := range e.rules {
		if f := r.Match(input); f != nil {
			findings = append(findings, *f)
		}
	}
	return findings
}

// ── Rule 1: Zero trades + EA contains OrderSend/Buy/Sell ─────────────

type ruleZeroTradesOrderSend struct{}

func (ruleZeroTradesOrderSend) ID() string { return "R01_zero_trades_ordersend" }

func (ruleZeroTradesOrderSend) Match(input RuleInput) *DiagnosticFinding {
	if input.TotalTrades > 0 {
		return nil
	}
	sourceUpper := strings.ToUpper(input.Source)
	hasOrderSend := strings.Contains(sourceUpper, "ORDERSEND")
	hasBuy := strings.Contains(sourceUpper, "BUY")
	hasSell := strings.Contains(sourceUpper, "SELL")
	if !hasOrderSend && !hasBuy && !hasSell {
		return nil
	}
	// Check for fatal blind spots
	for _, bs := range input.BlindSpots {
		if bs.Severity == interp.SeverityFatal {
			return &DiagnosticFinding{
				RuleID:   "R01_zero_trades_ordersend",
				Severity: "fatal",
				Title:    "Zero trades despite OrderSend/Buy/Sell calls",
				Detail:   fmt.Sprintf("EA has trade calls but produced 0 trades. Fatal blind spot: %s", bs.Builtin),
				Suggest:  fmt.Sprintf("Function %s is not implemented. This blocks all trading.", bs.Builtin),
			}
		}
	}
	// Check runtime blind spots for trade functions
	for _, rbs := range input.RuntimeBlinds {
		if rbs.Severity == interp.SeverityFatal {
			return &DiagnosticFinding{
				RuleID:   "R01_zero_trades_ordersend",
				Severity: "fatal",
				Title:    "Zero trades despite OrderSend/Buy/Sell calls",
				Detail:   fmt.Sprintf("EA has trade calls but produced 0 trades. Runtime blind spot: %s (hit %d times)", rbs.Builtin, rbs.Count),
				Suggest:  fmt.Sprintf("Function %s failed at runtime. Check implementation.", rbs.Builtin),
			}
		}
	}
	return &DiagnosticFinding{
		RuleID:   "R01_zero_trades_ordersend",
		Severity: "warning",
		Title:    "Zero trades despite OrderSend/Buy/Sell calls",
		Detail:   "EA contains trade calls but backtest produced 0 trades. No fatal blind spots detected.",
		Suggest:  "Check strategy parameters (e.g. Lots), entry conditions, and indicator values.",
	}
}

// ── Rule 2: Zero trades + EA has start() entry ───────────────────────

type ruleStartEntryNotMapped struct{}

func (ruleStartEntryNotMapped) ID() string { return "R02_start_entry" }

func (ruleStartEntryNotMapped) Match(input RuleInput) *DiagnosticFinding {
	if input.TotalTrades > 0 {
		return nil
	}
	// If OnTick was compiled (from start() or OnTick), the mapping worked.
	if input.HasOnTick {
		return nil
	}
	// Check if source has start() function
	if !strings.Contains(input.Source, "start()") && !strings.Contains(input.Source, "start ()") {
		return nil
	}
	return &DiagnosticFinding{
		RuleID:   "R02_start_entry",
		Severity: "fatal",
		Title:    "Classic MQL4 start() entry not mapped to OnTick",
		Detail:   "EA uses start() as entry point but it was not mapped to OnTick. Strategy code never executes.",
		Suggest:  "Add an OnTick() wrapper or ensure the compiler maps start() → OnTick.",
	}
}

// ── Rule 3: Zero trades + iMACD + MODE_SIGNAL ────────────────────────

type ruleMACDModeSignal struct{}

func (ruleMACDModeSignal) ID() string { return "R03_macd_mode_signal" }

func (ruleMACDModeSignal) Match(input RuleInput) *DiagnosticFinding {
	sourceUpper := strings.ToUpper(input.Source)
	if !strings.Contains(sourceUpper, "IMACD") {
		return nil
	}
	if !strings.Contains(sourceUpper, "MODE_SIGNAL") {
		return nil
	}
	// MODE_SIGNAL is in constants now, but check for regressions
	v, ok := interp.LookupMQLConstant("MODE_SIGNAL")
	if !ok {
		return &DiagnosticFinding{
			RuleID:   "R03_macd_mode_signal",
			Severity: "fatal",
			Title:    "MODE_SIGNAL constant missing",
			Detail:   "EA uses iMACD with MODE_SIGNAL but the constant is not defined. iMACD signal line will be incorrect.",
			Suggest:  "Add MODE_SIGNAL=1 to constants.go.",
		}
	}
	// Check if MODE_SIGNAL resolves to correct value
	if v.Kind == interp.ValInt && v.Int != 1 {
		return &DiagnosticFinding{
			RuleID:   "R03_macd_mode_signal",
			Severity: "fatal",
			Title:    "MODE_SIGNAL has incorrect value",
			Detail:   fmt.Sprintf("MODE_SIGNAL resolves to %d, expected 1. iMACD signal line will be wrong.", v.Int),
			Suggest:  "Fix MODE_SIGNAL value to 1 in constants.go.",
		}
	}
	return nil
}

// ── Rule 4: OrderType() == OP_BUY always false ───────────────────────

type ruleOrderTypeMapping struct{}

func (ruleOrderTypeMapping) ID() string { return "R04_ordertype_mapping" }

func (ruleOrderTypeMapping) Match(input RuleInput) *DiagnosticFinding {
	sourceUpper := strings.ToUpper(input.Source)
	if !strings.Contains(sourceUpper, "ORDERTYPE") {
		return nil
	}
	if !strings.Contains(sourceUpper, "OP_BUY") && !strings.Contains(sourceUpper, "OP_SELL") {
		return nil
	}
	// The fix mapped SideBuy→0(OP_BUY), SideSell→1(OP_SELL).
	// This rule is a regression guard — if the mapping reverts,
	// OrderType()==OP_BUY will always be false for buy positions.
	if input.TotalTrades > 0 {
		return nil
	}
	// If there are trades but they never close, mapping might be wrong.
	// We can't detect this precisely without runtime tracing, so
	// only fire if zero trades + OrderType comparison present.
	return &DiagnosticFinding{
		RuleID:   "R04_ordertype_mapping",
		Severity: "warning",
		Title:    "OrderType() comparison with OP_BUY/OP_SELL detected",
		Detail:   "EA compares OrderType() to OP_BUY/OP_SELL. If the value mapping is incorrect, position management logic will fail silently.",
		Suggest:  "Verify builtinOrderType returns OP_BUY(0)/OP_SELL(1), not PositionSide(1/-1).",
	}
}

// ── Rule 5: iCustom blind spot ───────────────────────────────────────

type ruleICustomBlindSpot struct{}

func (ruleICustomBlindSpot) ID() string { return "R05_icustom" }

func (ruleICustomBlindSpot) Match(input RuleInput) *DiagnosticFinding {
	sourceUpper := strings.ToUpper(input.Source)
	if !strings.Contains(sourceUpper, "ICUSTOM") {
		return nil
	}
	return &DiagnosticFinding{
		RuleID:   "R05_icustom",
		Severity: "fatal",
		Title:    "iCustom (custom indicator) is not supported",
		Detail:   "EA uses iCustom() which calls custom indicators. This function always returns 0, so entry/exit conditions depending on it will never trigger.",
		Suggest:  "Replace custom indicator calls with standard indicators, or inline the indicator logic.",
	}
}

// ── Rule 6: OrderSelect + MODE_HISTORY ───────────────────────────────

type ruleOrderSelectHistory struct{}

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
		Severity: "warning",
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
		Severity: "info",
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
				Severity: "fatal",
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
		Severity: "fatal",
		Title:    fmt.Sprintf("Missing indicator mode constants: %s", strings.Join(missing, ", ")),
		Detail:   fmt.Sprintf("EA uses indicator mode constants that are not defined: %s. These will resolve to 0, causing indicators to return the wrong line.", strings.Join(missing, ", ")),
		Suggest:  fmt.Sprintf("Add %s to constants.go with correct MQL values.", strings.Join(missing, ", ")),
	}
}
