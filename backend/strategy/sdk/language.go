package sdk

import "strings"

// StrategyLanguage describes the source language of a strategy code snippet.
type StrategyLanguage string

const (
	LangMQL    StrategyLanguage = "mql"
	LangPython StrategyLanguage = "python"
	LangGo     StrategyLanguage = "go"
	LangUnknown StrategyLanguage = "unknown"
)

// DetectLanguage classifies strategy source code as MQL, Python subset, or Go.
// Detection order: Go → MQL → Python → Unknown.
//
// Go: contains "package " + "alphaforge/strategy/sdk" import path.
// MQL: contains MQL lifecycle hooks (OnTick/OnBar/OnInit/OnDeinit/OnTimer) or
//      MQL declarations (#property, extern, input) and NOT Go package/import.
// Python: contains Python subset markers (StrategyBase, def on_init/on_bar/on_tick/on_deinit,
//         "from alphaforge") and NOT Go or MQL.
func DetectLanguage(code string) StrategyLanguage {
	if code == "" {
		return LangUnknown
	}
	if isGoSource(code) {
		return LangGo
	}
	if isMQLSource(code) {
		return LangMQL
	}
	if isPythonSource(code) {
		return LangPython
	}
	return LangUnknown
}

// IsMQL returns true if the code is MQL source.
func IsMQL(code string) bool { return DetectLanguage(code) == LangMQL }

// IsPython returns true if the code is Python subset source.
func IsPython(code string) bool { return DetectLanguage(code) == LangPython }

// IsGo returns true if the code is Go strategy source.
func IsGo(code string) bool { return DetectLanguage(code) == LangGo }

func isGoSource(code string) bool {
	return strings.Contains(code, "package ") && strings.Contains(code, "alphaforge/strategy/sdk")
}

func isMQLSource(code string) bool {
	return strings.Contains(code, "OnTick") || strings.Contains(code, "OnBar") ||
		strings.Contains(code, "OnInit") || strings.Contains(code, "OnDeinit") ||
		strings.Contains(code, "OnTimer") ||
		strings.Contains(code, "#property") || strings.Contains(code, "extern ") ||
		strings.Contains(code, "input ")
}

func isPythonSource(code string) bool {
	return strings.Contains(code, "StrategyBase") ||
		strings.Contains(code, "from alphaforge") ||
		strings.Contains(code, "def on_init") || strings.Contains(code, "def on_bar") ||
		strings.Contains(code, "def on_tick") || strings.Contains(code, "def on_deinit")
}
