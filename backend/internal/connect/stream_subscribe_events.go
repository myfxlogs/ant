package connect

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/pkg/logger"
	"go.uber.org/zap"
)

func (s *StreamService) acquireUserStreamSlot(userID string) (func(), error) {
	const maxStreamsPerUser = 10

	s.activeMu.Lock()
	defer s.activeMu.Unlock()

	count, exists := s.activeUserStreams[userID]
	if exists && count >= maxStreamsPerUser {
		return nil, fmt.Errorf("达到最大并发流数量限制: %d", maxStreamsPerUser)
	}

	s.activeUserStreams[userID] = count + 1

	release := func() {
		s.activeMu.Lock()
		defer s.activeMu.Unlock()
		s.activeUserStreams[userID]--
		if s.activeUserStreams[userID] <= 0 {
			delete(s.activeUserStreams, userID)
		}
	}

	return release, nil
}

func (s *StreamService) authorizeAccountStream(ctx context.Context, userID uuid.UUID, accountIDStr string) (*model.MTAccount, error) {
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("无效的账户ID"))
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("账户不存在"))
	}

	if account.UserID != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("无权限访问该账户"))
	}

	if account.IsDisabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("账户已停用"))
	}

	return account, nil
}

func (s *StreamService) SubscribeEvents(ctx context.Context, req *connect.Request[v1.SubscribeEventsRequest], stream *connect.ServerStream[v1.StreamEvent]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	releaseSlot, err := s.acquireUserStreamSlot(userID.String())
	if err != nil {
		return connect.NewError(connect.CodeResourceExhausted, err)
	}
	defer releaseSlot()

	requested := req.Msg.GetAccountIds()
	// MT-official-like: empty accountIds means "subscribe to all enabled accounts for this user".
	// This lets clients keep a single stable stream while server reconciles the subscription set.
	manageAllEnabled := len(requested) == 0

	curByAccount := make(map[string]string)
	for k, v := range req.Msg.GetLastEventIds() {
		if k == "" {
			continue
		}
		curByAccount[k] = v
	}
	legacyCursor := req.Msg.GetLastEventId()

	// Track current per-account subscribers for this stream.
	subsByAccount := make(map[string]*StreamSubscriber)
	readCancelByAccount := make(map[string]context.CancelFunc)
	var subsMu sync.Mutex

	outCh := make(chan *v1.StreamEvent, 2048)

	enqueueSnapshot := func(accountIDStr string) {
		if accountIDStr == "" {
			return
		}
		accountStream, ok := s.getAccountStream(accountIDStr)
		if !ok || accountStream == nil {
			return
		}
		if p := accountStream.getProfitSnapshot(); p != nil {
			sev := &v1.StreamEvent{
				Type:      "profit_update",
				AccountId: accountIDStr,
				Timestamp: timestamppb.Now(),
				Payload:   &v1.StreamEvent_ProfitUpdate{ProfitUpdate: p},
			}
			select {
			case outCh <- sev:
			default:
			}
		}
		if led := accountStream.GetLedgerSnapshot(); led != nil {
			sev := &v1.StreamEvent{
				Type:      "ledger_entry",
				AccountId: accountIDStr,
				Timestamp: timestamppb.Now(),
				Payload:   &v1.StreamEvent_LedgerEntry{LedgerEntry: led},
			}
			select {
			case outCh <- sev:
			default:
			}
		}
		for _, pos := range accountStream.GetPositionsSnapshot() {
			if pos == nil {
				continue
			}
			sev := &v1.StreamEvent{
				Type:      "position_update",
				AccountId: accountIDStr,
				Timestamp: timestamppb.Now(),
				Payload:   &v1.StreamEvent_PositionUpdate{PositionUpdate: pos},
			}
			select {
			case outCh <- sev:
			default:
			}
		}
		for _, d := range accountStream.GetDealsSnapshot() {
			if d == nil {
				continue
			}
			sev := &v1.StreamEvent{
				Type:      "deal_update",
				AccountId: accountIDStr,
				Timestamp: timestamppb.Now(),
				Payload:   &v1.StreamEvent_DealUpdate{DealUpdate: d},
			}
			select {
			case outCh <- sev:
			default:
			}
		}
		for _, o := range accountStream.getOpenedOrdersSnapshot() {
			if o == nil {
				continue
			}
			sev := &v1.StreamEvent{
				Type:      "order_update",
				AccountId: accountIDStr,
				Timestamp: timestamppb.Now(),
				Payload:   &v1.StreamEvent_OrderUpdate{OrderUpdate: o},
			}
			select {
			case outCh <- sev:
			default:
			}
		}
	}

	enqueueSyncEvent := func(accountIDStr string, ev *v1.SyncEvent) {
		if accountIDStr == "" || ev == nil {
			return
		}
		// If this instance is leader, also persist the sync event to Redis Streams.
		if s.isLeaderForAccount(accountIDStr) {
			s.publishSyncEvent(accountIDStr, ev)
		}
		sev := &v1.StreamEvent{
			Type:      "sync",
			AccountId: accountIDStr,
			Timestamp: timestamppb.Now(),
			Payload:   &v1.StreamEvent_Sync{Sync: ev},
		}
		select {
		case outCh <- sev:
		default:
		}
	}

	registerAccount := func(accountIDStr string) {
		if accountIDStr == "" {
			return
		}
		subsMu.Lock()
		if _, exists := subsByAccount[accountIDStr]; exists {
			subsMu.Unlock()
			return
		}
		subsMu.Unlock()

		account, err := s.authorizeAccountStream(ctx, userID, accountIDStr)
		if err != nil {
			logger.Warn("authorize account failed", zap.String("account_id", accountIDStr), zap.Error(err))
			return
		}

		if err := s.ensureAccountStream(ctx, account); err != nil {
			logger.Error("创建账户流失败", zap.String("account_id", accountIDStr), zap.Error(err))
			return
		}

		sub := &StreamSubscriber{
			ID:         uuid.New().String(),
			AccountID:   accountIDStr,
			UserID:      userID.String(),
			EventCh:     nil,
			QuoteCh:     nil,
			OrderCh:     nil,
			ProfitCh:    nil,
			Cancel:      nil,
			NeedsProfit: false,
			NeedsOrder:  false,
		}

		if err := s.registerSubscriber(sub); err != nil {
			logger.Error("注册订阅者失败", zap.String("account_id", accountIDStr), zap.Error(err))
			return
		}

		subsMu.Lock()
		subsByAccount[accountIDStr] = sub
		subsMu.Unlock()

		cursor := curByAccount[accountIDStr]
		if cursor == "" {
			cursor = legacyCursor
		}
		startAfter := curByAccount[accountIDStr]
		if startAfter == "" {
			startAfter = legacyCursor
		}
		if startAfter == "" {
			syncID := uuid.New().String()
			enqueueSyncEvent(accountIDStr, &v1.SyncEvent{
				SyncId: syncID,
				Phase:  v1.SyncPhase_SYNC_PHASE_STARTED,
				Reason: "subscribe",
			})
			enqueueSnapshot(accountIDStr)
			lastID := ""
			if s.redisClient != nil {
				lastID = s.latestStreamEventID(ctx, accountIDStr)
			}
			enqueueSyncEvent(accountIDStr, &v1.SyncEvent{
				SyncId:      syncID,
				Phase:       v1.SyncPhase_SYNC_PHASE_COMPLETED,
				Reason:      "subscribe",
				LastEventId: lastID,
			})
			startAfter = lastID
		}

		// Start Redis Streams reader loop (one per account) feeding outCh.
		if s.redisClient != nil {
			subsMu.Lock()
			_, have := readCancelByAccount[accountIDStr]
			subsMu.Unlock()
			if !have {
				rctx, cancel := context.WithCancel(ctx)
				subsMu.Lock()
				readCancelByAccount[accountIDStr] = cancel
				subsMu.Unlock()
				if s.goroutineMgr != nil {
					_, _ = s.goroutineMgr.Spawn("xread-"+accountIDStr, func(goroutineCtx context.Context) error {
						_ = s.readAccountStreamLoop(rctx, accountIDStr, startAfter, outCh)
						return goroutineCtx.Err()
					})
				} else {
					go func() {
						_ = s.readAccountStreamLoop(rctx, accountIDStr, startAfter, outCh)
					}()
				}
			}
		}
	}

	unregisterAccount := func(accountIDStr string) {
		subsMu.Lock()
		sub, ok := subsByAccount[accountIDStr]
		if ok {
			delete(subsByAccount, accountIDStr)
		}
		cancel := readCancelByAccount[accountIDStr]
		if cancel != nil {
			delete(readCancelByAccount, accountIDStr)
		}
		subsMu.Unlock()
		if cancel != nil {
			cancel()
		}
		if ok && sub != nil {
			s.unregisterSubscriber(sub)
		}
	}

	// Initial registration.
	if manageAllEnabled {
		accounts, err := s.accountRepo.GetByUserID(ctx, userID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("获取账户列表失败: %w", err))
		}
		for _, a := range accounts {
			if a == nil || a.IsDisabled {
				continue
			}
			registerAccount(a.ID.String())
		}
	} else {
		for _, id := range requested {
			registerAccount(id)
		}
	}

	runDispatcher := func(runCtx context.Context) error {
		defer func() {
			subsMu.Lock()
			toClose := make([]*StreamSubscriber, 0, len(subsByAccount))
			toCancel := make([]context.CancelFunc, 0, len(readCancelByAccount))
			for _, sub := range subsByAccount {
				if sub != nil {
					toClose = append(toClose, sub)
				}
			}
			for _, cancel := range readCancelByAccount {
				if cancel != nil {
					toCancel = append(toCancel, cancel)
				}
			}
			subsByAccount = make(map[string]*StreamSubscriber)
			readCancelByAccount = make(map[string]context.CancelFunc)
			subsMu.Unlock()
			for _, cancel := range toCancel {
				cancel()
			}
			for _, sub := range toClose {
				s.unregisterSubscriber(sub)
			}
		}()

		// event-driven reconcile for managed-all-enabled mode
		var (
			changesCh   <-chan accountEnabledChange
			unsubscribe func()
		)
		if manageAllEnabled {
			_, ch, un := s.subscribeAccountEnabledChanges(userID.String())
			changesCh = ch
			unsubscribe = un
			defer unsubscribe()
		}

		// periodic fallback reconcile (only for managed-all-enabled mode)
		reconcileTick := time.NewTicker(30 * time.Second)
		defer reconcileTick.Stop()
		heartbeatTick := time.NewTicker(25 * time.Second)
		defer heartbeatTick.Stop()

		for {
			select {
			case <-runCtx.Done():
				return runCtx.Err()
			case chg, ok := <-changesCh:
				if manageAllEnabled && ok {
					if chg.enabled {
						registerAccount(chg.accountID)
					} else {
						unregisterAccount(chg.accountID)
					}
				}
			case <-reconcileTick.C:
				if manageAllEnabled {
					accounts, err := s.accountRepo.GetByUserID(runCtx, userID)
					if err == nil {
						wanted := make(map[string]struct{}, len(accounts))
						for _, a := range accounts {
							if a == nil || a.IsDisabled {
								continue
							}
							wanted[a.ID.String()] = struct{}{}
						}
						subsMu.Lock()
						current := make([]string, 0, len(subsByAccount))
						for id := range subsByAccount {
							current = append(current, id)
						}
						subsMu.Unlock()
						for _, id := range current {
							if _, ok := wanted[id]; !ok {
								unregisterAccount(id)
							}
						}
						for id := range wanted {
							registerAccount(id)
						}
					}
				}
			case <-heartbeatTick.C:
				if err := stream.Send(&v1.StreamEvent{Type: "heartbeat", Timestamp: timestamppb.Now()}); err != nil {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					return err
				}
			default:
				// fall through to polling
			}

			select {
			case <-runCtx.Done():
				return runCtx.Err()
			case ev := <-outCh:
				if ev == nil {
					continue
				}
				if err := stream.Send(ev); err != nil {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					return err
				}
			case <-time.After(200 * time.Millisecond):
			}
		}
	}

	return runDispatcher(ctx)
}

func (s *StreamService) SendSystemTick(ctx context.Context) {
	if s == nil {
		return
	}
	sev := &v1.StreamEvent{Type: "system_tick", Timestamp: timestamppb.Now()}
	s.streamMu.RLock()
	streams := make([]*AccountStream, 0, len(s.accountStreams))
	for _, as := range s.accountStreams {
		if as != nil {
			streams = append(streams, as)
		}
	}
	s.streamMu.RUnlock()
	for _, as := range streams {
		as.mu.RLock()
		for _, sub := range as.Subscribers {
			if sub == nil {
				continue
			}
			select {
			case sub.EventCh <- sev:
			default:
			}
		}
		as.mu.RUnlock()
	}
}
