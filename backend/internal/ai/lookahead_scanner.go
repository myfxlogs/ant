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
	Passed     bool                 `json:"passed"`
	Violations []LookAheadViolation `json:"violations,omitempty"`
	Expression string               `json:"expression"`
}

// LookAheadScanner detects future references in DSL expressions.
type LookAheadScanner struct {
	patterns []futurePattern
}

type futurePattern struct {
	re  *regexp.Regexp
	msg string
}

// defaultScanner is a package-level singleton to avoid allocations per call in HasLookahead.
var defaultScanner = NewLookAheadScanner()

// NewLookAheadScanner creates a scanner with standard future-reference patterns.
func NewLookAheadScanner() *LookAheadScanner {
	return &LookAheadScanner{
		patterns: []futurePattern{
			// close[t+delta] or close[t+delta] — explicit future index.
			// Capture the digit to exclude t+0 (same-day reference, not lookahead).
			{re: regexp.MustCompile(`\w+\[t\s*\+\s*(\d+)\]`), msg: "explicit future index: $0"},
			// ref(source, -delta) where delta is negative offset (lookahead)
			{re: regexp.MustCompile(`ref\s*\(\s*\w+\s*,\s*(-\d+)\s*\)`), msg: "negative ref offset (future peek): $0"},
			// high[t+delta], low[t+delta], open[t+delta]
			{re: regexp.MustCompile(`(?:high|low|open)\s*\[\s*t\s*\+\s*\d+\s*\]`), msg: "future OHLC reference: $0"},
			// Ternary where condition references future: close[t+delta] where delta >= 1.
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

	// Deduplicate overlapping pattern matches by tracking matched byte ranges.
	seen := make(map[[2]int]bool)

	for _, p := range s.patterns {
		matches := p.re.FindAllStringSubmatchIndex(expression, -1)
		for _, m := range matches {
			start, end := m[0], m[1]

			// Skip if this byte range was already reported by a previous pattern.
			key := [2]int{start, end}
			if seen[key] {
				continue
			}
			seen[key] = true

			// Extract line/col from original expression.
			line := 1
			lastNewline := -1
			for i := 0; i < start && i < len(expression); i++ {
				if expression[i] == '\n' {
					line++
					lastNewline = i
				}
			}
			col := start - lastNewline - 1

			// Check if the offset is actually positive (future).
			pattern := expression[start:end]
			if isZeroOffset(pattern, m) {
				continue // skip t+0 (same-day reference, not lookahead)
			}
			if s.isNegativeRef(pattern, m, expression) {
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

// isZeroOffset checks if a matched t+delta pattern has delta=0,
// which is a same-day reference and not lookahead bias.
func isZeroOffset(pattern string, match []int) bool {
	// If we captured group 1 and it equals "0", skip.
	if len(match) >= 4 {
		// match[2], match[3] is the start/end of the first capture group.
		// The first pattern (`\w+\[t\s*\+\s*(\d+)\]`) captures the digit.
		captured := pattern[match[2]-match[0] : match[3]-match[0]]
		if captured == "0" {
			return true
		}
	}
	return false
}

// isNegativeRef checks if a ref() pattern has a positive offset (legitimate past ref).
//
// TODO: implement negative ref detection for ref(source, -delta)
// Currently always returns false (placeholder).
func (s *LookAheadScanner) isNegativeRef(pattern string, match []int, expression string) bool {
	return false
}

// isGtComparison checks if the match is part of a legitimate greater-than comparison.
//
// TODO: implement greater-than comparison detection for close > close[1]
// Currently always returns false (placeholder).
func (s *LookAheadScanner) isGtComparison(pattern string, match []int, expression string) bool {
	return false
}

// HasLookahead is a convenience function that returns true if the expression
// contains any future-looking references. Uses the package-level singleton scanner.
func HasLookahead(expression string) bool {
	return !defaultScanner.Scan(expression).Passed
}
