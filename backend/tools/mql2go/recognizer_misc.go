package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── Execution model ─────────────────────────────────────────────────

func detectExecCST(root *sitter.Node) ExecutionModel {
	hasBarGuard := detectBarGuardCST(root)

	for _, fn := range findFunctions(root) {
		name := funcName(fn)
		if name == "OnTick" {
			return ExecutionModel{Kind: ExecOnTick, HasBarGuard: hasBarGuard}
		}
	}
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		if hasGridFlagCST(body) {
			return ExecutionModel{Kind: ExecOnInitGrid}
		}
	}
	return ExecutionModel{Kind: ExecOnBar}
}

// detectBarGuardCST checks if the source contains a new-bar guard pattern.
// Common MQL patterns that indicate the strategy only trades on new bars:
//
//	Volume[0] > 1        — MQL4: first tick of new bar has volume=1
//	Volume[0] <= 1       — MQL4: inverted check
//	Bars != prevBars      — MQL4: bar count changed
//	iBars(NULL,0) != n    — MT4: bar count comparison
//	rates_total != prev   — MT5 OnCalculate
func detectBarGuardCST(root *sitter.Node) bool {
	found := false
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "binary_expression" && n.Type() != "subscript_expression" {
			return true
		}
		text := nodeText("", n)
		// Pattern: Volume[0] > 1 or Volume[0] <= 1
		if containsPattern(text, "Volume[0]", ">", "1") ||
			containsPattern(text, "Volume[0]", "<=", "1") ||
			containsPattern(text, "Volume[0]", ">=", "1") {
			found = true
			return false
		}
		// Pattern: Bars != prevBars or Bars > prevBars
		if containsPattern(text, "Bars", "!=") || containsPattern(text, "Bars", ">") {
			found = true
			return false
		}
		// Pattern: iBars(NULL, 0) != barCount
		if containsPattern(text, "iBars", "!=") {
			found = true
			return false
		}
		// Pattern: rates_total != prev_calculated (MT5)
		if containsPattern(text, "rates_total", "!=") {
			found = true
			return false
		}
		return true
	})
	return found
}

// containsPattern checks if all substrings appear in text (simple substring match).
func containsPattern(text string, substrs ...string) bool {
	for _, s := range substrs {
		if !strings.Contains(text, s) {
			return false
		}
	}
	return true
}

func hasGridFlagCST(body *sitter.Node) bool {
	found := false
	walkCST(body, func(n *sitter.Node) bool {
		if n.Type() == "identifier" && nodeText("", n) == "gridPlaced" {
			found = true
			return false
		}
		return true
	})
	return found
}

// ── State variables ─────────────────────────────────────────────────

func extractStateCST(source string, root *sitter.Node) []StateVar {
	var state []StateVar
	for i := 0; i < int(root.ChildCount()); i++ {
		n := root.Child(i)
		if n.Type() != "declaration" && n.Type() != "field_declaration" {
			continue
		}
		text := source[n.StartByte():n.EndByte()]
		if strings.Contains(text, "extern ") || strings.Contains(text, "input ") {
			continue
		}
		if childByType(source, n, "function_declarator") != nil {
			continue
		}
		var vt string
		if pt := childByType(source, n, "primitive_type"); pt != nil {
			vt = nodeText(source, pt)
		}
		init := childByType(source, n, "init_declarator")
		if init == nil {
			continue
		}
		name := ""
		if id := childByType(source, init, "identifier"); id != nil {
			name = nodeText(source, id)
		}
		if name == "" {
			continue
		}
		state = append(state, StateVar{Name: name, GoType: vt})
	}
	return state
}

// ── Sizing ──────────────────────────────────────────────────────────

func detectSizingCST(entries []EntryRule) *SizingRule {
	k := SizingFixed
	expr := "s.lotSize"
	for _, e := range entries {
		if strings.Contains(e.Volume, "MathPow") || strings.Contains(e.Volume, "*") {
			k = SizingMartingale
			expr = "s.baseLot"
			break
		}
	}
	return &SizingRule{Kind: k, Expression: expr}
}

// ── Timer ──────────────────────────────────────────────────────────

func detectTimerCST(root *sitter.Node) *TimerRule {
	var timer *TimerRule
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		if name == "EventSetTimer" || name == "EventSetMillisecondTimer" {
			secs := 300
			if name == "EventSetMillisecondTimer" {
				secs = 5
			}
			if name == "EventSetTimer" {
				args := childByType("", n, "argument_list")
				if args != nil {
					named := getNamedChildren(args)
					if len(named) > 0 {
						if nl := childByType("", named[0], "number_literal"); nl != nil {
							secs = parseInt(nodeText("", nl))
						}
					}
				}
			}
			timer = &TimerRule{IntervalSeconds: secs}
			return false
		}
		return true
	})
	return timer
}
