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

// mt4OpToSideAndType maps a broker-stated Op back to side/orderType.
// Inverse of mt4Op — PlaceOrder must populate rec.Side/rec.OrderType from
// the reply (never echo request values; zero values would misreport a
// sell-limit as a buy-market record downstream).
func mt4OpToSideAndType(op pb.Op) (mthub.Side, mthub.OrderType) {
	side := mthub.SideBuy
	ot := mthub.OrderMarket
	switch op {
	case pb.Op_Op_Sell:
		side = mthub.SideSell
	case pb.Op_Op_BuyLimit:
		ot = mthub.OrderLimit
	case pb.Op_Op_SellLimit:
		side, ot = mthub.SideSell, mthub.OrderLimit
	case pb.Op_Op_BuyStop:
		ot = mthub.OrderStop
	case pb.Op_Op_SellStop:
		side, ot = mthub.SideSell, mthub.OrderStop
	case pb.Op_Op_Balance:
		ot = mthub.OrderBalance
	case pb.Op_Op_Credit:
		ot = mthub.OrderCredit
	}
	return side, ot
}

func mt4Op(side mthub.Side, ot mthub.OrderType) (pb.Op, error) {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket:
		return pb.Op_Op_Buy, nil
	case side == mthub.SideSell && ot == mthub.OrderMarket:
		return pb.Op_Op_Sell, nil
	case side == mthub.SideBuy && ot == mthub.OrderLimit:
		return pb.Op_Op_BuyLimit, nil
	case side == mthub.SideSell && ot == mthub.OrderLimit:
		return pb.Op_Op_SellLimit, nil
	case side == mthub.SideBuy && ot == mthub.OrderStop:
		return pb.Op_Op_BuyStop, nil
	case side == mthub.SideSell && ot == mthub.OrderStop:
		return pb.Op_Op_SellStop, nil
	default:
		return 0, fmt.Errorf("mt4 unsupported order type: side=%d orderType=%d", side, ot)
	}
}

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return nil, fmt.Errorf("mt4 PlaceOrder: not connected")
	}
	if g.breaker != nil && !g.breaker.Allow() {
		return nil, mthub.ErrCircuitOpen
	}
	op, err := mt4Op(req.Side, req.OrderType)
	if err != nil {
		return nil, fmt.Errorf("mt4 PlaceOrder: %w", err)
	}
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
		Comment:    req.Comment,
		Slippage:   req.Deviation,
		Magic:      req.Magic,
	})
	if err != nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return nil, fmt.Errorf("mt4 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return nil, &mthub.BrokerRejectError{Op: "mt4 OrderSend", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
	}
	o := resp.GetResult()
	if o == nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return nil, fmt.Errorf("mt4 OrderSend: nil result")
	}
	if g.breaker != nil {
		g.breaker.OnSuccess()
	}
	rec := &mthub.OrderRecord{
		// mtapi OrderSend returns the full Order — map the broker receipt
		// verbatim. Fields absent from the response stay zero (= unknown);
		// never echo request values (VM-LIVE-PARITY-F1).
		Ticket:     int64(o.GetTicket()),
		OpenPrice:  decimal.NewFromFloat(o.GetOpenPrice()),
		Volume:     decimal.NewFromFloat(o.GetLots()),
		StopLoss:   decimal.NewFromFloat(o.GetStopLoss()),
		TakeProfit: decimal.NewFromFloat(o.GetTakeProfit()),
		Comment:    o.GetComment(),
		Magic:      o.GetMagicNumber(),
		AccountID:  req.AccountID,
		Canonical:  req.Canonical,
		SymbolRaw:  req.Canonical,
	}
	if ot := o.GetOpenTime(); ot != nil {
		rec.OpenTime = ot.AsTime()
	}
	// Side/OrderType come from the broker's stated order type, not the request
	// — zero values would inject a buy-market record for a sell-limit fill.
	rec.Side, rec.OrderType = mt4OpToSideAndType(o.GetType())
	// State must be assigned explicitly: OrderStatePending is the zero value,
	// so leaving it unset would report a market fill as pending.
	if o.GetType() == pb.Op_Op_Buy || o.GetType() == pb.Op_Op_Sell {
		rec.State = mthub.OrderStateOpen
	} else {
		rec.State = mthub.OrderStatePending
	}
	return rec, nil
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
		return &mthub.BrokerRejectError{Op: "mt4 OrderClose", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
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
		return &mthub.BrokerRejectError{Op: "mt4 OrderDelete", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
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
		return &mthub.BrokerRejectError{Op: "mt4 OrderModify", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
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
		ex := si.GetEx()
		param := &mthub.SymbolParam{
			Canonical:   c,
			SymbolRaw:   c,
			SpreadFloat: si.GetSpread() > 0,
		}
		if si != nil {
			param.Digits = si.GetDigits()
			// VM-LIVE-PARITY-F2: per-field flat→Ex fallback — flat zero means
			// unknown, so consult the rich SymbolInfoEx before giving up. Every
			// field resolves independently; a true zero stays zero.
			param.PointValue = decimal.NewFromFloat(si.GetPoint())
			if param.PointValue.IsZero() {
				param.PointValue = decimal.NewFromFloat(ex.GetPoint())
			}
			param.ContractSize = decimal.NewFromFloat(si.GetContractSize())
			if param.ContractSize.IsZero() {
				param.ContractSize = decimal.NewFromFloat(ex.GetContractSize())
			}
			param.LotSize = param.ContractSize
			param.StopLevel = si.GetStopsLevel()
			if param.StopLevel == 0 {
				param.StopLevel = ex.GetStopsLevel()
			}
			param.TickValue = decimal.NewFromFloat(ex.GetTickValue())
			param.TickSize = decimal.NewFromFloat(ex.GetTickSize())
			param.SwapLong = decimal.NewFromFloat(ex.GetSwapLong())
			param.SwapShort = decimal.NewFromFloat(ex.GetSwapShort())
		}
		if gp != nil {
			param.LotMin = decimal.NewFromFloat(gp.GetMinLot())
			param.LotMax = decimal.NewFromFloat(gp.GetMaxLot())
			param.LotStep = decimal.NewFromFloat(gp.GetLotStep())
		}
		// TradeMode semantics come from SymbolInfoEx.Trade (the trade-mode
		// enum); the group Execution mode is a different enum — writing it
		// here mislabeled every symbol (F2). Canonical enum:
		// 0=disabled,1=long_only,2=short_only,3=close_only,4=full.
		// VM-LIVE-VENUE-R2: Ex absent → -1 = unknown sentinel; the Go zero
		// value 0 would collide with real enum 0 (disabled / no-freeze /
		// instant). Ex present → broker facts verbatim. TradeExemode
		// canonical enum: 0=instant,1=request,2=market,3=exchange.
		param.TradeMode = -1
		param.FreezeLevel = -1
		param.TradeExemode = -1
		if ex != nil {
			param.TradeMode = ex.GetTrade()
			param.FreezeLevel = ex.GetFreezeLevel()
			param.TradeExemode = ex.GetExemode()
		}
		// Do not default ContractSize to 1; zero means "unknown" and triggers
		// fail-closed margin checks in the risk gate.
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
