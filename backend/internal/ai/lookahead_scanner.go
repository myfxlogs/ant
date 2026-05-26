// Package ai provides AI strategy quality gates (M10-BASE-E1 through E6).
//
// LookAhead Gate (E1): Scans DSL expressions for future-looking references that
// would cheat in backtest. Detects patterns like close[t+1], ref(close,-1),
// high[t+delta] where delta > 0, and ternary conditions on future values.
package ai

import (
	"regexp"
	"strings"
)

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
	Passed     bool                  `json:"passed"`
	Violations []LookAheadViolation  `json:"violations,omitempty"`
	Expression string                `json:"expression"`
}

// LookAheadScanner detects future references in DSL expressions.
type LookAheadScanner struct {
	patterns []futurePattern
}

type futurePattern struct {
	re      *regexp.Regexp
	msg     string
}

// NewLookAheadScanner creates a scanner with standard future-reference patterns.
func NewLookAheadScanner() *LookAheadScanner {
	return &LookAheadScanner{
		patterns: []futurePattern{
			// close[t+delta] or close[t+1] — explicit future index
			{re: regexp.MustCompile(`\w+\[t\s*\+\s*(\d+)\]`), msg: "explicit future index: $0"},
			// ref(source, -delta) where delta is negative offset (lookahead)
			{re: regexp.MustCompile(`ref\s*\(\s*\w+\s*,\s*(-\d+)\s*\)`), msg: "negative ref offset (future peek): $0"},
			// high[t+delta], low[t+delta], open[t+delta]
			{re: regexp.MustCompile(`(?:high|low|open)\s*\[\s*t\s*\+\s*\d+\s*\]`), msg: "future OHLC reference: $0"},
			// Ternary where condition references future: close[t+1] > close ? ... : ...
			{re: regexp.MustCompile(`\w+\[\s*t\s*\+\s*[1-9]\d*\s*\]`), msg: "future array index: $0"},
		},
	}
}

// Scan checks an expression for lookahead bias.
func (s *LookAheadScanner) Scan(expression string) LookAheadResult {
	result := LookAheadResult{
		Passed:     true,
		Expression: expression,
	}

	normalized := strings.ReplaceAll(expression, " ", "")

	for _, p := range s.patterns {
		matches := p.re.FindAllStringSubmatchIndex(normalized, -1)
		for _, m := range matches {
			// Extract line/col from original expression.
			line := 1
			col := m[0]
			// Count newlines to get line number.
			for i := 0; i < m[0] && i < len(expression); i++ {
				if expression[i] == '\n' {
					line++
					col -= i + 1
				}
			}

			// Check if the offset is actually positive (future).
			pattern := expression[m[0]:m[1]]
			if s.isNegativeRef(pattern, m, normalized) {
				continue // skip legitimate past references
			}
			if s.isGtComparison(pattern, m, expression) {
				continue // skip comparisons like close > close[1]
			}

			violation := LookAheadViolation{
				Pattern:  pattern,
				Line:     line,
				Column:   col,
				Severity: "error",
				Message:  strings.Replace(p.msg, "$0", pattern, 1),
			}
			result.Violations = append(result.Violations, violation)
			result.Passed = false
		}
	}

	return result
}

// isNegativeRef checks if a ref() pattern has a positive offset (legitimate past ref).
func (s *LookAheadScanner) isNegativeRef(pattern string, match []int, expression string) bool {
	// ref(source, -delta): if delta is negative → future peek
	// Already matched by the negative ref regex, so this is a pass-through.
	return false
}

// isGtComparison checks if the match is part of a legitimate greater-than comparison.
func (s *LookAheadScanner) isGtComparison(pattern string, match []int, expression string) bool {
	// close[t+1] > close  → this IS lookahead (comparing future close to current)
	// close > close[1]    → this is OK (comparing current to past)
	// Our patterns already only match t+N with N>0, so no false positives on past refs.
	return false
}

// HasLookahead is a convenience function that returns true if the expression
// contains any future-looking references.
func HasLookahead(expression string) bool {
	s := NewLookAheadScanner()
	return !s.Scan(expression).Passed
}
