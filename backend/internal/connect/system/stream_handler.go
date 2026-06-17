package system

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// formatPrice formats a float64 price with dynamic decimal precision:
// - Price > 100:  3 digits (JPY pairs, e.g. 149.250)
// - Price > 1:    5 digits (standard forex, e.g. 1.12345)
// - Price <= 1:   6 digits (crypto or fractional assets)
func formatPrice(p float64) string {
	switch {
	case p > 100:
		return fmt.Sprintf("%.3f", p)
	case p > 1:
		return fmt.Sprintf("%.5f", p)
	default:
		return fmt.Sprintf("%.6f", p)
	}
}

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

	profitSubs, err := s.setupProfitSubscriptions(ctx, userID, accountIDs, filterAll, accountSet)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer func() {
		for _, ps := range profitSubs {
			ps.cancel()
		}
	}()

	if filterAll && len(profitSubs) == 0 {
		// No accounts — stream is fully idle. Send periodic pings to prevent
		// Cloudflare/proxy from closing the idle HTTP/2 stream (100s timeout).
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

	var connectedIDs []string
	if !isReconnect {
		connectedIDs = s.sendInitialSnapshot(ctx, stream, userID, accountSet, filterAll)
		s.sendInitialPositionSnapshots(ctx, stream, connectedIDs)
	}

	snapSubs, err := s.setupSnapSubscriptions(ctx, userID, accountIDs, filterAll, accountSet)
	if err != nil {
		s.log.Warn("snapSubs setup failed", zap.Error(err))
	}
	defer func() {
		for _, ss := range snapSubs {
			ss.cancel()
		}
	}()

	statusSubs, err := s.setupStatusSubscriptions(ctx, userID, accountIDs, filterAll, accountSet)
	if err != nil {
		s.log.Warn("statusSubs setup failed", zap.Error(err))
	}
	defer func() {
		for _, ss := range statusSubs {
			ss.cancel()
		}
	}()

	loopCtx, loopCancel := context.WithCancel(ctx)
	defer loopCancel()

	profitCh, snapCh, statusCh, barCh, barCancel := s.initEventChannels(loopCtx, profitSubs, snapSubs, statusSubs, accountIDs, filterAll, accountSet)
	defer barCancel()

	snapKnownTickets := make(map[string]map[int64]bool)
	snapCount := make(map[string]int)
	recentlyClosed := make(map[string]map[int64]bool)

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-keepalive.C:
			// Keepalive ping prevents Cloudflare/proxy from closing idle HTTP/2 streams.
			if err := sendEvent(&antv1.StreamEvent{Type: "ping"}); err != nil {
				return err
			}

		case b, ok := <-barCh:
			if !ok {
				continue
			}
			if err := s.handleBarEvent(b, filterAll, accountSet, sendEvent); err != nil {
				return err
			}

		case ev, ok := <-orderCh:
			if !ok {
				return nil
			}
			if err := s.handleOrderEvent(ev, filterAll, accountSet, recentlyClosed, sendEvent); err != nil {
				return err
			}

		case sev, ok := <-statusCh:
			if !ok {
				statusCh = nil
				continue
			}
			if err := s.handleStatusEvent(sev, sendEvent); err != nil {
				return err
			}

		case pev, ok := <-profitCh:
			if !ok {
				profitCh = nil
				continue
			}
			if err := s.handleProfitEvent(pev, sendEvent); err != nil {
				return err
			}

		case snap, ok := <-snapCh:
			if !ok {
				snapCh = nil
				continue
			}
			if err := s.handleSnapEvent(snap, filterAll, accountSet,
				snapKnownTickets, snapCount, recentlyClosed, sendEvent); err != nil {
				return err
			}
		}
	}
}

// --- Bar subscription and event handling ---

func (s *StreamServer) forwardBarEvents(
	loopCtx context.Context,
	accountIDs []string,
	filterAll bool,
	accountSet map[string]bool,
) (chan *mthub.BarUpdate, func()) {
	type barSub struct {
		ch     <-chan *mthub.BarUpdate
		cancel func()
	}
	barSubs := make([]barSub, 0, len(accountIDs))
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribeBarUpdates(aid)
		barSubs = append(barSubs, barSub{ch, cancel})
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
	cancelAll := func() {
		for _, bs := range barSubs {
			bs.cancel()
		}
	}
	return barCh, cancelAll
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
