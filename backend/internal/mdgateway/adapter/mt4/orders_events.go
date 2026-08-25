package mt4

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt4 SubscribeOrderEvents: not connected")
	}
	g.mu.Lock()
	if g.cancelHubOrderSub != nil {
		g.cancelHubOrderSub()
	}
	ctx, g.cancelHubOrderSub = context.WithCancel(ctx)
	g.mu.Unlock()
	go g.orderEventLoop(ctx, h)
	return nil
}

func (g *Gateway) orderEventLoop(ctx context.Context, h mthub.OrderEventHandler) {
	defer func() {
		if r := recover(); r != nil {
			g.log.Error("mt4 order event recv panic", zap.Any("panic", r))
		}
	}()
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		g.mu.RLock()
		streamCli := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()
		if streamCli == nil || sid == "" {
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, streamMaxBackoff)
			continue
		}
		md := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			md.Set("authorization", "Bearer "+tok)
		}
		subCtx, cancel := context.WithCancel(ctx)
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := streamCli.OnOrderUpdate(subCtx, &pb.OnOrderUpdateRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt4 order event subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}
		backoff = time.Second
		g.recvOrderUpdates(ctx, cancel, stream, h, &backoff)
	}
}

func (g *Gateway) recvOrderUpdates(ctx context.Context, cancel context.CancelFunc,
	stream grpc.ServerStreamingClient[pb.OnOrderUpdateReply], h mthub.OrderEventHandler, backoff *time.Duration,
) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := stream.Recv()
		if err != nil {
			g.log.Warn("mt4 order event recv error", zap.Error(err))
			g.handleStreamError(ctx, err, backoff)
			return
		}
		if h == nil || msg.GetResult() == nil || msg.GetResult().GetUpdate() == nil {
			continue
		}
		upd := msg.GetResult().GetUpdate()
		o := upd.GetOrder()
		event := &mthub.OrderEvent{
			AccountID: g.cfg.AccountID,
			EventType: upd.GetAction().String(),
			Timestamp: time.Now(),
		}
		if o != nil {
			event.Ticket = int64(o.GetTicket())
		}
		h(event)
	}
}
