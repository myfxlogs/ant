package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	connectai "anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
)

// runAgentLoop is the single unified entry point for all Generator requests.
// It builds the full tool registry, uses the unified Python agent prompt,
// and delegates to AgentLoop. No pre-processing, no gates, no routing.
func (g *Generator) runAgentLoop(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	result := &generateState{}

	// ── Full tool registry ──
	registry := buildPythonToolRegistry(result, g.mkt, msg.BacktestConfig)
	// read_kline and read_backtest_log are Chat agent tools, not Generator tools.
	// Strategy logic doesn't depend on current market data.
	if g.dbExec != nil && g.dbQuery != nil {
		registry.WireMemoryDB(g.dbExec, g.dbQuery)
	}

	// ── Unified system prompt ──
	lang := ai.NormalizeLocale(msg.Locale)
	sysPrompt := ai.PythonAgentPrompt(lang)
	if msg.Symbol != "" || msg.Timeframe != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Workspace\nSymbol: %s\nTimeframe: %s", msg.Symbol, msg.Timeframe)
	}

	// ── User prompt ──
	userPrompt := msg.Message

	// ── Conversation history ──
	var history []systemai.ChatMessage
	if msg.ConversationId != "" && g.conversationRepo != nil {
		cid, err := uuid.Parse(msg.ConversationId)
		if err == nil {
			msgs, _ := g.conversationRepo.GetMessages(ctx, userID, cid)
			for _, m := range msgs {
				history = append(history, systemai.ChatMessage{
					Role:    m.Role,
					Content: m.Content,
				})
			}
		}
	}

	// ── Stream callbacks ──
	streamChunk := func(delta string) error {
		// Strip [THINK] blocks from streamed content — DeepSeek models output them
		// regardless of prompt instructions. User never sees reasoning traces.
		cleaned := stripThinkBlocks(delta)
		if cleaned == "" {
			return nil // nothing visible to stream
		}
		return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "generating", Delta: cleaned})
	}
	reasoningStream := func(delta string) error {
		return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "thinking", Delta: delta})
	}
	toolStream := func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
		switch tc.Name {
		case "compile_python":
			chunk := &antv1.AgentGenerateStrategyChunk{Phase: "compiling", PythonSource: result.PythonSource}
			if !tr.Success {
				chunk.CompileError = tr.Error
			}
			return streamOrAbort(chunk)
		case "read_kline":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning"})
		case "run_backtest":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "backtesting", Delta: tr.OutputJson})
		case "edit_code":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "editing", PythonSource: result.PythonSource})
		case "read_current_code":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "reading"})
		case "update_plan":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning", Delta: tr.OutputJson})
		}
		return nil
	}

	// ── Run the AgentLoop ──
	loop := connectai.NewAgentLoop(registry,
		func(llmCtx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			return g.aiSvc.ChatCompletionStreamWithTools(llmCtx, userID, messages, tools, onChunk)
		},
		streamChunk, toolStream, reasoningStream,
	)

	raw, loopErr := loop.RunWithHistory(ctx, sysPrompt, userPrompt, history, userID)
	g.log.Info("generator: loop done", zap.Int("raw_len", len(raw)), zap.Bool("has_err", loopErr != nil))

	// I1: PythonSource is ONLY set by write_strategy tool. Never overwrite it here.
	// ExtractCode from free text is for display only — not a source of truth (§3.1b前提2).
	cleaned := stripThinkBlocks(raw)
	displaySource := stripMarkdownFences(connectai.ExtractCode(cleaned))
	if displaySource == "" {
		displaySource = stripMarkdownFences(cleaned)
	}

	if loopErr != nil {
		g.log.Warn("generator: agent loop ended", zap.Error(loopErr))
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: result.PythonSource, Error: loopErr.Error()})
		return nil
	}

	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: result.PythonSource})
	return nil
}
