package system

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
)

func decToFloat(d decimal.Decimal) float64 {
	f, exact := d.Float64()
	// When !exact the decimal value exceeds float64 range/precision (e.g.,
	// extremely large/small values). Financial amounts (profit, margin, etc.)
	// typically fit within float64; lossy conversion is acceptable for display.
	_ = exact
	return f
}

func orderRecordToUpdateEvent(rec *mthub.OrderRecord, accountID string, eventType string, ticket int64) *antv1.OrderUpdateEvent {
	return &antv1.OrderUpdateEvent{
		AccountId: accountID,
		Ticket:    ticket,
		Symbol: func() string {
			if rec.Canonical != "" {
				return rec.Canonical
			}
			return rec.SymbolRaw
		}(),
		Type:       orderSideTypeLabel(rec.Side, rec.OrderType),
		Volume:     decToFloat(rec.Volume),
		OpenPrice:  decToFloat(rec.OpenPrice),
		Profit:     decToFloat(rec.Profit),
		Action:     eventType,
		ClosePrice: decToFloat(rec.ClosePrice),
		Swap:       decToFloat(rec.Swap),
		Commission: decToFloat(rec.Commission),
		Comment:    rec.Comment,
		OpenTime:   rec.OpenTime.Unix(),
		CloseTime:  rec.CloseTime.Unix(),
	}
}

func orderSideTypeLabel(side mthub.Side, ot mthub.OrderType) string {
	prefix := "buy"
	if side == mthub.SideSell {
		prefix = "sell"
	}
	switch ot {
	case mthub.OrderMarket:
		return prefix
	case mthub.OrderLimit:
		return prefix + "_limit"
	case mthub.OrderStop:
		return prefix + "_stop"
	case mthub.OrderStopLimit:
		return prefix + "_stop_limit"
	default:
		return prefix
	}
}

func profitEventToProto(pev *mthub.AccountProfitEvent) *antv1.ProfitUpdateEvent {
	orders := make([]*antv1.OrderProfitItem, 0, len(pev.Positions))
	for _, pos := range pev.Positions {
		orders = append(orders, &antv1.OrderProfitItem{
			Ticket:       pos.Ticket,
			Symbol:       pos.Symbol,
			Profit:       pos.Profit,
			Volume:       pos.Volume,
			CurrentPrice: pos.CurrentPrice,
		})
	}
	return &antv1.ProfitUpdateEvent{
		AccountId:     pev.AccountID,
		Balance:       pev.Balance,
		Credit:        pev.Credit,
		Equity:        pev.Equity,
		Profit:        pev.Profit,
		Margin:        pev.Margin,
		FreeMargin:    pev.FreeMargin,
		MarginLevel:   pev.MarginLevel,
		ProfitPercent: pev.ProfitPercent,
		Orders:        orders,
	}
}

func (s *StreamServer) sendInitialSnapshot(
	ctx context.Context,
	stream *connect.ServerStream[antv1.StreamEvent],
	userID string,
	accountSet map[string]bool,
	filterAll bool,
) (connectedIDs []string) {
	snapshots, err := s.platform.GetUserAccountSnapshots(ctx, userID)
	if err != nil {
		s.log.Warn("sendInitialSnapshot: GetUserAccountSnapshots failed", zap.Error(err))
		return nil
	}

	now := timestamppb.Now()
	for _, a := range snapshots {
		if !filterAll && !accountSet[a.ID] {
			continue
		}

		profit := a.Equity - a.Balance
		var profitPercent float64
		if a.Balance > 0 {
			profitPercent = profit / a.Balance * 100
		}

		if err := stream.Send(&antv1.StreamEvent{
			Type:      "profit_update",
			AccountId: a.ID,
			Timestamp: now,
			Payload: &antv1.StreamEvent_ProfitUpdate{
				ProfitUpdate: &antv1.ProfitUpdateEvent{
					AccountId:     a.ID,
					Balance:       a.Balance,
					Equity:        a.Equity,
					Profit:        profit,
					Credit:        a.Credit,
					Margin:        a.Margin,
					FreeMargin:    a.FreeMargin,
					MarginLevel:   a.MarginLevel,
					ProfitPercent: profitPercent,
				},
			},
		}); err != nil {
			s.log.Warn("sendInitialSnapshot: send profit_update failed", zap.String("account", a.ID), zap.Error(err))
		}

		if err := stream.Send(&antv1.StreamEvent{
			Type:      "account_status",
			AccountId: a.ID,
			Timestamp: now,
			Payload: &antv1.StreamEvent_AccountStatus{
				AccountStatus: &antv1.AccountStatusEvent{
					AccountId: a.ID,
					Status:    a.Status,
				},
			},
		}); err != nil {
			s.log.Warn("sendInitialSnapshot: send account_status failed", zap.String("account", a.ID), zap.Error(err))
		}

		if a.Status == "connected" {
			connectedIDs = append(connectedIDs, a.ID)
		}
	}
	return connectedIDs
}

func (s *StreamServer) sendInitialPositionSnapshots(
	ctx context.Context,
	stream *connect.ServerStream[antv1.StreamEvent],
	connectedIDs []string,
) {
	for _, aid := range connectedIDs {
		rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		orders, err := s.svc.OpenedOrders(rpcCtx, aid)
		cancel()
		if err != nil {
			s.log.Warn("sendInitialSnapshot: OpenedOrders failed, skipping position snapshot",
				zap.String("account", aid), zap.Error(err))
			continue
		}
		now := timestamppb.Now()
		positions := make([]*antv1.OrderUpdateEvent, 0, len(orders))
		for _, rec := range orders {
			positions = append(positions, orderRecordToUpdateEvent(rec, aid, "open", rec.Ticket))
		}
		if err := stream.Send(&antv1.StreamEvent{
			Type:      "position_snapshot",
			AccountId: aid,
			Timestamp: now,
			Payload: &antv1.StreamEvent_PositionSnapshot{
				PositionSnapshot: &antv1.PositionSnapshotEvent{
					AccountId: aid,
					Positions: positions,
				},
			},
		}); err != nil {
			s.log.Warn("sendInitialSnapshot: send position_snapshot failed", zap.String("account", aid), zap.Error(err))
		}
	}
}
