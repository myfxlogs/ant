package system

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/service"
)

// StreamServer implements the ant.v1.StreamServiceHandler interface.
type StreamServer struct {
	svc      *mthub.MtHubService
	platform *service.PlatformService
	log      *zap.Logger
}

var _ antv1c.StreamServiceHandler = (*StreamServer)(nil)

func NewStreamServer(svc *mthub.MtHubService, platform *service.PlatformService, log *zap.Logger) *StreamServer {
	return &StreamServer{svc: svc, platform: platform, log: log}
}

// SubscribeEvents streams aggregated events (order updates, profit, status) for given accounts.
func (s *StreamServer) SubscribeEvents(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeEventsRequest],
	stream *connect.ServerStream[antv1.StreamEvent],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	lastEventID := req.Header().Get("Last-Event-ID")
	isReconnect := lastEventID != ""
	if isReconnect {
		s.log.Info("SSE reconnect with Last-Event-ID", zap.String("last_id", lastEventID))
	}

	var eventID atomic.Int64

	sendEvent := func(ev *antv1.StreamEvent) error {
		id := eventID.Add(1)
		if err := stream.Send(ev); err != nil {
			return err
		}
		s.log.Debug("sent SSE event", zap.Int64("event_id", id), zap.String("type", ev.GetType()))
		return nil
	}

	orderCh, orderCancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
	defer orderCancel()

	accountSet := make(map[string]bool)
	for _, id := range req.Msg.AccountIds {
		accountSet[id] = true
	}
	filterAll := len(accountSet) == 0

	type profitSub struct {
		accountID string
		ch        <-chan *mthub.AccountProfitEvent
		cancel    func()
	}
	var profitSubs []profitSub
	defer func() {
		for _, ps := range profitSubs {
			ps.cancel()
		}
	}()

	accountIDs, err := s.platform.GetUserAccountIDs(ctx, userID)
	if err != nil {
		s.log.Warn("GetUserAccountIDs failed in SubscribeEvents", zap.Error(err))
	}
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribeAccountProfit(ctx, aid)
		profitSubs = append(profitSubs, profitSub{accountID: aid, ch: ch, cancel: cancel})
	}

	if filterAll && len(profitSubs) == 0 {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}

	var connectedIDs []string
	if !isReconnect {
		connectedIDs = s.sendInitialSnapshot(ctx, stream, userID, accountSet, filterAll)
		s.sendInitialPositionSnapshots(ctx, stream, connectedIDs)
	}

	type snapSub struct {
		accountID string
		ch        <-chan *mthub.PositionSnapshot
		cancel    func()
	}
	var snapSubs []snapSub
	defer func() {
		for _, ss := range snapSubs {
			ss.cancel()
		}
	}()
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribePositionSnapshots(ctx, aid)
		snapSubs = append(snapSubs, snapSub{accountID: aid, ch: ch, cancel: cancel})
	}

	// H2: Create a cancellable context for forwarder goroutines so they
	// are unblocked when the main loop exits (e.g., on stream.Send error).
	loopCtx, loopCancel := context.WithCancel(ctx)
	defer loopCancel()

	profitCh := make(chan *mthub.AccountProfitEvent, 64)
	for _, ps := range profitSubs {
		go func(ch <-chan *mthub.AccountProfitEvent) {
			for ev := range ch {
				select {
				case profitCh <- ev:
				case <-loopCtx.Done():
					return
				}
			}
		}(ps.ch)
	}

	snapCh := make(chan *mthub.PositionSnapshot, 64)
	for _, ss := range snapSubs {
		go func(ch <-chan *mthub.PositionSnapshot) {
			for ev := range ch {
				select {
				case snapCh <- ev:
				case <-loopCtx.Done():
					return
				}
			}
		}(ss.ch)
	}

	snapKnownTickets := make(map[string]map[int64]bool)
	// recentlyClosed tracks tickets seen as "close" events from orderCh
	// so the snapshot diff does not emit duplicate close events.
	recentlyClosed := make(map[string]map[int64]bool)

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-orderCh:
			if !ok {
				return nil
			}
			if !filterAll && !accountSet[ev.AccountID] {
				continue
			}
			if ev.Order == nil {
				continue
			}
			// Track close events from orderCh so the snapshot diff
			// does not emit duplicate close events.
			if ev.EventType == "close" {
				if recentlyClosed[ev.AccountID] == nil {
					recentlyClosed[ev.AccountID] = make(map[int64]bool)
				}
				recentlyClosed[ev.AccountID][ev.Ticket] = true
			}
			event := &antv1.StreamEvent{
				Type:      "order_update",
				AccountId: ev.AccountID,
				Timestamp: timestamppb.New(ev.Timestamp),
				Payload: &antv1.StreamEvent_OrderUpdate{
					OrderUpdate: orderRecordToUpdateEvent(ev.Order, ev.AccountID, ev.EventType, ev.Ticket),
				},
			}
			if err := sendEvent(event); err != nil {
				return fmt.Errorf("send order update event: %w", err)
			}
		case pev, ok := <-profitCh:
			if !ok {
				// Defensive: profitCh is never closed (goroutines only forward values),
				// but a nil-or-closed channel would spin in select.
				profitCh = nil
				continue
			}
			now := timestamppb.Now()
			if err := sendEvent(&antv1.StreamEvent{
				Type:      "profit_update",
				AccountId: pev.AccountID,
				Timestamp: now,
				Payload: &antv1.StreamEvent_ProfitUpdate{
					ProfitUpdate: profitEventToProto(pev),
				},
			}); err != nil {
				return fmt.Errorf("send profit update event: %w", err)
			}
		case snap, ok := <-snapCh:
			if !ok {
				// Defensive: snapCh is never closed; nil assignment prevents
				// future selects on this channel branch.
				snapCh = nil
				continue
			}
			if !filterAll && !accountSet[snap.AccountID] {
				continue
			}
			now := timestamppb.Now()

			if err := sendEvent(&antv1.StreamEvent{
				Type:      "account_status",
				AccountId: snap.AccountID,
				Timestamp: now,
				Payload: &antv1.StreamEvent_AccountStatus{
					AccountStatus: &antv1.AccountStatusEvent{
						AccountId: snap.AccountID,
						Status:    "connected",
					},
				},
			}); err != nil {
				return fmt.Errorf("send account_status event: %w", err)
			}

			currentTickets := make(map[int64]bool, len(snap.Positions))
			for _, pos := range snap.Positions {
				currentTickets[pos.Ticket] = true
			}
			if prev, ok := snapKnownTickets[snap.AccountID]; ok {
				for ticket := range prev {
					if !currentTickets[ticket] {
						// Skip tickets already reported as "close" via orderCh
						// to avoid duplicate close events.
						if closedForAccount := recentlyClosed[snap.AccountID]; closedForAccount != nil {
							if closedForAccount[ticket] {
								continue
							}
						}
						if err := sendEvent(&antv1.StreamEvent{
							Type:      "order_update",
							AccountId: snap.AccountID,
							Timestamp: now,
							Payload: &antv1.StreamEvent_OrderUpdate{
								OrderUpdate: &antv1.OrderUpdateEvent{
									AccountId: snap.AccountID,
									Ticket:    ticket,
									Action:    "close",
								},
							},
						}); err != nil {
							return fmt.Errorf("send order_update close event: %w", err)
						}
					}
				}
			}
			snapKnownTickets[snap.AccountID] = currentTickets
			// Clear recently-closed tickets for this account after snapshot
			// processing to avoid unbounded growth.
			delete(recentlyClosed, snap.AccountID)

			positions := make([]*antv1.OrderUpdateEvent, 0, len(snap.Positions))
			for _, pos := range snap.Positions {
				positions = append(positions, &antv1.OrderUpdateEvent{
					AccountId:  snap.AccountID,
					Ticket:     pos.Ticket,
					Symbol:     pos.Symbol,
					Type:       pos.Type,
					Volume:     pos.Volume,
					OpenPrice:  pos.OpenPrice,
					ClosePrice: pos.CurrentPrice,
					Profit:     pos.Profit,
					StopLoss:   pos.StopLoss,
					TakeProfit: pos.TakeProfit,
					Swap:       pos.Swap,
					Commission: pos.Commission,
					Comment:    pos.Comment,
					Action:     "open",
					OpenTime:   pos.OpenTime,
				})
			}
			if err := sendEvent(&antv1.StreamEvent{
				Type:      "position_snapshot",
				AccountId: snap.AccountID,
				Timestamp: now,
				Payload: &antv1.StreamEvent_PositionSnapshot{
					PositionSnapshot: &antv1.PositionSnapshotEvent{
						AccountId: snap.AccountID,
						Positions: positions,
					},
				},
			}); err != nil {
				return fmt.Errorf("send position_snapshot event: %w", err)
			}
		}
	}
}
