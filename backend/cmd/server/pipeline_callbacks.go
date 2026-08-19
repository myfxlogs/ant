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
	mthubSvc *mthub.MtHubService,
) func(accountID, userID string, o *mdtick.OrderUpdate) {
	return func(accountID, userID string, o *mdtick.OrderUpdate) {
		// Use a detached context with timeout — the gRPC stream context may expire
		// before the DB write completes, but the trade record must be persisted.
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		publishPositionSnapshot(snapshotBroker, accountID, userID, o)
		writeClosedTradeRecord(log, tradeRecordRepo, writeCtx, accountID, userID, o)

		// EXEC-2: Transition OMS order state based on broker update type.
		transitionOMSByUpdate(writeCtx, mthubSvc, accountID, o)

		// EXEC-3: Publish trade event to TradeBroker for strategy OnTrade callbacks.
		mthubSvc.PublishTradeEventFromUpdate(
			accountID, o.UpdateType, o.UpdateOrderType, o.UpdateSymbol,
			o.UpdateTicket, o.UpdateVolume, o.UpdateClosePrice,
			o.UpdateSL, o.UpdateTP, o.UpdateProfit, o.UpdateCommission, o.UpdateSwap,
		)
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
			Ticket: pos.Ticket, Symbol: pos.Symbol, Type: pos.Type, Magic: pos.Magic,
			Volume: pos.Volume, OpenPrice: pos.OpenPrice, CurrentPrice: pos.CurrentPrice,
			StopLoss: pos.StopLoss, TakeProfit: pos.TakeProfit,
			Profit: pos.Profit, Swap: pos.Swap, Commission: pos.Commission,
			Comment: pos.Comment, OpenTime: pos.OpenTime,
		})
	}
	broker.Publish(snapshot)
}

func publishProfitPositionSnapshot(broker *mthub.PositionSnapshotBroker, accountID, userID string, p *mdtick.ProfitUpdate) {
	snapshot := &mthub.PositionSnapshot{
		AccountID: accountID, UserID: userID, Platform: p.Platform,
		Balance: p.Balance, Credit: p.Credit, Equity: p.Equity,
		Margin: p.Margin, FreeMargin: p.FreeMargin, MarginLevel: p.MarginLevel,
		Profit: p.Profit, Positions: make([]mthub.PositionSnapshotItem, 0, len(p.Positions)),
	}
	for _, pos := range p.Positions {
		snapshot.Positions = append(snapshot.Positions, mthub.PositionSnapshotItem{
			Ticket: pos.Ticket, Symbol: pos.Symbol, Type: pos.Type, Magic: pos.Magic,
			Volume: pos.Volume, OpenPrice: pos.OpenPrice, CurrentPrice: pos.CurrentPrice,
			StopLoss: pos.StopLoss, TakeProfit: pos.TakeProfit,
			Profit: pos.Profit, Swap: pos.Swap, Commission: pos.Commission,
			Comment: pos.Comment, OpenTime: pos.OpenTime,
		})
	}
	broker.Publish(snapshot)
}

func writeClosedTradeRecord(log *zap.Logger, repo *repository.TradeRecordRepository, ctx context.Context, accountID, userID string, o *mdtick.OrderUpdate) {
	if !strings.EqualFold(o.UpdateType, "close") || o.UpdateCloseTime <= 0 {
		return
	}
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return
	}
	rec := &model.TradeRecord{
		UserID: userUUID, AccountID: uid, Ticket: o.UpdateTicket, Symbol: o.UpdateSymbol, OrderType: o.UpdateOrderType,
		Volume: o.UpdateVolume, OpenPrice: o.UpdateOpenPrice,
		ClosePrice: o.UpdateClosePrice, Profit: o.UpdateProfit,
		Swap: o.UpdateSwap, Commission: o.UpdateCommission,
		OpenTime: time.Unix(o.UpdateOpenTime, 0), CloseTime: time.Unix(o.UpdateCloseTime, 0),
		StopLoss: o.UpdateSL, TakeProfit: o.UpdateTP,
		OrderComment: o.UpdateComment, Platform: o.Platform,
	}
	if err := repo.Create(ctx, rec); err != nil {
		// Retry once with a fresh timeout — transient PG errors (pool exhaustion,
		// brief network flaps) should not lose trade records.
		retryCtx, retryCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer retryCancel()
		if retryErr := repo.Create(retryCtx, rec); retryErr != nil {
			log.Error("OnOrderUpdate: write closed trade failed after retry",
				zap.String("account", accountID), zap.Int64("ticket", o.UpdateTicket),
				zap.NamedError("first", err), zap.NamedError("retry", retryErr))
		}
	}
}

// transitionOMSByUpdate maps broker OnOrderUpdate event types to OMS state transitions.
// EXEC-2: Without this, orders stay stuck in SUBMITTED forever.
func transitionOMSByUpdate(ctx context.Context, svc *mthub.MtHubService, accountID string, o *mdtick.OrderUpdate) {
	ut := strings.ToLower(o.UpdateType)
	switch ut {
	case "close", "pending_close":
		svc.TransitionOrderByTicket(ctx, accountID, o.UpdateTicket, mthub.OMSStateFilled)
	case "delete":
		svc.TransitionOrderByTicket(ctx, accountID, o.UpdateTicket, mthub.OMSStateCancelled)
	default:
		svc.TransitionOrderByTicket(ctx, accountID, o.UpdateTicket, mthub.OMSStateWorking)
	}
}
