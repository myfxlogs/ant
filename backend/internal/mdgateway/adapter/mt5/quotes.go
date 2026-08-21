package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// defaultQuoteRecvTimeout is the maximum interval to wait for a quote frame
// before declaring the stream silent and retrying. See mt4/quotes.go for
// rationale.
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
		return fmt.Errorf("mt5: not connected")
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
		// LIVE-PRICE-4: Subscribe per-symbol to avoid atomic batch failure.
		subscribed := 0
		for _, sym := range syms {
			subCtx := metadata.NewOutgoingContext(ctx, subMd)
			resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: []string{sym}})
			if err != nil {
				g.log.Warn("mt5: subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
				continue
			}
			if e := resp.GetError(); e != nil && e.GetCode() != 0 {
				g.log.Warn("mt5: subscribe symbol rejected by mtapi", zap.String("sym", sym),
					zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
				continue
			}
			subscribed++
		}
		g.log.Info("mt5: subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
	}
	go g.recvLoop(ctx, handler)
	return nil
}

// AddSymbols subscribes to additional symbols on the existing MT5 session
// without starting a new quote stream. The existing recvLoop OnQuote stream
// will automatically deliver ticks for the newly added symbols.
func (g *Gateway) AddSymbols(ctx context.Context, symbols []string) error {
	g.mu.RLock()
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" {
		return fmt.Errorf("mt5 AddSymbols: not connected")
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
		return fmt.Errorf("mt5 AddSymbols: %w", err)
	}
	if e := resp.GetError(); e != nil && e.GetCode() != 0 {
		return fmt.Errorf("mt5 AddSymbols: mtapi rejected: code=%d msg=%s", e.GetCode(), e.GetMessage())
	}
	// Persist for re-subscription after reconnect.
	g.mu.Lock()
	g.subscribedSymbols = append(g.subscribedSymbols, symbols...)
	g.mu.Unlock()
	g.log.Info("mt5: added symbols", zap.Strings("syms", symbols))
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

		if err := g.ensureConnected(ctx, &backoff, maxBackoff); err != nil {
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
			g.log.Warn("mt5 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt5: quote stream active")
		for {
			timeout := g.quoteRecvTimeout()
			timer := time.NewTimer(timeout)

			type recvResult struct {
				resp *pb.OnQuoteReply
				err  error
			}
			ch := make(chan recvResult, 1)
			go func() {
				tick, err := stream.Recv()
				select {
				case ch <- recvResult{resp: tick, err: err}:
				case <-subCtx.Done():
				}
			}()

			select {
			case <-subCtx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				cancel()
				goto mt5QuoteLoopEnd
			case <-timer.C:
				g.log.Warn("mt5 quote stream silence timeout; retrying quote stream",
					zap.Duration("timeout", timeout), zap.String("account", g.cfg.AccountID))
				cancel()
				// Do NOT call Disconnect() — see mt4/quotes.go for rationale.
				g.sleep(ctx, backoff)
				backoff = minDuration(backoff*2, maxBackoff)
				goto mt5QuoteLoopEnd
			case r := <-ch:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				if r.err != nil {
					g.log.Warn("mt5 recv", zap.Error(r.err))
					cancel()
					g.handleStreamError(ctx, r.err, &backoff)
					goto mt5QuoteLoopEnd
				}
				q := r.resp.GetResult()
				if q == nil {
					continue
				}
				handler(&mdtick.Tick{
					UserID:        g.cfg.UserID,
					AccountID:     g.cfg.AccountID,
					Broker:        g.cfg.Broker,
					Platform:      "mt5",
					SymbolRaw:     q.GetSymbol(),
					Canonical:     "",
					TsUnixMs:      q.GetTime().AsTime().UnixMilli(),
					ArrivedUnixMs: Clk.Now().UTC().UnixMilli(),
					Bid:           decimal.NewFromFloat(q.GetBid()),
					Ask:           decimal.NewFromFloat(q.GetAsk()),
					BidVolume:     float64(q.GetVolume()),
				})
			}
		}
	mt5QuoteLoopEnd:
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
			g.log.Warn("mt5: re-subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
			continue
		}
		if e := resp.GetError(); e != nil && e.GetCode() != 0 {
			g.log.Warn("mt5: re-subscribe symbol rejected by mtapi", zap.String("sym", sym),
				zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
			continue
		}
		subscribed++
	}
	g.log.Info("mt5: re-subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
}
