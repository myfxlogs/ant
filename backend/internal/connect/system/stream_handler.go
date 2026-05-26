package connect

import (
	"context"

	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
)

// StreamServer implements the ant.v1.StreamServiceHandler interface.
type StreamServer struct {
	svc *mthub.MtHubService
	pg  *pgxpool.Pool
	log *zap.Logger
}

var _ antv1c.StreamServiceHandler = (*StreamServer)(nil)

func NewStreamServer(svc *mthub.MtHubService, pg *pgxpool.Pool, log *zap.Logger) *StreamServer {
	return &StreamServer{svc: svc, pg: pg, log: log}
}

func decToFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

func orderRecordToUpdateEvent(rec *mthub.OrderRecord, accountID string, eventType string, ticket int64) *antv1.OrderUpdateEvent {
	return &antv1.OrderUpdateEvent{
		AccountId: accountID,
		Ticket:    ticket,
		Symbol: func() string {
			if rec.Canonical != "" {
				return rec.Canonical
			}
			return rec.SymbolRaw
		}(),
		Type:       orderSideTypeLabel(rec.Side, rec.OrderType),
		Volume:     decToFloat(rec.Volume),
		OpenPrice:  decToFloat(rec.OpenPrice),
		Profit:     decToFloat(rec.Profit),
		Action:     eventType,
		ClosePrice: decToFloat(rec.ClosePrice),
		Swap:       decToFloat(rec.Swap),
		Commission: decToFloat(rec.Commission),
		Comment:    rec.Comment,
		OpenTime:   rec.OpenTime.Unix(),
		CloseTime:  rec.CloseTime.Unix(),
	}
}

