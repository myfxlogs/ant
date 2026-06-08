package service

import (
	"github.com/shopspring/decimal"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/notification"
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
	notifSender *notification.Sender,
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

	// Emit in-app notification for margin call events (all levels).
	if notifSender != nil {
		uid, err := uuid.Parse(userID)
		if err == nil {
			levelLabel := "Warning"
			if curLevel == int(MLevelCrit) {
				levelLabel = "Critical"
			} else if curLevel == int(MLevelCall) {
				levelLabel = "Margin Call"
			}
			_, _ = notifSender.Send(context.Background(), uid, "risk_alert",
				fmt.Sprintf("Margin %s: %s", levelLabel, accountID),
				fmt.Sprintf("Margin level %.1f%% (call level: %.1f%%)", marginLevel, callPct),
				fmt.Sprintf(`{"account_id":"%s","margin_level":%.1f,"call_pct":%.1f,"severity":%d}`, accountID, marginLevel, callPct, curLevel))
		}
	}
}

// AccountSyncService handles syncing account history from MT brokers to PG.
type AccountSyncService struct {
	tradeRecordRepo *repository.TradeRecordRepository
	mthubSvc        *mthub.MtHubService
	analyticsCache  *AnalyticsCache
	log             *zap.Logger
}

// NewAccountSyncService creates a new AccountSyncService.
func NewAccountSyncService(tradeRecordRepo *repository.TradeRecordRepository, mthubSvc *mthub.MtHubService, analyticsCache *AnalyticsCache, log *zap.Logger) *AccountSyncService {
	return &AccountSyncService{
		tradeRecordRepo: tradeRecordRepo,
		mthubSvc:        mthubSvc,
		analyticsCache:  analyticsCache,
		log:             log,
	}
}

// SyncAccountHistory fetches closed orders from MT broker in monthly chunks and
// writes them to trade_records.  Each chunk is committed independently — if a
// chunk fails (network timeout, broker error), earlier chunks are already saved
// and the next sync resumes from the last successful month.
func (s *AccountSyncService) SyncAccountHistory(accountID, userID string) {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return
	}

	// Determine start: last successful sync time or 1 year ago.
	userUUID, _ := uuid.Parse(userID)
	from := time.Now().AddDate(-1, 0, 0)
	if t, err := s.tradeRecordRepo.GetLastSyncTime(context.Background(), userUUID, uid); err == nil && t != nil {
		from = *t
	}

	// Sync in 3-month chunks to bound per-request latency and fault blast radius.
	chunkStart := from
	now := time.Now()
	total := 0
	for chunkStart.Before(now) {
		chunkEnd := chunkStart.AddDate(0, 3, 0)
		if chunkEnd.After(now) {
			chunkEnd = now
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		records, err := s.mthubSvc.OrderHistory(ctx, accountID, chunkStart, chunkEnd)
		cancel()
		if err != nil {
			s.log.Warn("syncHistory: chunk fetch failed",
				zap.String("account", accountID),
				zap.Time("chunkStart", chunkStart),
				zap.Error(err))
			return // resume from chunkStart next time
		}

		if len(records) > 0 {
			platform := s.mthubSvc.Platform(accountID)
			tradeRecs := s.convertRecords(accountID, uid, userUUID, platform, records)
			if err := s.tradeRecordRepo.BatchCreate(context.Background(), tradeRecs); err != nil {
				s.log.Warn("syncHistory: chunk insert failed",
					zap.String("account", accountID),
					zap.Time("chunkStart", chunkStart),
					zap.Error(err))
				return // resume from chunkStart next time
			}
			total += len(records)
		}

		// Update checkpoint so next sync resumes after this chunk.
		chunkStart = chunkEnd
	}

	if total > 0 {
		s.log.Info("syncHistory: synced", zap.String("account", accountID), zap.Int("count", total))
	}
}

// convertRecords maps mthub OrderRecords to model TradeRecords.
func (s *AccountSyncService) convertRecords(accountID string, uid, userID uuid.UUID, platform string, records []*mthub.OrderRecord) []*model.TradeRecord {
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
		case mthub.OrderBalance:
			ot = "BALANCE"
		case mthub.OrderCredit:
			ot = "CREDIT"
		}
		volume, vexact := r.Volume.Float64()
		openPrice, oexact := r.OpenPrice.Float64()
		closePrice, cexact := r.ClosePrice.Float64()
		profit, pexact := r.Profit.Float64()
		swap, sexact := r.Swap.Float64()
		commission, cmexact := r.Commission.Float64()
		if !vexact || !oexact || !cexact || !pexact || !sexact || !cmexact {
			s.log.Warn("syncHistory: precision loss converting decimal to float64",
				zap.String("account", accountID),
				zap.Int64("ticket", r.Ticket),
			)
		}
		tradeRecs = append(tradeRecs, &model.TradeRecord{
			UserID:       userID,
			AccountID:    uid,
			Ticket:       r.Ticket,
			Symbol:       r.SymbolRaw,
			OrderType:    ot,
			Volume:       decimal.NewFromFloat(volume),
			OpenPrice:    decimal.NewFromFloat(openPrice),
			ClosePrice:   decimal.NewFromFloat(closePrice),
			Profit:       decimal.NewFromFloat(profit),
			Swap:         decimal.NewFromFloat(swap),
			Commission:   decimal.NewFromFloat(commission),
			OpenTime:     r.OpenTime,
			CloseTime:    r.CloseTime,
			OrderComment: r.Comment,
			MagicNumber:  int(r.Magic),
			Platform:     platform,
		})
	}
	return tradeRecs
}

// MapSideToString converts an mthub.Side to a display string.
func MapSideToString(s mthub.Side) string {
	if s == mthub.SideBuy {
		return "buy"
	}
	if s == mthub.SideSell {
		return "sell"
	}
	return "unknown"
}
