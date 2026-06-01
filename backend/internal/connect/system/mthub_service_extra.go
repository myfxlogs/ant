package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
)

// PriceHistory returns K-line data from ClickHouse — the single source of truth.
// Broker symbol is resolved to canonical before querying.
func (s *MtHubServer) PriceHistory(ctx context.Context, req *connect.Request[antv1.PriceHistoryRequest]) (*connect.Response[antv1.PriceHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	period := m.Period
	if period == "" {
		period = "1h"
	}
	limit := m.Limit
	if limit <= 0 || limit > 1000 {
		limit = 300
	}

	var chFrom, chTo *time.Time
	if m.From != nil {
		t := m.From.AsTime()
		chFrom = &t
	}
	if m.To != nil {
		t := m.To.AsTime()
		chTo = &t
	}
	resolved := s.platform.ResolveSymbol(ctx, m.AccountId, m.Canonical)
	chBars, err := s.marketData.GetKlines(ctx, resolved, "", period, chFrom, chTo, limit)
	s.log.Info("PriceHistory",
		zap.String("input", m.Canonical),
		zap.String("resolved", resolved),
		zap.String("period", period),
		zap.String("account", m.AccountId),
		zap.Int("bars", len(chBars)),
		zap.Error(err),
	)
	if err != nil {
		s.log.Error("PriceHistory: get klines", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch price history"))
	}

	// CH data insufficient — fetch directly from broker for this period.
	// Async backfill will populate CH for subsequent timeframe switches.
	// Threshold: CH has <50 bars means stale/missing data; go to broker.
	if len(chBars) < 50 && m.AccountId != "" {
		chBars = s.brokerFallback(ctx, m.AccountId, resolved, period, int(limit))
	}

	out := make([]*antv1.OHLCV, 0, len(chBars))
	for _, b := range chBars {
		out = append(out, &antv1.OHLCV{
			OpenTime:  timestamppb.New(b.OpenTime()),
			CloseTime: timestamppb.New(b.OpenTime()),
			Open:      fmt.Sprintf("%.5f", b.Open),
			High:      fmt.Sprintf("%.5f", b.High),
			Low:       fmt.Sprintf("%.5f", b.Low),
			Close:     fmt.Sprintf("%.5f", b.Close),
			Volume:    b.Volume,
		})
	}
	return connect.NewResponse(&antv1.PriceHistoryResponse{Bars: out}), nil
}

// GetAccountStatus returns the current account connection state.
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
		LastTickAt: nil,
	}), nil
}

// SubscribeBars dynamically subscribes the gateway to a symbol's ticks and triggers
// a historical backfill from the broker into ClickHouse so the symbol has immediate
// K-line data.
func (s *MtHubServer) SubscribeBars(ctx context.Context, req *connect.Request[antv1.SubscribeBarsRequest]) (*connect.Response[antv1.SubscribeBarsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	if m.AccountId == "" || m.Symbol == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("account_id and symbol required"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, m.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	if err := s.svc.SubscribeSymbols(ctx, m.AccountId, []string{m.Symbol}); err != nil {
		s.log.Warn("SubscribeBars: failed to subscribe symbols",
			zap.String("account", m.AccountId), zap.String("symbol", m.Symbol), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to subscribe bars"))
	}

	// Backfill historical bars asynchronously — PriceHistory fallback handles
	// immediate needs for the requested period via direct broker fetch.
	go s.backfillKlines(m.AccountId, m.Symbol)

	return connect.NewResponse(&antv1.SubscribeBarsResponse{}), nil
}
