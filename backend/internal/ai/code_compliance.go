// code_compliance.go: 13-rule scanner for AI-generated strategy code (spec/26 Phase 1).
//
// Scans generated Python code for dangerous patterns before allowing backtest.
// Rules cover sandbox escapes, I/O, introspection, and structural requirements.

package ai

import (
	"regexp"
	"strings"
)

// ComplianceRule is a single check that scans Python code for a forbidden pattern.
type ComplianceRule struct {
	Name    string // human-readable rule name
	Pattern string // regex pattern to match
	Message string // message when rule is violated
	Severity string         // "block" (must fix) or "warn" (advisory)
	re       *regexp.Regexp // precompiled pattern
}

// ComplianceIssue describes a single rule violation.
type ComplianceIssue struct {
	RuleName string
	Message  string
	Severity string
	Line     int // approximate line number
}

// complianceRules defines all 13 forbidden-code rules (precompiled regex).
var complianceRules []ComplianceRule

func init() {
	raw := []ComplianceRule{
		{Name: "no_eval", Pattern: `\beval\s*\(`, Message: "禁止使用 eval()", Severity: "block"},
	{Name: "no_exec", Pattern: `\bexec\s*\(`, Message: "禁止使用 exec()", Severity: "block"},
	{Name: "no_compile", Pattern: `\bcompile\s*\(`, Message: "禁止使用 compile()", Severity: "block"},
	{Name: "no_import_dunder", Pattern: `__import__\s*\(`, Message: "禁止使用 __import__()", Severity: "block"},
	{Name: "no_os_import", Pattern: `\bimport\s+os\b`, Message: "禁止导入 os 模块", Severity: "block"},
	{Name: "no_subprocess", Pattern: `\bimport\s+subprocess\b|from\s+subprocess\b`, Message: "禁止导入 subprocess", Severity: "block"},
	{Name: "no_socket", Pattern: `\bimport\s+socket\b|from\s+socket\b`, Message: "禁止导入 socket", Severity: "block"},
	{Name: "no_pickle", Pattern: `\bimport\s+pickle\b|from\s+pickle\b`, Message: "禁止导入 pickle", Severity: "block"},
	{Name: "no_marshal", Pattern: `\bimport\s+marshal\b|from\s+marshal\b`, Message: "禁止导入 marshal", Severity: "block"},
	{Name: "no_file_open", Pattern: `\bopen\s*\(`, Message: "禁止文件 I/O (open)", Severity: "block"},
	{Name: "no_globals_locals", Pattern: `\b(globals|locals)\s*\(\s*\)`, Message: "禁止使用 globals()/locals()", Severity: "block"},
	{Name: "no_builtins_access", Pattern: `__builtins__`, Message: "禁止访问 __builtins__", Severity: "block"},
	{Name: "no_atexit_signal", Pattern: `\bimport\s+(atexit|signal)\b|from\s+(atexit|signal)\b`,
		Message: "禁止导入 atexit/signal", Severity: "block"},
}
	for i := range raw {
		raw[i].re = regexp.MustCompile(raw[i].Pattern)
		complianceRules = append(complianceRules, raw[i])
	}
}

// CodeComplianceScanner checks generated code against all 13 rules.
type CodeComplianceScanner struct {
	rules []ComplianceRule
}

// NewCodeComplianceScanner creates a scanner with the default 13 rules.
func NewCodeComplianceScanner() *CodeComplianceScanner {
	return &CodeComplianceScanner{rules: complianceRules}
}

// Scan runs all compliance rules against the given code.
// Returns blocking issues and warnings separately.
func (s *CodeComplianceScanner) Scan(code string) (blocks []ComplianceIssue, warns []ComplianceIssue) {
	lines := strings.Split(code, "\n")
	for _, rule := range s.rules {
		if loc := rule.re.FindStringIndex(code); loc != nil {
			lineNum := lineForOffset(lines, loc[0], code)
			issue := ComplianceIssue{
				RuleName: rule.Name,
				Message:  rule.Message,
				Severity: rule.Severity,
				Line:     lineNum,
			}
			if rule.Severity == "block" {
				blocks = append(blocks, issue)
			} else {
				warns = append(warns, issue)
			}
		}
	}
	return blocks, warns
}

// HasRequiredSignature checks that the code defines run(context) — the engine contract.
func (s *CodeComplianceScanner) HasRequiredSignature(code string) (bool, []string) {
	var missing []string
	if !regexp.MustCompile(`def\s+run\s*\(`).MatchString(code) {
		missing = append(missing, "缺少 run(context) 函数")
	}
	return len(missing) == 0, missing
}

// lineForOffset returns the 1-indexed line number for a byte offset.
func lineForOffset(lines []string, offset int, full string) int {
	if offset < 0 || offset > len(full) {
		return 0
	}
	prefix := full[:offset]
	return strings.Count(prefix, "\n") + 1
}
