// code_assist_handler_extract.go — Code extraction helpers extracted from code_assist_handler.go.
package ai

import (
	"strings"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func ExtractParams(code string) []*antv1.ParameterEntry {
	matches := paramPattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var entries []*antv1.ParameterEntry
	for _, m := range matches {
		name := m[2]
		if seen[name] {
			continue
		}
		seen[name] = true
		entries = append(entries, &antv1.ParameterEntry{
			Name:    name,
			Type:    paramType[m[1]],
			Default: strings.TrimSpace(m[3]),
		})
	}
	return entries
}

// extractCodeFromRepair attempts to salvage code from an LLM response
// that may contain explanatory text (3-tier extraction).
// When isMQL is true, MQL fenced blocks are tried before Go ones.
func extractCodeFromRepair(raw string, isMQL bool) string {
	var langs []string
	if isMQL {
		langs = []string{"mql4", "mql5", "mql", "go"}
	} else {
		langs = []string{"go", "mql4", "mql5", "mql"}
	}
	// Tier 1: extract from fenced code blocks
	for _, lang := range langs {
		if code := extractFencedCode(raw, lang); code != "" {
			return code
		}
	}
	// Tier 2: heuristic — find lines starting with known code markers
	if code := extractByHeuristic(raw); code != "" {
		return code
	}
	// Tier 3: unable to extract — return empty
	return ""
}

func extractFencedCode(raw, lang string) string {
	marker := "```" + lang
	start := strings.Index(raw, marker)
	if start < 0 {
		start = strings.Index(raw, "```")
		if start < 0 {
			return ""
		}
	}
	// Skip the opening fence line
	if nl := strings.Index(raw[start:], "\n"); nl >= 0 {
		start += nl + 1
	} else {
		return ""
	}
	end := strings.Index(raw[start:], "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func extractByHeuristic(raw string) string {
	raw = strings.TrimSpace(raw)
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Go markers
		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "import(") ||
			strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "//") {
			return strings.Join(lines[i:], "\n")
		}
		// MQL markers
		if strings.HasPrefix(trimmed, "#property") ||
			strings.HasPrefix(trimmed, "#include") ||
			strings.HasPrefix(trimmed, "#define") ||
			strings.HasPrefix(trimmed, "extern ") ||
			strings.HasPrefix(trimmed, "input ") ||
			strings.HasPrefix(trimmed, "int OnInit") ||
			strings.HasPrefix(trimmed, "void OnTick") ||
			strings.HasPrefix(trimmed, "void OnBar") ||
			strings.HasPrefix(trimmed, "int OnInit()") {
			return strings.Join(lines[i:], "\n")
		}
		return ""
	}
	return ""
}
