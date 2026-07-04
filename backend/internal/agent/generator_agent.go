package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	connectai "anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
)

// generatorDiscipline and generatorAgentSystemPrompt have been moved to internal/ai
// as i18n functions: ai.PythonGeneratorPrompt(lang) and ai.PythonAgentDiscipline(lang).
// The Generator prompt includes discipline internally.

// runAgentLoop runs the LLM-driven agent loop for the Generator.
// After the loop completes, code is extracted and sent to the frontend.
func (g *Generator) runAgentLoop(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	preProfile *antv1.StrategyProfile,
	sessionMem *SessionMemory,
	confirmedPlan *antv1.StrategyPlan,
	streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	result := &generateState{}

// ── Build tool registry — same tools as Conversate path
	registry := buildPythonToolRegistry(result)
	if g.mkt != nil {
		registry.AddPreTool(&connectai.ReadKlineTool{})
	}
	if g.btRepo != nil {
		registry.AddPreTool(&connectai.ReadBacktestLogTool{})
	}
	if g.dbExec != nil && g.dbQuery != nil {
		registry.WireMemoryDB(g.dbExec, g.dbQuery)
	}

	// ── Build system prompt ──
	lang := ai.NormalizeLocale(msg.Locale)
	sysPrompt := ai.PythonGeneratorPrompt(lang)
	if msg.Symbol != "" || msg.Timeframe != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Workspace\nSymbol: %s\nTimeframe: %s", msg.Symbol, msg.Timeframe)
	}

	// ── Build user prompt ──
	var userPrompt strings.Builder
	userPrompt.WriteString("## Strategy Request\n")
	userPrompt.WriteString(msg.Message)
	if preProfile != nil {
		userPrompt.WriteString("\n\n## Strategy Profile (AI-generated guidance)\n")
		userPrompt.WriteString(fmt.Sprintf("Type: %s\n", preProfile.StrategyType))
		userPrompt.WriteString(fmt.Sprintf("Description: %s\n", preProfile.Description))
		if len(preProfile.IndicatorsUsed) > 0 {
			userPrompt.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(preProfile.IndicatorsUsed, ", ")))
	}
		userPrompt.WriteString(fmt.Sprintf("Entry: %s\n", preProfile.EntryLogic))
		userPrompt.WriteString(fmt.Sprintf("Exit: %s\n", preProfile.ExitLogic))
		userPrompt.WriteString(fmt.Sprintf("Risk: %s\n", preProfile.RiskManagement))
	}
	if confirmedPlan != nil {
		userPrompt.WriteString("\n\n## Confirmed Plan (follow precisely)\n")
		userPrompt.WriteString(fmt.Sprintf("Type: %s\n", confirmedPlan.Type))
		userPrompt.WriteString(fmt.Sprintf("Entry: %s\n", confirmedPlan.Entry))
		userPrompt.WriteString(fmt.Sprintf("Exit: %s\n", confirmedPlan.Exit))
		userPrompt.WriteString(fmt.Sprintf("Risk: %s\n", confirmedPlan.Risk))
		userPrompt.WriteString(fmt.Sprintf("Market: %s\n", confirmedPlan.Market))
	}
	if sessionMem != nil {
		sessionMem.InjectIntoPrompt(&userPrompt)
	}
	userPrompt.WriteString("\n\nDiscuss the plan briefly, then generate the Python strategy code and present it to the user. Do NOT compile — the user will ask if they want verification.")

	// ── Stream callbacks — map AgentLoop events to Generator chunk format ──
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

	// ── Create and run the AgentLoop ──
	loop := connectai.NewAgentLoop(registry,
		func(llmCtx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			return g.aiSvc.ChatCompletionStreamWithTools(llmCtx, userID, messages, tools, onChunk)
	},
		streamChunk, toolStream,
	)
	loop.SetCurrentCode("")

	raw, loopErr := loop.Run(ctx, sysPrompt, userPrompt.String(), userID)
	if loopErr != nil {
		g.log.Warn("generator: agent loop failed", zap.Error(loopErr))
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
			Phase: "done",
			Error: fmt.Sprintf("agent loop: %v", loopErr),
	})
		return nil
	}

	// ── Extract code from the final LLM response ──
	pythonSource := stripMarkdownFences(connectai.ExtractCode(raw))
	if pythonSource == "" {
		pythonSource = raw
	}
	result.PythonSource = pythonSource

	// ── Send final chunk with the generated code ──
	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
		Phase:        "done",
		PythonSource: pythonSource,
	})
	return nil
}
