// Package ai provides AI strategy quality gates (M10-BASE-E1 through E6).
//
// LookAhead Gate (E1): Scans DSL expressions for future-looking references that
// would cheat in backtest. Detects patterns like close[t+1], ref(close,-1),
// high[t+delta] where delta > 0, and ternary conditions on future values.
package ai

import "regexp"

// LookAheadViolation describes a detected future-looking reference.
type LookAheadViolation struct {
	Pattern  string // the matched future-reference pattern
	Line     int    // line number (0 if not applicable)
	Column   int    // column offset
	Severity string // "error" or "warning"
	Message  string // human-readable description
}

// LookAheadResult is the outcome of scanning an expression for lookahead bias.
type LookAheadResult struct {
	Passed     bool
	Violations []LookAheadViolation
	Expression string
}

// LookAheadScanner detects future references in DSL expressions.
type LookAheadScanner struct {
	patterns []futurePattern
}

type futurePattern struct {
	re  *regexp.Regexp
	msg string
}

// defaultScanner is a package-level singleton to avoid allocations per call.
var defaultScanner = NewLookAheadScanner()

// NewLookAheadScanner creates a scanner with standard future-reference patterns.
func NewLookAheadScanner() *LookAheadScanner {
	return &LookAheadScanner{
		patterns: []futurePattern{
			// close[t+delta] — explicit future index. Captures the digit.
			{re: regexp.MustCompile(`\w+\[[tT]\s*\+\s*(\d+)\]`), msg: "explicit future index: $0"},
			// ref(source, -delta) where delta is negative offset (lookahead).
			{re: regexp.MustCompile(`ref\s*\(\s*\w+\s*,\s*(-\d+)\s*\)`), msg: "negative ref offset (future peek): $0"},
			// high[t+delta], low[t+delta], open[t+delta].
			{re: regexp.MustCompile(`(?:high|low|open)\s*\[\s*[tT]\s*\+\s*\d+\s*\]`), msg: "future OHLC reference: $0"},
			// Ternary where condition references future: close[t+delta] where delta >= 1.
			{re: regexp.MustCompile(`\w+\[\s*[tT]\s*\+\s*[1-9]\d*\s*\]`), msg: "future array index: $0"},
		},
	}
}

// checkMatch evaluates a single regex match for lookahead bias.
// Returns the violation and true if it's a genuine lookahead reference.
func (s *LookAheadScanner) checkMatch(expression string, p futurePattern, match []int, start, end int) (LookAheadViolation, bool) {
	// Compute line/column for the violation report.
	line := 1
	lastNewline := -1
	for i := 0; i < start && i < len(expression); i++ {
		if expression[i] == '\n' {
			line++
			lastNewline = i
		}
	}
	col := start - lastNewline - 1

	pattern := expression[start:end]

	// Filter out t+0 (same-day reference, not lookahead bias).
	if isZeroOffset(pattern, match, start) {
		return LookAheadViolation{}, false
	}
	// Filter out legitimate past references in ref(source, +delta).
	if s.isPastRef(expression, start) {
		return LookAheadViolation{}, false
	}

	return LookAheadViolation{
		Pattern:  pattern,
		Line:     line,
		Column:   col,
		Severity: "error",
		Message:  regexp.MustCompile(`\$0`).ReplaceAllString(p.msg, pattern),
	}, true
}

// Scan checks an expression for lookahead bias.
func (s *LookAheadScanner) Scan(expression string) LookAheadResult {
	result := LookAheadResult{Passed: true, Expression: expression}
	seen := make(map[[2]int]bool)

	for _, p := range s.patterns {
		matches := p.re.FindAllStringSubmatchIndex(expression, -1)
		for _, m := range matches {
			start, end := m[0], m[1]
			key := [2]int{start, end}
			if seen[key] {
				continue
			}
			seen[key] = true
			if viol, ok := s.checkMatch(expression, p, m, start, end); ok {
				result.Violations = append(result.Violations, viol)
				result.Passed = false
			}
		}
	}
	return result
}

// isZeroOffset checks if a matched t+delta pattern has delta=0,
// which is a same-day reference and not lookahead bias.
func isZeroOffset(pattern string, match []int, matchStart int) bool {
	if len(match) >= 4 {
		// match[2], match[3] is the start/end of the first capture group.
		captured := pattern[match[2]-matchStart : match[3]-matchStart]
		if captured == "0" {
			return true
		}
	}
	return false
}

// isPastRef checks if the match occurs inside a ref(..., +N) call
// where the offset is positive (legitimate past reference).
// The negative-offset case is already caught by the dedicated ref() pattern.
func (s *LookAheadScanner) isPastRef(expression string, matchStart int) bool {
	// Search backwards for "ref(" before this match position.
	prefix := expression[max(0, matchStart-50):matchStart]
	refIdx := -1
	for i := len(prefix) - 1; i >= 3; i-- {
		if prefix[i] == '(' && i >= 3 && prefix[i-3:i+1] == "ref(" {
			refIdx = matchStart - 50 + i
			break
		}
	}
	if refIdx < 0 {
		return false
	}
	// Check if the offset after the comma is positive (past reference).
	commaIdx := -1
	for i := refIdx + 4; i < len(expression) && i < matchStart; i++ {
		if expression[i] == ',' {
			commaIdx = i
			break
		}
	}
	if commaIdx < 0 {
		return false
	}
	// Skip whitespace after comma and check sign.
	for i := commaIdx + 1; i < len(expression) && i < matchStart; i++ {
		ch := expression[i]
		if ch == ' ' || ch == '\t' {
			continue
		}
		// Positive digit or '+' means past reference, not lookahead.
		if ch >= '0' && ch <= '9' || ch == '+' {
			return true
		}
		break
	}
	return false
}

// HasLookahead is a convenience function that returns true if the expression
// contains any future-looking references. Uses the package-level singleton scanner.
func HasLookahead(expression string) bool {
	return !defaultScanner.Scan(expression).Passed
}
