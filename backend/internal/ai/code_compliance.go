// code_compliance.go: structural quality checks for AI-generated strategy code.
//
// Security scanning (banned imports, dangerous builtins) is done by
// Python scan_security() — the single source of truth.  This file only
// contains Go-specific structural quality checks not covered by Python.

package ai

import (
	"regexp"
	"strings"
)

// ComplianceIssue describes a single rule violation.
type ComplianceIssue struct {
	RuleName string
	Message  string
	Severity string
	Line     int
}

// StructuralWarnings runs structural code-quality checks.
// These detect undefined variables, missing protections, and position sizing
// errors — things Python's validate_strategy_code doesn't cover.
func StructuralWarnings(code string) []string {
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
			continue
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
		hardcodedRe := regexp.MustCompile(`(?m)(?:calc_ema|calc_sma|sma|ema|period|len)\s*\(\s*\w+\s*,\s*` + regexp.QuoteMeta(defaultVal) + `\s*\)`)
		if hardcodedRe.MatchString(code) && !usedViaContext[name] {
			warns = append(warns, "参数硬编码: "+name+" 的默认值 "+defaultVal+" 直接写死在代码中，应使用变量 "+name)
		}
	}

	return warns
}

// HasRequiredSignature checks that the code defines a valid entry point.
// Accepts both legacy signal-dict (def run(context)) and SDK (class X(StrategyBase)).
func HasRequiredSignature(code string) (bool, []string) {
	var missing []string
	hasRun := regexp.MustCompile(`def\s+run\s*\(`).MatchString(code)
	hasSDK := regexp.MustCompile(`class\s+\w+\s*\(.*StrategyBase\s*\)`).MatchString(code)
	if !hasRun && !hasSDK {
		missing = append(missing, "缺少 run(context) 函数或 SDK 策略类 (class X(StrategyBase))")
	}
	return len(missing) == 0, missing
}
