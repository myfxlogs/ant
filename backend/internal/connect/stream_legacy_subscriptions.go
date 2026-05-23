package connect

import (
"context"
"errors"
"fmt"

"connectrpc.com/connect"
"github.com/google/uuid"

v1 "anttrader/gen/proto"
"anttrader/pkg/logger"

"go.uber.org/zap"
)

// CloseDisabledAccountStream 关闭已禁用账户的流
func (s *StreamService) CloseDisabledAccountStream(accountID uuid.UUID) {
	accountIDStr := accountID.String()
	s.closeAccountStream(accountIDStr, "disabled")
}

func (s *StreamService) SubscribeQuotes(ctx context.Context, req *connect.Request[v1.SubscribeQuotesRequest], stream *connect.ServerStream[v1.Quote]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	accountIDStr := req.Msg.GetAccountId()
	account, err := s.authorizeAccountStream(ctx, userID, accountIDStr)
	if err != nil {
		return err
	}

	err = s.ensureAccountStream(ctx, account)
	if err != nil {
		return err
	}

	subscriber := &StreamSubscriber{
		ID:        uuid.New().String(),
		AccountID: accountIDStr,
		UserID:    userID.String(),
		EventCh:   make(chan *v1.StreamEvent, 100),
		QuoteCh:   make(chan *v1.Quote, 100),
		OrderCh:   make(chan *v1.OrderUpdateEvent, 100),
		ProfitCh:  make(chan *v1.ProfitUpdateEvent, 1),
		Cancel:    nil,
		Symbols:   req.Msg.GetSymbols(),
		NeedsProfit: true,
		NeedsOrder:  false,
	}

	err = s.registerSubscriber(subscriber)
	if err != nil {
		return err
	}

	defer func() {
		s.unregisterSubscriber(subscriber)
	}()

	_, err = s.goroutineMgr.Spawn("quote-stream", func(goroutineCtx context.Context) error {
		for {
			select {
			case <-goroutineCtx.Done():
				return goroutineCtx.Err()
			case <-ctx.Done():
				return ctx.Err()
			case quote := <-subscriber.QuoteCh:
				if err := stream.Send(quote); err != nil {
					logger.Error("发送报价失败", zap.Error(err))
					return err
				}
			}
		}
	})

	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start quote stream: %w", err))
	}

	<-ctx.Done()
	return nil
}

func (s *StreamService) SubscribeOrderUpdates(ctx context.Context, req *connect.Request[v1.SubscribeOrderUpdatesRequest], stream *connect.ServerStream[v1.OrderUpdateEvent]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	accountIDStr := req.Msg.GetAccountId()
	account, err := s.authorizeAccountStream(ctx, userID, accountIDStr)
	if err != nil {
		return err
	}

	err = s.ensureAccountStream(ctx, account)
	if err != nil {
		return err
	}

	// MT-official-like: legacy API is a thin adapter over the same AccountStream snapshots/deltas.
	// Do NOT register a StreamSubscriber (avoids inflating total_subscribers).
	if s.connManager != nil {
		isLeader := true
		if s.sessionLeader != nil {
			isLeader = s.isLeaderForAccount(accountIDStr)
		}
		if isLeader {
			if uid, err := uuid.Parse(accountIDStr); err == nil {
				s.connManager.AddSubscription(uid, "legacy_order")
				defer s.connManager.RemoveSubscription(uid, "legacy_order")
			}
		}
	}

	accountStream, ok := s.getAccountStream(accountIDStr)
	if !ok || accountStream == nil {
		return connect.NewError(connect.CodeNotFound, errors.New("账户流不存在"))
	}

	// Send snapshot first.
	for _, o := range accountStream.getOpenedOrdersSnapshot() {
		if o == nil {
			continue
		}
		if err := stream.Send(o); err != nil {
			logger.Error("发送订单快照失败", zap.Error(err))
			return err
		}
	}

	// Then follow deltas via internal notify.
	lastVer := uint64(0)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-accountStream.orderNotifyCh():
			ev, ver := accountStream.getLastOrderDelta()
			if ver == 0 || ver == lastVer || ev == nil {
				continue
			}
			if err := stream.Send(ev); err != nil {
				logger.Error("发送订单更新失败", zap.Error(err))
				return err
			}
			lastVer = ver
		}
	}
}

func (s *StreamService) SubscribeProfitUpdates(ctx context.Context, req *connect.Request[v1.SubscribeProfitUpdatesRequest], stream *connect.ServerStream[v1.ProfitUpdateEvent]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	accountIDStr := req.Msg.GetAccountId()
	account, err := s.authorizeAccountStream(ctx, userID, accountIDStr)
	if err != nil {
		return err
	}

	err = s.ensureAccountStream(ctx, account)
	if err != nil {
		return err
	}

	// MT-official-like: legacy API is a thin adapter over the same AccountStream snapshots.
	// Do NOT register a StreamSubscriber (avoids inflating total_subscribers).
	if s.connManager != nil {
		isLeader := true
		if s.sessionLeader != nil {
			isLeader = s.isLeaderForAccount(accountIDStr)
		}
		if isLeader {
			if uid, err := uuid.Parse(accountIDStr); err == nil {
				s.connManager.AddSubscription(uid, "legacy_profit")
				defer s.connManager.RemoveSubscription(uid, "legacy_profit")
			}
		}
	}

	accountStream, ok := s.getAccountStream(accountIDStr)
	if !ok || accountStream == nil {
		return connect.NewError(connect.CodeNotFound, errors.New("账户流不存在"))
	}

	// Send latest snapshot immediately.
	if profit := accountStream.getProfitSnapshot(); profit != nil {
		if err := stream.Send(profit); err != nil {
			logger.Error("发送利润快照失败", zap.Error(err))
			return err
		}
	}

	// Then follow updates via internal notify.
	lastVer := uint64(0)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-accountStream.profitNotifyCh():
			p := accountStream.getProfitSnapshot()
			ver := accountStream.profitVer.Load()
			if ver == 0 || ver == lastVer || p == nil {
				continue
			}
			if err := stream.Send(p); err != nil {
				logger.Error("发送利润更新失败", zap.Error(err))
				return err
			}
			lastVer = ver
		}
	}
}
