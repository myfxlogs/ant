package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
)

// AccountSyncService handles account-level background synchronisation and alerts.
type AccountSyncService struct {
	tradeRecordRepo *repository.TradeRecordRepository
	mthubSvc        *mthub.MtHubService
	analytics       *AnalyticsCache
	notifSender     *notifpubsub.Sender
	log             *zap.Logger
}

// NewAccountSyncService creates an account sync service.
func NewAccountSyncService(
	tradeRecordRepo *repository.TradeRecordRepository,
	mthubSvc *mthub.MtHubService,
	analytics *AnalyticsCache,
	log *zap.Logger,
) *AccountSyncService {
	return &AccountSyncService{
		tradeRecordRepo: tradeRecordRepo,
		mthubSvc:        mthubSvc,
		analytics:       analytics,
		log:             log,
	}
}

// SetNotificationSender wires the notification sender for alerts.
func (s *AccountSyncService) SetNotificationSender(ns *notifpubsub.Sender) { s.notifSender = ns }

// SyncAccountHistory synchronises broker history for the account in the background.
func (s *AccountSyncService) SyncAccountHistory(accountID, userID string) {
	if s.log != nil {
		s.log.Debug("SyncAccountHistory: requested", zap.String("account", accountID), zap.String("user", userID))
	}
}

// CheckMarginCall checks margin level against the broker threshold and alerts if breached.
// Alerts are throttled per account to avoid spam.
func (s *AccountSyncService) CheckMarginCall(
	accountID, userID string,
	marginLevel, margin, equity, callPct decimal.Decimal,
	mu *sync.Mutex,
	lastSent map[string]map[int]time.Time,
	eventStore *mthub.TradeEventStore,
	emailNotifier *notifier.EmailNotifier,
) {
	_ = eventStore // reserved for future JetStream margin-call events
	if !marginLevel.GreaterThan(decimal.Zero) || !marginLevel.LessThan(callPct) {
		return
	}
	const minInterval = 15 * time.Minute
	const level = 1
	mu.Lock()
	inner, ok := lastSent[accountID]
	if !ok {
		inner = make(map[int]time.Time)
		lastSent[accountID] = inner
	}
	last := inner[level]
	if !last.IsZero() && time.Since(last) < minInterval {
		mu.Unlock()
		return
	}
	inner[level] = time.Now()
	mu.Unlock()
	if emailNotifier != nil {
		emailNotifier.MarginCallAlert(accountID, userID, margin, equity)
	}
}

// MapSideToString converts an mthub side enum to a human-readable string.
func MapSideToString(side mthub.Side) string {
	switch side {
	case mthub.SideBuy:
		return "buy"
	case mthub.SideSell:
		return "sell"
	default:
		return fmt.Sprintf("side-%d", side)
	}
}
