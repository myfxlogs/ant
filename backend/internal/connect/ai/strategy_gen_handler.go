package ai

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

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

func (s *StrategyGenServer) buildParamMap(m *antv1.GenerateStrategyRequest) map[string]string {
	pm := map[string]string{}
	if m.Symbol != "" {
		pm["symbol"] = m.Symbol
	}
	if m.Timeframe != "" {
		pm["timeframe"] = m.Timeframe
	}
	return pm
}

// loadHistory loads recent conversation messages and returns a summary string.
func (s *StrategyGenServer) loadHistory(ctx context.Context, userID uuid.UUID, convID string) string {
	if convID == "" {
		return ""
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return ""
	}
	msgs, err := s.convRepo.GetMessages(ctx, userID, cid)
	if err != nil || len(msgs) == 0 {
		return ""
	}
	// Keep last 6 messages for context.
	start := 0
	if len(msgs) > 6 {
		start = len(msgs) - 6
	}
	var sb strings.Builder
	for _, m := range msgs[start:] {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}
	return sb.String()
}

// extractCode extracts Python code from the LLM response.
// Handles both fenced code blocks and raw output.
func (s *StrategyGenServer) extractCode(raw string) string {
	// Try markdown code fence extraction.
	start := strings.Index(raw, "```python")
	if start < 0 {
		start = strings.Index(raw, "```")
	}
	if start >= 0 {
		rest := raw[start:]
		end := strings.Index(rest[3:], "```")
		if end >= 0 {
			code := rest[3 : end+3]
			code = strings.TrimPrefix(code, "python\n")
			code = strings.TrimPrefix(code, "python")
			return strings.TrimSpace(code)
		}
	}
	// If no fences, return raw (strip common preamble).
	raw = strings.TrimSpace(raw)
	// Remove leading non-code lines (explanatory text).
	lines := strings.Split(raw, "\n")
	var codeLines []string
	inCode := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") ||
			strings.HasPrefix(strings.TrimSpace(line), "def ") ||
			strings.HasPrefix(strings.TrimSpace(line), "@param") ||
			strings.HasPrefix(strings.TrimSpace(line), "class ") {
			inCode = true
		}
		if inCode {
			codeLines = append(codeLines, line)
		}
	}
	if len(codeLines) > 0 {
		return strings.Join(codeLines, "\n")
	}
	return raw
}

// collectComplianceIssues gathers all blocking issues into a string slice.
func (s *StrategyGenServer) collectComplianceIssues(blocks []ai.ComplianceIssue, missingSigs []string) []string {
	issues := make([]string, 0, len(blocks)+len(missingSigs))
	for _, b := range blocks {
		issues = append(issues, fmt.Sprintf("[%s] %s (line %d)", b.RuleName, b.Message, b.Line))
	}
	issues = append(issues, missingSigs...)
	return issues
}

// triggerBacktest creates a PENDING backtest run for the generated code.
// Returns empty string if backtest is not available (no repo, no code, or insufficient info).
func (s *StrategyGenServer) triggerBacktest(ctx context.Context, userID uuid.UUID, code, symbol, timeframe string) (string, error) {
	if s.backtestRepo == nil {
		return "", nil
	}
	if code == "" || symbol == "" || timeframe == "" {
		return "", nil
	}
	run := &repository.BacktestRun{
		ID:            uuid.New(),
		UserID:        userID,
		AccountID:     uuid.Nil,
		Symbol:        symbol,
		Timeframe:     timeframe,
		Mode:          "KLINE_RANGE",
		Status:        "PENDING",
		StrategyCode:  &code,
		InitialCapital: f64Ptr(10000),
			Commission:       f64Ptr(0.001),
			Slippage:         f64Ptr(0),
			Leverage:         f64Ptr(1),
			TradeDirection:   strPtr("both"),
			StrictMode:       bPtr(true),
		StrategyCodeHash: "",
		Error:         "",
		ExtraSymbols:  []string{},
	}
	id, err := s.backtestRepo.Create(ctx, run)
	if err != nil {
		return "", fmt.Errorf("create backtest run: %w", err)
	}
	return id.String(), nil
}

func f64Ptr(v float64) *float64 { return &v }
func strPtr(s string) *string { if s == "" { return nil }; return &s }
func bPtr(v bool) *bool { return &v }
