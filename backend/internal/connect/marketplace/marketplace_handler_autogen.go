package marketplace

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
)

// ── Rate limiter ─────────────────────────────────────────────────────────────

// autoGenerateLimiter enforces per-user rate limits and global concurrency.
type autoGenerateLimiter struct {
	userLimits sync.Map           // userID → *rate.Limiter
	globalSem  chan struct{}      // concurrency semaphore
}

func newAutoGenerateLimiter(globalConcurrency int, perUserPerHour int) *autoGenerateLimiter {
	return &autoGenerateLimiter{
		globalSem: make(chan struct{}, globalConcurrency),
	}
}

// perUserLimiter returns the rate limiter for a user, creating one if needed.
func (l *autoGenerateLimiter) perUserLimiter(userID string) *rate.Limiter {
	if v, ok := l.userLimits.Load(userID); ok {
		return v.(*rate.Limiter)
	}
	lim := rate.NewLimiter(rate.Every(time.Hour/10), 10) // 10 per hour
	actual, loaded := l.userLimits.LoadOrStore(userID, lim)
	if loaded {
		return actual.(*rate.Limiter)
	}
	return lim
}

func (l *autoGenerateLimiter) acquire(userID string) bool {
	if !l.perUserLimiter(userID).Allow() {
		return false
	}
	select {
	case l.globalSem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *autoGenerateLimiter) release() {
	<-l.globalSem
}

// ── GenerateAndPublish handler ───────────────────────────────────────────────

func (s *MarketplaceServer) GenerateAndPublish(
	ctx context.Context,
	req *connect.Request[antv1.GenerateAndPublishRequest],
	stream *connect.ServerStream[antv1.GenerateAndPublishEvent],
) error {
	if s.gen == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI strategy generation is not available"))
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id: %w", err))
	}

	msg := req.Msg
	if msg.Description == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("description is required"))
	}

	// Rate limit check.
	if !s.limiter().acquire(userID) {
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limit exceeded: too many generation requests"))
	}
	defer s.limiter().release()

	send := func(stage, message string, progress float64) error {
		return stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:    stage,
			Message:  message,
			Progress: progress,
		})
	}

	sendErr := func(stage, detail string, retryable bool) error {
		return stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:        "failed",
			ErrorStage:   stage,
			ErrorDetail:  detail,
			Retryable:    retryable,
		})
	}

	// ── Stage 1: AI generation (reuses existing Generator.Generate) ──
	if err := send("generating", "AI is generating strategy code...", 0.05); err != nil {
		return nil
	}

	// Build the agent request from the marketplace request.
	symbol := ""
	if len(msg.Symbols) > 0 {
		symbol = msg.Symbols[0]
	}
	agentReq := &antv1.AgentGenerateStrategyRequest{
		Message: msg.Description,
		Symbol:  symbol,
		Timeframe: msg.Timeframe,
		Locale:  msg.Language,
		BacktestConfig: &antv1.AgentBacktestConfig{
			Symbol:    symbol,
			Timeframe: msg.Timeframe,
			StartDateMs: 0, // default range
			EndDateMs:   0,
		},
	}

	// Capture the final state from the agent stream.
	var finalSource string
	var finalResult *antv1.AgentBacktestResult
	var genErr error

	genStream := func(chunk *antv1.AgentGenerateStrategyChunk) error {
		// Forward generating deltas to the client.
		if chunk.Delta != "" && (chunk.Phase == "generating" || chunk.Phase == "thinking") {
			if err := stream.Send(&antv1.GenerateAndPublishEvent{
				Stage:   "generating",
				Delta:   chunk.Delta,
				Progress: 0.1,
			}); err != nil {
				return err
			}
		}
		// Capture backtest progress.
		if chunk.Phase == "backtesting" && chunk.Result != nil {
			finalResult = chunk.Result
		}
		// Capture compile errors.
		if chunk.CompileError != "" {
			genErr = fmt.Errorf("compile failed: %s", chunk.CompileError)
		}
		// Capture backtest errors.
		if chunk.BacktestError != "" {
			genErr = fmt.Errorf("backtest failed: %s", chunk.BacktestError)
		}
		// Capture final source.
		if chunk.PythonSource != "" {
			finalSource = chunk.PythonSource
		}
		// On done phase, capture final result.
		if chunk.Phase == "done" {
			if chunk.PythonSource != "" {
				finalSource = chunk.PythonSource
			}
			if chunk.Result != nil {
				finalResult = chunk.Result
			}
			if chunk.Error != "" {
				genErr = fmt.Errorf("%s", chunk.Error)
			}
		}
		return nil
	}

	if err := s.gen.Generate(ctx, uid, agentReq, genStream); err != nil {
		s.log.Warn("GenerateAndPublish: generation failed", zap.Error(err))
		_ = sendErr("generating", err.Error(), true)
		return nil
	}

	if genErr != nil {
		_ = sendErr("generating", genErr.Error(), true)
		return nil
	}

	if finalSource == "" {
		_ = sendErr("generating", "AI did not produce any strategy code", true)
		return nil
	}

	// ── Stage 2: Quality evaluation ──
	if err := send("evaluating", "Evaluating backtest quality...", 0.85); err != nil {
		return nil
	}

	// Build BacktestSnapshot proto from the agent result.
	snapshotProto := buildSnapshotProto(finalResult)
	violations, qErr := s.svc.ValidateBacktestQuality(ctx, snapshotProto, "")
	if qErr != nil {
		s.log.Warn("GenerateAndPublish: quality validation error", zap.Error(qErr))
	}

	var violationInfos []*antv1.QualityViolationInfo
	for _, v := range violations {
		violationInfos = append(violationInfos, &antv1.QualityViolationInfo{
			Metric:    v.Metric,
			Actual:    v.Actual,
			Threshold: v.Threshold,
		})
	}

	if len(violations) > 0 {
		// Quality gate failed — return completed with violations, do not publish.
		_ = stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:      "completed",
			Message:    "Strategy generated but did not pass quality gates",
			Progress:   1.0,
			PythonSource: finalSource,
			Violations: violationInfos,
		})
		return nil
	}

	// ── Stage 3: Auto-publish ──
	if !msg.AutoPublish {
		// User wants to review before publishing.
		_ = stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:        "completed",
			Message:      "Strategy generated successfully, ready for review",
			Progress:     1.0,
			PythonSource: finalSource,
			Backtest:     buildSnapshot(finalResult),
		})
		return nil
	}

	if err := send("publishing", "Publishing to marketplace...", 0.95); err != nil {
		return nil
	}

	title := msg.TitleOverride
	if title == "" {
		title = generateTitle(msg.Description)
	}

	publishParams := marketplace.PublishParams{
		UserID:               userID,
		StrategyID:           uuid.New().String(),
		Title:                title,
		Description:          msg.Description,
		PriceModel:           marketplace.PriceModelFree,
		PriceAmount:          "0",
		AssetClass:           msg.AssetClass,
		Symbols:              msg.Symbols,
		Timeframe:            msg.Timeframe,
		RiskLevel:            msg.RiskLevel,
		BacktestSnapshotProto: snapshotProto,
	}

	publishID, pErr := s.svc.Publish(ctx, publishParams)
	if pErr != nil {
		s.log.Warn("GenerateAndPublish: publish failed", zap.Error(pErr))
		_ = sendErr("publishing", pErr.Error(), true)
		return nil
	}

	_ = stream.Send(&antv1.GenerateAndPublishEvent{
		Stage:        "completed",
		Message:      "Strategy generated and published successfully",
		Progress:     1.0,
		StrategyId:   publishParams.StrategyID,
		PublishId:    publishID,
		PythonSource: finalSource,
		Backtest:     buildSnapshot(finalResult),
	})
	return nil
}

