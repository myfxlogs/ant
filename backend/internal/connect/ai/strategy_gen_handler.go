package ai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

const maxClarificationRounds = 3

type StrategyGenServer struct {
	systemSvc    *systemai.Service
	templatesRepo *repository.AIStrategyTemplatesRepository
	convRepo     *repository.AIConversationRepository
	backtestRepo *repository.BacktestRunRepository
	log          *zap.Logger
}

var _ antv1c.StrategyGenerationServiceHandler = (*StrategyGenServer)(nil)

func NewStrategyGenServer(
	systemSvc *systemai.Service,
	templatesRepo *repository.AIStrategyTemplatesRepository,
	convRepo *repository.AIConversationRepository,
	backtestRepo *repository.BacktestRunRepository,
	log *zap.Logger,
) *StrategyGenServer {
	return &StrategyGenServer{
		systemSvc:    systemSvc,
		templatesRepo: templatesRepo,
		convRepo:     convRepo,
		backtestRepo: backtestRepo,
		log:          log,
	}
}

func (s *StrategyGenServer) GenerateStrategy(
	ctx context.Context,
	req *connect.Request[antv1.GenerateStrategyRequest],
	stream *connect.ServerStream[antv1.GenerateStrategyChunk],
) error {
	userID, err := userIDFromCtx(ctx)
	if err != nil { return err }
	m := req.Msg

	// ── Phase 3: FEEDBACK MODE ──
	if m.PreviousCode != "" && m.BacktestMetricsJson != "" {
		return s.handleFeedback(ctx, userID, m, stream)
	}

	if result := s.runClarification(m); result != nil {
		return stream.Send(&antv1.GenerateStrategyChunk{Phase: "clarifying", Questions: result.Questions})
	}

	tmpl, sysPrompt, userPrompt := s.buildStrategyPrompt(ctx, userID, m)
	if err := s.sendTemplateInfo(stream, tmpl); err != nil { return err }

	code, err := s.streamLLMCode(ctx, userID, sysPrompt, userPrompt, stream)
	if err != nil {
		s.log.Error("LLM stream failed", zap.Error(err))
		return stream.Send(&antv1.GenerateStrategyChunk{Phase: "done", Error: systemai.FriendlyError(err)})
	}

	if issues := s.runComplianceCheck(code); len(issues) > 0 {
		return stream.Send(&antv1.GenerateStrategyChunk{Phase: "compliance", Code: code, ComplianceIssues: issues})
	}

	runID, btErr := s.finalizeWithBacktest(ctx, userID, code, m.Symbol, m.Timeframe)

	// Auto-persist exchange to strategy session
	s.persistExchange(ctx, userID, m.ConversationId, m.Message, code)

	return stream.Send(&antv1.GenerateStrategyChunk{Phase: "done", Code: code, BacktestRunId: runID, Error: btErr})
}

func (s *StrategyGenServer) buildStrategyPrompt(ctx context.Context, userID uuid.UUID, m *antv1.GenerateStrategyRequest) (*repository.AIStrategyTemplate, string, string) {
	templates, _ := s.templatesRepo.ListActive(ctx)
	lib := ai.NewTemplateLibrary(templates)
	tmpl := lib.Match(m.Message)
	builder := ai.NewStrategyPromptBuilder()
	pp := &ai.PromptParams{
		Template: tmpl, Message: m.Message, Symbol: m.Symbol, Timeframe: m.Timeframe,
		ParamMap: s.buildParamMap(m), History: s.loadHistory(ctx, userID, m.ConversationId),
	}
	return tmpl, builder.BuildSystemPrompt(pp), builder.BuildUserPrompt(pp)
}

func (s *StrategyGenServer) sendTemplateInfo(stream *connect.ServerStream[antv1.GenerateStrategyChunk], tmpl *repository.AIStrategyTemplate) error {
	tmplName := ""
	if tmpl != nil { tmplName = tmpl.Name }
	return stream.Send(&antv1.GenerateStrategyChunk{Phase: "generating", TemplateName: tmplName})
}

func (s *StrategyGenServer) streamLLMCode(ctx context.Context, userID uuid.UUID, sysPrompt, userPrompt string, stream *connect.ServerStream[antv1.GenerateStrategyChunk]) (string, error) {
	var codeBuf strings.Builder
	err := s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
		"", func(chunk systemai.ChatStreamChunk) error {
			if err := stream.Send(&antv1.GenerateStrategyChunk{Phase: "generating", Delta: chunk.Content}); err != nil {
				return err
			}
			codeBuf.WriteString(chunk.Content)
			return nil
		})
	if err != nil { return "", err }
	return s.extractCode(codeBuf.String()), nil
}

func (s *StrategyGenServer) runComplianceCheck(code string) []string {
	scanner := ai.NewCodeComplianceScanner()
	blocks, _ := scanner.Scan(code)
	_, missingSigs := scanner.HasRequiredSignature(code)
	return s.collectComplianceIssues(blocks, missingSigs)
}

func (s *StrategyGenServer) finalizeWithBacktest(ctx context.Context, userID uuid.UUID, code, symbol, timeframe string) (string, string) {
	runID, err := s.triggerBacktest(ctx, userID, code, symbol, timeframe)
	if err != nil {
		s.log.Warn("auto-backtest trigger failed", zap.Error(err))
		return "", "backtest trigger failed: " + err.Error()
	}
	return runID, ""
}

