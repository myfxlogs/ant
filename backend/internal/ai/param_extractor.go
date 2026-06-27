// param_extractor.go: parse ctx.Param() calls from Go strategy code.
// Extracts tunable parameters with defaults for grid/random search in Smart Tuning.

package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TunableParam represents a single parameter extracted from strategy code.
type TunableParam struct {
	Name    string
	Type    string  // "int", "float", or "choice"
	Default float64
	Min     float64
	Max     float64
	Step    float64
	Choices []string
}

// paramCallPattern matches: ctx.Param("name", default)
var paramCallPattern = regexp.MustCompile(`ctx\.Param\(\s*["'](\w+)["']\s*,\s*([\d.]+)\s*\)`)

// ExtractParams parses ctx.Param() calls from Go strategy code.
// Returns nil if no parameters are found.
func ExtractParams(code string) []TunableParam {
	matches := paramCallPattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]TunableParam, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		p := TunableParam{Name: name}
		d, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		p.Default = d
		p.Type = paramType(d)
		p.Min = d
		p.Max = d
		p.Step = 0
		out = append(out, p)
	}
	return out
}

// paramType returns "int" if the value is integer-like, otherwise "float".
func paramType(val float64) string {
	if val == float64(int64(val)) {
		return "int"
	}
	return "float"
}

// HasTunableParams checks if the code contains any ctx.Param() calls.
func HasTunableParams(code string) bool {
	return strings.Contains(code, "ctx.Param(")
}

// ApplyOverrides injects parameter values into Go strategy code by replacing
// the default value in ctx.Param() calls.
func ApplyOverrides(code string, overrides map[string]interface{}) string {
	if len(overrides) == 0 {
		return code
	}
	result := code
	for name, val := range overrides {
		re := regexp.MustCompile(`(ctx\.Param\(\s*["']` + regexp.QuoteMeta(name) + `["']\s*,\s*)[\d.]+(\s*\))`)
		result = re.ReplaceAllString(result, "${1}"+formatGoValue(val)+"${2}")
	}
	return result
}

// formatGoValue converts a value to a Go-compatible literal string.
func formatGoValue(v interface{}) string {
	switch val := v.(type) {
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}
