package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
)

// SubscribeOrderUpdates streams order update events for a single account.
func (s *StreamServer) SubscribeOrderUpdates(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeOrderUpdatesRequest],
	stream *connect.ServerStream[antv1.OrderUpdateEvent],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	accountID := req.Msg.AccountId
	if accountID == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("account_id is required"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, accountID)
	if err != nil {
		s.log.Error("SubscribeOrderUpdates: UserOwnsAccount DB error", zap.String("account", accountID), zap.Error(err))
		return connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errors.New("account does not belong to user"))
	}

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
			if accountID != "" && ev.AccountID != accountID {
				continue
			}
			if ev.Order == nil {
				continue
			}
			protoEv := orderRecordToUpdateEvent(ev.Order, ev.AccountID, ev.EventType, ev.Ticket)
			if err := stream.Send(protoEv); err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("send order update event to single-account stream: %w", err))
			}
		}
	}
}

// SubscribeProfitUpdates streams profit/account-info updates for a single account.
func (s *StreamServer) SubscribeProfitUpdates(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeProfitUpdatesRequest],
	stream *connect.ServerStream[antv1.ProfitUpdateEvent],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	accountID := req.Msg.AccountId
	if accountID == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("account_id is required"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, accountID)
	if err != nil {
		s.log.Error("SubscribeProfitUpdates: UserOwnsAccount DB error", zap.String("account", accountID), zap.Error(err))
		return connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errors.New("account does not belong to user"))
	}

	ch, cancel := s.svc.SubscribeAccountProfit(ctx, accountID)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(profitEventToProto(ev)); err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("send profit update to single-account stream: %w", err))
			}
		}
	}
}

// SubscribeUserSummary streams aggregated user-level portfolio summary.
func (s *StreamServer) SubscribeUserSummary(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
	stream *connect.ServerStream[antv1.UserSummaryEvent],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	accountIDs, err := s.platform.GetUserAccountIDs(ctx, userID)
	if err != nil {
		s.log.Warn("GetUserAccountIDs failed in SubscribeUserSummary", zap.Error(err))
	}

	if ev := s.computeSummary(ctx, userID); ev != nil {
		if err := stream.Send(ev); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("send initial user summary: %w", err))
		}
	}

	if len(accountIDs) == 0 {
		return s.runSummaryKeepaliveOnly(ctx, stream)
	}

	profitCh, cancels := s.setupProfitForwarding(ctx, accountIDs)
	defer func() {
		for _, c := range cancels {
			c()
		}
	}()

	return s.runSummaryLoop(ctx, stream, userID, profitCh)
}

func (s *StreamServer) runSummaryKeepaliveOnly(ctx context.Context, stream *connect.ServerStream[antv1.UserSummaryEvent]) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := stream.Send(&antv1.UserSummaryEvent{}); err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("keepalive: %w", err))
			}
		}
	}
}

func (s *StreamServer) setupProfitForwarding(ctx context.Context, accountIDs []string) (<-chan *mthub.AccountProfitEvent, []func()) {
	profitCh := make(chan *mthub.AccountProfitEvent, len(accountIDs)*2)
	cancels := make([]func(), 0, len(accountIDs))
	for _, aid := range accountIDs {
		ch, cancel := s.svc.SubscribeAccountProfit(ctx, aid)
		cancels = append(cancels, cancel)
		go func(ch <-chan *mthub.AccountProfitEvent) {
			for ev := range ch {
				select {
				case profitCh <- ev:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}
	return profitCh, cancels
}

func (s *StreamServer) runSummaryLoop(ctx context.Context, stream *connect.ServerStream[antv1.UserSummaryEvent], userID string, profitCh <-chan *mthub.AccountProfitEvent) error {
	var lastSummary time.Time
	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-keepalive.C:
			if ev := s.computeSummary(ctx, userID); ev != nil {
				if err := stream.Send(ev); err != nil {
					return connect.NewError(connect.CodeInternal, err)
				}
				lastSummary = time.Now()
			}
		case _, ok := <-profitCh:
			if !ok {
				continue
			}
			if time.Since(lastSummary) < 5*time.Second {
				continue
			}
			if ev := s.computeSummary(ctx, userID); ev != nil {
				if err := stream.Send(ev); err != nil {
					return connect.NewError(connect.CodeInternal, fmt.Errorf("send recomputed user summary: %w", err))
				}
				lastSummary = time.Now()
			}
		}
	}
}

func (s *StreamServer) computeSummary(ctx context.Context, userID string) *antv1.UserSummaryEvent {
	summary, err := s.platform.GetUserAccountsSummary(ctx, userID)
	if err != nil {
		s.log.Warn("computeSummary: GetUserAccountsSummary failed", zap.String("userID", userID), zap.Error(err))
		return nil
	}
	return &antv1.UserSummaryEvent{
		TotalBalance:   summary.TotalBalance.String(),
		TotalEquity:    summary.TotalEquity.String(),
		TotalProfit:    summary.TotalProfit.String(),
		AccountCount:   summary.AccountCount,
		ConnectedCount: summary.ConnectedCount,
		UpdatedAt:      timestamppb.Now(),
	}
}