func (s *StrategyGenServer) runClarification(m *antv1.GenerateStrategyRequest) *ai.ClarificationResult {
	if m.ClarificationRound >= maxClarificationRounds {
		return nil
	}
	rules := defaultClarificationRules()
	engine := ai.NewClarificationEngine(rules)
	return engine.Check(m.Message)
}

// ── Phase 3: feedback mode ──

// handleFeedback executes the feedback iteration loop:
// parse backtest metrics → route feedback → build feedback prompt →
// stream LLM with section parsing → compliance check → auto-backtest.
func (s *StrategyGenServer) handleFeedback(
	ctx context.Context, userID uuid.UUID,
	m *antv1.GenerateStrategyRequest,
	stream *connect.ServerStream[antv1.GenerateStrategyChunk],
) error {
	// 1. Parse backtest metrics
	var metrics ai.FeedbackMetrics
	if err := json.Unmarshal([]byte(m.BacktestMetricsJson), &metrics); err != nil {
		s.log.Warn("feedback: parse backtest metrics failed", zap.Error(err))
		return stream.Send(&antv1.GenerateStrategyChunk{
			Phase: "done", Error: "failed to parse backtest metrics",
		})
	}

	// 2. Get feedback routing hints
	routing := ai.RouteFeedback(m.FeedbackMessage, &metrics)

	// 3. Build feedback prompt
	builder := ai.NewStrategyPromptBuilder()
	sysPrompt, userPrompt := builder.BuildFeedbackPrompt(&ai.FeedbackPromptParams{
		PreviousCode:    m.PreviousCode,
		Metrics:         &metrics,
		FeedbackMessage: m.FeedbackMessage,
		FeedbackHints:   routing.Reason,
	})

	// 4. Stream LLM response with progressive section parsing
	if err := stream.Send(&antv1.GenerateStrategyChunk{Phase: "analyzing"}); err != nil {
		return err
	}

	var fullBuf strings.Builder
	err := s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
		"", func(chunk systemai.ChatStreamChunk) error {
			fullBuf.WriteString(chunk.Content)
			sections := parseSections(fullBuf.String())
			return stream.Send(&antv1.GenerateStrategyChunk{
				Phase:    "generating",
				Delta:    chunk.Content,
				Analysis: sections.Analysis,
				Advice:   sections.Advice,
			})
		})
	if err != nil {
		s.log.Error("feedback: LLM stream failed", zap.Error(err))
		return stream.Send(&antv1.GenerateStrategyChunk{
			Phase: "done", Error: systemai.FriendlyError(err),
		})
	}

	// 5. Final parse: extract code from sections
	raw := fullBuf.String()
	fullSections := parseSections(raw)
	code := s.extractCode(fullSections.Code)

	// 6. Compliance check
	if issues := s.runComplianceCheck(code); len(issues) > 0 {
		return stream.Send(&antv1.GenerateStrategyChunk{
			Phase: "compliance", Code: code, ComplianceIssues: issues,
			Analysis: fullSections.Analysis, Advice: fullSections.Advice,
		})
	}

	// 7. Auto-backtest
	runID, btErr := s.finalizeWithBacktest(ctx, userID, code, m.Symbol, m.Timeframe)

	// 8. Persist exchange
	s.persistExchange(ctx, userID, m.ConversationId, m.FeedbackMessage, code)

	return stream.Send(&antv1.GenerateStrategyChunk{
		Phase: "done", Code: code, BacktestRunId: runID,
		Analysis: fullSections.Analysis, Advice: fullSections.Advice,
		Error: btErr,
	})
}

// persistExchange saves user+assistant messages to the conversation store.
func (s *StrategyGenServer) persistExchange(ctx context.Context, userID uuid.UUID, convID, userMsg, assistantMsg string) {
	if convID == "" || userMsg == "" || assistantMsg == "" {
		return
	}
	cid, parseErr := uuid.Parse(convID)
	if parseErr != nil {
		return
	}
	if _, err := s.convRepo.AddMessage(ctx, userID, cid, "user", userMsg); err != nil {
		s.log.Warn("persist user msg failed", zap.Error(err))
	}
	if _, err := s.convRepo.AddMessage(ctx, userID, cid, "assistant", assistantMsg); err != nil {
		s.log.Warn("persist assistant msg failed", zap.Error(err))
	}
	if err := s.convRepo.Touch(ctx, cid, userID); err != nil {
		s.log.Warn("touch session failed", zap.Error(err))
	}
}

func defaultClarificationRules() []ai.ClarificationRule {
	return []ai.ClarificationRule{
		{Keywords: []string{"稳健", "保守", "低风险", "安全", "稳当"},
			Questions: []string{"您能接受的最大回撤是多少？（例如：10% 以内）", "您偏好什么持仓周期？（日内/短线/中线/长线）"},
			ParamMap: map[string]string{"max_drawdown": "0.10", "risk_level": "low"}, Priority: 10},
		{Keywords: []string{"进攻", "激进", "高风险", "高收益", "快速"},
			Questions: []string{"您能接受的最大回撤是多少？（例如：30%）", "是否允许日内高频交易？"},
			ParamMap: map[string]string{"max_drawdown": "0.30", "risk_level": "high"}, Priority: 10},
		{Keywords: []string{"波段", "高抛低吸", "震荡", "短线", "日内"},
			Questions: []string{"您想操作的品种是什么？", "预计持仓时间是几小时还是几天？"},
			ParamMap: map[string]string{"strategy_family": "mean_reversion"}, Priority: 10},
	}
}

