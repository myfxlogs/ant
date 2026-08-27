package strategy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/ai"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notification"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
	"alphaforge/strategy/sdk"
)

// StrategyExecutionServer implements ant.v1c.StrategyRuntimeServiceHandler.
// Handles strategy execution via in-process Bytecode VM (MQL) and legacy GoExecutor (Go).
type StrategyExecutionServer struct {
	backtestRepo       *repository.BacktestRunRepository
	log                *zap.Logger
	marketDataRepo     repository.MarketDataStore
	barSource          BarSource           // unified bar data source (backtest or live)
	mtHub              *mthub.MtHubService // for live order submission
	paperEngine        PaperOrderExecutor  // for paper trading simulated fills
	pgListen           *pglisten.Listener
	notifSender        *notification.Sender
	onBacktestComplete func(ctx context.Context, run *repository.BacktestRun) // auto-gate hook

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

	// Strategy version history (snapshots, diff, rollback).
	versionRepo *repository.StrategyVersionRepository

	// Active session registry for monitoring + control.
	sessionRegistry *SessionRegistry

	// SSE heartbeat interval for WatchActiveStrategies (default 20s, injectable for tests).
	heartbeatInterval time.Duration

	// AccountLookup provides the MT4 account ID for bar data subscription in paper mode.
	accountLookup func(ctx context.Context, userID string) string

	// Push-first cancel: shared LISTEN on backtest_cancel channel.
	activeCancels   map[uuid.UUID]context.CancelFunc
	activeCancelsMu sync.Mutex

	// PositionCache holds push-based position snapshots from PositionSnapshotBroker.
	// Eliminates per-bar OpenedOrders polling (push-first architecture).

	// Failure signature persistence (B2-6).
	failureSigRepo *repository.FailureSignatureRepository
	posCache       *PositionCache

	// QuotaChecker enforces subscription plan limits (max strategies, live strategies).
	quotaChecker QuotaChecker

	// LEAKAGE-1: Bound account checker — enforces tier-based MT account binding.
	// Injected at construction; RunLiveStrategy (the shared chokepoint) calls it
	// before any live strategy launch. Non-bypassable: all live-launch paths
	// (StartStrategy, launchEventSession, dispatch) converge on RunLiveStrategy.
	boundSvc BoundAccountChecker

	// T5: Coverage checker — enforces fatal blind spot gate at live launch.
	// Checks latest SUCCEEDED backtest IsReliable + fatal blind spots, or
	// does compile+coverage analysis if no backtest exists. Non-bypassable:
	// all live-launch paths converge on RunLiveStrategy.
	coverageChecker CoverageChecker

	// QualityValidator checks backtest snapshot against marketplace quality gates (auto_gate preview).
	qualityValidator QualityValidator

	// GateEvalRepo persists 7-gate pipeline results + marketplace quality preview.
	gateEvalRepo *repository.GateEvaluationRepository

	// scheduleNameLookup resolves schedule ID → strategy name for ActiveStrategy proto.
	// nil = strategy_name left empty (frontend falls back to runId).
	scheduleNameLookup func(ctx context.Context, scheduleID uuid.UUID) string

	// brokerCompanyLookup resolves accountID → mt_accounts.broker_company.
	// Used by seedBarWindows to filter md_bars to the correct data source.
	brokerCompanyLookup func(ctx context.Context, accountID string) string

	// VM-TRADE-CONTEXT-6 S6/S7: server-side account truth lookups.
	// Each returns (value, error) so DB query errors can be distinguished
	// from legitimate false/empty values. Live mode requires all lookups
	// to succeed (fail-closed); paper mode tolerates errors (fail-open).
	accountLoginLookup        func(ctx context.Context, accountID string) (int64, error)
	accountIsDemoLookup       func(ctx context.Context, accountID string) (bool, error)
	accountConnectedLookup    func(ctx context.Context, accountID string) (bool, error)
	accountTradeAllowedLookup func(ctx context.Context, accountID string) (bool, error)
	accountIsInvestorLookup   func(ctx context.Context, accountID string) (bool, error)
}

// QualityValidator validates backtest quality for marketplace publishing (read-only preview).
type QualityValidator interface {
	ValidateBacktestQuality(ctx context.Context, snapshotProto []byte, strategyID string) ([]marketplace.QualityViolation, error)
}

// CoverageChecker checks whether a strategy is safe to run live (no fatal blind spots).
// T5: Symmetric with the publish gate (MQL-LOOP-1) — strategies with fatal coverage
// blind spots must not run on real accounts. Implemented by marketplace.Service.
type CoverageChecker interface {
	CheckLiveCoverage(ctx context.Context, strategyID, sourceCode string) error
}

// SetCoverageChecker injects the coverage checker for T5 live gate.
func (s *StrategyExecutionServer) SetCoverageChecker(c CoverageChecker) { s.coverageChecker = c }

// SetQualityValidator injects the marketplace quality validator for auto_gate preview.
func (s *StrategyExecutionServer) SetQualityValidator(v QualityValidator) { s.qualityValidator = v }

// SetGateEvalRepo injects the gate evaluation repository for persistence.
func (s *StrategyExecutionServer) SetGateEvalRepo(r *repository.GateEvaluationRepository) {
	s.gateEvalRepo = r
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
	PaperPnl(ctx context.Context, accountID, symbol string, bid, ask decimal.Decimal) (decimal.Decimal, error)
}

