package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

// SubscribeHistory replays recent order events for reconnecting clients.
// Uses broker OrderHistory + OpenedOrders to provide catch-up state without
// requiring a full NATS JetStream event store. Each order is sent as an
// order_update event, followed by a position_snapshot of currently open orders.
func (s *StreamServer) SubscribeHistory(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeHistoryRequest],
	stream *connect.ServerStream[antv1.StreamEvent],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	accountIDs := req.Msg.AccountIds
	if len(accountIDs) == 0 {
		// No accounts requested — fetch all user accounts.
		var err error
		accountIDs, err = s.platform.GetUserAccountIDs(ctx, userID)
		if err != nil {
			s.log.Warn("SubscribeHistory: GetUserAccountIDs failed", zap.Error(err))
			return nil
		}
	}

	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > 200 {
		limit = 50 // default cap
	}

	since := time.Now().Add(-24 * time.Hour)
	if req.Msg.Since != nil {
		since = req.Msg.Since.AsTime()
	}

	to := time.Now()
	totalSent := 0
	const accountTimeout = 5 * time.Second

	for _, accountID := range accountIDs {
		if totalSent >= limit {
			break
		}

		// Per-account timeout prevents a dead/non-responsive broker from blocking
		// history replay for all subsequent accounts.
		acctCtx, cancel := context.WithTimeout(ctx, accountTimeout)
		orders, err := s.svc.OrderHistory(acctCtx, accountID, since, to)
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				s.log.Warn("SubscribeHistory: OrderHistory timed out",
					zap.String("account", accountID), zap.Duration("timeout", accountTimeout))
			} else {
				s.log.Warn("SubscribeHistory: OrderHistory failed",
					zap.String("account", accountID), zap.Error(err))
			}
			continue
		}

		for _, rec := range orders {
			if totalSent >= limit {
				break
			}
			eventType := "history"
			if rec.CloseTime.IsZero() {
				eventType = "open"
			}
			protoEv := orderRecordToUpdateEvent(rec, accountID, eventType, rec.Ticket)
			if err := stream.Send(&antv1.StreamEvent{
				Type:      "order_update",
				AccountId: accountID,
				Timestamp: timestamppb.Now(),
				Payload:   &antv1.StreamEvent_OrderUpdate{OrderUpdate: protoEv},
			}); err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("send history event: %w", err))
			}
			totalSent++
		}

		// Append current open positions as a position_snapshot for this account.
		acctCtx2, cancel2 := context.WithTimeout(ctx, accountTimeout)
		opened, err := s.svc.OpenedOrders(acctCtx2, accountID)
		cancel2()
		if err == nil && len(opened) > 0 {
			now := timestamppb.Now()
			positions := make([]*antv1.OrderUpdateEvent, 0, len(opened))
			for _, rec := range opened {
				positions = append(positions, orderRecordToUpdateEvent(rec, accountID, "open", rec.Ticket))
			}
			if err := stream.Send(&antv1.StreamEvent{
				Type:      "position_snapshot",
				AccountId: accountID,
				Timestamp: now,
				Payload: &antv1.StreamEvent_PositionSnapshot{
					PositionSnapshot: &antv1.PositionSnapshotEvent{
						AccountId: accountID,
						Positions: positions,
					},
				},
			}); err != nil {
				s.log.Warn("SubscribeHistory: send position_snapshot failed",
					zap.String("account", accountID), zap.Error(err))
			}
		}
	}

	s.log.Info("SubscribeHistory: replay complete",
		zap.String("user", userID),
		zap.Int("accounts", len(accountIDs)),
		zap.Int("events_sent", totalSent),
	)

	return nil
}
