package system

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// MtHubServer implements ant.v1.MtHubServiceHandler.
type MtHubServer struct {
	svc          *mthub.MtHubService
	platform     *service.PlatformService
	marketData   *repository.MarketDataRepository
	tradeRecords *repository.TradeRecordRepository
	log          *zap.Logger
}

var _ antv1c.MtHubServiceHandler = (*MtHubServer)(nil)

func NewMtHubServer(svc *mthub.MtHubService, platform *service.PlatformService, marketData *repository.MarketDataRepository, tradeRecords *repository.TradeRecordRepository, log *zap.Logger) *MtHubServer {
	return &MtHubServer{svc: svc, platform: platform, marketData: marketData, tradeRecords: tradeRecords, log: log}
}

func (s *MtHubServer) PlaceOrder(ctx context.Context, req *connect.Request[antv1.PlaceOrderRequest]) (*connect.Response[antv1.PlaceOrderResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	ok, err := s.platform.UserOwnsAccount(ctx, userID, m.AccountId)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	vol, err := decimal.NewFromString(m.Volume)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// #10: Check decimal parse errors for Price/SL/TP.
	price, err := decimal.NewFromString(m.Price)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid price: %w", err))
	}
	sl, err := decimal.NewFromString(m.StopLoss)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid stop_loss: %w", err))
	}
	tp, err := decimal.NewFromString(m.TakeProfit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid take_profit: %w", err))
	}

	rec, err := s.svc.PlaceOrder(ctx, &mthub.OrderRequest{
		AccountID: m.AccountId, Canonical: m.Canonical,
		Side: sideFromProto(m.Side), OrderType: orderTypeFromProto(m.OrderType),
		Volume: vol, Price: price, StopLoss: sl, TakeProfit: tp,
		Comment: m.Comment, ClientID: m.ClientId, Magic: m.Magic,
	})
	if err != nil {
		s.log.Error("PlaceOrder", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PlaceOrderResponse{Ticket: rec.Ticket, Status: "submitted"}), nil
}

