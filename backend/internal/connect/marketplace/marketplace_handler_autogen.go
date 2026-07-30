package marketplace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

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

// limiter returns the auto-generate rate limiter, lazily initialized.
func (s *MarketplaceServer) limiter() *autoGenerateLimiter {
	s.limiterOnce.Do(func() {
		if s.autoLimiter == nil {
			s.autoLimiter = newAutoGenerateLimiter(4, 10)
		}
	})
	return s.autoLimiter
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

	symbol := ""
	if len(msg.Symbols) > 0 {
		symbol = msg.Symbols[0]
	}
	agentReq := &antv1.AgentGenerateStrategyRequest{
		Message:   msg.Description,
		Symbol:    symbol,
		Timeframe: msg.Timeframe,
		Locale:    msg.Language,
		BacktestConfig: &antv1.AgentBacktestConfig{
			Symbol:    symbol,
			Timeframe: msg.Timeframe,
		},
	}

	return s.runGeneratePipeline(ctx, uid, agentReq, msg.Description, msg.RiskLevel, symbol, msg.Timeframe, msg.AutoPublish, stream)
}

// runGeneratePipeline is the shared generation → quality → publish pipeline
// used by both GenerateAndPublish and GenerateFromTemplate.
func (s *MarketplaceServer) runGeneratePipeline(
	ctx context.Context,
	uid uuid.UUID,
	agentReq *antv1.AgentGenerateStrategyRequest,
	description string,
	riskLevel string,
	symbol string,
	timeframe string,
	autoPublish bool,
	stream *connect.ServerStream[antv1.GenerateAndPublishEvent],
) error {
	send := func(stage, message string, progress float64) error {
		return stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:    stage,
			Message:  message,
			Progress: progress,
		})
	}

	sendErr := func(stage, detail string, retryable bool) error {
		return stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:       "failed",
			ErrorStage:  stage,
			ErrorDetail: detail,
			Retryable:   retryable,
		})
	}

	if err := send("generating", "AI is generating strategy code...", 0.05); err != nil {
		return nil
	}

	var finalSource string
	var finalResult *antv1.AgentBacktestResult
	var genErr error

	genStream := func(chunk *antv1.AgentGenerateStrategyChunk) error {
		return genStreamChunk(chunk, stream, &finalSource, &finalResult, &genErr)
	}

	if err := s.gen.Generate(ctx, uid, agentReq, genStream); err != nil {
		s.log.Warn("autogen: generation failed", zap.Error(err))
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

	title := generateTitle(description)

	templateID, err := s.saveGeneratedStrategy(ctx, uid, title, description, finalSource)
	if err != nil {
		_ = sendErr("generating", fmt.Sprintf("failed to save strategy code: %v", err), true)
		return nil
	}

	if err := send("evaluating", "Evaluating backtest quality...", 0.85); err != nil {
		return nil
	}

	snapshotProto := buildSnapshotProto(finalResult)
	if s.handleQualityViolations(ctx, stream, finalResult, finalSource, templateID, snapshotProto) {
		return nil
	}

	if !autoPublish {
		_ = stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:        "completed",
			Message:      "Strategy generated successfully, ready for review",
			Progress:     1.0,
			StrategyId:   templateID,
			PythonSource: finalSource,
			Backtest:     buildSnapshot(finalResult),
		})
		return nil
	}

	return s.publishGeneratedStrategy(publishGeneratedParams{
		ctx:           ctx,
		stream:        stream,
		sendErr:       sendErr,
		uid:           uid,
		templateID:    templateID,
		title:         title,
		description:   description,
		symbol:        symbol,
		timeframe:     timeframe,
		riskLevel:     riskLevel,
		snapshotProto: snapshotProto,
		finalSource:   finalSource,
		finalResult:   finalResult,
	})
}

func (s *MarketplaceServer) handleQualityViolations(ctx context.Context, stream *connect.ServerStream[antv1.GenerateAndPublishEvent], finalResult *antv1.AgentBacktestResult, finalSource, templateID string, snapshotProto []byte) bool {
	violations, qErr := s.svc.ValidateBacktestQuality(ctx, snapshotProto, templateID)
	if qErr != nil {
		s.log.Warn("autogen: quality validation error", zap.Error(qErr))
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
		_ = stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:        "completed",
			Message:      "Strategy generated but did not pass quality gates",
			Progress:     1.0,
			StrategyId:   templateID,
			PythonSource: finalSource,
			Violations:   violationInfos,
		})
		return true
	}
	return false
}

