package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/ai"
	connectai "alphaforge/internal/connect/ai"
	systemai "alphaforge/internal/service/systemai"
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
	sessionID := msg.ConversationId
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Phase 2: credit pre-hold at session start.
	if g.creditSvc != nil {
		if err := g.creditSvc.PreHold(ctx, userID, sessionID, "", ""); err != nil {
			g.log.Warn("generator: credit pre-hold failed", zap.Error(err))
			_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
				Phase: "error",
				Error: "insufficient credits — please top up your account",
			})
			return nil
		}
	}

	registry := buildPythonToolRegistry(result, g.mkt, msg.BacktestConfig)
	if g.dbExec != nil && g.dbQuery != nil {
		registry.WireMemoryDB(g.dbExec, g.dbQuery)
	}

	sysPrompt := g.buildGeneratorSysPrompt(ctx, userID, msg)

	userPrompt := msg.Message
	if msg.CurrentCode != "" {
		userPrompt += "\n\n## Current Strategy Code\n```\n" + msg.CurrentCode + "\n```"
	}

	convID, history := g.loadConversationHistory(ctx, userID, msg, userPrompt)

	streamChunk, reasoningStream, toolStream := g.buildStreamCallbacks(result, streamOrAbort)

	// Create a session-scoped token counter so all LLM calls within this
	// agent loop share quota tracking. This prevents mid-session overshoot:
	// each round's pre-check sees the cumulative usage from prior rounds.
	quotaCtx, sessionQuota := systemai.WithSessionQuota(ctx)

	loop := connectai.NewAgentLoop(registry,
		func(llmCtx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			// Merge session quota context with the LLM call context.
			mergedCtx := context.WithValue(llmCtx, systemai.SessionQuotaKey, sessionQuota)
			return g.aiSvc.ChatCompletionStreamWithTools(mergedCtx, userID, messages, tools, onChunk)
		},
		streamChunk, toolStream, reasoningStream,
	)

	raw, loopErr := loop.RunWithHistory(quotaCtx, sysPrompt, userPrompt, history, userID)
	g.log.Info("generator: loop done", zap.Int("raw_len", len(raw)), zap.Bool("has_err", loopErr != nil))

	turnDataBytes := g.buildFinalTurnChunk(result, raw)

	g.persistConversation(ctx, userID, convID, userPrompt, raw, turnDataBytes)
	g.persistGeneratorMemory(ctx, userID, msg, result)

	// Phase 2: credit settlement at session end.
	if g.creditSvc != nil {
		if loopErr != nil {
			// Session failed — release the hold.
			if err := g.creditSvc.ReleaseHold(ctx, userID, sessionID); err != nil {
				g.log.Warn("generator: credit release failed", zap.Error(err))
			}
		} else {
			// Session succeeded — settle actual cost.
			// Token counts are tracked by the post-call biller; here we settle the hold with zero
			// since actual token costs are already deducted via the billing wire.
			if err := g.creditSvc.Settle(ctx, userID, sessionID, "", "", 0, 0); err != nil {
				g.log.Warn("generator: credit settle failed", zap.Error(err))
			}
		}
	}

	if loopErr != nil {
		g.log.Warn("generator: agent loop ended", zap.Error(loopErr))
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: result.PythonSource, Error: loopErr.Error()})
		return nil
	}

	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done", PythonSource: result.PythonSource})
	return nil
}

func (g *Generator) buildGeneratorSysPrompt(ctx context.Context, userID uuid.UUID, msg *antv1.AgentGenerateStrategyRequest) string {
	lang := ai.NormalizeLocale(msg.Locale)
	sysPrompt := ai.PythonAgentPrompt(lang)
	if msg.Symbol != "" || msg.Timeframe != "" || msg.AccountId != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Workspace\nSymbol: %s\nTimeframe: %s", msg.Symbol, msg.Timeframe)
		if msg.AccountId != "" {
			sysPrompt += fmt.Sprintf("\nAccount: %s", msg.AccountId)
		}
	}
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
	return sysPrompt
}

func (g *Generator) loadConversationHistory(ctx context.Context, userID uuid.UUID, msg *antv1.AgentGenerateStrategyRequest, userPrompt string) (uuid.UUID, []systemai.ChatMessage) {
	var convID uuid.UUID
	var history []systemai.ChatMessage
	if msg.ConversationId != "" && g.conversationRepo != nil {
		cid, err := uuid.Parse(msg.ConversationId)
		if err == nil {
			convID = cid
			if _, getErr := g.conversationRepo.GetByID(ctx, cid, userID); getErr != nil {
				title := userPrompt
				if runes := []rune(title); len(runes) > 80 {
					title = string(runes[:80])
				}
				if _, createErr := g.conversationRepo.CreateWithID(ctx, userID, cid, title); createErr != nil {
					g.log.Warn("generator: auto-create conversation failed", zap.Error(createErr))
				}
			}
			msgs, err := g.conversationRepo.GetMessages(ctx, userID, cid)
			if err != nil {
				g.log.Warn("generator: failed to load conversation history", zap.Error(err))
			}
			for _, m := range msgs {
				history = append(history, systemai.ChatMessage{
					Role:    m.Role,
					Content: m.Content,
				})
			}
		}
	}
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
	return convID, history
}

func (g *Generator) buildStreamCallbacks(result *generateState, streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error) (func(string) error, func(string) error, func(*antv1.ToolCall, *antv1.ToolResult) error) {
	streamChunk := func(delta string) error {
		cleaned := stripThinkBlocks(delta)
		if cleaned == "" {
			return nil
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
		case "read_kline":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "reading", Delta: structToJSON(tr.Output)})
		case "read_current_code":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "reading"})
		case "update_plan":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning", Delta: structToJSON(tr.Output)})
		}
		return nil
	}
	return streamChunk, reasoningStream, toolStream
}

func (g *Generator) buildFinalTurnChunk(result *generateState, raw string) []byte {
	if result.PythonSource == "" {
		return nil
	}
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
	data, err := proto.Marshal(finalChunk)
	if err != nil {
		return nil
	}
	return data
}

func (g *Generator) persistConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID, userPrompt, raw string, turnDataBytes []byte) {
	if convID == uuid.Nil || g.conversationRepo == nil {
		return
	}
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

func (g *Generator) persistGeneratorMemory(ctx context.Context, userID uuid.UUID, msg *antv1.AgentGenerateStrategyRequest, result *generateState) {
	if result.LastBacktest == nil || result.PythonSource == "" || g.memory == nil {
		return
	}
	symbol := msg.Symbol
	if symbol == "" && msg.BacktestConfig != nil {
		symbol = msg.BacktestConfig.Symbol
	}
	tf := msg.Timeframe
	if tf == "" && msg.BacktestConfig != nil {
		tf = msg.BacktestConfig.Timeframe
	}
	summary := fmt.Sprintf("%s %s: %d trades, %s%% win, %s%% return",
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

// structToJSON converts a *structpb.Struct to a JSON string for display.
func structToJSON(s *structpb.Struct) string {
	if s == nil {
		return ""
	}
	b, _ := protojson.Marshal(s)
	return string(b)
}
