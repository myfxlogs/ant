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
// Pre-checks workspace prerequisites before delegating to AgentLoop.
func (g *Generator) runAgentLoop(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	// ── Pre-flight gate: workspace must have symbol + timeframe ──
	if msg.Symbol == "" || msg.Timeframe == "" {
		return streamOrAbort(&antv1.AgentGenerateStrategyChunk{
			Phase: "error",
			Delta: "请先在 workspace 中选择交易品种（Symbol）和时间周期（Timeframe），再生成策略代码。",
		})
	}

	result := &generateState{}

	// ── Generator tool registry: compile_python only ──
	// Strategy code generation does not need market data or backtest logs.
	// Those tools belong to the Chat agent, not the Generator.
	registry := buildPythonToolRegistry(result)
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
