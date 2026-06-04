// Package ai provides the AI strategy generation pipeline (spec/26 Phase 1).
//
// clarification.go: fuzzy keyword detection + clarification question generation.
// Detects vague user input ("稳健", "激进", "做波段") and returns targeted questions
// to narrow down strategy requirements before code generation.

package ai

import "strings"

// ClarificationRule defines a fuzzy keyword pattern and its follow-up questions.
type ClarificationRule struct {
	Keywords  []string          // fuzzy keywords (Chinese)
	Questions []string          // clarifying questions
	ParamMap  map[string]string // default parameter mappings
	Priority  int               // higher = checked first
}

// ClarificationResult is the output of the clarification check.
type ClarificationResult struct {
	NeedsClarification bool
	Questions          []string
	ParamMap           map[string]string
	MatchedKeyword     string
}

// ClarificationEngine checks user messages for vague intent and generates questions.
type ClarificationEngine struct {
	rules []ClarificationRule
}

// NewClarificationEngine creates an engine with the given rules (loaded from DB at startup).
func NewClarificationEngine(rules []ClarificationRule) *ClarificationEngine {
	return &ClarificationEngine{rules: rules}
}

// Check scans the message against all rules and returns clarification questions
// if a fuzzy keyword is matched. Returns nil result if the message is clear enough.
func (e *ClarificationEngine) Check(message string) *ClarificationResult {
	msg := strings.ToLower(message)
	for _, rule := range e.rules {
		if matchAnyKeyword(msg, rule.Keywords) {
			return &ClarificationResult{
				NeedsClarification: true,
				Questions:          rule.Questions,
				ParamMap:           rule.ParamMap,
				MatchedKeyword:     firstMatch(msg, rule.Keywords),
			}
		}
	}
	return nil
}

// matchAnyKeyword returns true if any keyword is found in the message.
func matchAnyKeyword(msg string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(msg, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// firstMatch returns the first keyword found in the message.
func firstMatch(msg string, keywords []string) string {
	for _, kw := range keywords {
		if strings.Contains(msg, strings.ToLower(kw)) {
			return kw
		}
	}
	return ""
}
