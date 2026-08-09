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
	Type    string // "int", "float", or "choice"
	Default float64
	Min     float64
	Max     float64
	Step    float64
	Choices []string
}

// paramCallPattern matches: ctx.Param("name", default)
var paramCallPattern = regexp.MustCompile(`ctx\.Param\(\s*["'](\w+)["']\s*,\s*([\d.]+)\s*\)`)

// mqlExternPattern matches: extern int/double/bool Name = value;
// Captures type, name, and default value. Also matches MQL5 'input' keyword.
var mqlExternPattern = regexp.MustCompile(`(?:extern|input)\s+(int|long|double|float|bool|string|color)\s+(\w+)\s*=\s*([^;]+);`)

// ExtractParams parses ctx.Param() calls from Go strategy code
// and extern/input declarations from MQL source code.
// Returns nil if no parameters are found.
func ExtractParams(code string) []TunableParam {
	seen := map[string]bool{}
	var out []TunableParam

	// Go SDK: ctx.Param("name", default)
	for _, m := range paramCallPattern.FindAllStringSubmatch(code, -1) {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		d, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		p := TunableParam{Name: name, Default: d, Type: paramType(d), Min: d, Max: d}
		out = append(out, p)
	}

	// MQL: extern/input <type> Name = value;
	for _, m := range mqlExternPattern.FindAllStringSubmatch(code, -1) {
		name := m[2]
		if seen[name] {
			continue
		}
		seen[name] = true
		typeStr := m[1]
		valStr := strings.TrimSpace(m[3])
		// Strip string literal quotes — not numeric tunable
		if typeStr == "string" || typeStr == "color" {
			continue
		}
		var def float64
		if typeStr == "bool" {
			if strings.EqualFold(valStr, "true") {
				def = 1
			} else {
				def = 0
			}
		} else {
			d, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				continue
			}
			def = d
		}
		p := TunableParam{Name: name, Default: def, Type: paramType(def), Min: def, Max: def}
		out = append(out, p)
	}

	if len(out) == 0 {
		return nil
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

// ParamTypeString is the exported version of paramType for use by other packages.
func ParamTypeString(val float64) string {
	return paramType(val)
}

// HasTunableParams checks if the code contains any tunable parameters
// from ctx.Param() calls, @param annotations, or MQL extern/input declarations.
func HasTunableParams(code string) bool {
	return strings.Contains(code, "ctx.Param(") || strings.Contains(code, "@param") ||
		strings.Contains(code, "extern ") || strings.Contains(code, "input ")
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