func genStreamChunk(chunk *antv1.AgentGenerateStrategyChunk, stream *connect.ServerStream[antv1.GenerateAndPublishEvent], finalSource *string, finalResult **antv1.AgentBacktestResult, genErr *error) error {
	if chunk.Delta != "" && (chunk.Phase == "generating" || chunk.Phase == "thinking") {
		if err := stream.Send(&antv1.GenerateAndPublishEvent{
			Stage:    "generating",
			Delta:    chunk.Delta,
			Progress: 0.1,
		}); err != nil {
			return err
		}
	}
	if chunk.Phase == "backtesting" && chunk.Result != nil {
		*finalResult = chunk.Result
	}
	if chunk.CompileError != "" {
		*genErr = fmt.Errorf("compile failed: %s", chunk.CompileError)
	}
	if chunk.BacktestError != "" {
		*genErr = fmt.Errorf("backtest failed: %s", chunk.BacktestError)
	}
	if chunk.PythonSource != "" {
		*finalSource = chunk.PythonSource
	}
	if chunk.Phase == "done" {
		if chunk.PythonSource != "" {
			*finalSource = chunk.PythonSource
		}
		if chunk.Result != nil {
			*finalResult = chunk.Result
		}
		if chunk.Error != "" {
			*genErr = fmt.Errorf("%s", chunk.Error)
		}
	}
	return nil
}

func (s *MarketplaceServer) saveGeneratedStrategy(ctx context.Context, uid uuid.UUID, title, description, source string) (string, error) {
	var templateID string
	err := s.pgPool.QueryRow(ctx,
		`INSERT INTO strategy_templates (user_id, name, description, code, is_public, is_system, tags, use_count)
		 VALUES ($1, $2, $3, $4, false, false, '{}', 0)
		 RETURNING id`,
		uid, title, description, source,
	).Scan(&templateID)
	if err != nil {
		s.log.Warn("autogen: create strategy_template failed", zap.Error(err))
	}
	return templateID, err
}

type publishGeneratedParams struct {
	ctx           context.Context
	stream        *connect.ServerStream[antv1.GenerateAndPublishEvent]
	sendErr       func(string, string, bool) error
	uid           uuid.UUID
	templateID    string
	title         string
	description   string
	symbol        string
	timeframe     string
	riskLevel     string
	snapshotProto []byte
	finalSource   string
	finalResult   *antv1.AgentBacktestResult
}

func (s *MarketplaceServer) publishGeneratedStrategy(p publishGeneratedParams) error {
	if err := p.stream.Send(&antv1.GenerateAndPublishEvent{
		Stage:    "publishing",
		Message:  "Publishing to marketplace...",
		Progress: 0.95,
	}); err != nil {
		return nil
	}

	publishParams := marketplace.PublishParams{
		UserID:               p.uid.String(),
		StrategyID:           p.templateID,
		Title:                p.title,
		Description:          p.description,
		PriceModel:           marketplace.PriceModelFree,
		PriceAmount:          "0",
		AssetClass:           "forex",
		Symbols:              []string{p.symbol},
		Timeframe:            p.timeframe,
		RiskLevel:            p.riskLevel,
		BacktestSnapshotProto: p.snapshotProto,
	}

	publishID, pErr := s.svc.Publish(p.ctx, publishParams)
	if pErr != nil {
		s.log.Warn("autogen: publish failed", zap.Error(pErr))
		_ = p.sendErr("publishing", pErr.Error(), true)
		return nil
	}

	_ = p.stream.Send(&antv1.GenerateAndPublishEvent{
		Stage:        "completed",
		Message:      "Strategy generated and published successfully",
		Progress:     1.0,
		StrategyId:   p.templateID,
		PublishId:    publishID,
		PythonSource: p.finalSource,
		Backtest:     buildSnapshot(p.finalResult),
	})
	return nil
}
