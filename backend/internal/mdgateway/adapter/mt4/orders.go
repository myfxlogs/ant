package mt4

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const orderTimeout = 30 * time.Second

func mt4Op(side mthub.Side, ot mthub.OrderType) pb.Op {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket:
		return pb.Op_Op_Buy
	case side == mthub.SideSell && ot == mthub.OrderMarket:
		return pb.Op_Op_Sell
	case side == mthub.SideBuy && ot == mthub.OrderLimit:
		return pb.Op_Op_BuyLimit
	case side == mthub.SideSell && ot == mthub.OrderLimit:
		return pb.Op_Op_SellLimit
	case side == mthub.SideBuy && ot == mthub.OrderStop:
		return pb.Op_Op_BuyStop
	case side == mthub.SideSell && ot == mthub.OrderStop:
		return pb.Op_Op_SellStop
	default:
		return pb.Op_Op_Buy
	}
}

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return 0, fmt.Errorf("mt4 PlaceOrder: not connected")
	}
	if g.breaker != nil && !g.breaker.Allow() {
		return 0, mthub.ErrCircuitOpen
	}
	op := mt4Op(req.Side, req.OrderType)
	price := req.Price.InexactFloat64()
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	resp, err := tc.OrderSend(callCtx, &pb.OrderSendRequest{
		Id: sid, Symbol: req.Canonical, Operation: op,
		Volume:     req.Volume.InexactFloat64(),
		Price:      price,
		Stoploss:   req.StopLoss.InexactFloat64(),
		Takeprofit: req.TakeProfit.InexactFloat64(),
	})
	if err != nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt4 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt4 OrderSend: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt4 OrderSend: nil result")
	}
	if g.breaker != nil {
		g.breaker.OnSuccess()
	}
	return int64(resp.GetResult().GetTicket()), nil
}

func (g *Gateway) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		g.log.Warn("mt4 CloseOrder: not connected", zap.Bool("hasCli", tc != nil), zap.Bool("hasSid", sid != ""))
		return fmt.Errorf("mt4 CloseOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	l := lots.InexactFloat64()
	g.log.Info("mt4 CloseOrder: sending", zap.Int64("ticket", ticket), zap.Float64("lots", l), zap.String("sid", truncSid(sid)))
	resp, err := tc.OrderClose(callCtx, &pb.OrderCloseRequest{Id: sid, Ticket: int32(ticket), Lots: l})
	if err != nil {
		g.log.Error("mt4 OrderClose: gRPC error", zap.Error(err))
		return fmt.Errorf("mt4 OrderClose: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt4 OrderClose: broker error", zap.Int32("code", int32(resp.GetError().GetCode())), zap.String("msg", resp.GetError().GetMessage()))
		return fmt.Errorf("mt4 OrderClose: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	g.log.Info("mt4 CloseOrder: success", zap.Int64("ticket", ticket))
	return nil
}

// DeleteOrder cancels a pending order using MT4 OrderDelete.
// MT4 has a dedicated OrderDelete RPC — OrderClose only works for open positions.
func (g *Gateway) DeleteOrder(ctx context.Context, ticket int64) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		g.log.Warn("mt4 DeleteOrder: not connected", zap.Bool("hasCli", tc != nil), zap.Bool("hasSid", sid != ""))
		return fmt.Errorf("mt4 DeleteOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	g.log.Info("mt4 DeleteOrder: sending", zap.Int64("ticket", ticket), zap.String("sid", truncSid(sid)))
	resp, err := tc.OrderDelete(callCtx, &pb.OrderDeleteRequest{Id: sid, Ticket: int32(ticket)})
	if err != nil {
		g.log.Error("mt4 OrderDelete: gRPC error", zap.Error(err))
		return fmt.Errorf("mt4 OrderDelete: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt4 OrderDelete: broker error", zap.Int32("code", int32(resp.GetError().GetCode())), zap.String("msg", resp.GetError().GetMessage()))
		return fmt.Errorf("mt4 OrderDelete: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	g.log.Info("mt4 DeleteOrder: success", zap.Int64("ticket", ticket))
	return nil
}

func (g *Gateway) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return fmt.Errorf("mt4 ModifyOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	resp, err := tc.OrderModify(callCtx, &pb.OrderModifyRequest{
		Id: sid, Ticket: int32(ticket),
		Stoploss: sl.InexactFloat64(), Takeprofit: tp.InexactFloat64(),
		Price: price.InexactFloat64(),
	})
	if err != nil {
		return fmt.Errorf("mt4 OrderModify: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("mt4 OrderModify: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 FetchSymbolParams: not connected")
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
			return nil, fmt.Errorf("mt4 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return nil, fmt.Errorf("mt4 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil {
			continue
		}
		si := r.GetSymbol()
		gp := r.GetGroupParams()
		param := &mthub.SymbolParam{
			Canonical:   c,
			SymbolRaw:   c,
			SpreadFloat: si.GetSpread() > 0,
		}
		if si != nil {
			param.Digits = si.GetDigits()
			param.StopLevel = si.GetStopsLevel()
			param.PointValue = decimal.NewFromFloat(si.GetPoint())
		}
		if gp != nil {
			param.LotMin = decimal.NewFromFloat(gp.GetMinLot())
			param.LotMax = decimal.NewFromFloat(gp.GetMaxLot())
			param.LotStep = decimal.NewFromFloat(gp.GetLotStep())
			param.TradeMode = gp.GetExecution()
		}
		// MT4 does not expose ContractSize (LotSize) via SymbolParams; default is 1.
		if param.LotSize.IsZero() {
			param.LotSize = decimal.NewFromInt(1)
		}
		out = append(out, param)
	}
	return out, nil
}

// FetchPriceHistory fetches K-line bars from the broker (MT4 QuoteHistory RPC).
// Delegates to GetPriceHistory to avoid duplicating the RPC call and auth logic.
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

// FetchAllSymbols returns all available symbol names from the broker (MT4 Symbols RPC).
func (g *Gateway) FetchAllSymbols(ctx context.Context) ([]string, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 FetchAllSymbols: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx2 := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.Symbols(ctx2, &pb.SymbolsRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt4 Symbols: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 Symbols: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt4 SubscribeOrderEvents: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := streamCli.OnOrderUpdate(ctx, &pb.OnOrderUpdateRequest{Id: sid})
	if err != nil {
		return fmt.Errorf("mt4 OnOrderUpdate: %w", err)
	}
	g.mu.Lock()
	if g.cancelHubOrderSub != nil {
		g.cancelHubOrderSub()
	}
	ctx, g.cancelHubOrderSub = context.WithCancel(ctx)
	g.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.log.Error("mt4 order event recv panic", zap.Any("panic", r))
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			msg, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				g.log.Warn("mt4 order event recv error", zap.Error(err))
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
	}()
	return nil
}

func truncSid(s string) string {
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s
}
