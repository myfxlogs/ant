package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
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
func (s *AccountSyncService) CheckMarginCall(
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

	// Emit in-app notification for margin call events (all levels).
	if s.notifSender != nil {
		uid, err := uuid.Parse(userID)
		if err == nil {
			levelLabel := "Warning"
			if curLevel == int(MLevelCrit) {
				levelLabel = "Critical"
			} else if curLevel == int(MLevelCall) {
				levelLabel = "Margin Call"
			}
			data, _ := json.Marshal(map[string]interface{}{
				"account_id":   accountID,
				"margin_level": marginLevel,
				"call_pct":     callPct,
				"severity":     curLevel,
			})
			_, _ = s.notifSender.Send(context.Background(), uid, "risk_alert",
				fmt.Sprintf("Margin %s: %s", levelLabel, accountID),
				fmt.Sprintf("Margin level %.1f%% (call level: %.1f%%)", marginLevel, callPct),
				string(data))
		}
	}
}

// AccountSyncService handles syncing account history from MT brokers to PG.
type AccountSyncService struct {
	tradeRecordRepo *repository.TradeRecordRepository
	mthubSvc        *mthub.MtHubService
	analyticsCache  *AnalyticsCache
	log             *zap.Logger
	notifSender     *notification.Sender
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

// SetNotificationSender injects the notification sender for margin call events.
func (s *AccountSyncService) SetNotificationSender(ns *notification.Sender) { s.notifSender = ns }

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
		tradeRecs = append(tradeRecs, &model.TradeRecord{
			UserID:       userID,
			AccountID:    uid,
			Ticket:       r.Ticket,
			Symbol:       r.SymbolRaw,
			OrderType:    ot,
			Volume:       r.Volume,
			OpenPrice:    r.OpenPrice,
			ClosePrice:   r.ClosePrice,
			Profit:       r.Profit,
			Swap:         r.Swap,
			Commission:   r.Commission,
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
