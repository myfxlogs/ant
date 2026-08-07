package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
)

// StreamOrderEvents streams real-time order events for the authenticated user.
func (s *MtHubServer) StreamOrderEvents(ctx context.Context, req *connect.Request[antv1.StreamOrderEventsRequest], stream *connect.ServerStream[antv1.OrderEvent]) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	filterAccountID := req.Msg.AccountId
	ch, cancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if filterAccountID != "" && ev.AccountID != filterAccountID {
				continue
			}
			if err := stream.Send(toProtoOrderEvent(ev)); err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("send order event to stream: %w", err))
			}
		}
	}
}

// SyncOrderHistory fetches order history from the MT broker and writes it to trade_records.
func (s *MtHubServer) SyncOrderHistory(ctx context.Context, req *connect.Request[antv1.SyncOrderHistoryRequest]) (*connect.Response[antv1.SyncOrderHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	accountID := req.Msg.AccountId
	ok, err := s.platform.UserOwnsAccount(ctx, userID, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}

	uid, err := uuid.Parse(accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id"))
	}

	from := time.Now().AddDate(-1, 0, 0)
	parsedUID, _ := uuid.Parse(userID)
	lastTime, err := s.tradeRecords.GetLastSyncTime(ctx, parsedUID, uid)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.log.Error("SyncOrderHistory: get last sync time failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else if lastTime != nil {
		from = *lastTime
	}
	to := time.Now()

	records, err := s.svc.OrderHistory(ctx, accountID, from, to)
	if err != nil {
		s.log.Error("SyncOrderHistory: fetch from broker", zap.String("account", accountID), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	platform := s.svc.Platform(accountID)

	tradeRecs := make([]*model.TradeRecord, 0, len(records))
	parsedUID, _ = uuid.Parse(userID)
	for _, r := range records {
		rec := orderRecordToTradeRecord(r, uid, parsedUID, platform)
		tradeRecs = append(tradeRecs, rec)
	}

	if err := s.tradeRecords.BatchCreate(ctx, tradeRecs); err != nil {
		s.log.Error("SyncOrderHistory: batch create", zap.String("account", accountID), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.log.Info("SyncOrderHistory: synced",
		zap.String("account", accountID),
		zap.Int("records", len(tradeRecs)))
	return connect.NewResponse(&antv1.SyncOrderHistoryResponse{SyncedRecords: int64(len(tradeRecs))}), nil
}

// ClosedTradeParams holds parameters for WriteClosedTrade.
type ClosedTradeParams struct {
	AccountID, Platform, OrderType, Symbol, Comment                 string
	Ticket                                                          int64
	Volume, OpenPrice, ClosePrice, Profit, Swap, Commission, SL, TP decimal.Decimal
	OpenTime, CloseTime                                             int64
}

func (s *MtHubServer) WriteClosedTrade(ctx context.Context, p ClosedTradeParams) error {
	uid, err := uuid.Parse(p.AccountID)
	if err != nil {
		return err
	}
	rec := &model.TradeRecord{
		UserID:       uuid.Nil,
		AccountID:    uid,
		Ticket:       p.Ticket,
		Symbol:       p.Symbol,
		OrderType:    p.OrderType,
		Volume:       p.Volume,
		OpenPrice:    p.OpenPrice,
		ClosePrice:   p.ClosePrice,
		Profit:       p.Profit,
		Swap:         p.Swap,
		Commission:   p.Commission,
		OpenTime:     time.Unix(p.OpenTime, 0),
		CloseTime:    time.Unix(p.CloseTime, 0),
		StopLoss:     p.SL,
		TakeProfit:   p.TP,
		OrderComment: p.Comment,
		Platform:     p.Platform,
	}
	return s.tradeRecords.Create(ctx, rec)
}

func orderRecordToTradeRecord(r *mthub.OrderRecord, accountID, userID uuid.UUID, platform string) *model.TradeRecord {
	rec := &model.TradeRecord{
		UserID:       userID,
		AccountID:    accountID,
		Ticket:       r.Ticket,
		Symbol:       r.SymbolRaw,
		OrderType:    r.OrderTypeString(),
		Volume:       r.Volume,
		OpenPrice:    r.OpenPrice,
		ClosePrice:   r.ClosePrice,
		Profit:       r.Profit,
		Swap:         r.Swap,
		Commission:   r.Commission,
		OpenTime:     r.OpenTime,
		CloseTime:    r.CloseTime,
		StopLoss:     r.StopLoss,
		TakeProfit:   r.TakeProfit,
		OrderComment: r.Comment,
		MagicNumber:  int(r.Magic),
		Platform:     platform,
	}
	return rec
}

func toProtoOrderEvent(ev *mthub.OrderEvent) *antv1.OrderEvent {
	order := &antv1.OrderRecord{}
	if ev.Order != nil {
		order.Ticket = ev.Order.Ticket
	}
	return &antv1.OrderEvent{
		AccountId: ev.AccountID, Ticket: ev.Ticket,
		EventType: ev.EventType, Timestamp: timestamppb.New(ev.Timestamp),
		Order: order,
	}
}