func orderSideTypeLabel(side mthub.Side, ot mthub.OrderType) string {
	prefix := "buy"
	if side == mthub.SideSell {
		prefix = "sell"
	}
	switch ot {
	case mthub.OrderMarket:
		return prefix
	case mthub.OrderLimit:
		return prefix + "_limit"
	case mthub.OrderStop:
		return prefix + "_stop"
	case mthub.OrderStopLimit:
		return prefix + "_stop_limit"
	default:
		return prefix
	}
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

	// Subscribe to order events from mthub.
	orderCh, orderCancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
	defer orderCancel()

	accountSet := make(map[string]bool)
	for _, id := range req.Msg.AccountIds {
		accountSet[id] = true
	}
	filterAll := len(accountSet) == 0

	// Find all accounts for this user and subscribe to profit events.
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

	accountIDs := loadUserAccountIDs(ctx, s.pg, userID)
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribeAccountProfit(ctx, aid)
		profitSubs = append(profitSubs, profitSub{accountID: aid, ch: ch, cancel: cancel})
	}

	if filterAll && len(profitSubs) == 0 {
		// No accounts — keep the stream alive with periodic keepalive.
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

	// Send initial profit + status snapshot from DB so the frontend
	// has data immediately, even before gateway profit events arrive.
	connectedIDs := s.sendInitialSnapshot(ctx, stream, userID, accountSet, filterAll)

	// Fetch positions once for connected accounts and stream as a single
	// position_snapshot per account.  After this, OnOrderUpdate (snapCh)
	// provides all subsequent real-time position data.
	for _, aid := range connectedIDs {
		rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		orders, err := s.svc.OpenedOrders(rpcCtx, aid)
		cancel()
		if err != nil {
			continue
		}
		now := timestamppb.Now()
		positions := make([]*antv1.OrderUpdateEvent, 0, len(orders))
		for _, rec := range orders {
			positions = append(positions, orderRecordToUpdateEvent(rec, aid, "open", rec.Ticket))
		}
		stream.Send(&antv1.StreamEvent{
			Type:      "position_snapshot",
			AccountId: aid,
			Timestamp: now,
			Payload: &antv1.StreamEvent_PositionSnapshot{
				PositionSnapshot: &antv1.PositionSnapshotEvent{
					AccountId: aid,
					Positions: positions,
				},
			},
		})
	}

	// Subscribe to position snapshots (full OpenedOrders from OnOrderUpdate stream).
	// These provide real-time position data with all fields in one event.
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

	// Multiplex order events + profit events + snapshots into the SSE stream.
	profitCh := make(chan *mthub.AccountProfitEvent, 64)
	for _, ps := range profitSubs {
		go func(ch <-chan *mthub.AccountProfitEvent) {
			for ev := range ch {
				select {
				case profitCh <- ev:
				case <-ctx.Done():
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
				case <-ctx.Done():
					return
				}
			}
		}(ss.ch)
	}

	// Track known tickets per account for close detection on full snapshots.
	snapKnownTickets := make(map[string]map[int64]bool)

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
			event := &antv1.StreamEvent{
				Type:      "order_update",
				AccountId: ev.AccountID,
				Timestamp: timestamppb.New(ev.Timestamp),
				Payload: &antv1.StreamEvent_OrderUpdate{
					OrderUpdate: orderRecordToUpdateEvent(ev.Order, ev.AccountID, ev.EventType, ev.Ticket),
				},
			}
			if err := stream.Send(event); err != nil {
				return fmt.Errorf("send order update event: %w", err)
			}
		case pev, ok := <-profitCh:
			if !ok {
				continue
			}
			now := timestamppb.Now()
			if err := stream.Send(&antv1.StreamEvent{
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
				snapCh = nil
				continue
			}
			if !filterAll && !accountSet[snap.AccountID] {
				continue
			}
			now := timestamppb.Now()

			// Send account_status.
			stream.Send(&antv1.StreamEvent{
				Type:      "account_status",
				AccountId: snap.AccountID,
				Timestamp: now,
				Payload: &antv1.StreamEvent_AccountStatus{
					AccountStatus: &antv1.AccountStatusEvent{
						AccountId: snap.AccountID,
						Status:    "connected",
					},
				},
			})

			// Detect closed positions (disappeared from snapshot).
			currentTickets := make(map[int64]bool, len(snap.Positions))
			for _, pos := range snap.Positions {
				currentTickets[pos.Ticket] = true
			}
			if prev, ok := snapKnownTickets[snap.AccountID]; ok {
				for ticket := range prev {
					if !currentTickets[ticket] {
						stream.Send(&antv1.StreamEvent{
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
						})
					}
				}
			}
			snapKnownTickets[snap.AccountID] = currentTickets

			// Send ALL positions as a SINGLE position_snapshot event — ticket,
			// symbol, lots, prices, profit all in one message. No per-position flicker.
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
			stream.Send(&antv1.StreamEvent{
				Type:      "position_snapshot",
				AccountId: snap.AccountID,
				Timestamp: now,
				Payload: &antv1.StreamEvent_PositionSnapshot{
					PositionSnapshot: &antv1.PositionSnapshotEvent{
						AccountId: snap.AccountID,
						Positions: positions,
					},
				},
			})
		}
	}
}

func profitEventToProto(pev *mthub.AccountProfitEvent) *antv1.ProfitUpdateEvent {
	orders := make([]*antv1.OrderProfitItem, 0, len(pev.Positions))
	for _, pos := range pev.Positions {
		orders = append(orders, &antv1.OrderProfitItem{
			Ticket:       pos.Ticket,
			Symbol:       pos.Symbol,
			Profit:       pos.Profit,
			Volume:       pos.Volume,
			CurrentPrice: pos.CurrentPrice,
		})
	}
	return &antv1.ProfitUpdateEvent{
		AccountId:     pev.AccountID,
		Balance:       pev.Balance,
		Credit:        pev.Credit,
		Equity:        pev.Equity,
		Profit:        pev.Profit,
		Margin:        pev.Margin,
		FreeMargin:    pev.FreeMargin,
		MarginLevel:   pev.MarginLevel,
		ProfitPercent: pev.ProfitPercent,
		Orders:        orders,
	}
}

