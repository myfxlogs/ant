package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
)

// AccountSyncService handles account-level background synchronisation and alerts.
type AccountSyncService struct {
	tradeRecordRepo  *repository.TradeRecordRepository
	mthubSvc         *mthub.MtHubService
	analytics        *AnalyticsCache
	scheduleResolver mthub.ScheduleResolver
	notifSender      *notifpubsub.Sender
	log              *zap.Logger
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

// SetScheduleResolver injects the schedule resolver for trade attribution (ARCH-4 step⑥).
func (s *AccountSyncService) SetScheduleResolver(r mthub.ScheduleResolver) { s.scheduleResolver = r }

// SyncAccountHistory synchronises broker order history into trade_records.
// Called on gateway connect/disconnect to catch orders missed during disconnect gaps.
// Synchronous — caller decides whether to run in a goroutine. Errors are logged, not returned.
func (s *AccountSyncService) SyncAccountHistory(accountID, userID string) {
	if s.log != nil {
		s.log.Info("SyncAccountHistory: starting", zap.String("account", accountID), zap.String("user", userID))
	}
	s.syncAccountHistory(context.Background(), accountID, userID)
}

func (s *AccountSyncService) syncAccountHistory(ctx context.Context, accountID, userID string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	uid, err := uuid.Parse(userID)
	if err != nil {
		s.log.Error("SyncAccountHistory: invalid userID", zap.String("userID", userID), zap.Error(err))
		return
	}
	accID, err := uuid.Parse(accountID)
	if err != nil {
		s.log.Error("SyncAccountHistory: invalid accountID", zap.String("accountID", accountID), zap.Error(err))
		return
	}

	from := time.Now().AddDate(-1, 0, 0)
	lastTime, err := s.tradeRecordRepo.GetLastSyncTime(ctx, uid, accID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.log.Warn("SyncAccountHistory: get last sync time failed", zap.String("account", accountID), zap.Error(err))
	} else if lastTime != nil {
		from = *lastTime
	}
	to := time.Now()

	records, err := s.mthubSvc.OrderHistory(ctx, accountID, from, to)
	if err != nil {
		s.log.Warn("SyncAccountHistory: fetch from broker failed", zap.String("account", accountID), zap.Error(err))
		return
	}
	if len(records) == 0 {
		s.log.Info("SyncAccountHistory: no new records", zap.String("account", accountID))
		return
	}

	platform := s.mthubSvc.Platform(accountID)
	tradeRecs := make([]*model.TradeRecord, 0, len(records))
	for _, r := range records {
		tradeRecs = append(tradeRecs, orderRecordToTradeRecord(ctx, r, accID, uid, platform, s.scheduleResolver, s.log))
	}

	if err := s.tradeRecordRepo.BatchCreate(ctx, tradeRecs); err != nil {
		s.log.Error("SyncAccountHistory: batch create failed", zap.String("account", accountID), zap.Error(err))
		return
	}

	s.log.Info("SyncAccountHistory: synced",
		zap.String("account", accountID),
		zap.Int("records", len(tradeRecs)))
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

// orderRecordToTradeRecord converts an mthub.OrderRecord to a model.TradeRecord.
// REUSE: mthub.OrderRecord.OrderTypeString @ mthub/order_types.go
func orderRecordToTradeRecord(ctx context.Context, r *mthub.OrderRecord, accountID, userID uuid.UUID, platform string, resolver mthub.ScheduleResolver, log *zap.Logger) *model.TradeRecord {
	return &model.TradeRecord{
		UserID:       userID,
		AccountID:    accountID,
		Ticket:       r.Ticket,
		Symbol:       r.SymbolRaw,
		OrderType:    r.OrderTypeString(),
		Volume:       r.Volume,
		OpenPrice:    r.OpenPrice,
		ClosePrice:   r.ClosePrice,
		Profit:       r.Profit,
		Swap:         r.Swap,
		Commission:   r.Commission,
		OpenTime:     r.OpenTime,
		CloseTime:    r.CloseTime,
		StopLoss:     r.StopLoss,
		TakeProfit:   r.TakeProfit,
		OrderComment: r.Comment,
		MagicNumber:  int(r.Magic),
		ScheduleID:   mthub.ResolveScheduleID(ctx, resolver, log, accountID, int32(r.Magic)),
		Platform:     platform,
	}
}
