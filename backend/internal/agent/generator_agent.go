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
	registry := buildPythonToolRegistry(result)
	if g.mkt != nil {
		registry.AddPreTool(connectai.NewReadKlineTool(g.mkt))
	}
	if g.btRepo != nil {
		registry.AddPreTool(connectai.NewReadBacktestLogTool(g.btRepo))
	}
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

	// ── Stream callbacks ──
	streamChunk := func(delta string) error {
		return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "generating", Delta: delta})
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
		}
		return nil
	}

	// ── Run the AgentLoop ──
	loop := connectai.NewAgentLoop(registry,
		func(llmCtx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			return g.aiSvc.ChatCompletionStreamWithTools(llmCtx, userID, messages, tools, onChunk)
		},
		streamChunk, toolStream,
	)

	raw, loopErr := loop.Run(ctx, sysPrompt, userPrompt, userID)

	// Extract code regardless of loopErr — the LLM may have generated valid code
	// even if max rounds was exceeded.
	cleaned := stripThinkBlocks(raw)
	pythonSource := stripMarkdownFences(connectai.ExtractCode(cleaned))
	if pythonSource == "" {
		pythonSource = stripMarkdownFences(cleaned)
	}
	result.PythonSource = pythonSource

	if loopErr != nil {
		g.log.Warn("generator: agent loop ended", zap.Error(loopErr))
		// If we got code despite the error, send it with a warning instead of failing.
		if pythonSource != "" {
			_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: pythonSource})
			return nil
		}
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", Error: loopErr.Error()})
		return nil
	}

	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: pythonSource})
	return nil
}
