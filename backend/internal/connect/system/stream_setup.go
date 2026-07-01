package system

import (
	"context"
	"sync/atomic"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
)

// setupProfitSubscriptions subscribes to account profit updates for the user.
// Returns the list of active subscriptions (empty if no accounts match).
func (s *StreamServer) setupProfitSubscriptions(
	ctx context.Context,
	userID string,
	accountIDs []string,
	filterAll bool,
	accountSet map[string]bool,
) ([]profitSub, error) {
	var profitSubs []profitSub
	var err error
	accountIDs, err = s.platform.GetUserAccountIDs(ctx, userID)
	if err != nil {
		s.log.Warn("GetUserAccountIDs failed in SubscribeEvents", zap.Error(err))
		return nil, err
	}
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribeAccountProfit(ctx, aid)
		profitSubs = append(profitSubs, profitSub{accountID: aid, ch: ch, cancel: cancel})
	}
	return profitSubs, nil
}

// setupSnapSubscriptions subscribes to position snapshot updates for the user.
func (s *StreamServer) setupSnapSubscriptions(
	ctx context.Context,
	userID string,
	accountIDs []string,
	filterAll bool,
	accountSet map[string]bool,
) ([]snapSub, error) {
	var snapSubs []snapSub
	var err error
	accountIDs, err = s.platform.GetUserAccountIDs(ctx, userID)
	if err != nil {
		s.log.Warn("GetUserAccountIDs failed for snapSubs", zap.Error(err))
		return nil, err
	}
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribePositionSnapshots(ctx, aid)
		snapSubs = append(snapSubs, snapSub{accountID: aid, ch: ch, cancel: cancel})
	}
	return snapSubs, nil
}

// setupStatusSubscriptions subscribes to account status updates for the user.
func (s *StreamServer) setupStatusSubscriptions(
	ctx context.Context,
	userID string,
	accountIDs []string,
	filterAll bool,
	accountSet map[string]bool,
) ([]statusSub, error) {
	var statusSubs []statusSub
	var err error
	accountIDs, err = s.platform.GetUserAccountIDs(ctx, userID)
	if err != nil {
		s.log.Warn("GetUserAccountIDs failed for statusSubs", zap.Error(err))
		return nil, err
	}
	for _, aid := range accountIDs {
		if !filterAll && !accountSet[aid] {
			continue
		}
		ch, cancel := s.svc.SubscribeAccountStatus(aid)
		statusSubs = append(statusSubs, statusSub{accountID: aid, ch: ch, cancel: cancel})
	}
	return statusSubs, nil
}

// initEventChannels creates the forwarded event channels and returns them
// along with a combined cleanup function.
func (s *StreamServer) initEventChannels(
	loopCtx context.Context,
	profitSubs []profitSub,
	snapSubs []snapSub,
	statusSubs []statusSub,
	accountIDs []string,
	filterAll bool,
	accountSet map[string]bool,
) (profitCh <-chan *mthub.AccountProfitEvent, snapCh <-chan *mthub.PositionSnapshot, statusCh <-chan *mthub.AccountStatusEvent, barCh <-chan *mthub.BarUpdate, barDropCh <-chan *mthub.BarDropEvent, barCancel func()) {
	profitCh = s.forwardProfitEvents(loopCtx, profitSubs)
	snapCh = s.forwardSnapEvents(loopCtx, snapSubs)
	statusCh = s.forwardStatusEvents(loopCtx, statusSubs)
	barCh, barDropCh, barCancel = s.forwardBarEvents(loopCtx, accountIDs, filterAll, accountSet)
	return
}

// buildSendEvent creates a closure for sending stream events with auto-incrementing IDs.
func buildSendEvent(stream *connect.ServerStream[antv1.StreamEvent], log *zap.Logger) func(*antv1.StreamEvent) error {
	var eventID atomic.Int64
	return func(ev *antv1.StreamEvent) error {
		id := eventID.Add(1)
		if err := stream.Send(ev); err != nil {
			return err
		}
		log.Debug("sent SSE event", zap.Int64("event_id", id), zap.String("type", ev.GetType()))
		return nil
	}
}
