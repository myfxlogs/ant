package mt4

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// defaultQuoteRecvTimeout is the maximum interval to wait for a quote frame
// before declaring the stream silent and retrying. mtapi proxy can keep the
// gRPC connection alive (keepalive pings succeed) but stop pushing quote
// data — in this state, stream.Recv() blocks forever and the strategy
// starves for ticks. 45s is chosen because: (1) forex ticks arrive multiple
// times per second during market hours, so 45s of silence is definitively
// abnormal; (2) it's long enough to avoid false positives during brief
// broker-side gaps (e.g. between sessions); (3) it's short enough that a
// stuck stream is recovered in under a minute rather than blocking
// indefinitely.
const defaultQuoteRecvTimeout = 45 * time.Second

// quoteRecvTimeout returns the silence timeout for the quote stream. Tests
// override this via quoteRecvTimeoutOverride to keep test runtime low.
func (g *Gateway) quoteRecvTimeout() time.Duration {
	if g.quoteRecvTimeoutOverride > 0 {
		return g.quoteRecvTimeoutOverride
	}
	return defaultQuoteRecvTimeout
}

func (g *Gateway) Subscribe(ctx context.Context, syms []string, handler mdtick.TickHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt4: not connected")
	}
	// Persist symbols for re-subscription after reconnect.
	g.mu.Lock()
	g.subscribedSymbols = append(g.subscribedSymbols[:0], syms...)
	g.mu.Unlock()
	if sub != nil && len(syms) > 0 {
		subMd := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			subMd.Set("authorization", "Bearer "+tok)
		}
		// LIVE-PRICE-4: Subscribe per-symbol to avoid atomic batch failure
		// when one symbol doesn't exist on the broker. Non-existent symbols
		// are skipped (logged) instead of failing the entire subscription.
		subscribed := 0
		for _, sym := range syms {
			subCtx := metadata.NewOutgoingContext(ctx, subMd)
			resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: []string{sym}})
			if err != nil {
				g.log.Warn("mt4: subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
				continue
			}
			if e := resp.GetError(); e != nil && e.GetCode() != 0 {
				g.log.Warn("mt4: subscribe symbol rejected by mtapi", zap.String("sym", sym),
					zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
				continue
			}
			subscribed++
		}
		g.log.Info("mt4: subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
	}
	go g.recvLoop(ctx, handler)
	return nil
}

// AddSymbols subscribes to additional symbols on the existing MT4 session
// without starting a new quote stream. The existing recvLoop OnQuote stream
// will automatically deliver ticks for the newly added symbols.
func (g *Gateway) AddSymbols(ctx context.Context, symbols []string) error {
	g.mu.RLock()
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" {
		return fmt.Errorf("mt4 AddSymbols: not connected")
	}
	if len(symbols) == 0 {
		return nil
	}
	subMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		subMd.Set("authorization", "Bearer "+tok)
	}
	subCtx := metadata.NewOutgoingContext(ctx, subMd)
	resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: symbols})
	if err != nil {
		return fmt.Errorf("mt4 AddSymbols: %w", err)
	}
	if e := resp.GetError(); e != nil && e.GetCode() != 0 {
		return fmt.Errorf("mt4 AddSymbols: mtapi rejected: code=%d msg=%s", e.GetCode(), e.GetMessage())
	}
	// Persist for re-subscription after reconnect.
	g.mu.Lock()
	g.subscribedSymbols = append(g.subscribedSymbols, symbols...)
	g.mu.Unlock()
	g.log.Info("mt4: added symbols", zap.Strings("syms", symbols))
	return nil
}

func (g *Gateway) recvLoop(ctx context.Context, handler mdtick.TickHandler) {
	const maxBackoff = 5 * time.Minute
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// QUOTE-RECONNECT-LOOP S3: ensureConnected never returns an error
		// (it logs + sleeps + returns nil on Connect failure). The loop
		// only exits on ctx.Done(). The old code returned on
		// ensureConnected error, permanently killing the quote stream.
		_ = g.ensureConnected(ctx, &backoff, maxBackoff)
		if ctx.Err() != nil {
			return
		}

		g.reSubscribeSymbols(ctx)

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()
		if sc == nil || sid == "" {
			g.sleep(ctx, time.Second)
			continue
		}

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			md.Set("authorization", "Bearer "+tok)
		}
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnQuote(subCtx, &pb.OnQuoteRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt4 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt4: quote stream active")
		for {
			action := g.recvQuoteFrame(ctx, subCtx, stream, handler, &backoff, maxBackoff)
			if action == quoteActionContinue {
				continue
			}
			cancel()
			break
		}
	}
}

