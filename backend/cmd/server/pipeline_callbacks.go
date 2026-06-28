package main

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/service"
)

// buildOnOrderUpdate creates the OnOrderUpdate callback for the mdgateway pipeline.
// It updates PG metrics, publishes profit/snapshot events, feeds platform aggregator,
// and auto-writes closed orders to trade_records.
func buildOnOrderUpdate(
	log *zap.Logger,
	accountSvc *service.AccountService,
	accountBroker *mthub.AccountProfitBroker,
	snapshotBroker *mthub.PositionSnapshotBroker,
	tradeRecordRepo *repository.TradeRecordRepository,
	platformAgg **risksvc.PlatformAggregator,
) func(accountID, userID string, o *mdtick.OrderUpdate) {
	return func(accountID, userID string, o *mdtick.OrderUpdate) {
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		userUID, err := uuid.Parse(userID)
		if err != nil { log.Warn("OnOrderUpdate: invalid user UUID", zap.String("userID", userID), zap.Error(err)); return }
		if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, o.Balance.InexactFloat64(), o.Equity.InexactFloat64(), o.Credit.InexactFloat64(), o.Margin.InexactFloat64(), o.FreeMargin.InexactFloat64(), o.MarginLevel.InexactFloat64()); err != nil {
			log.Warn("OnOrderUpdate: pg update failed", zap.String("account", accountID), zap.Error(err))
		}
		publishProfitEvent(accountBroker, accountID, userID, o)
		// Update in-memory summary cache for SSE SubscribeUserSummary.
		accountSvc.UpdateSummaryCache(userID, accountID, o.Balance.InexactFloat64(), o.Equity.InexactFloat64(), "connected")
		publishPositionSnapshot(snapshotBroker, accountID, userID, o)
		feedPlatformAggregator(platformAgg, accountID, o)
		writeClosedTradeRecord(log, tradeRecordRepo, writeCtx, accountID, o)
	}
}

func publishProfitEvent(broker *mthub.AccountProfitBroker, accountID, userID string, o *mdtick.OrderUpdate) {
	broker.Publish(&mthub.AccountProfitEvent{
		AccountID: accountID, UserID: userID, Platform: o.Platform,
		Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
		Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
		Profit: o.Profit, ProfitPercent: o.ProfitPercent, Status: "connected", Timestamp: time.Now(),
		Positions: mapOrderPositions(o.Positions),
	})
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

func feedPlatformAggregator(platformAgg **risksvc.PlatformAggregator, accountID string, o *mdtick.OrderUpdate) {
	(*platformAgg).ClearAccount(accountID)
	for _, pos := range o.Positions {
		netVol := pos.Volume.InexactFloat64()
		if pos.Type == "sell" { netVol = -netVol }
		(*platformAgg).UpdatePosition(accountID, &risksvc.AggregatorPosition{
			Canonical: pos.Symbol, NetVolume: netVol, Notional: pos.Volume.Mul(pos.CurrentPrice).Mul(decimal.NewFromInt(100000)).InexactFloat64(), Margin: 0,
		})
	}
}

func mapOrderPositions(positions []mdtick.OrderUpdatePosition) []mthub.AccountProfitPosition {
	out := make([]mthub.AccountProfitPosition, 0, len(positions))
	for _, pos := range positions {
		out = append(out, mthub.AccountProfitPosition{
			Ticket: pos.Ticket, Symbol: pos.Symbol, Profit: pos.Profit,
			Volume: pos.Volume, CurrentPrice: pos.CurrentPrice,
		})
	}
	return out
}

func writeClosedTradeRecord(log *zap.Logger, repo *repository.TradeRecordRepository, ctx context.Context, accountID string, o *mdtick.OrderUpdate) {
	if !strings.EqualFold(o.UpdateType, "close") || o.UpdateCloseTime <= 0 { return }
	uid, err := uuid.Parse(accountID)
	if err != nil { return }
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
