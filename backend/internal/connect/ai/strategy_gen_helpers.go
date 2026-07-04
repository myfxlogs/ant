// Package ai — StrategyGen helper functions.
package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"anttrader/internal/pkg/ptr"
	"anttrader/internal/repository"
)

// ExtractCode extracts Go code from an LLM response string.
// Handles markdown code fences, truncated output, and heuristic line-based extraction.
func ExtractCode(raw string) string {
	// Try to extract from markdown code block (```python, ```go, or ```)
	start := strings.Index(raw, "```python")
	if start < 0 {
		start = strings.Index(raw, "```go")
	}
	if start < 0 {
		start = strings.Index(raw, "```")
	}
	if start >= 0 {
		rest := raw[start:]
		fenceLen := 3
		if strings.HasPrefix(rest, "```python") {
			fenceLen = 9
		} else if strings.HasPrefix(rest, "```go") {
			fenceLen = 5
		} else if strings.HasPrefix(rest, "```golang") {
			fenceLen = 9
		}
		end := strings.Index(rest[fenceLen:], "```")
		if end >= 0 {
			code := rest[fenceLen : end+fenceLen]
			code = strings.TrimPrefix(code, "python\n")
			code = strings.TrimPrefix(code, "python")
			code = strings.TrimPrefix(code, "go\n")
			code = strings.TrimPrefix(code, "go")
			code = strings.TrimPrefix(code, "golang\n")
			code = strings.TrimPrefix(code, "golang")
			code = strings.TrimSpace(code)
			return fixUnclosedBraces(code)
		}
		// Code block not closed (streaming truncated): extract everything after opening fence
		code := rest[fenceLen:]
		code = strings.TrimPrefix(code, "python\n")
		code = strings.TrimPrefix(code, "python")
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

// fixUnclosedBraces adds missing closing braces for truncated code.
func fixUnclosedBraces(code string) string {
	opens := strings.Count(code, "{")
	closes := strings.Count(code, "}")
	if closes < opens {
		for i := closes; i < opens; i++ {
			code += "\n}"
		}
	}
	return strings.TrimSpace(code)
}

// CreateBacktestRun is a standalone helper used by StrategyPlanServer.
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

