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

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
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

	ch, cancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
	defer cancel()
	accountID := req.Msg.AccountId

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
				return fmt.Errorf("send order update event to single-account stream: %w", err)
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
				return fmt.Errorf("send profit update to single-account stream: %w", err)
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
			return fmt.Errorf("send initial user summary: %w", err)
		}
	}

	if len(accountIDs) == 0 {
		<-ctx.Done()
		return nil
	}

	profitCh := make(chan *mthub.AccountProfitEvent, len(accountIDs)*2)
	cancels := make([]func(), 0, len(accountIDs))
	defer func() {
		for _, c := range cancels {
			c()
		}
	}()

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

	var lastSummary time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-profitCh:
			if !ok {
				// Defensive: profitCh is never closed; goroutines only forward values.
				continue
			}
			// Throttle: recompute summary at most once every 5 seconds
			// to avoid flooding the DB with redundant aggregate queries.
			if time.Since(lastSummary) < 5*time.Second {
				continue
			}
			if ev := s.computeSummary(ctx, userID); ev != nil {
				if err := stream.Send(ev); err != nil {
					return fmt.Errorf("send recomputed user summary: %w", err)
				}
				lastSummary = time.Now()
			}
		}
	}
}

func (s *StreamServer) computeSummary(ctx context.Context, userID string) *antv1.UserSummaryEvent {
	summary, err := s.platform.GetUserAccountsSummary(ctx, userID)
	if err != nil {
		return nil
	}
	return &antv1.UserSummaryEvent{
		TotalBalance:   summary.TotalBalance,
		TotalEquity:    summary.TotalEquity,
		TotalProfit:    summary.TotalProfit,
		AccountCount:   summary.AccountCount,
		ConnectedCount: summary.ConnectedCount,
		UpdatedAt:      timestamppb.Now(),
	}
}

// SubscribeHistory replays historical events (bounded).
// TODO(H15): Deferred — requires event store persistence and replay
// infrastructure (NATS JetStream consumer) before it can serve historical
// event replays to newly connected clients.
func (s *StreamServer) SubscribeHistory(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeHistoryRequest],
	stream *connect.ServerStream[antv1.StreamEvent],
) error {
	return nil
}
