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

// CodeComplianceScanner checks generated code against all rules (sandbox + code quality).
type CodeComplianceScanner struct {
	rules []ComplianceRule
}

// NewCodeComplianceScanner creates a scanner with all compliance rules.
func NewCodeComplianceScanner() *CodeComplianceScanner {
	return &CodeComplianceScanner{rules: complianceRules}
}

// StructuralWarnings runs structural code-quality checks beyond regex sandbox rules.
// These detect undefined variables, missing protections, and position sizing errors.
func (s *CodeComplianceScanner) StructuralWarnings(code string) []string {
	var warns []string

	// 1. @param declared but not used (no variable definition or context.get)
	paramRe := regexp.MustCompile(`#\s*@param\s+(\w+)`)
	contextGetRe := regexp.MustCompile(`context\.get\(['"](\w+)['"]`)
	variableDefRe := regexp.MustCompile(`(?m)^(\w+)\s*=\s*`)

	declaredParams := map[string]bool{}
	for _, m := range paramRe.FindAllStringSubmatch(code, -1) {
		declaredParams[m[1]] = true
	}
	usedViaContext := map[string]bool{}
	for _, m := range contextGetRe.FindAllStringSubmatch(code, -1) {
		usedViaContext[m[1]] = true
	}
	definedVars := map[string]bool{}
	for _, m := range variableDefRe.FindAllStringSubmatch(code, -1) {
		definedVars[m[1]] = true
	}
	for name := range declaredParams {
		if !usedViaContext[name] && !definedVars[name] {
			warns = append(warns, "未定义变量: "+name+" — @param 声明了但代码中未从 context.get() 读取，也未定义同名变量，会导致 NameError")
		}
	}

	// 2. @strategy entryPct not read from context
	strategyRe := regexp.MustCompile(`#\s*@strategy\s+(\w+)`)
	for _, m := range strategyRe.FindAllStringSubmatch(code, -1) {
		name := m[1]
		if name == "tradeDirection" || name == "leverage" {
			continue // these are engine-level, not code-level
		}
		if !usedViaContext[name] && !definedVars[name] {
			warns = append(warns, "未定义变量: "+name+" — @strategy 声明了但未从 context.get() 读取")
		}
	}

	// 3. stop_loss/take_profit = 0 in hold path
	if strings.Contains(code, "'signal': 'hold'") || strings.Contains(code, "\"signal\": \"hold\"") {
		if strings.Contains(code, "stop_loss': 0.0") || strings.Contains(code, "stop_loss\": 0.0") {
			warns = append(warns, "持有时止损为 0：当 position is not None 时应返回实际的止损价格，不能设为 0")
		}
	}

	// 4. Position sizing using current balance instead of initial_balance
	if strings.Contains(code, "context.get('balance'") || strings.Contains(code, "context['balance']") {
		if strings.Contains(code, "volume") || strings.Contains(code, "手数") || strings.Contains(code, "lot") {
			warns = append(warns, "仓位大小使用了当前余额 context['balance']，应该使用 context.get('initial_balance') 避免每根 bar 重复计算")
		}
	}

	// 5. Hardcoded values matching @param defaults
	for name := range declaredParams {
		defRe := regexp.MustCompile(`#\s*@param\s+` + regexp.QuoteMeta(name) + `\s+(\d+(?:\.\d+)?)`)
		match := defRe.FindStringSubmatch(code)
		if match == nil {
			continue
		}
		defaultVal := match[1]
		// Check if the default value is hardcoded literally (not via context.get or variable)
		hardcodedRe := regexp.MustCompile(`(?m)(?:calc_ema|calc_sma|sma|ema|period|len)\s*\(\s*\w+\s*,\s*` + regexp.QuoteMeta(defaultVal) + `\s*\)`)
		if hardcodedRe.MatchString(code) && !usedViaContext[name] {
			warns = append(warns, "参数硬编码: "+name+" 的默认值 "+defaultVal+" 直接写死在代码中，应使用变量 "+name)
		}
	}

	return warns
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

// HasRequiredSignature checks that the code defines a valid entry point.
// Accepts both legacy signal-dict (def run(context)) and SDK (class X(StrategyBase)).
func (s *CodeComplianceScanner) HasRequiredSignature(code string) (bool, []string) {
	var missing []string
	hasRun := regexp.MustCompile(`def\s+run\s*\(`).MatchString(code)
	hasSDK := regexp.MustCompile(`class\s+\w+\s*\(.*StrategyBase\s*\)`).MatchString(code)
	if !hasRun && !hasSDK {
		missing = append(missing, "缺少 run(context) 函数或 SDK 策略类 (class X(StrategyBase))")
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