// ── GenerateFromTemplate handler ─────────────────────────────────────────────

func (s *MarketplaceServer) GenerateFromTemplate(
	ctx context.Context,
	req *connect.Request[antv1.GenerateFromTemplateRequest],
	stream *connect.ServerStream[antv1.GenerateAndPublishEvent],
) error {
	// TODO: implement template-based generation in 2.3
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GenerateFromTemplate not yet implemented"))
}

// ── ListStrategyTemplates handler ────────────────────────────────────────────

func (s *MarketplaceServer) ListStrategyTemplates(
	ctx context.Context,
	req *connect.Request[antv1.ListStrategyTemplatesRequest],
) (*connect.Response[antv1.ListStrategyTemplatesResponse], error) {
	// TODO: implement in 2.3
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("ListStrategyTemplates not yet implemented"))
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// limiter returns the auto-generate rate limiter, lazily initialized.
func (s *MarketplaceServer) limiter() *autoGenerateLimiter {
	s.limiterOnce.Do(func() {
		if s.autoLimiter == nil {
			s.autoLimiter = newAutoGenerateLimiter(4, 10)
		}
	})
	return s.autoLimiter
}

func buildSnapshotProto(result *antv1.AgentBacktestResult) []byte {
	if result == nil {
		return nil
	}
	snap := &antv1.BacktestSnapshot{
		TotalReturn:   result.TotalReturn,
		AnnualReturn:  result.AnnualReturn,
		MaxDrawdown:   result.MaxDrawdown,
		SharpeRatio:   result.SharpeRatio,
		WinRate:       result.WinRate,
		TotalTrades:   result.TotalTrades,
	}
	data, err := proto.Marshal(snap)
	if err != nil {
		return nil
	}
	return data
}

func buildSnapshot(result *antv1.AgentBacktestResult) *antv1.BacktestSnapshot {
	if result == nil {
		return nil
	}
	return &antv1.BacktestSnapshot{
		TotalReturn:   result.TotalReturn,
		AnnualReturn:  result.AnnualReturn,
		MaxDrawdown:   result.MaxDrawdown,
		SharpeRatio:   result.SharpeRatio,
		WinRate:       result.WinRate,
		TotalTrades:   result.TotalTrades,
	}
}

func generateTitle(description string) string {
	words := strings.Fields(description)
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, " ")
}
