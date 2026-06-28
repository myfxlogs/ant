// Package ai — StrategyGen helper functions.
package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/pkg/ptr"
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

// ExtractCode extracts Go code from an LLM response string.
// Handles markdown code fences, truncated output, and heuristic line-based extraction.
func ExtractCode(raw string) string {
	// Try to extract from markdown code block (```go or ```)
	start := strings.Index(raw, "```go")
	if start < 0 {
		start = strings.Index(raw, "```")
	}
	if start >= 0 {
		rest := raw[start:]
		fenceLen := 3
		if strings.HasPrefix(rest, "```go") {
			fenceLen = 5
		} else if strings.HasPrefix(rest, "```golang") {
			fenceLen = 9
		}
		end := strings.Index(rest[fenceLen:], "```")
		if end >= 0 {
			code := rest[fenceLen : end+fenceLen]
			code = strings.TrimPrefix(code, "go\n")
			code = strings.TrimPrefix(code, "go")
			code = strings.TrimPrefix(code, "golang\n")
			code = strings.TrimPrefix(code, "golang")
			code = strings.TrimSpace(code)
			return fixUnclosedBraces(code)
		}
		// Code block not closed (streaming truncated): extract everything after opening fence
		code := rest[fenceLen:]
		code = strings.TrimPrefix(code, "go\n")
		code = strings.TrimPrefix(code, "go")
		code = strings.TrimPrefix(code, "golang\n")
		code = strings.TrimPrefix(code, "golang")
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
		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") {
			inCode = true
		}
		if inCode {
			codeLines = append(codeLines, line)
		}
	}
	if len(codeLines) > 0 {
		return fixUnclosedBraces(strings.Join(codeLines, "\n"))
	}
	return "" // no code found — don't treat entire response as code
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

// CreateBacktestRun is a standalone helper used by both StrategyGenServer and StrategyPlanServer.
func CreateBacktestRun(ctx context.Context, repo *repository.BacktestRunRepository, userID uuid.UUID, code, symbol, timeframe string) (string, error) {
	return triggerBacktest(ctx, repo, userID, code, symbol, timeframe)
}

func triggerBacktest(ctx context.Context, repo *repository.BacktestRunRepository, userID uuid.UUID, code, symbol, timeframe string) (string, error) {
	if repo == nil {
		return "", nil
	}
	if code == "" {
		return "", fmt.Errorf("回测跳过：代码为空")
	}
	if symbol == "" || timeframe == "" {
		return "", fmt.Errorf("回测跳过：未选择交易品种和时间周期。请在顶部工具栏选择后再生成策略。")
	}
	run := &repository.BacktestRun{
		ID: uuid.New(), UserID: userID, AccountID: uuid.Nil,
		Symbol: symbol, Timeframe: timeframe, Mode: "KLINE_RANGE", Status: "PENDING",
		StrategyCode: &code,
		InitialCapital: ptr.Decimal(decimal.NewFromInt(10000)),
		Commission:     ptr.Decimal(decimal.NewFromFloat(0.001)),
		Slippage:       ptr.Decimal(decimal.Zero),
		Leverage:       ptr.Decimal(decimal.NewFromInt(1)),
		TradeDirection: ptr.Str("both"), StrictMode: ptr.Bool(true), ExtraSymbols: []string{},
	}
	id, err := repo.Create(ctx, run)
	if err != nil {
		return "", fmt.Errorf("create backtest run: %w", err)
	}
	return id.String(), nil
}

// triggerBacktest delegates to the standalone helper for backward compatibility.
func (s *StrategyGenServer) triggerBacktest(ctx context.Context, userID uuid.UUID, code, symbol, timeframe string) (string, error) {
	return CreateBacktestRun(ctx, s.backtestRepo, userID, code, symbol, timeframe)
}


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
