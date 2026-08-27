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
	resolver mthub.ScheduleResolver,
) func(accountID, userID string, o *mdtick.OrderUpdate) {
	return func(accountID, userID string, o *mdtick.OrderUpdate) {
		// Use a detached context with timeout — the gRPC stream context may expire
		// before the DB write completes, but the trade record must be persisted.
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		publishPositionSnapshot(snapshotBroker, accountID, userID, o)
		writeClosedTradeRecord(log, tradeRecordRepo, resolver, writeCtx, accountID, userID, o)

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
	now := time.Now()
	snapshot := &mthub.PositionSnapshot{
		AccountID: accountID, UserID: userID, Platform: o.Platform,
		Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
		Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
		Profit: o.Profit, FinancialsSource: "order_stream", CapturedAt: now,
		PositionsAuthoritative: true,
		// B6: positions provenance — broker event time and source.
		PositionsCapturedAt: now,
		PositionsSource:     "order_stream",
		Positions:           make([]mthub.PositionSnapshotItem, 0, len(o.Positions)),
		PendingOrders:       make([]mthub.PositionSnapshotItem, 0),
		// LIVE-ORDER-REENTRY-1: carry triggering update metadata for barrier confirmation.
		UpdateTicket: o.UpdateTicket,
		UpdateType:   o.UpdateType,
		UpdateMagic:  o.UpdateMagic,
	}
	for _, pos := range o.Positions {
		item := mthub.PositionSnapshotItem{
			Ticket: pos.Ticket, Symbol: pos.Symbol, Type: pos.Type, Magic: pos.Magic,
			Volume: pos.Volume, OpenPrice: pos.OpenPrice, CurrentPrice: pos.CurrentPrice,
			StopLoss: pos.StopLoss, TakeProfit: pos.TakeProfit,
			Profit: pos.Profit, Swap: pos.Swap, Commission: pos.Commission,
			Comment: pos.Comment, OpenTime: pos.OpenTime,
		}
		// LIVE-MQL-ORDER-CONTEXT-1: split market positions from pending orders
		// by order type. MQL4 OrdersTotal = positions + pending, but OrderSelect
		// must distinguish them for OrderType/OrderMagicNumber.
		if mdtick.IsPendingOrderType(pos.Type) {
			snapshot.PendingOrders = append(snapshot.PendingOrders, item)
		} else {
			snapshot.Positions = append(snapshot.Positions, item)
		}
	}
	broker.Publish(snapshot)
}

func publishProfitPositionSnapshot(broker *mthub.PositionSnapshotBroker, accountID, userID string, p *mdtick.ProfitUpdate) {
	snapshot := &mthub.PositionSnapshot{
		AccountID: accountID, UserID: userID, Platform: p.Platform,
		Balance: p.Balance, Credit: p.Credit, Equity: p.Equity,
		Margin: p.Margin, FreeMargin: p.FreeMargin, MarginLevel: p.MarginLevel,
		Profit: p.Profit, Leverage: p.Leverage,
		FinancialsAuthoritative: p.FinancialSource == mdtick.FinancialsSourceAccountSummary,
		FinancialsSource:        p.FinancialSource, CapturedAt: p.CapturedAt,
		PositionsAuthoritative: p.PositionsAuthoritative,
		// B6: positions provenance from profit stream.
		PositionsCapturedAt: p.CapturedAt,
		PositionsSource:     "profit_stream",
		Positions:           make([]mthub.PositionSnapshotItem, 0, len(p.Positions)),
		PendingOrders:       make([]mthub.PositionSnapshotItem, 0),
	}
	for _, pos := range p.Positions {
		item := mthub.PositionSnapshotItem{
			Ticket: pos.Ticket, Symbol: pos.Symbol, Type: pos.Type, Magic: pos.Magic,
			Volume: pos.Volume, OpenPrice: pos.OpenPrice, CurrentPrice: pos.CurrentPrice,
			StopLoss: pos.StopLoss, TakeProfit: pos.TakeProfit,
			Profit: pos.Profit, Swap: pos.Swap, Commission: pos.Commission,
			Comment: pos.Comment, OpenTime: pos.OpenTime,
		}
		if mdtick.IsPendingOrderType(pos.Type) {
			snapshot.PendingOrders = append(snapshot.PendingOrders, item)
		} else {
			snapshot.Positions = append(snapshot.Positions, item)
		}
	}
	broker.Publish(snapshot)
}

// buildClosedTradeRecord constructs a TradeRecord from a closed-order update.
// Returns nil if the update is not a close event or account/user IDs are invalid.
// FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION (修复 B): Magic + ScheduleID
// are resolved here so live-closed trades carry strategy attribution, mirroring
// the SyncOrderHistory path (orderRecordToTradeRecord).
func buildClosedTradeRecord(log *zap.Logger, resolver mthub.ScheduleResolver, ctx context.Context, accountID, userID string, o *mdtick.OrderUpdate) *model.TradeRecord {
	if !strings.EqualFold(o.UpdateType, "close") || o.UpdateCloseTime <= 0 {
		return nil
	}
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return nil
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil
	}
	return &model.TradeRecord{
		UserID: userUUID, AccountID: uid, Ticket: o.UpdateTicket, Symbol: o.UpdateSymbol, OrderType: o.UpdateOrderType,
		Volume: o.UpdateVolume, OpenPrice: o.UpdateOpenPrice,
		ClosePrice: o.UpdateClosePrice, Profit: o.UpdateProfit,
		Swap: o.UpdateSwap, Commission: o.UpdateCommission,
		OpenTime: time.Unix(o.UpdateOpenTime, 0), CloseTime: time.Unix(o.UpdateCloseTime, 0),
		StopLoss: o.UpdateSL, TakeProfit: o.UpdateTP,
		OrderComment: o.UpdateComment, Platform: o.Platform,
		MagicNumber: int(o.UpdateMagic),
		ScheduleID:  mthub.ResolveScheduleID(ctx, resolver, log, uid, o.UpdateMagic),
	}
}

func writeClosedTradeRecord(log *zap.Logger, repo *repository.TradeRecordRepository, resolver mthub.ScheduleResolver, ctx context.Context, accountID, userID string, o *mdtick.OrderUpdate) {
	rec := buildClosedTradeRecord(log, resolver, ctx, accountID, userID, o)
	if rec == nil {
		return
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
