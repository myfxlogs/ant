// code_compliance.go: structural quality checks for AI-generated Go strategy code.
//
// Checks for Go SDK strategy compliance: interface implementation, parameter
// usage, decimal types, and structural quality issues.

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

// StructuralWarnings runs structural code-quality checks on Go strategy code.
func StructuralWarnings(code string) []string {
	var warns []string

	// 1. ctx.Param declared but not used in struct fields
	paramRe := regexp.MustCompile(`ctx\.Param\(\s*["'](\w+)["']`)
	structFieldRe := regexp.MustCompile(`(?m)^\s+(\w+)\s+(int|float64|string|bool|decimal\.Decimal)`)

	declaredParams := map[string]bool{}
	for _, m := range paramRe.FindAllStringSubmatch(code, -1) {
		declaredParams[m[1]] = true
	}
	definedFields := map[string]bool{}
	for _, m := range structFieldRe.FindAllStringSubmatch(code, -1) {
		definedFields[m[1]] = true
	}

	// 2. Check for float64 usage in monetary calculations
	if strings.Contains(code, "float64") && (strings.Contains(code, "price") || strings.Contains(code, "volume") || strings.Contains(code, "balance")) {
		warns = append(warns, "金额计算使用了 float64 — 应使用 decimal.Decimal 避免精度丢失")
	}

	// 3. Stop-loss/take-profit = 0 in signal
	if strings.Contains(code, "StopLoss:") || strings.Contains(code, "TakeProfit:") {
		if strings.Contains(code, "decimal.Zero") || strings.Contains(code, "StopLoss: 0") || strings.Contains(code, "TakeProfit: 0") {
			warns = append(warns, "止损或止盈设为 0 — 持仓时必须返回实际的止损止盈价格")
		}
	}

	// 4. Missing OnDeinit
	if !strings.Contains(code, "func (s *MyStrategy) OnDeinit") && !regexp.MustCompile(`func \(\w+ \*\w+\) OnDeinit`).MatchString(code) {
		warns = append(warns, "缺少 OnDeinit 方法 — 必须实现完整的 sdk.Strategy 接口")
	}

	// 5. Hardcoded numeric values matching ctx.Param defaults
	for name := range declaredParams {
		defRe := regexp.MustCompile(`ctx\.Param\(\s*["']` + regexp.QuoteMeta(name) + `["']\s*,\s*(\d+(?:\.\d+)?)`)
		match := defRe.FindStringSubmatch(code)
		if match == nil {
			continue
		}
		defaultVal := match[1]
		hardcodedRe := regexp.MustCompile(`(?m)(?:decimal\.NewFromFloat|decimal\.NewFromString|decimal\.NewFromInt)\(\s*` + regexp.QuoteMeta(defaultVal) + `\s*\)`)
		if hardcodedRe.MatchString(code) {
			warns = append(warns, "参数硬编码: "+name+" 的默认值 "+defaultVal+" 直接写死在代码中，应使用变量")
		}
	}

	return warns
}

// HasRequiredSignature checks that the code defines a valid Go SDK strategy.
// Accepts any type name implementing sdk.Strategy (OnInit/OnBar/OnDeinit).
func HasRequiredSignature(code string) (bool, []string) {
	var missing []string
	hasOnInit := regexp.MustCompile(`func\s+\(\w+\s+\*\w+\)\s+OnInit\s*\(`).MatchString(code)
	hasOnBar := regexp.MustCompile(`func\s+\(\w+\s+\*\w+\)\s+OnBar\s*\(`).MatchString(code)
	if !hasOnInit {
		missing = append(missing, "缺少 OnInit(ctx sdk.Context) error 方法")
	}
	if !hasOnBar {
		missing = append(missing, "缺少 OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) 方法")
	}
	return len(missing) == 0, missing
}
