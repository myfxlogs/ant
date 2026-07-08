package agent

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/internal/service/systemai"
)


// Generator orchestrates the strategy generation Agent loop:
// intent → profile → plan → AgentLoop (generate/compile/backtest/fix).
type Generator struct {
	mkt              repository.MarketDataStore
	btRepo           *repository.BacktestRunRepository
	dbExec           func(ctx context.Context, sql string, args ...any) error
	dbQuery          func(ctx context.Context, sql string, args ...any) (string, error)
	aiSvc            *systemai.Service
	log              *zap.Logger
	cache            *LLCache
	memory           *MemoryStore
	conversationRepo *repository.AIConversationRepository
}

// SetConversationRepo injects the conversation store for multi-turn history.
func (g *Generator) SetConversationRepo(r *repository.AIConversationRepository) {
	g.conversationRepo = r
}

// NewGenerator creates the strategy generation orchestrator.
func NewGenerator(aiSvc *systemai.Service, log *zap.Logger, cache *LLCache, memory *MemoryStore, mkt repository.MarketDataStore, btRepo *repository.BacktestRunRepository, dbExec func(ctx context.Context, sql string, args ...any) error, dbQuery func(ctx context.Context, sql string, args ...any) (string, error)) *Generator {
	return &Generator{aiSvc: aiSvc, log: log, cache: cache, memory: memory, mkt: mkt, btRepo: btRepo, dbExec: dbExec, dbQuery: dbQuery}
}

// generateState tracks mutable state during AgentLoop execution — tools update
// compile/backtest results so runAgentLoop can inspect final state after completion.
type generateState struct {
	PythonSource  string
	CompileError  string
	BacktestError string
	LastBacktest  *backtestSummary // captured on write_strategy success for persistent memory
}

// Generate runs the unified AgentLoop — single path, full tools, no pre-processing.
// The LLM decides autonomously: discuss, query data, generate code, compile.
func (g *Generator) Generate(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	stream func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	streamOrAbort := func(chunk *antv1.AgentGenerateStrategyChunk) error {
		if err := stream(chunk); err != nil {
			g.log.Info("generator: client disconnected, aborting", zap.Error(err))
			return err
		}
		return nil
	}
	return g.runAgentLoop(ctx, userID, msg, streamOrAbort)
}
