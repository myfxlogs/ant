package system

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

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

	profitCh, snapCh, statusCh := s.initEventChannels(loopCtx, profitSubs, snapSubs, statusSubs)

	snapKnownTickets := make(map[string]map[int64]bool)
	snapCount := make(map[string]int)
	recentlyClosed := make(map[string]map[int64]bool)

	return s.runEventLoop(eventLoopConfig{
		ctx:          ctx,
		sendEvent:    sendEvent,
		orderCh:      orderCh,
		statusCh:     statusCh,
		profitCh:     profitCh,
		snapCh:       snapCh,
		filterAll:    filterAll,
		accountSet:   accountSet,
		snapKnown:    snapKnownTickets,
		snapCount:    snapCount,
		recentClosed: recentlyClosed,
	})
}

type eventLoopConfig struct {
	ctx          context.Context
	sendEvent    func(*antv1.StreamEvent) error
	orderCh      <-chan *mthub.OrderEvent
	statusCh     <-chan *mthub.AccountStatusEvent
	profitCh     <-chan *mthub.AccountProfitEvent
	snapCh       <-chan *mthub.PositionSnapshot
	filterAll    bool
	accountSet   map[string]bool
	snapKnown    map[string]map[int64]bool
	snapCount    map[string]int
	recentClosed map[string]map[int64]bool
}

func (s *StreamServer) runEventLoop(cfg eventLoopConfig) error {
	h := &eventLoopHandlers{s: s, filterAll: cfg.filterAll, accountSet: cfg.accountSet,
		snapKnownTickets: cfg.snapKnown, snapCount: cfg.snapCount, recentlyClosed: cfg.recentClosed, sendEvent: cfg.sendEvent}

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cfg.ctx.Done())},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(keepalive.C)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cfg.orderCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cfg.statusCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cfg.profitCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cfg.snapCh)},
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
			return true, nil
		}
		return false, h.handleOrder(val.Interface().(*mthub.OrderEvent))
	case 3:
		if !ok {
			cases[3].Chan = reflect.Value{}
			return false, nil
		}
		return false, h.handleStatus(val.Interface().(*mthub.AccountStatusEvent))
	case 4:
		if !ok {
			cases[4].Chan = reflect.Value{}
			return false, nil
		}
		return false, h.handleProfit(val.Interface().(*mthub.AccountProfitEvent))
	case 5:
		if !ok {
			cases[5].Chan = reflect.Value{}
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