// quoteFrameAction represents the outcome of a single recvQuoteFrame call.
type quoteFrameAction int

const (
	quoteActionContinue quoteFrameAction = iota // got a tick, keep looping
	quoteActionBreak                            // silence timeout or error, break inner loop
)

// recvQuoteFrame receives one quote frame with a silence timeout. On silence
// timeout or Recv error, returns quoteActionBreak (caller cancels + retries
// the stream). On successful tick, dispatches via handler and returns
// quoteActionContinue.
// Do NOT call Disconnect() on silence — that tears down the shared connection
// including the profit stream. mtapi proxy can keep the gRPC connection alive
// (keepalive pings succeed) but stop pushing quote data. Without this silence
// timeout, stream.Recv() blocks forever and the strategy starves for ticks.
func (g *Gateway) recvQuoteFrame(ctx context.Context, subCtx context.Context,
	stream pb.Streams_OnQuoteClient, handler mdtick.TickHandler, backoff *time.Duration, maxBackoff time.Duration,
) quoteFrameAction {
	timeout := g.quoteRecvTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	type recvResult struct {
		resp *pb.OnQuoteReply
		err  error
	}
	ch := make(chan recvResult, 1)
	go func() {
		quote, err := stream.Recv()
		select {
		case ch <- recvResult{resp: quote, err: err}:
		case <-subCtx.Done():
		}
	}()

	select {
	case <-subCtx.Done():
		return quoteActionBreak
	case <-timer.C:
		g.log.Warn("mt4 quote stream silence timeout; retrying quote stream",
			zap.Duration("timeout", timeout), zap.String("account", g.cfg.AccountID))
		g.sleep(ctx, *backoff)
		*backoff = minDuration(*backoff*2, maxBackoff)
		return quoteActionBreak
	case r := <-ch:
		if r.err != nil {
			g.log.Warn("mt4 recv", zap.Error(r.err))
			g.handleStreamError(ctx, r.err, backoff)
			return quoteActionBreak
		}
		q := r.resp.GetResult()
		if q == nil {
			return quoteActionContinue
		}
		handler(&mdtick.Tick{
			UserID:        g.cfg.UserID,
			AccountID:     g.cfg.AccountID,
			Broker:        g.cfg.Broker,
			Platform:      "mt4",
			SymbolRaw:     q.GetSymbol(),
			Canonical:     "",
			TsUnixMs:      q.GetTime().AsTime().UnixMilli(),
			ArrivedUnixMs: Clk.Now().UTC().UnixMilli(),
			Bid:           decimal.NewFromFloat(q.GetBid()),
			Ask:           decimal.NewFromFloat(q.GetAsk()),
		})
		return quoteActionContinue
	}
}

func (g *Gateway) reSubscribeSymbols(ctx context.Context) {
	g.mu.RLock()
	syms := append([]string{}, g.subscribedSymbols...)
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" || len(syms) == 0 {
		return
	}
	subMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		subMd.Set("authorization", "Bearer "+tok)
	}
	// LIVE-PRICE-4: Per-symbol re-subscribe, skip failures.
	subscribed := 0
	for _, sym := range syms {
		subCtx := metadata.NewOutgoingContext(ctx, subMd)
		resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: []string{sym}})
		if err != nil {
			g.log.Warn("mt4: re-subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
			continue
		}
		if e := resp.GetError(); e != nil && e.GetCode() != 0 {
			g.log.Warn("mt4: re-subscribe symbol rejected by mtapi", zap.String("sym", sym),
				zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
			continue
		}
		subscribed++
	}
	g.log.Info("mt4: re-subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
}
