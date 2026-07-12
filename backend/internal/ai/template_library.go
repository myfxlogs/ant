// template_library.go: strategy template matching for AI generation (spec/26 Phase 1).
//
// Matches user intent (from clarification or direct description) to the best
// platform strategy template based on category keywords and strategy family hints.

package ai

import (
	"strings"

	"alphaforge/internal/repository"
)

// categoryKeywords maps strategy categories to Chinese keywords for matching.
var categoryKeywords = map[string][]string{
	"trend_following": {"趋势", "跟踪", "顺势", "趋势跟踪", "均线", "双均线", "ma", "sma", "ema"},
	"mean_reversion":  {"回归", "均值", "布林带", "布林", "bollinger", "震荡", "波段", "高抛低吸", "rsi"},
	"breakout":        {"突破", "donchian", "唐奇安", "最高价", "最低价", "突破买入"},
	"grid":            {"网格", "grid", "区间", "震荡网格"},
	"martingale":      {"马丁", "martingale", "倍投", "加仓"},
}

// TemplateLibrary selects strategy templates based on user input.
type TemplateLibrary struct {
	templates []repository.AIStrategyTemplate
}

// NewTemplateLibrary creates a library with the given templates (loaded from DB).
func NewTemplateLibrary(templates []repository.AIStrategyTemplate) *TemplateLibrary {
	return &TemplateLibrary{templates: templates}
}

// MatchByFamily selects a template by explicit strategy family name.
// Returns nil if no template matches the family.
func (l *TemplateLibrary) MatchByFamily(family string) *repository.AIStrategyTemplate {
	if family == "" || family == "unknown" || len(l.templates) == 0 {
		return nil
	}
	for i := range l.templates {
		if l.templates[i].Category == family {
			return &l.templates[i]
		}
	}
	return nil
}

// Match finds the best template for the given message. Returns nil if no match.
// It first checks for explicit category keywords, then falls back to name matching.
func (l *TemplateLibrary) Match(message string) *repository.AIStrategyTemplate {
	msg := strings.ToLower(message)
	if len(l.templates) == 0 {
		return nil
	}

	// Phase 1: category keyword match
	bestCat := l.matchCategory(msg)
	if bestCat != "" {
		if t := l.topByCategory(bestCat); t != nil {
			return t
		}
	}

	// Phase 2: fallback — name keyword match
	if t := l.matchByName(msg); t != nil {
		return t
	}

	// Phase 3: default — return first template (trend_following if available)
	for i := range l.templates {
		if l.templates[i].Category == "trend_following" {
			return &l.templates[i]
		}
	}
	return &l.templates[0]
}

// matchCategory returns the best-matching category based on keyword count.
func (l *TemplateLibrary) matchCategory(msg string) string {
	var bestCat string
	bestScore := 0
	for cat, keywords := range categoryKeywords {
		score := countMatches(msg, keywords)
		if score > bestScore {
			bestScore = score
			bestCat = cat
		}
	}
	if bestScore > 0 {
		return bestCat
	}
	return ""
}

// topByCategory returns the first active template in the given category.
func (l *TemplateLibrary) topByCategory(category string) *repository.AIStrategyTemplate {
	for i := range l.templates {
		if l.templates[i].Category == category {
			return &l.templates[i]
		}
	}
	return nil
}

// matchByName checks template names and descriptions for keyword matches.
func (l *TemplateLibrary) matchByName(msg string) *repository.AIStrategyTemplate {
	for i := range l.templates {
		name := strings.ToLower(l.templates[i].Name)
		desc := strings.ToLower(l.templates[i].DescriptionZh)
		if strings.Contains(msg, name) || strings.Contains(name, msg) ||
			(desc != "" && strings.Contains(msg, desc)) {
			return &l.templates[i]
		}
	}
	return nil
}

// countMatches counts how many keywords appear in the message.
func countMatches(msg string, keywords []string) int {
	n := 0
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			n++
		}
	}
	return n
}
