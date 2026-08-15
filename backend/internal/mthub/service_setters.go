package mthub

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
	"alphaforge/internal/usermgr"
)

// SetKillSwitch injects the global kill switch for emergency stop (V3-R-5).
func (s *MtHubService) SetKillSwitch(ks KillSwitchGate) { s.killSwitch = ks }

// SetReconcileTrigger injects the reconciliation trigger for SUBMIT-STUCK-RACE
// fix (Task 2). When a broker fill event arrives before ticket backfill,
// TransitionOrderByTicket triggers reconciliation as a fallback.
func (s *MtHubService) SetReconcileTrigger(f func(accountID string)) { s.reconcileTrigger = f }

// AccountStateProvider fetches account state for risk evaluation.
// Returns risk.AccountState (single source of truth, shared with Gate).
type AccountStateProvider func(ctx context.Context, accountID string) (*risk.AccountState, error)

// SetBrokerRegistry injects the multi-broker registry (M12-C2).
// When configured, external components (e.g. AlgoExecutor) can resolve
// broker adapters by platform name. Existing PlaceOrder flow is unaffected.
func (s *MtHubService) SetBrokerRegistry(r BrokerRegistry) { s.brokerRegistry = r }

// BrokerRegistry returns the configured multi-broker registry, or nil.
func (s *MtHubService) BrokerRegistry() BrokerRegistry { return s.brokerRegistry }

// AccountOwnerVerifier checks that a user owns the given account.
type AccountOwnerVerifier func(ctx context.Context, userID, accountID string) (bool, error)

// SetAccountOwnerVerifier injects the account ownership checker.
func (s *MtHubService) SetAccountOwnerVerifier(v AccountOwnerVerifier) { s.accountOwnerVerifier = v }

// SetLogger injects a zap logger for error reporting.
func (s *MtHubService) SetLogger(l *zap.Logger) { s.logger = l }

// SetBarBroker injects the bar update broker for real-time K-line push.
func (s *MtHubService) SetBarBroker(b *BarBroker) { s.barBroker = b }

// SetTickBroker injects the tick broker for real-time quote (Bid/Ask) push.
func (s *MtHubService) SetTickBroker(b *TickBroker) { s.tickBroker = b }

// SetTradeBroker injects the trade event broker for fill/close/modify events.
func (s *MtHubService) SetTradeBroker(b *TradeBroker) { s.tradeBroker = b }

// SetStatusBroker injects the account status broker for real-time connection state push.
func (s *MtHubService) SetStatusBroker(b *AccountStatusBroker) { s.statusBroker = b }

// SetGuard injects the mandatory safety net (kill switch, duplicate, max lot size).
func (s *MtHubService) SetGuard(g *risk.Guard) { s.guard = g }

// SetOmsWriter injects the OMS state writer for order lifecycle tracking (S1.2).
func (s *MtHubService) SetOmsWriter(w *OmsWriter) { s.omsWriter = w }

// SetGate injects the risk gate for pre-trade evaluation (D6-A single chokepoint).
func (s *MtHubService) SetGate(g *risk.Gate) { s.gate = g }

// SetUserLimiter injects the per-user rate limiter (nil-safe).
func (s *MtHubService) SetUserLimiter(l *usermgr.UserLimiter) { s.userLimiter = l }

// SetCostEstimator injects the pre-trade cost estimator (M10-BASE-D2).
func (s *MtHubService) SetCostEstimator(e costsvc.CostEstimator) { s.costEstimator = e }

// SetAccountStateProvider injects the account state fetcher for risk evaluation.
func (s *MtHubService) SetAccountStateProvider(p AccountStateProvider) { s.accountStateProvider = p }

// ScheduleResolver maps a live trade's magic number back to its schedule ID.
// Used by orderRecordToTradeRecord to attribute trades to strategies (ARCH-4 step⑥).
type ScheduleResolver interface {
	ResolveScheduleIDByMagic(ctx context.Context, accountID uuid.UUID, magic int32) (*uuid.UUID, error)
}
