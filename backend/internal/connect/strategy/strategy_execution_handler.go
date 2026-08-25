package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notification"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
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
	// VM-API-TRUTH-3 round 4: returns (value, error) so DB query errors
	// are distinguishable from real empty results.
	brokerCompanyLookup func(ctx context.Context, accountID string) (string, error)

	// accountLoginLookup resolves accountID → mt_accounts.login (int64).
	// VM-TRADE-CONTEXT-6: used by buildLiveContext to inject Login into
	// LiveStrategyContext so AccountNumber() works during OnInit.
	// VM-API-TRUTH-3 round 4: returns (value, error) so DB query errors
	// are distinguishable from real login=0.
	accountLoginLookup func(ctx context.Context, accountID string) (int64, error)

	// accountIsDemoLookup resolves accountID → mt_accounts.account_type == 'demo'.
	// VM-API-TRUTH-3: used by buildLiveContext to inject IsDemo into
	// LiveStrategyContext so IsDemo() reflects the real account type.
	// VM-API-TRUTH-3 round 4: returns (value, error) so DB query errors
	// are distinguishable from real false.
	accountIsDemoLookup func(ctx context.Context, accountID string) (bool, error)

	// accountConnectedLookup resolves accountID → mt_accounts.account_status == 'connected'.
	// VM-API-TRUTH-3: used by buildLiveContext to inject IsConnected from the
	// authoritative account_status column, not a hardcoded constant.
	// VM-API-TRUTH-3 round 4: returns (value, error) so DB query errors
	// are distinguishable from real false.
	accountConnectedLookup func(ctx context.Context, accountID string) (bool, error)

	// accountIsInvestorLookup resolves accountID → mt_accounts.is_investor.
	// VM-API-TRUTH-3 round 4: Investor/read-only accounts cannot trade even
	// when account_status == 'connected'. Used to gate IsTradeAllowed.
	accountIsInvestorLookup func(ctx context.Context, accountID string) (bool, error)

	// accountTradeAllowedLookup resolves accountID → whether trading is permitted.
	// VM-API-TRUTH-3 round 4: returns (value, error). The caller (buildLiveContext)
	// combines this with accountIsInvestorLookup — if either returns true for
	// "cannot trade" or errors, IsTradeAllowed is false (fail-closed).
	accountTradeAllowedLookup func(ctx context.Context, accountID string) (bool, error)
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
func (s *StrategyExecutionServer) SetBrokerCompanyLookup(f func(ctx context.Context, accountID string) (string, error)) {
	s.brokerCompanyLookup = f
}

// SetAccountLoginLookup injects the accountID → login resolver.
// VM-TRADE-CONTEXT-6: used by buildLiveContext to fill LiveStrategyContext.Login.
// VM-API-TRUTH-3 round 4: returns (value, error) so DB errors propagate.
func (s *StrategyExecutionServer) SetAccountLoginLookup(f func(ctx context.Context, accountID string) (int64, error)) {
	s.accountLoginLookup = f
}

// SetAccountIsDemoLookup injects the accountID → isDemo resolver.
// VM-API-TRUTH-3: used by buildLiveContext to fill LiveStrategyContext.IsDemo.
// VM-API-TRUTH-3 round 4: returns (value, error) so DB errors propagate.
func (s *StrategyExecutionServer) SetAccountIsDemoLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountIsDemoLookup = f
}

// SetAccountConnectedLookup injects the accountID → isConnected resolver.
// VM-API-TRUTH-3: used by buildLiveContext to fill LiveStrategyContext.IsConnected
// from the authoritative account_status column.
// VM-API-TRUTH-3 round 4: returns (value, error) so DB errors propagate.
func (s *StrategyExecutionServer) SetAccountConnectedLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountConnectedLookup = f
}

// SetAccountIsInvestorLookup injects the accountID → isInvestor resolver.
// VM-API-TRUTH-3 round 4: Investor/read-only accounts cannot trade even
// when connected. Used to gate IsTradeAllowed.
func (s *StrategyExecutionServer) SetAccountIsInvestorLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountIsInvestorLookup = f
}

// SetAccountTradeAllowedLookup injects the accountID → isTradeAllowed resolver.
// VM-API-TRUTH-3: used by buildLiveContext to fill LiveStrategyContext.IsTradeAllowed.
// VM-API-TRUTH-3 round 4: returns (value, error). buildLiveContext also checks
// is_investor — investor accounts get IsTradeAllowed=false even if this returns true.
func (s *StrategyExecutionServer) SetAccountTradeAllowedLookup(f func(ctx context.Context, accountID string) (bool, error)) {
	s.accountTradeAllowedLookup = f
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

func (s *StrategyExecutionServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
