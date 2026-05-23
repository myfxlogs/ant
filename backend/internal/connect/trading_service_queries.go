package connect

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/service"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"
)

func (s *TradingService) GetPositions(ctx context.Context, req *connect.Request[v1.GetPositionsRequest]) (*connect.Response[v1.GetPositionsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	startedAt := time.Now()
	positions, err := s.tradingSvc.GetPositions(ctx, uid, accountID)
	if err != nil {
		logger.Error("获取持仓失败", zap.String("account_id", accountID.String()), zap.Error(err))
		s.logPositionsDiagnostic(ctx, uid, accountID, "failed", 0, time.Since(startedAt), err,
			map[string]interface{}{"transient": isTransientGetPositionsFailure(err)})
		fallback := s.positionsFromStreamSnapshot(req.Msg.AccountId)
		// R0: snapshot fallback is emergency-only for expected transient MT/runtime failures (parity doc).
		if len(fallback) > 0 && isTransientGetPositionsFailure(err) {
			s.logPositionsDiagnostic(ctx, uid, accountID, "fallback_snapshot", len(fallback), time.Since(startedAt), nil,
				map[string]interface{}{"fallback_policy": "transient_only"})
			return connect.NewResponse(&v1.GetPositionsResponse{
				Positions: fallback,
			}), nil
		}
		if len(fallback) > 0 && !isTransientGetPositionsFailure(err) {
			logger.Warn("positions: stream snapshot available but skipped (non-transient native error)",
				zap.String("account_id", accountID.String()),
				zap.Int("snapshot_orders", len(fallback)),
				zap.Error(err))
		}
		s.logPositionsDiagnostic(ctx, uid, accountID, "failed_no_snapshot", 0, time.Since(startedAt), err,
			map[string]interface{}{"had_stream_fallback": len(fallback) > 0})
		// Degrade to empty positions for expected transient MT runtime failures.
		// This prevents frontend hard errors when MT bridge is reconnecting or temporarily unavailable.
		if errors.Is(err, service.ErrAccountNotConnected) ||
			errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return connect.NewResponse(&v1.GetPositionsResponse{
				Positions: []*v1.Order{},
			}), nil
		}
		if err != nil && (containsErrorText(err.Error(), "timed out") || containsErrorText(err.Error(), "timeout")) {
			return connect.NewResponse(&v1.GetPositionsResponse{
				Positions: []*v1.Order{},
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetPositionsResponse{
		Positions: make([]*v1.Order, len(positions)),
	}
	if len(positions) == 0 {
		s.logPositionsDiagnostic(ctx, uid, accountID, "empty", 0, time.Since(startedAt), nil,
			map[string]interface{}{"source": "native"})
	}

	for i, pos := range positions {
		response.Positions[i] = convertPositionResponse(pos, req.Msg.AccountId)
	}

	if len(positions) > 0 {
		s.logPositionsDiagnostic(ctx, uid, accountID, "success", len(positions), time.Since(startedAt), nil,
			map[string]interface{}{"source": "native"})
	}
	return connect.NewResponse(response), nil
}

func isTransientGetPositionsFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrAccountNotConnected) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var toe *faultisol.TimeoutError
	if errors.As(err, &toe) {
		return true
	}
	if faultisol.IsTimeoutError(err) {
		return true
	}
	return containsErrorText(err.Error(), "timed out") || containsErrorText(err.Error(), "timeout")
}

func (s *TradingService) logPositionsDiagnostic(ctx context.Context, userID, accountID uuid.UUID, stage string, count int, elapsed time.Duration, cause error, extras ...map[string]interface{}) {
	if s.logSvc == nil {
		return
	}
	log := model.NewSystemOperationLog(userID, model.OperationTypeUpdate, "trading_positions", "get_positions_diagnostic")
	log.ResourceType = "mt_account"
	log.ResourceID = accountID
	log.DurationMs = elapsed.Milliseconds()
	nv := map[string]interface{}{
		"account_id":       accountID.String(),
		"stage":            stage,
		"positions_count":  count,
		"elapsed_ms":       elapsed.Milliseconds(),
	}
	for _, ex := range extras {
		for k, v := range ex {
			nv[k] = v
		}
	}
	log.NewValue = nv
	if cause != nil {
		log.Status = model.OperationStatusFailed
		log.ErrorMessage = cause.Error()
		nv := map[string]interface{}{
			"account_id":       accountID.String(),
			"stage":            stage,
			"positions_count":  count,
			"elapsed_ms":       elapsed.Milliseconds(),
			"error":            cause.Error(),
		}
		for _, ex := range extras {
			for k, v := range ex {
				nv[k] = v
			}
		}
		log.NewValue = nv
	} else {
		log.Status = model.OperationStatusSuccess
	}
	logCtx := ctx
	if logCtx == nil || logCtx.Err() != nil {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		logCtx = bg
	}
	if err := s.logSvc.LogOperation(logCtx, log); err != nil {
		logger.Warn("positions diagnostic log persist failed",
			zap.String("account_id", accountID.String()),
			zap.String("stage", stage),
			zap.Error(err))
	}
}

func (s *TradingService) positionsFromStreamSnapshot(accountID string) []*v1.Order {
	if s == nil || s.streamSvc == nil || accountID == "" {
		return nil
	}
	accountStream, ok := s.streamSvc.getAccountStream(accountID)
	if !ok || accountStream == nil {
		return nil
	}

	opened := accountStream.getOpenedOrdersSnapshot()
	profitSnap := accountStream.getProfitSnapshot()
	profitByTicket := make(map[int64]*v1.OrderProfitItem)
	if profitSnap != nil {
		for _, item := range profitSnap.GetOrders() {
			if item == nil || item.GetTicket() <= 0 {
				continue
			}
			profitByTicket[item.GetTicket()] = item
		}
	}
	if len(opened) == 0 && len(profitByTicket) == 0 {
		return nil
	}

	out := make([]*v1.Order, 0, len(opened)+len(profitByTicket))
	for _, ev := range opened {
		if ev == nil || ev.GetTicket() <= 0 {
			continue
		}
		profitItem := profitByTicket[ev.GetTicket()]
		o := &v1.Order{
			Ticket:      ev.GetTicket(),
			Symbol:      ev.GetSymbol(),
			Type:        ev.GetType(),
			Volume:      ev.GetVolume(),
			OpenPrice:   ev.GetOpenPrice(),
			ClosePrice:  ev.GetClosePrice(),
			StopLoss:    ev.GetStopLoss(),
			TakeProfit:  ev.GetTakeProfit(),
			Profit:      ev.GetProfit(),
			Swap:        ev.GetSwap(),
			Commission:  ev.GetCommission(),
			Comment:     ev.GetComment(),
			AccountId:   accountID,
		}
		if profitItem != nil {
			if p := profitItem.GetProfit(); p != 0 {
				o.Profit = p
			}
			if cp := profitItem.GetCurrentPrice(); cp != 0 {
				o.ClosePrice = cp
			}
			if v := profitItem.GetVolume(); v > 0 {
				o.Volume = v
			}
		}
		if ev.GetOpenTime() > 0 {
			o.OpenTime = timestamppb.New(time.Unix(ev.GetOpenTime(), 0))
		}
		if ev.GetCloseTime() > 0 {
			o.CloseTime = timestamppb.New(time.Unix(ev.GetCloseTime(), 0))
		}
		out = append(out, o)
	}
	seen := make(map[int64]struct{}, len(out))
	for _, o := range out {
		if o != nil && o.Ticket > 0 {
			seen[o.Ticket] = struct{}{}
		}
	}
	// If opened-order snapshot is empty or stale, fallback to profit snapshot orders.
	for ticket, p := range profitByTicket {
		if _, exists := seen[ticket]; exists {
			continue
		}
		out = append(out, &v1.Order{
			Ticket:     ticket,
			Symbol:     p.GetSymbol(),
			Type:       "buy",
			Volume:     p.GetVolume(),
			OpenPrice:  0,
			ClosePrice: p.GetCurrentPrice(),
			Profit:     p.GetProfit(),
			AccountId:  accountID,
			Comment:    "snapshot_fallback",
		})
	}
	return out
}

func containsErrorText(msg string, sub string) bool {
	if msg == "" || sub == "" {
		return false
	}
	// Case-insensitive contains without extra dependency.
	m := []byte(msg)
	s := []byte(sub)
	for i := range m {
		if m[i] >= 'A' && m[i] <= 'Z' {
			m[i] += 32
		}
	}
	for i := range s {
		if s[i] >= 'A' && s[i] <= 'Z' {
			s[i] += 32
		}
	}
	if len(s) > len(m) {
		return false
	}
	for i := 0; i <= len(m)-len(s); i++ {
		if string(m[i:i+len(s)]) == string(s) {
			return true
		}
	}
	return false
}

func (s *TradingService) GetPendingOrders(ctx context.Context, req *connect.Request[v1.GetPendingOrdersRequest]) (*connect.Response[v1.GetPendingOrdersResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	return connect.NewResponse(&v1.GetPendingOrdersResponse{
		Orders: []*v1.Order{},
	}), nil
}

func (s *TradingService) GetOrderHistory(ctx context.Context, req *connect.Request[v1.GetOrderHistoryRequest]) (*connect.Response[v1.GetOrderHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var from, to time.Time
	if req.Msg.From != "" {
		from, err = time.Parse(time.RFC3339, req.Msg.From)
		if err != nil {
			from, _ = time.Parse("2006-01-02", req.Msg.From)
		}
	} else {
		from = time.Now().AddDate(-10, 0, 0)
	}

	if req.Msg.To != "" {
		to, err = time.Parse(time.RFC3339, req.Msg.To)
		if err != nil {
			to, _ = time.Parse("2006-01-02", req.Msg.To)
		}
	} else {
		to = time.Now()
	}

	orders, err := s.tradingSvc.GetOrderHistory(ctx, uid, accountID, from, to)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetOrderHistoryResponse{
		Orders:   make([]*v1.Order, len(orders)),
		Total:    int32(len(orders)),
		Page:     req.Msg.Page,
		PageSize: req.Msg.PageSize,
	}

	for i, order := range orders {
		response.Orders[i] = convertHistoryOrderResponse(order, req.Msg.AccountId)
	}

	return connect.NewResponse(response), nil
}

func (s *TradingService) SyncOrderHistory(ctx context.Context, req *connect.Request[v1.SyncOrderHistoryRequest]) (*connect.Response[v1.SyncOrderHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	to := time.Now()
	from := to.AddDate(-1, 0, 0)

	count, err := s.tradingSvc.SyncOrderHistory(ctx, uid, accountID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrAccountNotConnected) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.SyncOrderHistoryResponse{SyncedRecords: int32(count)}), nil
}