func (s *StrategyExecutionServer) SetBarSource(bs BarSource)                      { s.barSource = bs }
func (s *StrategyExecutionServer) SetMtHub(h *mthub.MtHubService)                 { s.mtHub = h }
func (s *StrategyExecutionServer) SetPaperEngine(pe PaperOrderExecutor)           { s.paperEngine = pe }
func (s *StrategyExecutionServer) SetGoExecutor(ge *GoExecutor)                   { s.goExecutor = ge }
func (s *StrategyExecutionServer) SetRunRepo(r *repository.StrategyRunRepository) { s.runRepo = r }
func (s *StrategyExecutionServer) SetImportedRepo(r *repository.ImportedStrategyRepository) {
	s.importedRepo = r
}
func (s *StrategyExecutionServer) SetVersionRepo(r *repository.StrategyVersionRepository) {
	s.versionRepo = r
}
func (s *StrategyExecutionServer) SetSessionRegistry(r *SessionRegistry) { s.sessionRegistry = r }
func (s *StrategyExecutionServer) SetAccountLookup(f func(ctx context.Context, userID string) string) {
	s.accountLookup = f
}
func (s *StrategyExecutionServer) SetPositionCache(pc *PositionCache) { s.posCache = pc }
func (s *StrategyExecutionServer) SetScheduleNameLookup(f func(ctx context.Context, scheduleID uuid.UUID) string) {
	s.scheduleNameLookup = f
}
func (s *StrategyExecutionServer) SetBrokerCompanyLookup(f func(ctx context.Context, accountID string) string) {
	s.brokerCompanyLookup = f
}

// VM-TRADE-CONTEXT-6 S7: setters for server-side account truth lookups.
func (s *StrategyExecutionServer) SetAccountLoginLookup(f func(ctx context.Context, accountID string) (int64, error)) {
	s.accountLoginLookup = f
}
func (s *StrategyExecutionServer) SetAccountIsDemoLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountIsDemoLookup = f
}
func (s *StrategyExecutionServer) SetAccountConnectedLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountConnectedLookup = f
}
func (s *StrategyExecutionServer) SetAccountTradeAllowedLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountTradeAllowedLookup = f
}
func (s *StrategyExecutionServer) SetAccountIsInvestorLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountIsInvestorLookup = f
}

// QuotaChecker provides subscription plan limit checks.
// Implemented by service.QuotaChecker.
type QuotaChecker interface {
	CheckStrategyLimit(userID uuid.UUID, currentCount int) bool
	CheckLiveStrategyLimit(userID uuid.UUID, currentLive int) bool
	CheckBacktestDailyLimit(userID uuid.UUID, todayCount int) bool
}

func (s *StrategyExecutionServer) SetQuotaChecker(qc QuotaChecker) { s.quotaChecker = qc }

// SetBoundSvc injects the bound account checker (LEAKAGE-1).
func (s *StrategyExecutionServer) SetBoundSvc(b BoundAccountChecker) { s.boundSvc = b }

// checkBoundAccount is the shared helper used by RunLiveStrategy (chokepoint)
// and pre-checks in StartStrategy/launchEventSession/dispatch.
// Returns nil if boundSvc is nil or accountID is Nil (no-op).
func (s *StrategyExecutionServer) checkBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	if s.boundSvc == nil || accountID == uuid.Nil {
		return nil
	}
	return s.boundSvc.EnsureBoundAccount(ctx, userID, accountID)
}

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
func (s *StrategyExecutionServer) SetFailureSignatureRepo(repo *repository.FailureSignatureRepository) {
	s.failureSigRepo = repo
}
func (s *StrategyExecutionServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) {
	s.onBacktestComplete = fn
}

// userIDRequire extracts and validates the authenticated user ID from context.
func userIDRequire(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return id, nil
}

func (s *StrategyExecutionServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	_, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	// GoExecutor removed (Gap 3). Go strategies must be converted to MQL for Bytecode VM.
	if isGoStrategy(req.Msg.Code) {
		return nil, connect.NewError(connect.CodeUnimplemented,
			fmt.Errorf("go strategy execution has been retired — please convert your strategy to MQL"))
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
	_, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

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

	// Synchronous backtest is deprecated.
	// Use StartBacktestRun for async execution via the Go-native backtest engine.
	s.log.Debug("Backtest: deprecated sync endpoint called", zap.String("userID", uid.String()))
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success: false,
		Error:   "use StartBacktestRun for async backtesting via the Go-native engine",
	}), nil
}

func (s *StrategyExecutionServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetStrategyTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetStrategyTemplatesResponse{Templates: builtinTemplates()}), nil
}

// ExecuteLive runs strategy code against a live bar stream.
// Go-native path retired (GoExecutor removed); MQL path uses in-process Bytecode VM.
func (s *StrategyExecutionServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}

	// Go-native compilation path: generated Go strategy via go run.
	// GoExecutor removed (Gap 3). Go strategies must be converted to MQL for Bytecode VM.
	if isGoStrategy(req.Msg.StrategyCode) {
		return nil, connect.NewError(connect.CodeUnimplemented,
			fmt.Errorf("go strategy live execution has been retired — please convert your strategy to MQL"))
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

	// Python path: in-process Bytecode VM execution (same VM, different compiler).
	if sdk.IsPython(req.Msg.StrategyCode) {
		resp, err := s.executePythonVMLive(ctx, req.Msg)
		if err != nil {
			s.log.Warn("vm python live execution failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("python live execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	return connect.NewResponse(&antv1.ExecuteLiveResponse{Success: false, Error: "live execution not available"}), nil
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
	return sdk.IsGo(code)
}

// isMQLStrategy returns true if the code looks like MQL source (not Go, not Python).
func isMQLStrategy(code string) bool {
	return sdk.IsMQL(code)
}
