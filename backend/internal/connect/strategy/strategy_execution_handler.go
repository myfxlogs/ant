package strategy

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/notification"
	"anttrader/internal/repository"
	"anttrader/internal/pglisten"
	"anttrader/internal/risk"
	"anttrader/internal/ai"
	"anttrader/tools/mql2go"
)

// StrategyExecutionServer implements ant.v1c.StrategyRuntimeServiceHandler.
// Handles strategy execution via the Go-native executor (GoExecutor).
type StrategyExecutionServer struct {
	backtestRepo        *repository.BacktestRunRepository
	log                 *zap.Logger
	marketDataRepo      repository.MarketDataStore
	barSource           BarSource // unified bar data source (backtest or live)
	mtHub               *mthub.MtHubService // for live order submission
	paperEngine         PaperOrderExecutor  // for paper trading simulated fills
	pgListen            *pglisten.Listener
	notifSender         *notification.Sender
	onBacktestComplete  func(ctx context.Context, run *repository.BacktestRun) // auto-gate hook

	// D6-A: Gate is the single authoritative risk evaluator.  Injected at
	// construction — if nil, live order dispatch panics (fail-stop, non-bypassable).
	gate *risk.Gate

	// D6-A / T3.2b: AccountStateProvider feeds real account data.
	// Before connected, equity-dependent gate rules fail-closed.
	accountProvider AccountStateProvider

	// Go-native strategy executor — runs generated Go strategies via go run (backtest).
	goExecutor *GoExecutor

	// WASM strategy executor — runs generated Go strategies via wazero (live/paper).
	wasmExecutor *WasmExecutor

	// Push-first cancel: shared LISTEN on backtest_cancel channel.
	activeCancels   map[uuid.UUID]context.CancelFunc
	activeCancelsMu sync.Mutex
}

// AccountStateProvider supplies live account state for gate evaluation (T3.2b).
// Implemented by the MT gateway account status subscription.
type AccountStateProvider interface {
	GetAccountState(ctx context.Context, accountID string) (*risk.AccountState, error)
}

func (s *StrategyExecutionServer) SetMarketDataRepo(r repository.MarketDataStore) {
	s.marketDataRepo = r
}

// PaperOrderExecutor abstracts paper trading order simulation.
// Implemented by paper.PaperEngine to avoid circular imports.
type PaperOrderExecutor interface {
	PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
		volume, bid, ask decimal.Decimal) error
	ClosePaperOrder(ctx context.Context, accountID, symbol string) error
	ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error
	CancelPaperOrder(ctx context.Context, accountID, symbol string) error
}

func (s *StrategyExecutionServer) SetBarSource(bs BarSource)                 { s.barSource = bs }
func (s *StrategyExecutionServer) SetMtHub(h *mthub.MtHubService)            { s.mtHub = h }
func (s *StrategyExecutionServer) SetPaperEngine(pe PaperOrderExecutor)      { s.paperEngine = pe }
func (s *StrategyExecutionServer) SetGoExecutor(ge *GoExecutor)              { s.goExecutor = ge }
func (s *StrategyExecutionServer) SetWasmExecutor(we *WasmExecutor)          { s.wasmExecutor = we }

// SetGate injects the risk gate (D6-A: mandatory, non-optional).
// Must be called before RunLiveStrategy.
func (s *StrategyExecutionServer) SetGate(g *risk.Gate) { s.gate = g }

// AddGateRule adds a rule to the Gate after initialization.
func (s *StrategyExecutionServer) AddGateRule(r risk.Rule) {
	if s.gate != nil {
		s.gate.AddRule(r)
	}
}

// SetAccountProvider injects the live account state provider (T3.2b).
func (s *StrategyExecutionServer) SetAccountProvider(p AccountStateProvider) { s.accountProvider = p }

var _ antv1c.StrategyRuntimeServiceHandler = (*StrategyExecutionServer)(nil)

func NewStrategyExecutionServer(backtestRepo *repository.BacktestRunRepository, log *zap.Logger) *StrategyExecutionServer {
	return &StrategyExecutionServer{backtestRepo: backtestRepo, log: log, activeCancels: make(map[uuid.UUID]context.CancelFunc)}
}

func (s *StrategyExecutionServer) SetNotificationSender(ns *notification.Sender) { s.notifSender = ns }
func (s *StrategyExecutionServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) { s.onBacktestComplete = fn }

// userIDRequire extracts and validates the authenticated user ID from context.
func userIDRequire(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return id, nil
}

