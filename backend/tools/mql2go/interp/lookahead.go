package interp

import (
	"fmt"
	"strconv"
	"strings"
)

// LookaheadViolation describes a detected future-data access in the strategy.
type LookaheadViolation struct {
	Function  string // "Close", "iMA", etc.
	ShiftExpr string // the shift expression as source text
	ShiftVal  int    // evaluated shift value (0 if unknown)
	IsLiteral bool   // true if shift was a compile-time constant
	Severity  string // SeverityFatal for definite, SeverityWarning for potential
	Message   string // human-readable description
}

// indicatorShiftArgIdx maps indicator function names to the position of the
// shift argument in their argument list (0-indexed). The shift is the last
// parameter in all MQL4/MQL5 indicator functions.
var indicatorShiftArgIdx = map[string]int{
	// Standard indicators
	"iMA": 6, "iRSI": 4, "iATR": 3, "iBands": 7, "iBollinger": 7,
	"iMACD": 7, "iStochastic": 8, "iCCI": 4, "iADX": 5, "iMomentum": 4,
	"iWPR": 3, "iMFI": 3, "iOBV": 3, "iSAR": 4, "iStdDev": 6,
	// Extended indicators (MQL4/MQL5 shared)
	"iAlligator": 11, "iIchimoku": 6, "iEnvelopes": 8, "iDeMarker": 3,
	"iOsMA": 6, "iRVI": 4, "iForce": 5, "iFractals": 3, "iGator": 11,
	"iAC": 2, "iAD": 2, "iAO": 2, "iBearsPower": 4, "iBullsPower": 4,
	"iBWMFI": 2,
	// MQL5-only
	"iAMA": 7, "iDEMA": 5, "iTEMA": 5, "iFrAMA": 5, "iVIDyA": 7,
	"iTriX": 5, "iADXWilder": 5, "iChaikin": 4, "iVolumes": 2,
	// Market data access
	"iClose": 2, "iOpen": 2, "iHigh": 2, "iLow": 2, "iTime": 2, "iVolume": 2,
	// OnArray variants
	"iMAOnArray": 5, "iRSIOnArray": 3, "iATROnArray": 3, "iBandsOnArray": 6,
	"iStdDevOnArray": 5, "iMomentumOnArray": 3, "iCCIOnArray": 3, "iMACDOnArray": 6,
}

// seriesNames are the MQL4 predefined series variables accessed via subscript.
var seriesNames = map[string]bool{
	"Close": true, "Open": true, "High": true, "Low": true,
	"Volume": true, "Time": true,
}

// DetectLookahead performs a static analysis pass over the IR to find
// future-data access patterns. It checks:
//   - Series subscript with negative shift: Close[-1], High[-2], etc.
//   - Indicator calls with negative shift parameter: iMA(..., -1), etc.
//
// A shift is "definite lookahead" when it can be evaluated to a negative
// constant at compile time. A non-constant shift expression is "potential
// lookahead" (warning level) since we cannot rule out negative values.
func DetectLookahead(ir *IR) []LookaheadViolation {
	var violations []LookaheadViolation
	seen := make(map[string]bool) // deduplicate by function+shiftExpr

	visit := func(e *Expr) {
		switch e.Kind {
		case ExprSubscript:
			if seriesNames[e.Name] {
				v := checkShiftExpr(e.Name, e.Index)
				if v != nil && !seen[v.Function+":"+v.ShiftExpr] {
					seen[v.Function+":"+v.ShiftExpr] = true
					violations = append(violations, *v)
				}
			}
		case ExprCall:
			shiftIdx, ok := indicatorShiftArgIdxByName(e.Name)
			if ok && shiftIdx < len(e.Args) {
				v := checkShiftExpr(e.Name, &e.Args[shiftIdx])
				if v != nil && !seen[v.Function+":"+v.ShiftExpr] {
					seen[v.Function+":"+v.ShiftExpr] = true
					violations = append(violations, *v)
				}
			}
		}
	}

	walkIR(ir, visit)
	return violations
}

// indicatorShiftArgIdxByName returns the shift argument index for a given
// indicator function name, or (0, false) if not an indicator.
func indicatorShiftArgIdxByName(name string) (int, bool) {
	idx, ok := indicatorShiftArgIdx[name]
	return idx, ok
}

