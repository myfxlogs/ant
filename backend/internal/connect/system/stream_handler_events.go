package system

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// --- Channel forwarders (fan-in from per-account subscriptions) ---

type profitSub struct {
	accountID string
	ch        <-chan *mthub.AccountProfitEvent
	cancel    func()
}

type snapSub struct {
	accountID string
	ch        <-chan *mthub.PositionSnapshot
	cancel    func()
}

func (s *StreamServer) forwardProfitEvents(
	loopCtx context.Context,
	profitSubs []profitSub,
) chan *mthub.AccountProfitEvent {
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
	return profitCh
}

func (s *StreamServer) forwardSnapEvents(
	loopCtx context.Context,
	snapSubs []snapSub,
) chan *mthub.PositionSnapshot {
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
	return snapCh
}

type statusSub struct {
	accountID string
	ch        <-chan *mthub.AccountStatusEvent
	cancel    func()
}

func (s *StreamServer) forwardStatusEvents(
	loopCtx context.Context,
	statusSubs []statusSub,
) chan *mthub.AccountStatusEvent {
	statusCh := make(chan *mthub.AccountStatusEvent, 64)
	for _, ss := range statusSubs {
		go func(ch <-chan *mthub.AccountStatusEvent) {
			for ev := range ch {
				select {
				case statusCh <- ev:
				case <-loopCtx.Done():
					return
				}
			}
		}(ss.ch)
	}
	return statusCh
}

// --- Per-event-type handlers for the main SSE loop ---

func (s *StreamServer) handleOrderEvent(
	ev *mthub.OrderEvent,
	filterAll bool,
	accountSet map[string]bool,
	recentlyClosed map[string]map[int64]bool,
	sendEvent func(*antv1.StreamEvent) error,
) error {
	if !filterAll && !accountSet[ev.AccountID] {
		return nil
	}
	if ev.Order == nil {
		return nil
	}
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
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send order update event: %w", err))
	}
	return nil
}

func (s *StreamServer) handleProfitEvent(
	pev *mthub.AccountProfitEvent,
	sendEvent func(*antv1.StreamEvent) error,
) error {
	now := timestamppb.Now()
	if err := sendEvent(&antv1.StreamEvent{
		Type:      "profit_update",
		AccountId: pev.AccountID,
		Timestamp: now,
		Payload: &antv1.StreamEvent_ProfitUpdate{
			ProfitUpdate: profitEventToProto(pev),
		},
	}); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send profit update event: %w", err))
	}
	return nil
}

// handleSnapEvent processes a full position snapshot: account_status ping,
// close-diff detection vs previous snapshot, and full position_snapshot emit.
func (s *StreamServer) handleSnapEvent(
	snap *mthub.PositionSnapshot,
	filterAll bool,
	accountSet map[string]bool,
	snapKnownTickets map[string]map[int64]bool,
	snapCount map[string]int,
	recentlyClosed map[string]map[int64]bool,
	sendEvent func(*antv1.StreamEvent) error,
) error {
	if !filterAll && !accountSet[snap.AccountID] {
		return nil
	}
	now := timestamppb.Now()

	// 1. Emit account_status heartbeat.
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
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send account_status event: %w", err))
	}

	// 2. Periodic cleanup: reset known tickets every 100 snapshots.
	snapCount[snap.AccountID]++
	if snapCount[snap.AccountID] >= 100 {
		delete(snapKnownTickets, snap.AccountID)
		snapCount[snap.AccountID] = 0
	}

	// 3. Diff against previous snapshot to emit close events.
	currentTickets, err := s.emitSnapCloseDiff(snap, snapKnownTickets, recentlyClosed, now, sendEvent)
	if err != nil {
		return err
	}
	snapKnownTickets[snap.AccountID] = currentTickets
	delete(recentlyClosed, snap.AccountID)

	// 4. Build and emit full position_snapshot.
	return s.emitPositionSnapshot(snap, now, sendEvent)
}

// emitSnapCloseDiff detects positions that disappeared since the last snapshot
// and emits close events, skipping tickets already reported via the order channel.
func (s *StreamServer) emitSnapCloseDiff(
	snap *mthub.PositionSnapshot,
	snapKnownTickets map[string]map[int64]bool,
	recentlyClosed map[string]map[int64]bool,
	now *timestamppb.Timestamp,
	sendEvent func(*antv1.StreamEvent) error,
) (map[int64]bool, error) {
	currentTickets := make(map[int64]bool, len(snap.Positions))
	for _, pos := range snap.Positions {
		currentTickets[pos.Ticket] = true
	}
	prev, hasPrev := snapKnownTickets[snap.AccountID]
	if !hasPrev {
		return currentTickets, nil
	}
	for ticket := range prev {
		if currentTickets[ticket] {
			continue
		}
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
			return currentTickets, connect.NewError(connect.CodeInternal, fmt.Errorf("send order_update close event: %w", err))
		}
	}
	return currentTickets, nil
}

// emitPositionSnapshot builds and sends the full position list for a snapshot.
func (s *StreamServer) emitPositionSnapshot(
	snap *mthub.PositionSnapshot,
	now *timestamppb.Timestamp,
	sendEvent func(*antv1.StreamEvent) error,
) error {
	positions := make([]*antv1.OrderUpdateEvent, 0, len(snap.Positions))
	for _, pos := range snap.Positions {
		positions = append(positions, &antv1.OrderUpdateEvent{
			AccountId:  snap.AccountID,
			Ticket:     pos.Ticket,
			Symbol:     pos.Symbol,
			Type:       pos.Type,
			Volume:     pos.Volume.String(),
			OpenPrice:  pos.OpenPrice.String(),
			ClosePrice: pos.CurrentPrice.String(),
			Profit:     pos.Profit.String(),
			StopLoss:   pos.StopLoss.String(),
			TakeProfit: pos.TakeProfit.String(),
			Swap:       pos.Swap.String(),
			Commission: pos.Commission.String(),
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
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send position_snapshot event: %w", err))
	}
	return nil
}

// handleStatusEvent sends an account_status SSE event for a connection state change.
func (s *StreamServer) handleStatusEvent(
	ev *mthub.AccountStatusEvent,
	sendEvent func(*antv1.StreamEvent) error,
) error {
	if err := sendEvent(&antv1.StreamEvent{
		Type:      "account_status",
		AccountId: ev.AccountID,
		Timestamp: timestamppb.New(ev.Timestamp),
		Payload: &antv1.StreamEvent_AccountStatus{
			AccountStatus: &antv1.AccountStatusEvent{
				AccountId: ev.AccountID,
				Status:    ev.Status,
				Message:   ev.Message,
			},
		},
	}); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send account_status event: %w", err))
	}
	return nil
}
