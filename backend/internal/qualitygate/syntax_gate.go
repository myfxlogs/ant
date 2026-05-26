package qualitygate

import (
	"context"
	"strings"
)

// SyntaxGate validates strategy code for basic correctness.
// It checks: non-empty code, import validity, syntax markers.
type SyntaxGate struct {
	// RequiredImports is a list of packages that must be imported.
	RequiredImports []string
	// ForbiddenPatterns are regex-like strings that should not appear.
	ForbiddenPatterns []string
	// MinCodeLength is the minimum number of characters required.
	MinCodeLength int
}

func (g *SyntaxGate) Name() string { return "syntax" }

func (g *SyntaxGate) Evaluate(_ context.Context, info *StrategyInfo) *GateResult {
	if g.MinCodeLength <= 0 {
		g.MinCodeLength = 10
	}

	result := &GateResult{
		Gate:   g.Name(),
		Passed: true,
		Score:  1.0,
		Details: map[string]interface{}{},
	}

	code := strings.TrimSpace(info.Code)
	if len(code) < g.MinCodeLength {
		result.Passed = false
		result.Severity = SeverityCritical
		result.Score = 0
		result.Reason = "code is too short or empty"
		return result
	}

	codeLower := strings.ToLower(code)
	warnings := make([]string, 0)

	// Check required imports
	for _, imp := range g.RequiredImports {
		if !strings.Contains(codeLower, strings.ToLower(imp)) {
			warnings = append(warnings, "missing import: "+imp)
		}
	}

	// Check forbidden patterns
	for _, pat := range g.ForbiddenPatterns {
		if strings.Contains(codeLower, strings.ToLower(pat)) {
			result.Passed = false
			result.Severity = SeverityError
			result.Reason = "forbidden pattern found: " + pat
			result.Score = 0.3
			return result
		}
	}

	// Check for basic structure markers
	hasFunctionDef := strings.Contains(code, "def ") || strings.Contains(code, "void ") || strings.Contains(code, "function ")
	hasReturn := strings.Contains(codeLower, "return")
	hasEntry := strings.Contains(codeLower, "onbar") || strings.Contains(codeLower, "ontick") ||
		strings.Contains(codeLower, "on_calculate") || strings.Contains(codeLower, "next(")

	if !hasFunctionDef && !hasEntry {
		warnings = append(warnings, "no function definition or entry point found")
		result.Score -= 0.1
	}
	if !hasReturn && !hasEntry {
		warnings = append(warnings, "no return statement or entry point found")
		result.Score -= 0.1
	}

	if len(warnings) > 0 {
		result.Warnings = warnings
		if result.Score < 0.7 {
			result.Severity = SeverityWarning
		}
	}

	result.Details["code_length"] = len(code)
	result.Details["language"] = info.Language
	result.Details["warnings"] = len(warnings)

	return result
}