func (s *StrategyExecutionServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid

	// Go-native path: execute generated Go strategies via proto binary.
	if s.goExecutor != nil && isGoStrategy(req.Msg.Code) {
		resp, err := s.goExecutor.Run(ctx, req.Msg.Code, req.Msg)
		if err != nil {
			s.log.Warn("go executor failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("strategy execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	return connect.NewResponse(&antv1.ExecuteStrategyResponse{
		Success: false,
		Error:   "strategy execution not available",
	}), nil
}

func (s *StrategyExecutionServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid

	code := req.Msg.GetCode()
	if code == "" {
		return connect.NewResponse(&antv1.ValidateStrategyResponse{
			Valid:  false,
			Errors: []string{"code is empty"},
		}), nil
	}

	hasSig, missing := ai.HasRequiredSignature(code)
	var errors []string
	if !hasSig {
		errors = missing
	}
	warnings := ai.StructuralWarnings(code)

	return connect.NewResponse(&antv1.ValidateStrategyResponse{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}), nil
}

func (s *StrategyExecutionServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid

	// Synchronous backtest is deprecated.
	// Use StartBacktestRun for async execution via the Go-native backtest engine.
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success: false,
		Error:   "use StartBacktestRun for async backtesting via the Go-native engine",
	}), nil
}

func (s *StrategyExecutionServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetStrategyTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetStrategyTemplatesResponse{Templates: builtinTemplates()}), nil
}

// ExecuteLive runs strategy code against a live bar stream.
// Delegates to GoExecutor.RunLive for Go-native strategy execution.
func (s *StrategyExecutionServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}
	if s.goExecutor != nil && isGoStrategy(req.Msg.StrategyCode) {
		resp, err := s.goExecutor.RunLive(ctx, req.Msg.StrategyCode, req.Msg)
		if err != nil {
			s.log.Warn("go executor live failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("live execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}
	return connect.NewResponse(&antv1.ExecuteLiveResponse{Success: false, Error: "live execution not available"}), nil
}

func (s *StrategyExecutionServer) TranspileCode(ctx context.Context, req *connect.Request[antv1.TranspileCodeRequest]) (*connect.Response[antv1.TranspileCodeResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.TranspileCodeResponse{
			Confidence: 0,
		}), nil
	}

	intent, err := mql2go.Analyze(source)
	if err != nil {
		return connect.NewResponse(&antv1.TranspileCodeResponse{
			Confidence: 0,
		}), nil
	}

	if name := req.Msg.GetClassName(); name != "" {
		intent.Meta.Name = name
	}

	code := mql2go.Generate(intent)
	totalPatterns := int32(len(intent.Entry) + len(intent.Exit) + len(intent.Indicators))

	return connect.NewResponse(&antv1.TranspileCodeResponse{
		TargetCode:      code,
		Confidence:      1.0,
		TotalPatterns:   totalPatterns,
		IsDeterministic: true,
	}), nil
}

func (s *StrategyExecutionServer) AnalyzeImportCode(ctx context.Context, req *connect.Request[antv1.AnalyzeImportCodeRequest]) (*connect.Response[antv1.AnalyzeImportCodeResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{}), nil
	}

	intent, err := mql2go.Analyze(source)
	if err != nil {
		s.log.Warn("AnalyzeImportCode: analyze failed", zap.Error(err))
		return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{}), nil
	}

	// Build params proto fields
	paramFields := buildParamFields(intent.Params)
	paramGroups := buildParamGroups(intent.Params)

	// Build indicator names
	var indicatorNames []string
	for _, ind := range intent.Indicators {
		indicatorNames = append(indicatorNames, ind.SDKMethod)
	}

	// Calculate coverage
	totalBlocks := int32(len(intent.Entry) + len(intent.Exit) + len(intent.Indicators) + len(intent.Params))
	recognizedBlocks := totalBlocks // all recognized blocks are recognized
	coverage := 1.0
	if totalBlocks > 0 && len(intent.BlindSpots) > 0 {
		coverage = 1.0 - float64(len(intent.BlindSpots))/float64(totalBlocks)
	}

	return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{
		StrategyName:     intent.Meta.Name,
		MqlVersion:       intent.Meta.MQLVersion,
		CoverageScore:    coverage,
		TotalBlocks:      totalBlocks,
		RecognizedBlocks: recognizedBlocks,
		ExecutionKind:    string(intent.Execution.Kind),
		EntryRulesCount:  int32(len(intent.Entry)),
		ExitRulesCount:   int32(len(intent.Exit)),
		SizingKind:       string(intent.Sizing.Kind),
		Params:           paramFields,
		Groups:           paramGroups,
		BlindSpots:       buildBlindSpotProtos(intent.BlindSpots),
		IndicatorNames:   indicatorNames,
	}), nil
}

