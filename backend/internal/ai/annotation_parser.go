package ai

import (
	"regexp"
	"strconv"
	"strings"
)

// TunableParamFromAnnotation represents a parameter extracted from @param annotations.
// It has range info (min/max/step) that ctx.Param() calls don't have.
type TunableParamFromAnnotation struct {
	Name    string
	Default float64
	Min     float64
	Max     float64
	Step    float64
	HasRange bool
}

// StrategyDirectiveAnnotation represents a @strategy directive from code comments.
type StrategyDirectiveAnnotation struct {
	Key   string
	Value string
}

// paramAnnotPattern matches: # @param name default [range=min:max:step] or // @param name default [range=min:max:step]
// Works for Python (#), MQL (//), and Go (//) comments.
var paramAnnotPattern = regexp.MustCompile(`(?:#|//)\s*@param\s+(\w+)\s+(\S+)(?:\s+range=([\d.]+):([\d.]+):([\d.]+))?`)

// strategyAnnotPattern matches: # @strategy key value or // @strategy key value
var strategyAnnotPattern = regexp.MustCompile(`(?:#|//)\s*@strategy\s+(\w+)\s+(\S+)`)

// ExtractParamAnnotations parses @param annotations from code comments.
// Returns parameters with range info for sweep dimension generation.
func ExtractParamAnnotations(code string) []TunableParamFromAnnotation {
	matches := paramAnnotPattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]TunableParamFromAnnotation, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		def, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		p := TunableParamFromAnnotation{
			Name:    name,
			Default: def,
			Min:     def,
			Max:     def,
		}
		if m[3] != "" && m[4] != "" && m[5] != "" {
			minVal, err1 := strconv.ParseFloat(m[3], 64)
			maxVal, err2 := strconv.ParseFloat(m[4], 64)
			stepVal, err3 := strconv.ParseFloat(m[5], 64)
			if err1 == nil && err2 == nil && err3 == nil && stepVal > 0 && maxVal > minVal {
				p.Min = minVal
				p.Max = maxVal
				p.Step = stepVal
				p.HasRange = true
			}
		}
		out = append(out, p)
	}
	return out
}

// ExtractStrategyDirectives parses @strategy annotations from code comments.
func ExtractStrategyDirectives(code string) []StrategyDirectiveAnnotation {
	matches := strategyAnnotPattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]StrategyDirectiveAnnotation, 0, len(matches))
	for _, m := range matches {
		out = append(out, StrategyDirectiveAnnotation{
			Key:   m[1],
			Value: m[2],
		})
	}
	return out
}

// ExtractParamsWithAnnotations combines ctx.Param() extraction with @param annotation parsing.
// @param annotations take precedence because they include range info.
// Falls back to ctx.Param() for params without @param annotations.
func ExtractParamsWithAnnotations(code string) []TunableParam {
	annotParams := ExtractParamAnnotations(code)
	ctxParams := ExtractParams(code)

	byName := map[string]*TunableParam{}
	order := []string{}

	// First pass: ctx.Param() calls (base params with defaults)
	for i := range ctxParams {
		p := ctxParams[i]
		byName[p.Name] = &p
		order = append(order, p.Name)
	}

	// Second pass: @param annotations (override with range info)
	for _, ap := range annotParams {
		if existing, ok := byName[ap.Name]; ok {
			existing.Min = ap.Min
			existing.Max = ap.Max
			existing.Step = ap.Step
		} else {
			p := TunableParam{
				Name:    ap.Name,
				Default: ap.Default,
				Min:     ap.Min,
				Max:     ap.Max,
				Step:    ap.Step,
			}
			p.Type = paramType(p.Default)
			byName[ap.Name] = &p
			order = append(order, ap.Name)
		}
	}

	out := make([]TunableParam, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// HasParamAnnotations checks if the code contains any @param annotations.
func HasParamAnnotations(code string) bool {
	return strings.Contains(code, "@param")
}
