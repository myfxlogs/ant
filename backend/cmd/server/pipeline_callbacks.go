package main

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
)

// buildOnOrderUpdate creates the OnOrderUpdate callback for the mdgateway pipeline.
// It publishes position snapshots (read-only display) and records closed trades.
// Profit/equity DB writes belong to the profit pipeline (OnAccountProfit).
// Risk calculation (feedPlatformAggregator) is decoupled — it subscribes to
// PositionSnapshotBroker independently.
func buildOnOrderUpdate(
	log *zap.Logger,
	snapshotBroker *mthub.PositionSnapshotBroker,
	tradeRecordRepo *repository.TradeRecordRepository,
) func(accountID, userID string, o *mdtick.OrderUpdate) {
	return func(accountID, userID string, o *mdtick.OrderUpdate) {
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		publishPositionSnapshot(snapshotBroker, accountID, userID, o)
		writeClosedTradeRecord(log, tradeRecordRepo, writeCtx, accountID, o)
	}
}

func publishPositionSnapshot(broker *mthub.PositionSnapshotBroker, accountID, userID string, o *mdtick.OrderUpdate) {
	snapshot := &mthub.PositionSnapshot{
		AccountID: accountID, UserID: userID, Platform: o.Platform,
		Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
		Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
		Profit: o.Profit, Positions: make([]mthub.PositionSnapshotItem, 0, len(o.Positions)),
	}
	for _, pos := range o.Positions {
		snapshot.Positions = append(snapshot.Positions, mthub.PositionSnapshotItem{
			Ticket: pos.Ticket, Symbol: pos.Symbol, Type: pos.Type,
			Volume: pos.Volume, OpenPrice: pos.OpenPrice, CurrentPrice: pos.CurrentPrice,
			StopLoss: pos.StopLoss, TakeProfit: pos.TakeProfit,
			Profit: pos.Profit, Swap: pos.Swap, Commission: pos.Commission,
			Comment: pos.Comment, OpenTime: pos.OpenTime,
		})
	}
	broker.Publish(snapshot)
}

func writeClosedTradeRecord(log *zap.Logger, repo *repository.TradeRecordRepository, ctx context.Context, accountID string, o *mdtick.OrderUpdate) {
	if !strings.EqualFold(o.UpdateType, "close") || o.UpdateCloseTime <= 0 {
		return
	}
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return
	}
	rec := &model.TradeRecord{
		AccountID: uid, Ticket: o.UpdateTicket, Symbol: o.UpdateSymbol, OrderType: o.UpdateOrderType,
		Volume: o.UpdateVolume, OpenPrice: o.UpdateOpenPrice,
		ClosePrice: o.UpdateClosePrice, Profit: o.UpdateProfit,
		Swap: o.UpdateSwap, Commission: o.UpdateCommission,
		OpenTime: time.Unix(o.UpdateOpenTime, 0), CloseTime: time.Unix(o.UpdateCloseTime, 0),
		StopLoss: o.UpdateSL, TakeProfit: o.UpdateTP,
		OrderComment: o.UpdateComment, Platform: o.Platform,
	}
	if err := repo.Create(ctx, rec); err != nil {
		log.Warn("OnOrderUpdate: write closed trade failed", zap.String("account", accountID), zap.Int64("ticket", o.UpdateTicket), zap.Error(err))
	}
}
