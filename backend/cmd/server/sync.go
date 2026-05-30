package main

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
)

// syncAccountHistory fetches closed orders from the MT broker and writes them
// to trade_records. Called on OnBrokerInfo and OnAccountDisconnect events.
func syncAccountHistory(
	accountID string,
	tradeRecordRepo *repository.TradeRecordRepository,
	mthubSvc *mthub.MtHubService,
	log *zap.Logger,
) {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	from := time.Now().AddDate(-1, 0, 0)
	if t, err := tradeRecordRepo.GetLastSyncTime(ctx, uid); err == nil && t != nil {
		from = *t
	}
	to := time.Now()

	records, err := mthubSvc.OrderHistory(ctx, accountID, from, to)
	if err != nil {
		log.Warn("syncHistory: fetch failed", zap.String("account", accountID), zap.Error(err))
		return
	}

	platform := mthubSvc.Platform(accountID)
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
	if err := tradeRecordRepo.BatchCreate(ctx, tradeRecs); err != nil {
		log.Warn("syncHistory: batch create failed", zap.String("account", accountID), zap.Error(err))
	} else {
		log.Info("syncHistory: synced", zap.String("account", accountID), zap.Int("count", len(tradeRecs)))
	}
}
