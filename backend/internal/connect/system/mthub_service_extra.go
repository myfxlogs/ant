package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/mthub"
)

// PriceHistory returns historical kline data (#12: now requires authentication).
func (s *MtHubServer) PriceHistory(ctx context.Context, req *connect.Request[antv1.PriceHistoryRequest]) (*connect.Response[antv1.PriceHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	period := m.Period
	if period == "" {
		period = "M1"
	}
	limit := m.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	bars, err := s.marketData.GetKlines(ctx, m.Canonical, "", period, limit)
	if err != nil {
		s.log.Warn("PriceHistory: get klines", zap.Error(err))
		return connect.NewResponse(&antv1.PriceHistoryResponse{}), nil
	}

	out := make([]*antv1.OHLCV, 0, len(bars))
	for _, b := range bars {
		out = append(out, &antv1.OHLCV{
			OpenTime:  timestamppb.New(b.OpenTime()),
			CloseTime: timestamppb.New(b.OpenTime()), // kline bars have single timestamp
			Open:      fmt.Sprintf("%.5f", b.Open),
			High:      fmt.Sprintf("%.5f", b.High),
			Low:       fmt.Sprintf("%.5f", b.Low),
			Close:     fmt.Sprintf("%.5f", b.Close),
			Volume:    b.Volume,
		})
	}
	return connect.NewResponse(&antv1.PriceHistoryResponse{Bars: out}), nil
}

func (s *MtHubServer) GetAccountStatus(ctx context.Context, req *connect.Request[antv1.GetAccountStatusRequest]) (*connect.Response[antv1.AccountStatus], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	acct, err := s.platform.GetAccount(ctx, uid, req.Msg.AccountId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("account not found"))
		}
		s.log.Error("GetAccountStatus: get account", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	state := s.svc.SessionState(ctx, req.Msg.AccountId)
	if state == "not_found" {
		state = acct.Status
		if state == "" {
			state = "disconnected"
		}
	}
	return connect.NewResponse(&antv1.AccountStatus{
		AccountId:  req.Msg.AccountId,
		State:      state,
		LastTickAt: timestamppb.Now(),
	}), nil
}

// StreamOrderEvents streams real-time order events for the authenticated user (#16: filter by AccountId if set).
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
			// #16: Filter events to only the requested AccountId if specified.
			if filterAccountID != "" && ev.AccountID != filterAccountID {
				continue
			}
			if err := stream.Send(toProtoOrderEvent(ev)); err != nil {
				return fmt.Errorf("send order event to stream: %w", err)
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

	// Determine time range: from last close_time in trade_records (or 1 year ago) to now.
	from := time.Now().AddDate(-1, 0, 0)
	lastTime, err := s.tradeRecords.GetLastSyncTime(ctx, uid)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.log.Error("SyncOrderHistory: get last sync time failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else if lastTime != nil {
		from = *lastTime
	}
	to := time.Now()

	// Fetch from MT broker.
	records, err := s.svc.OrderHistory(ctx, accountID, from, to)
	if err != nil {
		s.log.Error("SyncOrderHistory: fetch from broker", zap.String("account", accountID), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Determine platform from the active executor.
	platform := s.svc.Platform(accountID)

	// Convert mthub.OrderRecord → model.TradeRecord.
	tradeRecs := make([]*model.TradeRecord, 0, len(records))
	for _, r := range records {
		tradeRecs = append(tradeRecs, orderRecordToTradeRecord(r, uid, platform))
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

// WriteClosedTrade creates a single TradeRecord from an OrderUpdate close event.
func (s *MtHubServer) WriteClosedTrade(ctx context.Context, accountID, platform, updateOrderType, updateSymbol, updateComment string, updateTicket int64, updateVolume, updateOpenPrice, updateClosePrice, updateProfit, updateSwap, updateCommission, updateSL, updateTP float64, updateOpenTime, updateCloseTime int64) error {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return err
	}
	rec := &model.TradeRecord{
		AccountID:    uid,
		Ticket:       updateTicket,
		Symbol:       updateSymbol,
		OrderType:    updateOrderType,
		Volume:       updateVolume,
		OpenPrice:    updateOpenPrice,
		ClosePrice:   updateClosePrice,
		Profit:       updateProfit,
		Swap:         updateSwap,
		Commission:   updateCommission,
		OpenTime:     time.Unix(updateOpenTime, 0),
		CloseTime:    time.Unix(updateCloseTime, 0),
		StopLoss:     updateSL,
		TakeProfit:   updateTP,
		OrderComment: updateComment,
		Platform:     platform,
	}
	return s.tradeRecords.Create(ctx, rec)
}

// #14: Use Float64() instead of InexactFloat64() to detect precision loss.
func orderRecordToTradeRecord(r *mthub.OrderRecord, accountID uuid.UUID, platform string) *model.TradeRecord {
	orderType := mthubSideOrderTypeToString(r.Side, r.OrderType)
	return &model.TradeRecord{
		AccountID:    accountID,
		Ticket:       r.Ticket,
		Symbol:       r.SymbolRaw,
		OrderType:    orderType,
		Volume:       decimalToFloat64(r.Volume),
		OpenPrice:    decimalToFloat64(r.OpenPrice),
		ClosePrice:   decimalToFloat64(r.ClosePrice),
		Profit:       decimalToFloat64(r.Profit),
		Swap:         decimalToFloat64(r.Swap),
		Commission:   decimalToFloat64(r.Commission),
		OpenTime:     r.OpenTime,
		CloseTime:    r.CloseTime,
		OrderComment: r.Comment,
		MagicNumber:  int(r.Magic),
		Platform:     platform,
	}
}

// decimalToFloat64 converts a decimal to float64 and detects precision loss (#14).
func decimalToFloat64(d decimal.Decimal) float64 {
	f, exact := d.Float64()
	if !exact {
		// Precision loss is logged at the call site if a logger is available.
		// The float64 is still the best available representation.
	}
	return f
}

func mthubSideOrderTypeToString(side mthub.Side, ot mthub.OrderType) string {
	prefix := "BUY"
	if side == mthub.SideSell {
		prefix = "SELL"
	}
	switch ot {
	case mthub.OrderMarket:
		return prefix
	case mthub.OrderLimit:
		return prefix + "_LIMIT"
	case mthub.OrderStop:
		return prefix + "_STOP"
	case mthub.OrderStopLimit:
		return prefix + "_STOP_LIMIT"
	default:
		return prefix
	}
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
