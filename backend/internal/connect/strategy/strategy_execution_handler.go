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
	"anttrader/strategy/runner"
	"anttrader/tools/mql2go"
)

// StrategyExecutionServer implements ant.v1c.StrategyRuntimeServiceHandler.
// Handles strategy execution via in-process Bytecode VM (MQL) and legacy GoExecutor (Go).
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
	// ADR-0023: retained for legacy Go strategies; all new strategies use MQL→Bytecode VM.
	goExecutor *GoExecutor

	// Strategy run + signal persistence.
	runRepo *repository.StrategyRunRepository

	// Imported strategy persistence (raw MQL source of truth).
	importedRepo *repository.ImportedStrategyRepository

	// Active session registry for monitoring + control.
	sessionRegistry *SessionRegistry

	// AccountLookup provides the MT4 account ID for bar data subscription in paper mode.
	accountLookup func(ctx context.Context, userID string) string

	// Push-first cancel: shared LISTEN on backtest_cancel channel.
	activeCancels   map[uuid.UUID]context.CancelFunc
	activeCancelsMu sync.Mutex

	// PositionCache holds push-based position snapshots from PositionSnapshotBroker.
	// Eliminates per-bar OpenedOrders polling (push-first architecture).
	posCache *PositionCache
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
func (s *StrategyExecutionServer) SetRunRepo(r *repository.StrategyRunRepository) { s.runRepo = r }
func (s *StrategyExecutionServer) SetImportedRepo(r *repository.ImportedStrategyRepository) { s.importedRepo = r }
func (s *StrategyExecutionServer) SetSessionRegistry(r *SessionRegistry)           { s.sessionRegistry = r }
func (s *StrategyExecutionServer) SetAccountLookup(f func(ctx context.Context, userID string) string) { s.accountLookup = f }
func (s *StrategyExecutionServer) SetPositionCache(pc *PositionCache)                                  { s.posCache = pc }

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
	if isGoStrategy(req.Msg.Code) {
		if s.goExecutor == nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Go strategy executor not configured"))
		}
		resp, err := s.goExecutor.Run(ctx, req.Msg.Code, req.Msg)
		if err != nil {
			s.log.Warn("go executor failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("strategy execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	// MQL source requires bar data to produce signals — use StartBacktestRun or ExecuteLive.
	if isMQLStrategy(req.Msg.Code) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("MQL strategies require bar data — use StartBacktestRun or ExecuteLive"))
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
// Delegates to GoExecutor.RunLive for Go-native strategy execution,
// or executeVMLive for MQL in-process Bytecode VM path.
func (s *StrategyExecutionServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}

	// Go-native compilation path: generated Go strategy via go run.
	if isGoStrategy(req.Msg.StrategyCode) {
		if s.goExecutor == nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Go strategy executor not configured"))
		}
		resp, err := s.goExecutor.RunLive(ctx, req.Msg.StrategyCode, req.Msg)
		if err != nil {
			s.log.Warn("go executor live failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("live execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	// MQL path: in-process Bytecode VM execution.
	if isMQLStrategy(req.Msg.StrategyCode) {
		resp, err := s.executeVMLive(ctx, req.Msg)
		if err != nil {
			s.log.Warn("vm live execution failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("live execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	return connect.NewResponse(&antv1.ExecuteLiveResponse{Success: false, Error: "live execution not available"}), nil
}

// executeVMLive runs a single live event via the in-process Bytecode VM.
// MQL source → CompileMQL → VMRunner → runner.Runner → dispatch event → ExecuteLiveResponse.
func (s *StrategyExecutionServer) executeVMLive(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	strategy, err := mql2go.CompileMQL(req.StrategyCode)
	if err != nil {
		return nil, fmt.Errorf("compile MQL: %w", err)
	}

	// Build runner config from bar context (first request must have bar_context).
	bctx := req.GetBarContext()
	if bctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "first request must have bar_context for initialization"}, nil
	}

	params := make(map[string]string)
	for _, p := range bctx.GetParams() {
		params[p.GetKey()] = p.GetValue()
	}

	r := runner.New(runner.Config{
		Symbol:    bctx.Symbol,
		Timeframe: bctx.Timeframe,
		Params:    params,
		Mode:      bctx.Mode,
	})
	r.SetStrategy(strategy)

	if err := r.Init(ctx); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}, nil
	}

	// Dispatch based on request type
	switch req.GetRequestType() {
	case antv1.RequestType_REQUEST_TYPE_BAR:
		return vmHandleBar(r, bctx), nil

	case antv1.RequestType_REQUEST_TYPE_TICK:
		return vmHandleTick(r, req.GetTickContext()), nil

	case antv1.RequestType_REQUEST_TYPE_TRADE:
		return vmHandleTrade(r, req.GetTradeContext()), nil

	case antv1.RequestType_REQUEST_TYPE_TIMER:
		return vmHandleTimer(r, req.GetTimerContext()), nil

	default:
		if bctx != nil {
			return vmHandleBar(r, bctx), nil
		}
		return &antv1.ExecuteLiveResponse{Success: false, Error: "unknown request type"}, nil
	}
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
	return len(code) > 0 && strings.Contains(code, "anttrader/strategy/sdk")
}

// isMQLStrategy returns true if the code looks like MQL source (not Go).
// MQL code lacks Go package/import declarations and contains MQL patterns.
func isMQLStrategy(code string) bool {
	if len(code) == 0 {
		return false
	}
	if isGoStrategy(code) {
		return false
	}
	return strings.Contains(code, "OnBar") || strings.Contains(code, "OnTick") ||
		strings.Contains(code, "OnInit") || strings.Contains(code, "OnTimer") ||
		strings.Contains(code, "OnDeinit")
}

