package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
	"anttrader/internal/repository"
)

// MarginLevel represents severity levels for margin call detection (B-2.3).
type MarginLevel int

const (
	MLevelWarn MarginLevel = 1 // Level 1 (预警): margin_level <= call_pct * 1.5
	MLevelCall MarginLevel = 2 // Level 2 (警告): margin_level <= call_pct
	MLevelCrit MarginLevel = 3 // Level 3 (危急): margin_level <= call_pct * 0.7
)

// CheckMarginCall evaluates margin level against per-broker thresholds and publishes
// events + sends emails at 3 severity levels with independent cooldowns.
// Pure logic with no PG or other side effects except email and event publishing.
func CheckMarginCall(
	accountID, userID string,
	marginLevel, margin, equity, callPct float64,
	mu *sync.Mutex,
	lastSent map[string]map[int]time.Time,
	eventStore *mthub.TradeEventStore,
	emailNotifier *notifier.EmailNotifier,
) {
	mu.Lock()
	defer mu.Unlock()

	if lastSent[accountID] == nil {
		lastSent[accountID] = make(map[int]time.Time)
	}

	now := time.Now()

	// Determine current severity level.
	var curLevel int
	switch {
	case marginLevel <= callPct*0.7:
		curLevel = int(MLevelCrit)
	case marginLevel <= callPct:
		curLevel = int(MLevelCall)
	case marginLevel <= callPct*1.5:
		curLevel = int(MLevelWarn)
	default:
		delete(lastSent, accountID)
		return
	}

	cooldown := 5 * time.Minute
	if curLevel == int(MLevelCrit) {
		cooldown = 1 * time.Minute
	}

	if since := now.Sub(lastSent[accountID][curLevel]); since < cooldown {
		return
	}
	lastSent[accountID][curLevel] = now

	eventStore.Publish(context.Background(), &mthub.TradeEvent{
		EventID:   fmt.Sprintf("mc-%s-%d-%d", accountID, now.Unix(), curLevel),
		EventType: mthub.TradeEventOrderMarginCall,
		AccountID: accountID,
		UserID:    userID,
	})

	if curLevel >= int(MLevelCall) && emailNotifier != nil {
		emailNotifier.MarginCallAlert(accountID, userID, margin, equity)
	}
}

// AccountSyncService handles syncing account history from MT brokers to PG.
type AccountSyncService struct {
	tradeRecordRepo *repository.TradeRecordRepository
	mthubSvc        *mthub.MtHubService
	log             *zap.Logger
}

// NewAccountSyncService creates a new AccountSyncService.
func NewAccountSyncService(tradeRecordRepo *repository.TradeRecordRepository, mthubSvc *mthub.MtHubService, log *zap.Logger) *AccountSyncService {
	return &AccountSyncService{
		tradeRecordRepo: tradeRecordRepo,
		mthubSvc:        mthubSvc,
		log:             log,
	}
}

// SyncAccountHistory fetches closed orders from MT broker and writes them to trade_records.
func (s *AccountSyncService) SyncAccountHistory(accountID string) {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	from := time.Now().AddDate(-1, 0, 0)
	if t, err := s.tradeRecordRepo.GetLastSyncTime(ctx, uid); err == nil && t != nil {
		from = *t
	}
	to := time.Now()

	records, err := s.mthubSvc.OrderHistory(ctx, accountID, from, to)
	if err != nil {
		s.log.Warn("syncHistory: fetch failed", zap.String("account", accountID), zap.Error(err))
		return
	}

	platform := s.mthubSvc.Platform(accountID)
	tradeRecs := make([]*model.TradeRecord, 0, len(records))
	for _, r := range records {
		ot := "BUY"
		if r.Side == mthub.SideSell {
			ot = "SELL"
		}
		switch r.OrderType {
		case mthub.OrderMarket:
		case mthub.OrderLimit:
			ot += "_LIMIT"
		case mthub.OrderStop:
			ot += "_STOP"
		case mthub.OrderStopLimit:
			ot += "_STOP_LIMIT"
		}
		tradeRecs = append(tradeRecs, &model.TradeRecord{
			AccountID:    uid,
			Ticket:       r.Ticket,
			Symbol:       r.SymbolRaw,
			OrderType:    ot,
			Volume:       r.Volume.InexactFloat64(),
			OpenPrice:    r.OpenPrice.InexactFloat64(),
			ClosePrice:   r.ClosePrice.InexactFloat64(),
			Profit:       r.Profit.InexactFloat64(),
			Swap:         r.Swap.InexactFloat64(),
			Commission:   r.Commission.InexactFloat64(),
			OpenTime:     r.OpenTime,
			CloseTime:    r.CloseTime,
			OrderComment: r.Comment,
			MagicNumber:  int(r.Magic),
			Platform:     platform,
		})
	}
	if err := s.tradeRecordRepo.BatchCreate(ctx, tradeRecs); err != nil {
		s.log.Warn("syncHistory: batch create failed", zap.String("account", accountID), zap.Error(err))
	} else {
		s.log.Info("syncHistory: synced", zap.String("account", accountID), zap.Int("count", len(tradeRecs)))
	}
}

// MapSideToString converts an mthub.Side to a display string.
func MapSideToString(s mthub.Side) string {
	if s == mthub.SideBuy {
		return "buy"
	}
	if s == mthub.SideSell {
		return "sell"
	}
	return "buy"
}
