package system

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
	antdecimal "alphaforge/internal/pkg/decimal"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

// formatPrice delegates to the shared decimal utility.
func formatPrice(p decimal.Decimal) string { return antdecimal.FormatPrice(p) }

// StreamServer implements the ant.v1.StreamServiceHandler interface.
type StreamServer struct {
	svc            *mthub.MtHubService
	platform       *service.PlatformService
	marketDataRepo repository.MarketDataStore
	log            *zap.Logger
}

var _ antv1c.StreamServiceHandler = (*StreamServer)(nil)

func NewStreamServer(svc *mthub.MtHubService, platform *service.PlatformService, log *zap.Logger) *StreamServer {
	return &StreamServer{svc: svc, platform: platform, log: log}
}

// SubscribeEvents streams aggregated events (order updates, profit, status, bars) for given accounts.
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

	sendEvent := buildSendEvent(stream, s.log)

	orderCh, orderCancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
	defer orderCancel()

	accountSet := make(map[string]bool)
	for _, id := range req.Msg.AccountIds {
		accountSet[id] = true
	}
	filterAll := len(accountSet) == 0

	accountIDs, err := s.platform.GetUserAccountIDs(ctx, userID)
	if err != nil {
		s.log.Warn("GetUserAccountIDs failed in SubscribeEvents", zap.Error(err))
	}

	profitSubs := s.setupProfitSubscriptions(ctx, accountIDs, filterAll, accountSet)
	defer func() {
		for _, ps := range profitSubs {
			ps.cancel()
		}
	}()

	if filterAll && len(profitSubs) == 0 {
		return s.runKeepaliveOnly(ctx, sendEvent)
	}

	var connectedIDs []string
	if !isReconnect {
		connectedIDs = s.sendInitialSnapshot(ctx, stream, userID, accountSet, filterAll)
		s.sendInitialPositionSnapshots(ctx, stream, connectedIDs)
	}

	snapSubs := s.setupSnapSubscriptions(ctx, accountIDs, filterAll, accountSet)
	defer func() {
		for _, ss := range snapSubs {
			ss.cancel()
		}
	}()

	statusSubs := s.setupStatusSubscriptions(accountIDs, filterAll, accountSet)
	defer func() {
		for _, ss := range statusSubs {
			ss.cancel()
		}
	}()

	loopCtx, loopCancel := context.WithCancel(ctx)
	defer loopCancel()

	profitCh, snapCh, statusCh, barCh, barDropCh, barCancel := s.initEventChannels(loopCtx, profitSubs, snapSubs, statusSubs, accountIDs, filterAll, accountSet)
	defer barCancel()

	snapKnownTickets := make(map[string]map[int64]bool)
	snapCount := make(map[string]int)
	recentlyClosed := make(map[string]map[int64]bool)

	return s.runEventLoop(ctx, sendEvent, barCh, barDropCh, orderCh, statusCh, profitCh, snapCh,
		filterAll, accountSet, snapKnownTickets, snapCount, recentlyClosed)
}

func (s *StreamServer) runEventLoop(
	ctx context.Context,
	sendEvent func(*antv1.StreamEvent) error,
	barCh <-chan *mthub.BarUpdate,
	barDropCh <-chan *mthub.BarDropEvent,
	orderCh <-chan *mthub.OrderEvent,
	statusCh <-chan *mthub.AccountStatusEvent,
	profitCh <-chan *mthub.AccountProfitEvent,
	snapCh <-chan *mthub.PositionSnapshot,
	filterAll bool,
	accountSet map[string]bool,
	snapKnownTickets map[string]map[int64]bool,
	snapCount map[string]int,
	recentlyClosed map[string]map[int64]bool,
) error {
	h := &eventLoopHandlers{s: s, filterAll: filterAll, accountSet: accountSet,
		snapKnownTickets: snapKnownTickets, snapCount: snapCount, recentlyClosed: recentlyClosed, sendEvent: sendEvent}

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(keepalive.C)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(barCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(barDropCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orderCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(statusCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(profitCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(snapCh)},
	}

	for {
		chosen, val, ok := reflect.Select(cases)
		done, err := h.dispatch(chosen, val, ok, cases)
		if done {
			return err
		}
		if err != nil {
			return err
		}
	}
}

func (h *eventLoopHandlers) dispatch(chosen int, val reflect.Value, ok bool, cases []reflect.SelectCase) (done bool, err error) {
	switch chosen {
	case 0:
		return true, nil
	case 1:
		return false, h.sendEvent(&antv1.StreamEvent{Type: "ping"})
	case 2:
		if !ok {
			return false, nil
		}
		return false, h.handleBar(val.Interface().(*mthub.BarUpdate))
	case 3:
		if !ok {
			cases[3].Chan = reflect.Value{}
			return false, nil
		}
		return false, h.handleBarDrop(val.Interface().(*mthub.BarDropEvent))
	case 4:
		if !ok {
			return true, nil
		}
		return false, h.handleOrder(val.Interface().(*mthub.OrderEvent))
	case 5:
		if !ok {
			cases[5].Chan = reflect.Value{}
			return false, nil
		}
		return false, h.handleStatus(val.Interface().(*mthub.AccountStatusEvent))
	case 6:
		if !ok {
			cases[6].Chan = reflect.Value{}
			return false, nil
		}
		return false, h.handleProfit(val.Interface().(*mthub.AccountProfitEvent))
	case 7:
		if !ok {
			cases[7].Chan = reflect.Value{}
			return false, nil
		}
		return false, h.handleSnap(val.Interface().(*mthub.PositionSnapshot))
	}
	return false, nil
}

type eventLoopHandlers struct {
	s                *StreamServer
	filterAll        bool
	accountSet       map[string]bool
	snapKnownTickets map[string]map[int64]bool
	snapCount        map[string]int
	recentlyClosed   map[string]map[int64]bool
	sendEvent        func(*antv1.StreamEvent) error
}

func (h *eventLoopHandlers) handleBar(b *mthub.BarUpdate) error {
	return h.s.handleBarEvent(b, h.filterAll, h.accountSet, h.sendEvent)
}

func (h *eventLoopHandlers) handleBarDrop(drop *mthub.BarDropEvent) error {
	return handleBarDropEvent(drop, h.filterAll, h.accountSet, h.sendEvent)
}

func (h *eventLoopHandlers) handleOrder(ev *mthub.OrderEvent) error {
	return h.s.handleOrderEvent(ev, h.filterAll, h.accountSet, h.recentlyClosed, h.sendEvent)
}

func (h *eventLoopHandlers) handleStatus(sev *mthub.AccountStatusEvent) error {
	return h.s.handleStatusEvent(sev, h.sendEvent)
}

func (h *eventLoopHandlers) handleProfit(pev *mthub.AccountProfitEvent) error {
	return h.s.handleProfitEvent(pev, h.sendEvent)
}

func (h *eventLoopHandlers) handleSnap(snap *mthub.PositionSnapshot) error {
	return h.s.handleSnapEvent(snap, h.filterAll, h.accountSet,
		h.snapKnownTickets, h.snapCount, h.recentlyClosed, h.sendEvent)
}

func (s *StreamServer) runKeepaliveOnly(ctx context.Context, sendEvent func(*antv1.StreamEvent) error) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := sendEvent(&antv1.StreamEvent{Type: "ping"}); err != nil {
				return err
			}
		}
	}
}

