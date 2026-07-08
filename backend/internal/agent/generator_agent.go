package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

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

	// ── Structured memory injection (§4 Layer 2) ──
	if g.memory != nil {
		session, err := g.memory.LoadSessionMemory(ctx, userID, msg.Symbol, msg.Timeframe)
		if err != nil {
			g.log.Warn("generator: failed to load session memory", zap.Error(err))
		} else if session != nil {
			var memSB strings.Builder
			session.InjectIntoPrompt(&memSB)
			if memStr := memSB.String(); memStr != "" {
				sysPrompt += "\n" + memStr
			}
		}
	}

	// ── User prompt ──
	userPrompt := msg.Message

	// ── Conversation history ──
	var convID uuid.UUID
	var history []systemai.ChatMessage
	if msg.ConversationId != "" && g.conversationRepo != nil {
		cid, err := uuid.Parse(msg.ConversationId)
		if err == nil {
			convID = cid
			// Ensure conversation row exists (frontend may send fresh UUID not yet persisted).
			if _, getErr := g.conversationRepo.GetByID(ctx, cid, userID); getErr != nil {
				title := userPrompt
				if runes := []rune(title); len(runes) > 80 {
					title = string(runes[:80])
				}
				if _, createErr := g.conversationRepo.CreateWithID(ctx, userID, cid, title); createErr != nil {
					g.log.Warn("generator: auto-create conversation failed", zap.Error(createErr))
				}
			}
			msgs, _ := g.conversationRepo.GetMessages(ctx, userID, cid)
			for _, m := range msgs {
				history = append(history, systemai.ChatMessage{
					Role:    m.Role,
					Content: m.Content,
				})
			}
		}
	}
	// Auto-create conversation if no ID was provided at all.
	if convID == uuid.Nil && g.conversationRepo != nil {
		title := userPrompt
		if runes := []rune(title); len(runes) > 80 {
			title = string(runes[:80])
		}
		conv, err := g.conversationRepo.Create(ctx, userID, title)
		if err != nil {
			g.log.Warn("generator: failed to create conversation", zap.Error(err))
		} else {
			convID = conv.ID
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
		return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "thinking", Reasoning: delta})
	}
	toolStream := func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
		switch tc.Name {
		case "write_strategy":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{
				Phase:        "generating",
				PythonSource: result.PythonSource,
			})
		case "edit_code":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{
				Phase:        "editing",
				PythonSource: result.PythonSource,
			})
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

	// Serialize the final turn chunk for full replay on resume.
	var turnDataBytes []byte
	if result.PythonSource != "" {
		finalChunk := &antv1.AgentGenerateStrategyChunk{
			Phase:        "done",
			Delta:        raw,
			PythonSource: result.PythonSource,
		}
		if result.LastBacktest != nil {
			finalChunk.Result = &antv1.AgentBacktestResult{
				Success:      true,
				TotalTrades:  int32(result.LastBacktest.TotalTrades),
				WinRate:      result.LastBacktest.WinRate,
				TotalReturn:  result.LastBacktest.TotalReturn,
				MaxDrawdown:  result.LastBacktest.MaxDrawdown,
				SharpeRatio:  result.LastBacktest.SharpeRatio,
			}
		}
		if result.CompileError != "" {
			finalChunk.CompileError = result.CompileError
		}
		if result.BacktestError != "" {
			finalChunk.BacktestError = result.BacktestError
		}
		if data, err := proto.Marshal(finalChunk); err == nil {
			turnDataBytes = data
		}
	}

	// Persist conversation messages for multi-turn context.
	if convID != uuid.Nil && g.conversationRepo != nil {
		if _, err := g.conversationRepo.AddMessage(ctx, userID, convID, "user", userPrompt, nil); err != nil {
			g.log.Warn("generator: failed to save user message", zap.Error(err))
		}
		if raw != "" {
			if _, err := g.conversationRepo.AddMessage(ctx, userID, convID, "assistant", raw, turnDataBytes); err != nil {
				g.log.Warn("generator: failed to save assistant message", zap.Error(err))
			}
		}
		_ = g.conversationRepo.Touch(ctx, convID, userID)
	}

	// Persist structured memory when a strategy was successfully delivered (deterministic, zero LLM call).
	if result.LastBacktest != nil && result.PythonSource != "" && g.memory != nil {
		symbol := msg.Symbol
		if symbol == "" && msg.BacktestConfig != nil {
			symbol = msg.BacktestConfig.Symbol
		}
		tf := msg.Timeframe
		if tf == "" && msg.BacktestConfig != nil {
			tf = msg.BacktestConfig.Timeframe
		}
		summary := fmt.Sprintf("%s %s: %d trades, %.1f%% win, %.2f%% return",
			symbol, tf,
			result.LastBacktest.TotalTrades,
			result.LastBacktest.WinRate,
			result.LastBacktest.TotalReturn)
		fp := fmt.Sprintf("%x", sha256.Sum256([]byte(result.PythonSource)))
		_, err := g.memory.StoreExperience(ctx, userID, "strategy",
			summary, fp, nil, symbol+" "+tf)
		if err != nil {
			g.log.Warn("generator: failed to store experience", zap.Error(err))
		}
	}

	// I1: PythonSource is ONLY set by write_strategy tool. Never overwrite it here (§3.1b前提2).
	if loopErr != nil {
		g.log.Warn("generator: agent loop ended", zap.Error(loopErr))
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: result.PythonSource, Error: loopErr.Error()})
		return nil
	}

	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: result.PythonSource})
	return nil
}
