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
		userUID, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			log.Warn("OnOrderUpdate: invalid user UUID", zap.String("userID", userID), zap.Error(parseErr))
			return
		}
		if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, o.Balance, o.Equity, o.Credit, o.Margin, o.FreeMargin, o.MarginLevel); err != nil {
			log.Warn("OnOrderUpdate: pg update failed", zap.String("account", accountID), zap.Error(err))
		}

		accountBroker.Publish(&mthub.AccountProfitEvent{
			AccountID: accountID, UserID: userID, Platform: o.Platform,
			Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
			Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
			Profit: o.Profit, ProfitPercent: o.ProfitPercent, Status: "connected", Timestamp: time.Now(),
			Positions: func() []mthub.AccountProfitPosition {
				out := make([]mthub.AccountProfitPosition, 0, len(o.Positions))
				for _, pos := range o.Positions {
					out = append(out, mthub.AccountProfitPosition{
						Ticket: pos.Ticket, Symbol: pos.Symbol,
						Profit: pos.Profit, Volume: pos.Volume,
						CurrentPrice: pos.CurrentPrice,
					})
				}
				return out
			}(),
		})

		snapshot := &mthub.PositionSnapshot{
			AccountID:   accountID, UserID: userID, Platform: o.Platform,
			Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
			Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
			Profit: o.Profit,
			Positions: make([]mthub.PositionSnapshotItem, 0, len(o.Positions)),
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
		snapshotBroker.Publish(snapshot)

		(*platformAgg).ClearAccount(accountID)
		for _, pos := range o.Positions {
			netVol := pos.Volume
			if pos.Type == "sell" {
				netVol = -netVol
			}
			(*platformAgg).UpdatePosition(accountID, &risksvc.AggregatorPosition{
				Canonical: pos.Symbol,
				NetVolume: netVol,
				Notional:  pos.Volume * pos.CurrentPrice * 100000,
				Margin:    0,
			})
		}

		if strings.EqualFold(o.UpdateType, "close") && o.UpdateCloseTime > 0 {
			uid, err := uuid.Parse(accountID)
			if err == nil {
				rec := &model.TradeRecord{
					AccountID:    uid,
					Ticket:       o.UpdateTicket,
					Symbol:       o.UpdateSymbol,
					OrderType:    o.UpdateOrderType,
					Volume:       decimal.NewFromFloat(o.UpdateVolume),
					OpenPrice:    decimal.NewFromFloat(o.UpdateOpenPrice),
					ClosePrice:   decimal.NewFromFloat(o.UpdateClosePrice),
					Profit:       decimal.NewFromFloat(o.UpdateProfit),
					Swap:         decimal.NewFromFloat(o.UpdateSwap),
					Commission:   decimal.NewFromFloat(o.UpdateCommission),
					OpenTime:     time.Unix(o.UpdateOpenTime, 0),
					CloseTime:    time.Unix(o.UpdateCloseTime, 0),
					StopLoss:     decimal.NewFromFloat(o.UpdateSL),
					TakeProfit:   decimal.NewFromFloat(o.UpdateTP),
					OrderComment: o.UpdateComment,
					Platform:     o.Platform,
				}
				if err := tradeRecordRepo.Create(writeCtx, rec); err != nil {
					log.Warn("OnOrderUpdate: write closed trade failed",
						zap.String("account", accountID), zap.Int64("ticket", o.UpdateTicket), zap.Error(err))
				}
			}
		}
	}
}