func handleBarDropEvent(drop *mthub.BarDropEvent, filterAll bool, accountSet map[string]bool, sendEvent func(*antv1.StreamEvent) error) error {
	if !filterAll && !accountSet[drop.AccountID] {
		return nil
	}
	return sendEvent(&antv1.StreamEvent{
		Type:      "risk_alert",
		AccountId: drop.AccountID,
		Timestamp: timestamppb.Now(),
		Payload: &antv1.StreamEvent_RiskAlert{
			RiskAlert: &antv1.RiskAlertEvent{
				AccountId: drop.AccountID,
				AlertType: "bars_dropped",
				Message:   "Real-time bars are being dropped due to slow processing. Strategy execution may be delayed.",
				Value:     fmt.Sprintf("%d", drop.TotalDrops),
			},
		},
	})
}

// --- Bar subscription and event handling ---

func (s *StreamServer) forwardBarEvents(
	loopCtx context.Context,
	accountIDs []string,
	filterAll bool,
	accountSet map[string]bool,
) (chan *mthub.BarUpdate, <-chan *mthub.BarDropEvent, func()) {
	type barSub struct {
		ch     <-chan *mthub.BarUpdate
		cancel func()
	}
	barSubs := make([]barSub, 0, len(accountIDs))
	type dropSub struct {
		ch     <-chan *mthub.BarDropEvent
		cancel func()
	}
	dropSubs := make([]dropSub, 0, len(accountIDs))
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribeBarUpdates(aid)
		barSubs = append(barSubs, barSub{ch, cancel})
		dCh, dCancel := s.svc.SubscribeBarDrops(aid)
		dropSubs = append(dropSubs, dropSub{ch: dCh, cancel: dCancel})
	}
	barCh := make(chan *mthub.BarUpdate, 64)
	for _, bs := range barSubs {
		go func(ch <-chan *mthub.BarUpdate) {
			for ev := range ch {
				select {
				case barCh <- ev:
				case <-loopCtx.Done():
					return
				}
			}
		}(bs.ch)
	}
	barDropCh := make(chan *mthub.BarDropEvent, 4)
	for _, ds := range dropSubs {
		go func(ch <-chan *mthub.BarDropEvent) {
			for ev := range ch {
				select {
				case barDropCh <- ev:
				case <-loopCtx.Done():
					return
				}
			}
		}(ds.ch)
	}
	cancelAll := func() {
		for _, bs := range barSubs {
			bs.cancel()
		}
		for _, ds := range dropSubs {
			ds.cancel()
		}
	}
	return barCh, barDropCh, cancelAll
}

func (s *StreamServer) handleBarEvent(
	b *mthub.BarUpdate,
	filterAll bool,
	accountSet map[string]bool,
	sendEvent func(*antv1.StreamEvent) error,
) error {
	if !filterAll && !accountSet[b.AccountID] {
		return nil
	}
	t := time.UnixMilli(b.OpenTime)
	if err := sendEvent(&antv1.StreamEvent{
		Type:      "bar_update",
		AccountId: b.AccountID,
		Timestamp: timestamppb.New(t),
		Payload: &antv1.StreamEvent_BarUpdate{
			BarUpdate: &antv1.BarUpdateEvent{
				AccountId: b.AccountID,
				Symbol:    b.Symbol,
				Period:    b.Period,
				OpenTime:  timestamppb.New(t),
				Open:      formatPrice(b.Open),
				High:      formatPrice(b.High),
				Low:       formatPrice(b.Low),
				Close:     formatPrice(b.Close),
				Bid:       formatPrice(b.Bid),
				Ask:       formatPrice(b.Ask),
				Volume:    b.Volume,
				Closed:    b.Closed,
			},
		},
	}); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send bar_update event: %w", err))
	}
	return nil
}
