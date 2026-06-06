package ai

import (
	"context"
	"strings"

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
	if err != nil {
		return err
	}
	m := req.Msg

	// Phase 1a: Clarification
	if result := s.runClarification(m); result != nil {
		return stream.Send(&antv1.GenerateStrategyChunk{
			Phase:     "clarifying",
			Questions: result.Questions,
		})
	}

	// Phase 1b: Template selection
	templates, _ := s.templatesRepo.ListActive(ctx)
	lib := ai.NewTemplateLibrary(templates)
	tmpl := lib.Match(m.Message)

	// Phase 1c: Build prompt
	paramMap := s.buildParamMap(m)
	builder := ai.NewStrategyPromptBuilder()
	pp := &ai.PromptParams{
		Template:  tmpl,
		Message:   m.Message,
		Symbol:    m.Symbol,
		Timeframe: m.Timeframe,
		ParamMap:  paramMap,
		History:   s.loadHistory(ctx, userID, m.ConversationId),
	}
	sysPrompt := builder.BuildSystemPrompt(pp)
	userPrompt := builder.BuildUserPrompt(pp)

	// Phase 1d: Template info
	tmplName := ""
	if tmpl != nil {
		tmplName = tmpl.Name
	}
	if err := stream.Send(&antv1.GenerateStrategyChunk{
		Phase:        "generating",
		TemplateName: tmplName,
	}); err != nil {
		return err
	}

	// Phase 1d: LLM stream
	var codeBuf strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		}, "",
		func(chunk systemai.ChatStreamChunk) error {
			if err := stream.Send(&antv1.GenerateStrategyChunk{
				Phase: "generating",
				Delta: chunk.Content,
			}); err != nil {
				return err
			}
			codeBuf.WriteString(chunk.Content)
			return nil
		})
	if err != nil {
		s.log.Error("LLM stream failed", zap.Error(err))
		return stream.Send(&antv1.GenerateStrategyChunk{
			Phase: "done",
			Error: systemai.FriendlyError(err),
		})
	}

	code := s.extractCode(codeBuf.String())

	// Phase 1e: Compliance
	scanner := ai.NewCodeComplianceScanner()
	blocks, _ := scanner.Scan(code)
	_, missingSigs := scanner.HasRequiredSignature(code)

	allIssues := s.collectComplianceIssues(blocks, missingSigs)
	if len(allIssues) > 0 {
		return stream.Send(&antv1.GenerateStrategyChunk{
			Phase:             "compliance",
			Code:              code,
			ComplianceIssues:  allIssues,
		})
	}

	// Phase 1f: Auto-backtest
	runID, err := s.triggerBacktest(ctx, userID, code, m.Symbol, m.Timeframe)
	phase := "done"
	var btErr string
	if err != nil {
		s.log.Warn("auto-backtest trigger failed", zap.Error(err))
		btErr = "backtest trigger failed: " + err.Error()
		phase = "backtest"
	}
	if runID == "" && err == nil {
		btErr = ""
	}

	return stream.Send(&antv1.GenerateStrategyChunk{
		Phase:         phase,
		Code:          code,
		BacktestRunId: runID,
		Error:         btErr,
	})
}

func (s *StrategyGenServer) runClarification(m *antv1.GenerateStrategyRequest) *ai.ClarificationResult {
	if m.ClarificationRound >= maxClarificationRounds {
		return nil
	}
	rules := defaultClarificationRules()
	engine := ai.NewClarificationEngine(rules)
	return engine.Check(m.Message)
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

