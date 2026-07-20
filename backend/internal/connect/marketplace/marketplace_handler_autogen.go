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

	// ── Stage 1: AI generation ──
	if err := send("generating", "AI is generating strategy code...", 0.05); err != nil {
		return nil
	}

	var finalSource string
	var finalResult *antv1.AgentBacktestResult
	var genErr error

	genStream := func(chunk *antv1.AgentGenerateStrategyChunk) error {
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
			finalResult = chunk.Result
		}
		if chunk.CompileError != "" {
			genErr = fmt.Errorf("compile failed: %s", chunk.CompileError)
		}
		if chunk.BacktestError != "" {
			genErr = fmt.Errorf("backtest failed: %s", chunk.BacktestError)
		}
		if chunk.PythonSource != "" {
			finalSource = chunk.PythonSource
		}
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

	// ── Stage 1b: Persist source code ──
	// Always save the generated code to strategy_templates so it survives
	// across page refreshes and is available for pricing/publish later.
	title := generateTitle(description)

	var templateID string
	if qErr := s.pgPool.QueryRow(ctx,
		`INSERT INTO strategy_templates (user_id, name, description, code, is_public, is_system, tags, use_count)
		 VALUES ($1, $2, $3, $4, false, false, '{}', 0)
		 RETURNING id`,
		uid, title, description, finalSource,
	).Scan(&templateID); qErr != nil {
		s.log.Warn("autogen: create strategy_template failed", zap.Error(qErr))
		_ = sendErr("generating", fmt.Sprintf("failed to save strategy code: %v", qErr), true)
		return nil
	}

	// ── Stage 2: Quality evaluation ──
	if err := send("evaluating", "Evaluating backtest quality...", 0.85); err != nil {
		return nil
	}

	snapshotProto := buildSnapshotProto(finalResult)
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
		return nil
	}

	// ── Stage 3: Auto-publish ──
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

	if err := send("publishing", "Publishing to marketplace...", 0.95); err != nil {
		return nil
	}

	publishParams := marketplace.PublishParams{
		UserID:               uid.String(),
		StrategyID:           templateID,
		Title:                title,
		Description:          description,
		PriceModel:           marketplace.PriceModelFree,
		PriceAmount:          "0",
		AssetClass:           "forex",
		Symbols:              []string{symbol},
		Timeframe:            timeframe,
		RiskLevel:            riskLevel,
		BacktestSnapshotProto: snapshotProto,
	}

	publishID, pErr := s.svc.Publish(ctx, publishParams)
	if pErr != nil {
		s.log.Warn("autogen: publish failed", zap.Error(pErr))
		_ = sendErr("publishing", pErr.Error(), true)
		return nil
	}

	_ = stream.Send(&antv1.GenerateAndPublishEvent{
		Stage:        "completed",
		Message:      "Strategy generated and published successfully",
		Progress:     1.0,
		StrategyId:   templateID,
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
	if s.gen == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI strategy generation is not available"))
	}
	if s.pgPool == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("database not configured"))
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
	if msg.TemplateId == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("template_id is required"))
	}

	// Look up template.
	var templateKey, templateType, nameI18n, descI18n, riskLevel string
	err = s.pgPool.QueryRow(ctx,
		`SELECT template_key, template_type, name_i18n, description_i18n, default_risk_level
		 FROM strategy_parameter_templates WHERE id=$1 AND enabled=true`,
		msg.TemplateId).Scan(&templateKey, &templateType, &nameI18n, &descI18n, &riskLevel)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("template not found: %w", err))
	}

	lang := langFromAccept(req.Header().Get("Accept-Language"))
	tmplName := pickLocalized(nameI18n, lang)
	tmplDesc := pickLocalized(descI18n, lang)

	// Build natural language description from template + parameters.
	description := fmt.Sprintf("Generate a %s strategy using the '%s' template (%s). Parameters: %s. Symbol: %s, Timeframe: %s.",
		templateType, tmplName, tmplDesc, msg.ParametersJson, msg.Symbol, msg.Timeframe)

	// Rate limit.
	if !s.limiter().acquire(userID) {
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limit exceeded"))
	}
	defer s.limiter().release()

	// Build agent request.
	agentReq := &antv1.AgentGenerateStrategyRequest{
		Message:   description,
		Symbol:    msg.Symbol,
		Timeframe: msg.Timeframe,
		Locale:    lang,
		BacktestConfig: &antv1.AgentBacktestConfig{
			Symbol:    msg.Symbol,
			Timeframe: msg.Timeframe,
		},
	}

	// Reuse the same generation → quality → publish pipeline.
	return s.runGeneratePipeline(ctx, uid, agentReq, description, riskLevel, msg.Symbol, msg.Timeframe, msg.AutoPublish, stream)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

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
