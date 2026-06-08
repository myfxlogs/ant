package asset_analysis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/analysis"
	"anttrader/internal/interceptor"
	"anttrader/internal/service/systemai"
)

// AssetAnalysisServer implements the AssetAnalysisService ConnectRPC handler.
type AssetAnalysisServer struct {
	analyzer *analysis.Analyzer
	aiSvc    *systemai.Service
	log      *zap.Logger
}

var _ antv1c.AssetAnalysisServiceHandler = (*AssetAnalysisServer)(nil)

// NewAssetAnalysisServer creates an AssetAnalysisService handler.
func NewAssetAnalysisServer(
	analyzer *analysis.Analyzer,
	aiSvc *systemai.Service,
	log *zap.Logger,
) *AssetAnalysisServer {
	return &AssetAnalysisServer{
		analyzer: analyzer,
		aiSvc:    aiSvc,
		log:      log,
	}
}

// AnalyzeAsset performs comprehensive asset analysis and streams results
// progressively via SSE. Each frame populates a different phase:
//
//	mtf_outlook → sr_levels → volatility → ai_recommendation → complete
func (s *AssetAnalysisServer) AnalyzeAsset(
	ctx context.Context,
	req *connect.Request[antv1.AnalyzeAssetRequest],
	stream *connect.ServerStream[antv1.AnalyzeAssetResponse],
) error {
	userID, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("invalid user_id: %w", err))
	}

	symbol := req.Msg.Symbol
	if symbol == "" {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("symbol is required"))
	}

	primaryTF := req.Msg.Timeframe
	if primaryTF == "" {
		primaryTF = "D1"
	}

	klineCount := req.Msg.KlineCount
	if klineCount <= 0 {
		klineCount = 200
	}

	// Run analysis, streaming each phase.
	result, analyzeErr := s.analyzer.Analyze(ctx, symbol, req.Msg.AccountId, primaryTF, klineCount,
		func(phase string, r *analysis.AnalysisResult) error {
			resp := analysisResultToProto(phase, r)
			return stream.Send(resp)
		},
	)

	// If analysis failed entirely, send error frame.
	if analyzeErr != nil && result == nil {
		return stream.Send(&antv1.AnalyzeAssetResponse{
			Phase: "complete",
			Error: analyzeErr.Error(),
		})
	}

	// AI recommendation: build prompt and call LLM.
	if s.aiSvc != nil && result != nil {
		prompt := analysis.BuildAIPrompt(symbol, result)
		sysPrompt := "You are a quantitative trading analyst. Given market data analysis, recommend a trading strategy. Be concise and specific. Use markdown."
		messages := systemai.BuildChatMessages(sysPrompt, prompt, nil)

		recommendation, aiErr := s.aiSvc.ChatCompletion(ctx, userID, messages, "")
		if aiErr != nil {
			s.log.Warn("analysis: AI recommendation failed",
				zap.String("symbol", symbol),
				zap.Error(aiErr),
			)
			// Graceful degradation: send empty recommendation.
			recommendation = "_AI recommendation unavailable. Please configure an AI provider in Settings._"
		}
		result.AIRecommendation = recommendation

		// Stream the AI recommendation frame.
		if err := stream.Send(analysisResultToProto("ai_recommendation", result)); err != nil {
			return err
		}
	}

	// Final completion frame.
	return stream.Send(&antv1.AnalyzeAssetResponse{Phase: "complete"})
}

func analysisResultToProto(phase string, r *analysis.AnalysisResult) *antv1.AnalyzeAssetResponse {
	resp := &antv1.AnalyzeAssetResponse{
		Phase:            phase,
		Error:            r.Error,
		VolatilityState:  r.VolatilityState,
		VolatilityValue:  r.VolatilityValue,
		AiRecommendation: r.AIRecommendation,
		MultiTf: &antv1.MultiTfOutlook{
			H1: tfOutlookToProto(r.MTF.H1),
			H4: tfOutlookToProto(r.MTF.H4),
			D1: tfOutlookToProto(r.MTF.D1),
			W1: tfOutlookToProto(r.MTF.W1),
		},
	}

	for _, lvl := range r.KeyLevels {
		resp.KeyLevels = append(resp.KeyLevels, &antv1.SRLevel{
			Price:    lvl.Price,
			Type:     lvl.Type,
			Strength: lvl.Strength,
			Touches:  lvl.Touches,
		})
	}

	return resp
}

func tfOutlookToProto(o analysis.TfOutlook) *antv1.TfOutlook {
	return &antv1.TfOutlook{
		Trend:          o.Trend,
		Strength:       o.Strength,
		EmaGapPct:      o.EMAGapPct,
		PriceChangePct: o.PriceChangePct,
	}
}
