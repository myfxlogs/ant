// Package ai — code compliance checker for AI-generated strategies (M5-5).
// Validates that generated Python code passes sandbox scan and doesn't
// contain dangerous patterns before being sent to strategy-service.
package ai

import (
	"regexp"
	"strings"
)

// CodeComplianceResult is the output of an AI code compliance check.
type CodeComplianceResult struct {
	Passed     bool     `json:"passed"`
	Violations []string `json:"violations,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// bannedPatterns are Go regexps for patterns banned in AI-generated strategy code.
var bannedPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`import\s+os\b`), "banned import: os"},
	{regexp.MustCompile(`import\s+subprocess\b`), "banned import: subprocess"},
	{regexp.MustCompile(`import\s+sys\b`), "banned import: sys"},
	{regexp.MustCompile(`import\s+socket\b`), "banned import: socket"},
	{regexp.MustCompile(`import\s+requests\b`), "banned import: requests"},
	{regexp.MustCompile(`import\s+shutil\b`), "banned import: shutil"},
	{regexp.MustCompile(`import\s+pickle\b`), "banned import: pickle"},
	{regexp.MustCompile(`from\s+os\s+import`), "banned from-import: os"},
	{regexp.MustCompile(`from\s+subprocess\s+import`), "banned from-import: subprocess"},
	{regexp.MustCompile(`\beval\s*\(`), "banned builtin: eval()"},
	{regexp.MustCompile(`\bexec\s*\(`), "banned builtin: exec()"},
	{regexp.MustCompile(`\bcompile\s*\(`), "banned builtin: compile()"},
	{regexp.MustCompile(`__import__\s*\(`), "banned: __import__()"},
	{regexp.MustCompile(`\bopen\s*\(`), "potentially unsafe: open()"},
}

// CheckCode performs a Go-level compliance scan of AI-generated Python strategy code.
// This is a fast pre-check before sending code to the Python sandbox scan.
func CheckCode(code string) *CodeComplianceResult {
	result := &CodeComplianceResult{Passed: true}

	if len(code) > 10000 {
		result.Violations = append(result.Violations, "code too long (>10000 chars)")
		result.Passed = false
		return result
	}

	// Check syntax: must contain at least one function definition
	if !strings.Contains(code, "def ") {
		result.Warnings = append(result.Warnings, "no function definition found")
	}

	for _, bp := range bannedPatterns {
		if bp.pattern.MatchString(code) {
			result.Violations = append(result.Violations, bp.desc)
		}
	}

	if len(result.Violations) > 0 {
		result.Passed = false
	}
	return result
}
