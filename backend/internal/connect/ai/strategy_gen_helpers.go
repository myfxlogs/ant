// Package ai — StrategyGen helper functions.
package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

func (s *StrategyGenServer) buildParamMap(m *antv1.GenerateStrategyRequest) map[string]string {
	pm := map[string]string{}
	if m.Symbol != "" {
		pm["symbol"] = m.Symbol
	}
	if m.Timeframe != "" {
		pm["timeframe"] = m.Timeframe
	}
	return pm
}

// loadHistory loads recent conversation messages and returns a summary string.
func (s *StrategyGenServer) loadHistory(ctx context.Context, userID uuid.UUID, convID string) string {
	if convID == "" {
		return ""
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return ""
	}
	msgs, err := s.convRepo.GetMessages(ctx, userID, cid)
	if err != nil || len(msgs) == 0 {
		return ""
	}
	start := 0
	if len(msgs) > 6 {
		start = len(msgs) - 6
	}
	var sb strings.Builder
	for _, m := range msgs[start:] {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}
	return sb.String()
}

// ExtractCode extracts Python code from an LLM response string.
// Handles markdown code fences, truncated output, and heuristic line-based extraction.
func ExtractCode(raw string) string {
	// Try to extract from markdown code block
	start := strings.Index(raw, "```python")
	if start < 0 {
		start = strings.Index(raw, "```")
	}
	if start >= 0 {
		rest := raw[start:]
		fenceLen := 3
		if strings.HasPrefix(rest, "```python") {
			fenceLen = 9
		}
		end := strings.Index(rest[fenceLen:], "```")
		if end >= 0 {
			code := rest[fenceLen : end+fenceLen]
			code = strings.TrimPrefix(code, "python\n")
			code = strings.TrimPrefix(code, "python")
			code = strings.TrimSpace(code)
			return fixUnclosedBraces(code)
		}
		// Code block not closed (streaming truncated): extract everything after opening fence
		code := rest[fenceLen:]
		code = strings.TrimPrefix(code, "python\n")
		code = strings.TrimPrefix(code, "python")
		code = strings.TrimSpace(code)
		return fixUnclosedBraces(code)
	}
	// Fallback: heuristic line-based extraction
	raw = strings.TrimSpace(raw)
	lines := strings.Split(raw, "\n")
	var codeLines []string
	inCode := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "@param") ||
			strings.HasPrefix(trimmed, "class ") {
			inCode = true
		}
		if inCode {
			codeLines = append(codeLines, line)
		}
	}
	if len(codeLines) > 0 {
		return fixUnclosedBraces(strings.Join(codeLines, "\n"))
	}
	return fixUnclosedBraces(raw)
}

// fixUnclosedBraces adds missing closing braces/returns for truncated code.
func fixUnclosedBraces(code string) string {
	// Count braces
	opens := strings.Count(code, "{")
	closes := strings.Count(code, "}")
	if closes < opens {
		// Add missing closing braces for the return dict
		for i := closes; i < opens; i++ {
			code += "\n}"
		}
	}
	// Add missing return if needed
	code = strings.TrimSpace(code)
	if !strings.HasSuffix(code, "}") && !strings.Contains(code, "return ") {
		// Don't try to guess — just ensure braces balance
	}
	// Ensure the last return statement has complete closing
	if strings.Count(code, "{") > strings.Count(code, "}") {
		missing := strings.Count(code, "{") - strings.Count(code, "}")
		for i := 0; i < missing; i++ {
			code += "\n}"
		}
	}
	return code
}

// collectComplianceIssues gathers all blocking issues into a string slice.
func (s *StrategyGenServer) collectComplianceIssues(blocks []ai.ComplianceIssue, missingSigs []string) []string {
	issues := make([]string, 0, len(blocks)+len(missingSigs))
	for _, b := range blocks {
		issues = append(issues, fmt.Sprintf("[%s] %s (line %d)", b.RuleName, b.Message, b.Line))
	}
	issues = append(issues, missingSigs...)
	return issues
}

// triggerBacktest creates a PENDING backtest run for the generated code.
func (s *StrategyGenServer) triggerBacktest(ctx context.Context, userID uuid.UUID, code, symbol, timeframe string) (string, error) {
	if s.backtestRepo == nil {
		return "", nil
	}
	if code == "" || symbol == "" || timeframe == "" {
		return "", nil
	}
	run := &repository.BacktestRun{
		ID:            uuid.New(),
		UserID:        userID,
		AccountID:     uuid.Nil,
		Symbol:        symbol,
		Timeframe:     timeframe,
		Mode:          "KLINE_RANGE",
		Status:        "PENDING",
		StrategyCode:  &code,
		InitialCapital: f64Ptr(10000),
			Commission:       f64Ptr(0.001),
			Slippage:         f64Ptr(0),
			Leverage:         f64Ptr(1),
			TradeDirection:   strPtr("both"),
			StrictMode:       bPtr(true),
		StrategyCodeHash: "",
		Error:         "",
		ExtraSymbols:  []string{},
	}
	id, err := s.backtestRepo.Create(ctx, run)
	if err != nil {
		return "", fmt.Errorf("create backtest run: %w", err)
	}
	return id.String(), nil
}

func f64Ptr(v float64) *float64 { return &v }
func strPtr(s string) *string { if s == "" { return nil }; return &s }
func bPtr(v bool) *bool { return &v }

// ── Phase 3: section parsing for feedback mode ──

// parsedSections holds the extracted <section> blocks from LLM feedback output.
type parsedSections struct {
	Analysis string
	Advice   string
	Code     string
}

// parseSections extracts <section type="..."> blocks from raw LLM output.
// Partial output is fine — missing sections are empty strings.
func parseSections(raw string) parsedSections {
	var s parsedSections
	s.Analysis = extractSection(raw, "analysis")
	s.Advice = extractSection(raw, "advice")
	s.Code = extractSection(raw, "code")
	return s
}

// extractSection matches <section type="TYPE"> ... </section> and returns the inner content.
func extractSection(raw, sectionType string) string {
	re := regexp.MustCompile(
		`(?s)<section\s+type="` + regexp.QuoteMeta(sectionType) + `">(.*?)</section>`,
	)
	m := re.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