// sendInitialSnapshot queries the DB for current account state and sends
// profit_update + account_status events to the stream.  Returns the list of
// connected account IDs so the caller can fetch positions asynchronously.
func (s *StreamServer) sendInitialSnapshot(
	ctx context.Context,
	stream *connect.ServerStream[antv1.StreamEvent],
	userID string,
	accountSet map[string]bool,
	filterAll bool,
) (connectedIDs []string) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, account_status, balance, equity, credit, margin, free_margin, margin_level
		FROM mt_accounts WHERE user_id = $1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	now := timestamppb.Now()
	for rows.Next() {
		var id, status string
		var balance, equity, credit, margin, freeMargin, marginLevel float64
		if err := rows.Scan(&id, &status, &balance, &equity, &credit, &margin, &freeMargin, &marginLevel); err != nil {
			continue
		}
		if !filterAll && !accountSet[id] {
			continue
		}

		profit := equity - balance
		var profitPercent float64
		if balance > 0 {
			profitPercent = profit / balance * 100
		}

		// Profit update.
		stream.Send(&antv1.StreamEvent{
			Type:      "profit_update",
			AccountId: id,
			Timestamp: now,
			Payload: &antv1.StreamEvent_ProfitUpdate{
				ProfitUpdate: &antv1.ProfitUpdateEvent{
					AccountId:     id,
					Balance:       balance,
					Equity:        equity,
					Profit:        profit,
					Credit:        credit,
					Margin:        margin,
					FreeMargin:    freeMargin,
					MarginLevel:   marginLevel,
					ProfitPercent: profitPercent,
				},
			},
		})

		// Account status.
		stream.Send(&antv1.StreamEvent{
			Type:      "account_status",
			AccountId: id,
			Timestamp: now,
			Payload: &antv1.StreamEvent_AccountStatus{
				AccountStatus: &antv1.AccountStatusEvent{
					AccountId: id,
					Status:    status,
				},
			},
		})

		if status == "connected" {
			connectedIDs = append(connectedIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		s.log.Warn("sendInitialSnapshot rows iteration error", zap.Error(err))
	}
	return connectedIDs
}

// loadUserAccountIDs queries mt_accounts for all account IDs belonging to a user.
func loadUserAccountIDs(ctx context.Context, pg *pgxpool.Pool, userID string) []string {
	rows, err := pg.Query(ctx, `SELECT id FROM mt_accounts WHERE user_id = $1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return ids
}

// SubscribeHistory replays historical events (bounded).
func (s *StreamServer) SubscribeHistory(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeHistoryRequest],
	stream *connect.ServerStream[antv1.StreamEvent],
) error {
	return nil
}

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
	if !userOwnsAccount(ctx, s.pg, userID, accountID) {
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

	// Find all accounts for this user and subscribe to profit events.
	accountIDs := loadUserAccountIDs(ctx, s.pg, userID)

	// Send initial summary from current DB state.
	if ev := computeSummary(ctx, s.pg, userID); ev != nil {
		if err := stream.Send(ev); err != nil {
			return fmt.Errorf("send initial user summary: %w", err)
		}
	}

	if len(accountIDs) == 0 {
		<-ctx.Done()
		return nil
	}

	// Subscribe to profit events for all accounts and recompute summary on each update.
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

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-profitCh:
			if !ok {
				continue
			}
			// Recompute summary from DB on each profit event.
			if ev := computeSummary(ctx, s.pg, userID); ev != nil {
				if err := stream.Send(ev); err != nil {
					return fmt.Errorf("send recomputed user summary: %w", err)
				}
			}
		}
	}
}

func computeSummary(ctx context.Context, pg *pgxpool.Pool, userID string) *antv1.UserSummaryEvent {
	rows, err := pg.Query(ctx, `
		SELECT balance, equity, account_status
		FROM mt_accounts WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var totalBalance, totalEquity, totalProfit float64
	var accountCount, connectedCount int32
	for rows.Next() {
		var balance, equity float64
		var status string
		if err := rows.Scan(&balance, &equity, &status); err != nil {
			continue
		}
		totalBalance += balance
		totalEquity += equity
		totalProfit += equity - balance
		accountCount++
		if status == "connected" {
			connectedCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil
	}

	return &antv1.UserSummaryEvent{
		TotalBalance:   totalBalance,
		TotalEquity:    totalEquity,
		TotalProfit:    totalProfit,
		AccountCount:   accountCount,
		ConnectedCount: connectedCount,
		UpdatedAt:      timestamppb.Now(),
	}
}

func userOwnsAccount(ctx context.Context, pg *pgxpool.Pool, userID, accountID string) bool {
	var exists bool
	err := pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1 AND user_id = $2)`,
		accountID, userID,
	).Scan(&exists)
	return err == nil && exists
}
