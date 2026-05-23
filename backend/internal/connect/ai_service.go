package connect

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type AIService struct {
	aiReportSvc *service.AIReportService
	aiCfgSvc    *service.AIConfigService
	aiAgentSvc  *service.AIAgentService
	aiManager   *ai.Manager
	convRepo    *repository.AIConversationRepository
	wfRepo      *repository.AIWorkflowRepository
	pythonSvc   *service.PythonStrategyService
}

func NewAIService(aiReportSvc *service.AIReportService, aiCfgSvc *service.AIConfigService, aiAgentSvc *service.AIAgentService, aiManager *ai.Manager, convRepo *repository.AIConversationRepository, wfRepo *repository.AIWorkflowRepository, pythonSvc *service.PythonStrategyService) *AIService {
	return &AIService{
		aiReportSvc: aiReportSvc,
		aiCfgSvc:    aiCfgSvc,
		aiAgentSvc:  aiAgentSvc,
		aiManager:   aiManager,
		convRepo:    convRepo,
		wfRepo:      wfRepo,
		pythonSvc:   pythonSvc,
	}
}

func (s *AIService) GetAIReports(ctx context.Context, req *connect.Request[v1.GetAIReportsRequest]) (*connect.Response[v1.GetAIReportsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	_ = userID

	return connect.NewResponse(&v1.GetAIReportsResponse{
		Reports: []*v1.AIReport{},
	}), nil
}

func (s *AIService) GenerateReport(ctx context.Context, req *connect.Request[v1.GenerateReportRequest]) (*connect.Response[v1.GenerateReportResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	now := time.Now()
	var start time.Time
	switch req.Msg.Period {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	case "quarter":
		start = now.AddDate(0, -3, 0)
	case "year":
		start = now.AddDate(-1, 0, 0)
	default:
		start = now.AddDate(0, -1, 0)
	}

	report, err := s.aiReportSvc.GenerateReport(ctx, userID, accountID, start, now)
	if err != nil {
		if errors.Is(err, service.ErrNoTradeData) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.ai.no_trade_data_available"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.GenerateReportResponse{
		Report: &v1.AIReport{
			Id:              uuid.New().String(),
			AccountId:       req.Msg.AccountId,
			ReportType:      req.Msg.ReportType,
			Title:           "ai.reports.tradeAnalysis.title",
			Content:         report.Summary + "\n\n" + report.RiskAssessment,
			Recommendations: report.Suggestions,
			CreatedAt:       timestamppb.Now(),
		},
	}), nil
}

func (s *AIService) Chat(ctx context.Context, req *connect.Request[v1.ChatRequest]) (*connect.Response[v1.ChatResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	provider, aiMessages, convID, notConfigured, prepErr := s.prepareChatInvocation(ctx, userID, req.Msg)
	if prepErr != nil {
		var cerr *connect.Error
		if errors.As(prepErr, &cerr) {
			return nil, cerr
		}
		return nil, connect.NewError(connect.CodeInternal, prepErr)
	}
	if notConfigured {
		return connect.NewResponse(&v1.ChatResponse{Message: "errors.ai.not_configured", Suggestions: []string{}}), nil
	}

	llmCtx, cancel := context.WithTimeout(ctx, chatProviderDeadline)
	defer cancel()
	resp, err := service.ChatWithRetry(llmCtx, provider, aiMessages)
	if err != nil {
		return nil, mapChatProviderError(err)
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("errors.ai.provider_returned_empty_message"))
	}

	persistChatTurn(ctx, s, convID, userID, req.Msg.GetMessage(), resp.Content)

	return connect.NewResponse(&v1.ChatResponse{
		Message:     resp.Content,
		Suggestions: []string{},
	}), nil
}

// ChatStream streams assistant tokens; see docs/AI对话体验与可靠性优化.md
func (s *AIService) ChatStream(ctx context.Context, req *connect.Request[v1.ChatRequest], stream *connect.ServerStream[v1.ChatStreamChunk]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	provider, aiMessages, convID, notConfigured, prepErr := s.prepareChatInvocation(ctx, userID, req.Msg)
	if prepErr != nil {
		var cerr *connect.Error
		if errors.As(prepErr, &cerr) {
			return cerr
		}
		return connect.NewError(connect.CodeInternal, prepErr)
	}
	if notConfigured {
		_ = stream.Send(&v1.ChatStreamChunk{Done: true, ErrorMessage: "errors.ai.not_configured"})
		return nil
	}

	llmCtx, cancel := context.WithTimeout(ctx, chatProviderDeadline)
	defer cancel()

	full, usage, err := service.StreamChatWithRetry(llmCtx, provider, aiMessages, func(delta string) {
		if delta == "" {
			return
		}
		_ = stream.Send(&v1.ChatStreamChunk{Delta: delta})
	})
	if err != nil {
		_ = stream.Send(&v1.ChatStreamChunk{Done: true, ErrorMessage: err.Error()})
		return nil
	}
	if strings.TrimSpace(full) == "" {
		_ = stream.Send(&v1.ChatStreamChunk{Done: true, ErrorMessage: "errors.ai.provider_returned_empty_message"})
		return nil
	}

	persistChatTurn(ctx, s, convID, userID, req.Msg.GetMessage(), full)
	_ = stream.Send(&v1.ChatStreamChunk{
		Done:               true,
		PromptTokens:       int32(usage.PromptTokens),
		CompletionTokens:   int32(usage.CompletionTokens),
	})
	return nil
}
// extractContextField 从 context 字符串中提取 "Key: value" 格式的字段值
func extractContextField(ctx, key string) string {
	prefix := key + ":"
	idx := strings.Index(ctx, prefix)
	if idx < 0 {
		return ""
	}
	rest := ctx[idx+len(prefix):]
	if j := strings.IndexAny(rest, "\n\r"); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}