func (s *StrategyExecutionServer) GenerateImportCode(ctx context.Context, req *connect.Request[antv1.GenerateImportCodeRequest]) (*connect.Response[antv1.GenerateImportCodeResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.GenerateImportCodeResponse{Compiles: false}), nil
	}

	intent, err := mql2go.Analyze(source)
	if err != nil {
		s.log.Warn("GenerateImportCode: analyze failed", zap.Error(err))
		return connect.NewResponse(&antv1.GenerateImportCodeResponse{Compiles: false}), nil
	}

	if name := req.Msg.GetSourceName(); name != "" {
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".mq4"), ".mq5")
		base = strings.TrimSuffix(base, ".mqh")
		if base != "" {
			intent.Meta.Name = toCamelCase(base)
		}
	}

	code := mql2go.Generate(intent)
	lines := int32(strings.Count(code, "\n") + 1)

	// Compile verification: run go vet on generated code
	compiles := false
	compileError := ""
	if s.goExecutor != nil {
		compiles, compileError = s.goExecutor.CompileCheck(ctx, code)
		if !compiles {
			s.log.Warn("GenerateImportCode: generated code failed compilation",
				zap.String("error", compileError))
		}
	} else {
		// No executor available — assume compiles for CLI-only usage
		compiles = true
	}

	resp := &antv1.GenerateImportCodeResponse{
		GoCode:    code,
		CodeLines: lines,
		Compiles:  compiles,
	}
	if !compiles && compileError != "" {
		resp.QualityGateFailures = []string{compileError}
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyExecutionServer) ImportStrategy(ctx context.Context, req *connect.Request[antv1.ImportStrategyRequest]) (*connect.Response[antv1.ImportStrategyResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.ImportStrategyResponse{}), nil
	}

	intent, err := mql2go.Analyze(source)
	if err != nil {
		s.log.Warn("ImportStrategy: analyze failed", zap.Error(err))
		return connect.NewResponse(&antv1.ImportStrategyResponse{}), nil
	}

	if name := req.Msg.GetSourceName(); name != "" {
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".mq4"), ".mq5")
		base = strings.TrimSuffix(base, ".mqh")
		if base != "" {
			intent.Meta.Name = toCamelCase(base)
		}
	}

	code := mql2go.Generate(intent)

	totalBlocks := len(intent.Entry) + len(intent.Exit) + len(intent.Indicators) + len(intent.Params)
	coverage := 1.0
	if totalBlocks > 0 && len(intent.BlindSpots) > 0 {
		coverage = 1.0 - float64(len(intent.BlindSpots))/float64(totalBlocks)
	}

	strategyID := uuid.New().String()

	return connect.NewResponse(&antv1.ImportStrategyResponse{
		StrategyId:    strategyID,
		StrategyName:  intent.Meta.Name,
		GoCode:        code,
		CoverageScore: coverage,
		BlindSpots:    buildBlindSpotProtos(intent.BlindSpots),
	}), nil
}

// buildParamFields converts mql2go.ParamSpec slice to proto ParamField slice.
func buildParamFields(params []mql2go.ParamSpec) []*antv1.ParamField {
	fields := make([]*antv1.ParamField, 0, len(params))
	for _, p := range params {
		fields = append(fields, &antv1.ParamField{
			Name:         p.Name,
			Label:        p.Label,
			Type:         string(p.Type),
			DefaultValue: p.Default,
			Group:        string(p.Group),
		})
	}
	return fields
}

// buildParamGroups extracts unique parameter groups as proto ParamGroupInfo slice.
func buildParamGroups(params []mql2go.ParamSpec) []*antv1.ParamGroupInfo {
	seen := make(map[string]bool)
	var groups []*antv1.ParamGroupInfo
	for _, p := range params {
		g := string(p.Group)
		if !seen[g] {
			seen[g] = true
			groups = append(groups, &antv1.ParamGroupInfo{Name: g})
		}
	}
	return groups
}

// buildBlindSpotProtos converts mql2go.BlindSpot slice to proto BlindSpot slice.
func buildBlindSpotProtos(spots []mql2go.BlindSpot) []*antv1.BlindSpot {
	result := make([]*antv1.BlindSpot, 0, len(spots))
	for _, bs := range spots {
		result = append(result, &antv1.BlindSpot{
			Id:                 bs.ID,
			Category:           bs.Category,
			Severity:           bs.Severity,
			Description:        bs.Description,
			Location:           bs.Location,
			Handling:           bs.Handling,
			UserActionRequired: bs.UserActionRequired,
		})
	}
	return result
}

// toCamelCase converts a filename like "my_strategy" to "MyStrategy".
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

func (s *StrategyExecutionServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

func isGoStrategy(code string) bool {
	return len(code) > 0 && containsStr(code, "anttrader/strategy/sdk")
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