// checkShiftExpr evaluates the shift expression and returns a violation if
// it represents a lookahead (negative shift) or potential lookahead.
func checkShiftExpr(funcName string, shiftExpr *Expr) *LookaheadViolation {
	if shiftExpr == nil {
		return nil
	}

	shiftText := exprToText(shiftExpr)
	shiftVal, ok := evalConstInt(shiftExpr)

	if ok {
		if shiftVal < 0 {
			return &LookaheadViolation{
				Function:  funcName,
				ShiftExpr: shiftText,
				ShiftVal:  shiftVal,
				IsLiteral: true,
				Severity:  SeverityFatal,
				Message:   fmt.Sprintf("%s: negative shift %d accesses future bar data", funcName, shiftVal),
			}
		}
		// Non-negative constant shift — fine.
		return nil
	}

	// Non-constant shift expression — we can't prove it's non-negative.
	// Only flag if the expression contains a subtraction or negation that
	// could produce a negative value.
	if couldBeNegative(shiftExpr) {
		return &LookaheadViolation{
			Function:  funcName,
			ShiftExpr: shiftText,
			ShiftVal:  0,
			IsLiteral: false,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("%s: shift expression '%s' could be negative (potential lookahead)", funcName, shiftText),
		}
	}

	return nil
}

// evalConstInt attempts to evaluate an expression to a constant int.
// Returns (value, true) if the expression is a compile-time integer constant.
func evalConstInt(e *Expr) (int, bool) {
	if e == nil {
		return 0, false
	}
	switch e.Kind {
	case ExprLiteral:
		switch e.Val.Kind {
		case ValInt:
			return int(e.Val.Int), true
		case ValDecimal:
			if e.Val.Decimal.IsInteger() {
				return int(e.Val.Decimal.IntPart()), true
			}
		}
	case ExprConst:
		// MQL constants (OP_BUY, MODE_SIGNAL, etc.) resolve to int values
		// via the IR's Enums map, but we don't have access to it here.
		// Constants are never used as shift values in practice.
		return 0, false
	case ExprUnary:
		if e.Op == "-" {
			v, ok := evalConstInt(&e.Args[0])
			if ok {
				return -v, true
			}
		}
		if e.Op == "+" {
			return evalConstInt(&e.Args[0])
		}
	case ExprBinary:
		switch e.Op {
		case "+", "-":
			l, lok := evalConstInt(&e.Args[0])
			r, rok := evalConstInt(&e.Args[1])
			if lok && rok {
				if e.Op == "+" {
					return l + r, true
				}
				return l - r, true
			}
		case "*":
			l, lok := evalConstInt(&e.Args[0])
			r, rok := evalConstInt(&e.Args[1])
			if lok && rok {
				return l * r, true
			}
		}
	}
	return 0, false
}

// couldBeNegative checks if an expression contains operators that could
// produce a negative result (subtraction, negation).
func couldBeNegative(e *Expr) bool {
	if e == nil {
		return false
	}
	switch e.Kind {
	case ExprUnary:
		if e.Op == "-" {
			return true
		}
	case ExprBinary:
		if e.Op == "-" {
			return true
		}
		// Recurse into sub-expressions
		for i := range e.Args {
			if couldBeNegative(&e.Args[i]) {
				return true
			}
		}
	case ExprVar:
		// A bare variable as shift could be negative at runtime.
		// Only flag if it's not a known parameter (parameters are typically
		// non-negative, but we can't verify without the params list).
		return true
	}
	return false
}

// exprToText renders an expression to a readable string for diagnostics.
func exprToText(e *Expr) string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case ExprLiteral:
		switch e.Val.Kind {
		case ValInt:
			return strconv.FormatInt(int64(e.Val.Int), 10)
		case ValDecimal:
			return e.Val.Decimal.String()
		case ValString:
			return strconv.Quote(e.Val.Str)
		case ValBool:
			if e.Val.Bool {
				return "true"
			}
			return "false"
		}
		return ""
	case ExprVar, ExprConst:
		return e.Name
	case ExprUnary:
		return e.Op + exprToText(&e.Args[0])
	case ExprBinary:
		return exprToText(&e.Args[0]) + " " + e.Op + " " + exprToText(&e.Args[1])
	case ExprCall:
		args := make([]string, 0, len(e.Args))
		for i := range e.Args {
			args = append(args, exprToText(&e.Args[i]))
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")"
	case ExprSubscript:
		return e.Name + "[" + exprToText(e.Index) + "]"
	case ExprTernary:
		return exprToText(e.Cond) + " ? " + exprToText(e.ThenExpr) + " : " + exprToText(e.ElseExpr)
	}
	return ""
}
