package marketplace

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

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