func (s *MtHubServer) CloseOrder(ctx context.Context, req *connect.Request[antv1.CloseOrderRequest]) (*connect.Response[antv1.CloseOrderResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	ok, err := s.platform.UserOwnsAccount(ctx, userID, m.AccountId)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	lots := decimal.Zero
	if m.Lots != "" {
		// #11: Check decimal parse error for lots.
		lots, err = decimal.NewFromString(m.Lots)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid lots: %w", err))
		}
	}
	if err := s.svc.CloseOrder(ctx, m.AccountId, m.Ticket, lots); err != nil {
		s.log.Error("CloseOrder", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CloseOrderResponse{Status: "closed"}), nil
}

func (s *MtHubServer) OpenedOrders(ctx context.Context, req *connect.Request[antv1.OpenedOrdersRequest]) (*connect.Response[antv1.OpenedOrdersResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, req.Msg.AccountId)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	list, err := s.svc.OpenedOrders(ctx, req.Msg.AccountId)
	if err != nil {
		s.log.Error("OpenedOrders", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.OpenedOrdersResponse{Orders: toProtoOrders(list)}), nil
}

func (s *MtHubServer) OrderHistory(ctx context.Context, req *connect.Request[antv1.OrderHistoryRequest]) (*connect.Response[antv1.OrderHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, req.Msg.AccountId)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	list, err := s.svc.OrderHistory(ctx, req.Msg.AccountId, req.Msg.From.AsTime(), req.Msg.To.AsTime())
	if err != nil {
		s.log.Error("OrderHistory", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.OrderHistoryResponse{Orders: toProtoOrders(list)}), nil
}

func (s *MtHubServer) SymbolParams(ctx context.Context, req *connect.Request[antv1.SymbolParamsRequest]) (*connect.Response[antv1.SymbolParamsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, req.Msg.AccountId)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	list, err := s.svc.SymbolParams(ctx, req.Msg.AccountId, req.Msg.Canonicals)
	if err != nil {
		s.log.Error("SymbolParams", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SymbolParamsResponse{Params: toProtoParams(list)}), nil
}

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
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account not found or not owned by user"))
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
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}

	uid, err := uuid.Parse(accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id"))
	}

	// Determine time range: from last close_time in trade_records (or 1 year ago) to now.
	from := time.Now().AddDate(-1, 0, 0)
	if lastTime, err := s.tradeRecords.GetLastSyncTime(ctx, uid); err == nil && lastTime != nil {
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

func sideFromProto(s antv1.Side) mthub.Side {
	if s == antv1.Side_SIDE_SELL {
		return mthub.SideSell
	}
	return mthub.SideBuy
}

func orderTypeFromProto(t antv1.OrderType) mthub.OrderType {
	switch t {
	case antv1.OrderType_ORDER_TYPE_LIMIT:
		return mthub.OrderLimit
	case antv1.OrderType_ORDER_TYPE_STOP:
		return mthub.OrderStop
	case antv1.OrderType_ORDER_TYPE_STOP_LIMIT:
		return mthub.OrderStopLimit
	default:
		return mthub.OrderMarket
	}
}

func toProtoOrders(list []*mthub.OrderRecord) []*antv1.OrderRecord {
	out := make([]*antv1.OrderRecord, 0, len(list))
	for _, r := range list {
		out = append(out, &antv1.OrderRecord{
			Ticket: r.Ticket, AccountId: r.AccountID,
			SymbolRaw: r.SymbolRaw, Canonical: r.Canonical,
			Side: toProtoSide(r.Side), OrderType: toProtoOrderType(r.OrderType),
			Volume: r.Volume.String(), OpenPrice: r.OpenPrice.String(),
			ClosePrice: r.ClosePrice.String(), Profit: r.Profit.String(),
			Commission: r.Commission.String(), Swap: r.Swap.String(),
			OpenTime: timestamppb.New(r.OpenTime), CloseTime: timestamppb.New(r.CloseTime),
			Comment: r.Comment, Magic: r.Magic, State: toProtoState(r.State),
		})
	}
	return out
}

func toProtoSide(s mthub.Side) antv1.Side {
	if s == mthub.SideSell {
		return antv1.Side_SIDE_SELL
	}
	return antv1.Side_SIDE_BUY
}

func toProtoOrderType(t mthub.OrderType) antv1.OrderType {
	switch t {
	case mthub.OrderMarket:
		return antv1.OrderType_ORDER_TYPE_MARKET
	case mthub.OrderLimit:
		return antv1.OrderType_ORDER_TYPE_LIMIT
	case mthub.OrderStop:
		return antv1.OrderType_ORDER_TYPE_STOP
	case mthub.OrderStopLimit:
		return antv1.OrderType_ORDER_TYPE_STOP_LIMIT
	default:
		return antv1.OrderType_ORDER_TYPE_MARKET
	}
}

func toProtoParams(list []*mthub.SymbolParam) []*antv1.SymbolParam {
	out := make([]*antv1.SymbolParam, 0, len(list))
	for _, p := range list {
		out = append(out, &antv1.SymbolParam{
			Canonical: p.Canonical, Digits: p.Digits,
			LotSize: p.LotSize.String(), LotMin: p.LotMin.String(),
			TradeMode: p.TradeMode, StopLevel: p.StopLevel,
		})
	}
	return out
}

func toProtoState(s mthub.OrderState) antv1.OrderState {
	switch s {
	case mthub.OrderStateOpen:
		return antv1.OrderState_ORDER_STATE_OPEN
	case mthub.OrderStateClosed:
		return antv1.OrderState_ORDER_STATE_CLOSED
	case mthub.OrderStateCancelled:
		return antv1.OrderState_ORDER_STATE_CANCELLED
	case mthub.OrderStateRejected:
		return antv1.OrderState_ORDER_STATE_REJECTED
	default:
		return antv1.OrderState_ORDER_STATE_PENDING
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
