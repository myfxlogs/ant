package strategy

import (
	"context"
	"fmt"

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

	// Go-native strategy executor — runs generated Go strategies.
	goExecutor *GoExecutor
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
		volume decimal.Decimal, bid, ask float64) error
	ClosePaperOrder(ctx context.Context, accountID, symbol string) error
	ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error
	CancelPaperOrder(ctx context.Context, accountID, symbol string) error
}

func (s *StrategyExecutionServer) SetBarSource(bs BarSource)                 { s.barSource = bs }
func (s *StrategyExecutionServer) SetMtHub(h *mthub.MtHubService)            { s.mtHub = h }
func (s *StrategyExecutionServer) SetPaperEngine(pe PaperOrderExecutor)      { s.paperEngine = pe }
func (s *StrategyExecutionServer) SetGoExecutor(ge *GoExecutor)              { s.goExecutor = ge }

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
	return &StrategyExecutionServer{backtestRepo: backtestRepo, log: log}
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
	// MQL→Go migration is handled by tools/mql2go.
	// TranspileCode is deprecated.
	return connect.NewResponse(&antv1.TranspileCodeResponse{
		IsDeterministic: false,
		GapSamples:       []string{"DEPRECATED: use StrategyImport for MQL→Go migration"},
		Confidence:       0,
	}), nil
}

func (s *StrategyExecutionServer) AnalyzeImportCode(ctx context.Context, req *connect.Request[antv1.AnalyzeImportCodeRequest]) (*connect.Response[antv1.AnalyzeImportCodeResponse], error) {
	return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{}), nil
}

func (s *StrategyExecutionServer) GenerateImportCode(ctx context.Context, req *connect.Request[antv1.GenerateImportCodeRequest]) (*connect.Response[antv1.GenerateImportCodeResponse], error) {
	return connect.NewResponse(&antv1.GenerateImportCodeResponse{Compiles: false}), nil
}

func (s *StrategyExecutionServer) ImportStrategy(ctx context.Context, req *connect.Request[antv1.ImportStrategyRequest]) (*connect.Response[antv1.ImportStrategyResponse], error) {
	return connect.NewResponse(&antv1.ImportStrategyResponse{}), nil
}

func (s *StrategyExecutionServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

func isGoStrategy(code string) bool {
	return len(code) > 0 && (containsStr(code, "anttrader/strategy/sdk") || containsStr(code, "package "))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
