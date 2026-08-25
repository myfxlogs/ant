package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchSymbolParams: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	out := make([]*mthub.SymbolParam, 0, len(canonicals))
	for _, c := range canonicals {
		ctx2 := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.SymbolParams(ctx2, &pb.SymbolParamsRequest{Id: sid, Symbol: c})
		if err != nil {
			return nil, fmt.Errorf("mt5 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return nil, fmt.Errorf("mt5 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil {
			continue
		}
		si := r.GetSymbolInfo()
		sg := r.GetSymbolGroup()
		out = append(out, &mthub.SymbolParam{
			Canonical:    c,
			SymbolRaw:    c,
			Digits:       si.GetDigits(),
			TradeMode:    int32(sg.GetTradeMode()),
			StopLevel:    sg.GetSL(),
			PointValue:   decimal.NewFromFloat(si.GetTickValue()),
			ContractSize: decimal.NewFromFloat(si.GetContractSize()),
			LotSize:      decimal.NewFromFloat(si.GetContractSize()),
			LotStep:      decimal.NewFromFloat(sg.GetLotsStep()),
			LotMin:       decimal.NewFromFloat(sg.GetMinLots()),
			LotMax:       decimal.NewFromFloat(sg.GetMaxLots()),
			SpreadFloat:  si.GetSpread() > 0,
		})
	}
	return out, nil
}

// FetchPriceHistory fetches K-line bars via the broker PriceHistory RPC in quotes.go.
func (g *Gateway) FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*mthub.Bar, error) {
	bars, err := g.GetPriceHistory(ctx, "", symbol, period, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]*mthub.Bar, 0, len(bars))
	for _, b := range bars {
		out = append(out, &mthub.Bar{
			Time:   time.UnixMilli(b.OpenTsUnixMs),
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: decimal.NewFromFloat(b.Volume),
		})
	}
	return out, nil
}

// FetchAllSymbols returns all available symbol names from the broker.
func (g *Gateway) FetchAllSymbols(ctx context.Context) ([]string, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchAllSymbols: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx2 := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.SymbolList(ctx2, &pb.SymbolListRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 SymbolList: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 SymbolList: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt5 SubscribeOrderEvents: not connected")
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
			g.log.Error("mt5 order event recv panic", zap.Any("panic", r))
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
			g.log.Warn("mt5 order event subscribe", zap.Error(err), zap.Duration("backoff", backoff))
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
			g.log.Warn("mt5 order event recv error", zap.Error(err))
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
			EventType: upd.GetType().String(),
			Timestamp: time.Now(),
		}
		if o != nil {
			event.Ticket = o.GetTicket()
		}
		h(event)
	}
}

func truncSid(s string) string {
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s
}
