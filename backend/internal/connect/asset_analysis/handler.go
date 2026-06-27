package asset_analysis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/analysis"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
	"anttrader/internal/service/systemai"
)

// AssetAnalysisServer implements the AssetAnalysisService ConnectRPC handler.
type AssetAnalysisServer struct {
	analyzer    *analysis.Analyzer
	aiSvc       *systemai.Service
	platformSvc *service.PlatformService
	log         *zap.Logger
}

var _ antv1c.AssetAnalysisServiceHandler = (*AssetAnalysisServer)(nil)

// NewAssetAnalysisServer creates an AssetAnalysisService handler.
func NewAssetAnalysisServer(
	analyzer *analysis.Analyzer,
	aiSvc *systemai.Service,
	platformSvc *service.PlatformService,
	log *zap.Logger,
) *AssetAnalysisServer {
	return &AssetAnalysisServer{
		analyzer:    analyzer,
		aiSvc:       aiSvc,
		platformSvc: platformSvc,
		log:         log,
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

	accountID := req.Msg.AccountId
	if accountID == "" {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("account_id is required"))
	}

	// Resolve broker from account so analysis uses the user's actual broker data.
	broker, err := s.platformSvc.GetAccountBroker(ctx, accountID)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("resolve broker for account %s: %w", accountID, err))
	}
	if broker == "" {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("broker not found for account %s", accountID))
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
	result, analyzeErr := s.analyzer.Analyze(ctx, symbol, broker, primaryTF, klineCount,
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
		lang := detectLang(req.Header().Get("Accept-Language"))
		prompt := analysis.BuildAIPrompt(symbol, result) + langInstruction(lang)
		sysPrompt := buildSystemPrompt(lang)
		messages := systemai.BuildChatMessages(sysPrompt, prompt, nil)

		recommendation, aiErr := s.aiSvc.ChatCompletion(ctx, userID, messages)
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
			Price:    strconv.FormatFloat(lvl.Price, 'f', -1, 64),
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

// detectLang maps Accept-Language header to a prompt language tag.
func detectLang(acceptLang string) string {
	if acceptLang == "" {
		return "en"
	}
	// Parse primary language from header like "zh-CN,zh;q=0.9,en;q=0.8"
	primary := acceptLang
	if idx := strings.IndexByte(acceptLang, ','); idx > 0 {
		primary = acceptLang[:idx]
	}
	if idx := strings.IndexByte(primary, ';'); idx > 0 {
		primary = primary[:idx]
	}
	primary = strings.TrimSpace(primary)
	switch {
	case strings.HasPrefix(primary, "zh"):
		return "zh"
	case strings.HasPrefix(primary, "ja"):
		return "ja"
	case strings.HasPrefix(primary, "vi"):
		return "vi"
	default:
		return "en"
	}
}

// buildSystemPrompt returns a quant analyst system prompt in the user's language.
func buildSystemPrompt(lang string) string {
	switch lang {
	case "zh":
		return "你是一名量化交易分析师。根据提供的市场数据分析，推荐交易策略。要求简洁具体，使用 Markdown 格式，用中文回复。"
	case "ja":
		return "あなたはクオンツトレーディングアナリストです。提供された市場データ分析に基づいて、取引戦略を推奨してください。簡潔かつ具体的に、Markdown形式で日本語で回答してください。"
	case "vi":
		return "Bạn là nhà phân tích giao dịch định lượng. Dựa trên phân tích dữ liệu thị trường được cung cấp, hãy đề xuất chiến lược giao dịch. Hãy ngắn gọn và cụ thể, sử dụng định dạng Markdown, trả lời bằng tiếng Việt."
	default:
		return "You are a quantitative trading analyst. Given market data analysis, recommend a trading strategy. Be concise and specific. Use markdown."
	}
}

// langInstruction returns a prompt suffix instructing the LLM to respond in the user's language.
func langInstruction(lang string) string {
	switch lang {
	case "zh":
		return "\n\n请用中文回复。"
	case "ja":
		return "\n\n日本語で回答してください。"
	case "vi":
		return "\n\nVui lòng trả lời bằng tiếng Việt."
	default:
		return ""
	}
}
