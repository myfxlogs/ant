// param_extractor.go: parse @param annotations from Python strategy code.
// Extracts tunable parameters with ranges for grid/random search in Smart Tuning.

package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TunableParam represents a single @param annotation extracted from strategy code.
type TunableParam struct {
	Name    string
	Type    string  // "int", "float", or "choice"
	Default float64
	Min     float64
	Max     float64
	Step    float64
	Choices []string
}

// paramPattern matches: @param name default_value range=min:max:step
var paramPattern = regexp.MustCompile(`@param\s+(\w+)\s+([\d.]+)(?:\s+range=([\d.]+):([\d.]+):([\d.]+))?`)

// ExtractParams parses @param annotations from Python strategy code.
// Returns nil if no annotations are found.
func ExtractParams(code string) []TunableParam {
	matches := paramPattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]TunableParam, 0, len(matches))
	for _, m := range matches {
		p := TunableParam{Name: m[1]}
		d, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		p.Default = d

		if m[3] != "" {
			// Has range=min:max:step
			p.Min, _ = strconv.ParseFloat(m[3], 64)
			p.Max, _ = strconv.ParseFloat(m[4], 64)
			p.Step, _ = strconv.ParseFloat(m[5], 64)
			p.Type = paramType(p.Min, p.Max, p.Step)
		} else {
			p.Type = "float"
			p.Min = d
			p.Max = d
			p.Step = 0
		}
		out = append(out, p)
	}
	return out
}

// paramType returns "int" if the range and step are integer-like, otherwise "float".
func paramType(min, max, step float64) string {
	if step == float64(int64(step)) &&
		min == float64(int64(min)) &&
		max == float64(int64(max)) {
		return "int"
	}
	return "float"
}

// HasTunableParams checks if the code contains any @param annotations.
func HasTunableParams(code string) bool {
	return strings.Contains(code, "@param")
}

// ApplyOverrides injects parameter values into strategy code by replacing
// @param annotations with explicit variable assignments.
func ApplyOverrides(code string, overrides map[string]interface{}) string {
	if len(overrides) == 0 {
		return code
	}
	lines := strings.Split(code, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		cleaned := strings.TrimPrefix(trimmed, "# ")
		cleaned = strings.TrimPrefix(cleaned, "#")
		if !strings.HasPrefix(cleaned, "@param") {
			out = append(out, line)
			continue
		}
		m := paramPattern.FindStringSubmatch(cleaned)
		if m == nil {
			out = append(out, line)
			continue
		}
		name := m[1]
		if val, ok := overrides[name]; ok {
			out = append(out, name+" = "+formatValue(val))
		} else {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// formatValue converts a value to a Python-compatible string.
func formatValue(v interface{}) string {
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
