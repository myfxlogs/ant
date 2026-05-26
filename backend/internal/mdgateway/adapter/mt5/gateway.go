package mt5
import (
	"context"; "fmt"; "strings"; "sync"; "time"
	pb "anttrader/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"; "go.uber.org/zap"
	"crypto/tls"
	"google.golang.org/grpc"; "google.golang.org/grpc/credentials"; "google.golang.org/grpc/metadata"
)
type Gateway struct {
	cfg mdtick.AccountConfig; log *zap.Logger
	mu sync.RWMutex; conn *grpc.ClientConn
	client pb.MT5Client; connCli pb.ConnectionClient; streamCli pb.StreamsClient; qhCli pb.QuoteHistoryClient; subCli pb.SubscriptionsClient; tradingCli pb.TradingClient
	sessionID string; cancelSub context.CancelFunc; cancelProfitSub context.CancelFunc; cancelOrderUpdateSub context.CancelFunc
}
func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway { return &Gateway{cfg: cfg, log: log} }
func (g *Gateway) Platform() string { return "mt5" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }
func (g *Gateway) Connect(ctx context.Context) error {
	// Resolve mtapi gateway: broker config > mtapi.io fallback.
	gateway := g.cfg.MtapiHost
	if gateway == "" || gateway == g.cfg.BrokerHost { gateway = "mt5grpc3.mtapi.io:443" }
	if !strings.Contains(gateway, ":") { gateway += ":443" }
	conn, err := grpc.DialContext(ctx, gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
	)
	if err != nil { return fmt.Errorf("mt5 dial: %w", err) }
	g.mu.Lock(); g.conn=conn; g.client=pb.NewMT5Client(conn); g.connCli=pb.NewConnectionClient(conn); g.streamCli=pb.NewStreamsClient(conn); g.qhCli=pb.NewQuoteHistoryClient(conn); g.subCli=pb.NewSubscriptionsClient(conn); g.tradingCli=pb.NewTradingClient(conn); g.mu.Unlock()
	// Connect to broker via mtapi (alfq pattern: id="mdgw-<login>", no authorization header, no Id in ConnectRequest for MT5).
	tempID := "mdgw-" + g.cfg.Login
	md := metadata.New(map[string]string{"id": tempID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	brokerHost := g.cfg.BrokerHost
	if idx := strings.LastIndex(brokerHost, ":"); idx > 0 { brokerHost = brokerHost[:idx] }
	loginResp, err := g.connCli.Connect(loginCtx, &pb.ConnectRequest{Host: brokerHost, Port: 443, User: strToUint64(g.cfg.Login), Password: g.cfg.Password})
	if err != nil { g.conn.Close(); g.conn=nil; return fmt.Errorf("mt5 login: %w", err) }
	token := loginResp.GetResult()
	respErr := loginResp.GetError()
	g.log.Info("mt5 connect response", zap.String("token", token), zap.Any("error", respErr), zap.String("host", brokerHost), zap.String("gateway", gateway))
	if token == "" {
		errMsg := "empty token"
		if respErr != nil { errMsg = fmt.Sprintf("code=%d msg=%s", respErr.GetCode(), respErr.GetMessage()) }
		g.conn.Close(); g.conn=nil; return fmt.Errorf("mt5 login: %s", errMsg)
	}
	g.mu.Lock(); g.sessionID=token; g.mu.Unlock()
	return nil
}
func (g *Gateway) Disconnect(ctx context.Context) error {
	g.mu.Lock(); defer g.mu.Unlock()
	if g.cancelSub != nil { g.cancelSub(); g.cancelSub = nil }
	if g.cancelProfitSub != nil { g.cancelProfitSub(); g.cancelProfitSub = nil }
	if g.cancelOrderUpdateSub != nil { g.cancelOrderUpdateSub(); g.cancelOrderUpdateSub = nil }
	if g.conn != nil { g.conn.Close(); g.conn = nil }
	g.client=nil; g.connCli=nil; g.streamCli=nil; g.qhCli=nil; g.subCli=nil; g.tradingCli=nil; g.sessionID=""; return nil
}
func (g *Gateway) Subscribe(ctx context.Context, syms []string, handler mdtick.TickHandler) error {
	g.mu.RLock(); sc := g.streamCli; sub := g.subCli; sid := g.sessionID; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt5: not connected") }
	if sub != nil && len(syms) > 0 {
		subMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
		subCtx := metadata.NewOutgoingContext(ctx, subMd)
		if _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms}); err != nil {
			g.log.Warn("mt5: subscribe symbols failed", zap.Strings("syms", syms), zap.Error(err))
		} else {
			g.log.Info("mt5: subscribed symbols", zap.Strings("syms", syms))
		}
	}
	go g.recvLoop(ctx, handler)
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

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnQuote(subCtx, &pb.OnQuoteRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt5: quote stream active")
		for {
			tick, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 recv", zap.Error(err))
				cancel()
				break
			}
			q := tick.GetResult()
			if q == nil { continue }
			handler(&mdtick.Tick{
				UserID: g.cfg.UserID, AccountID: g.cfg.AccountID, Broker: g.cfg.Broker, Platform: "mt5",
				SymbolRaw: q.GetSymbol(), Canonical: "", TsUnixMs: q.GetTime().AsTime().UnixMilli(),
				ArrivedUnixMs: Clk.Now().UTC().UnixMilli(),
				Bid: decimal.NewFromFloat(q.GetBid()), Ask: decimal.NewFromFloat(q.GetAsk()),
				BidVolume: float64(q.GetVolume()),
			})
		}
	}
}

func (g *Gateway) ensureConnected(ctx context.Context, backoff *time.Duration, maxBackoff time.Duration) error {
	g.mu.RLock()
	conn := g.conn
	g.mu.RUnlock()
	if conn != nil { return nil }
	if err := g.Connect(ctx); err != nil {
		g.log.Warn("mt5 reconnect failed", zap.Error(err), zap.Duration("backoff", *backoff))
		g.sleep(ctx, *backoff)
		*backoff = minDuration(*backoff*2, maxBackoff)
		return fmt.Errorf("mt5 reconnect: %w", err)
	}
	return nil
}

func (g *Gateway) sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b { return a }
	return b
}
func strToUint64(s string) uint64 {
	var v uint64
	for _, c := range s { if c >= '0' && c <= '9' { v = v*10 + uint64(c-'0') } }
	return v
}

// fetchAndPublish calls AccountSummary (canonical MQL5 values) and publishes
// via the handler. If AccountSummary fails, falls back to stream-derived values
// from p; if p is nil (initial call before any stream frame), skips.
func (g *Gateway) fetchAndPublish(ctx context.Context, sid string, p *pb.ProfitUpdate, handler mdtick.ProfitHandler) {
	var balance, equity, profit, margin, freeMargin, marginLevel, credit float64

	asMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	sctx, scancel := context.WithTimeout(ctx, 3*time.Second)
	defer scancel()
	asCtx := metadata.NewOutgoingContext(sctx, asMd)
	acct, err := g.client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})

	if err == nil && acct != nil && acct.GetResult() != nil {
		s := acct.GetResult()
		balance = s.GetBalance()
		equity = s.GetEquity()
		profit = s.GetProfit()
		margin = s.GetMargin()
		freeMargin = s.GetFreeMargin()
		marginLevel = s.GetMarginLevel()
		credit = s.GetCredit()
	} else if p != nil {
		g.log.Debug("mt5 AccountSummary failed; falling back to stream frame",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		balance = p.GetBalance()
		equity = p.GetEquity()
		profit = equity - balance
		margin = p.GetMargin()
		freeMargin = p.GetFreeMargin()
		marginLevel = p.GetMarginLevel()
		credit = p.GetCredit()
	} else {
		g.log.Warn("mt5 initial AccountSummary failed; no data to publish",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		return
	}

	var profitPercent float64
	if balance > 0 {
		profitPercent = profit / balance * 100
	}

	var positions []mdtick.ProfitPosition
	if p != nil {
		positions = make([]mdtick.ProfitPosition, 0, len(p.GetOrders()))
		for _, o := range p.GetOrders() {
			positions = append(positions, mdtick.ProfitPosition{
				Ticket: o.GetTicket(), Symbol: o.GetSymbol(),
				Profit: o.GetProfit(), Volume: o.GetLots(),
				CurrentPrice: o.GetOpenPrice(),
			})
		}
	}

	handler(&mdtick.ProfitUpdate{
		AccountID: g.cfg.AccountID, Platform: "mt5",
		Balance: balance, Credit: credit, Equity: equity,
		Margin: margin, FreeMargin: freeMargin, MarginLevel: marginLevel,
		Profit: profit, ProfitPercent: profitPercent, Positions: positions,
	})
}

func (g *Gateway) SubscribeProfit(ctx context.Context, handler mdtick.ProfitHandler) error {
	g.mu.RLock(); sc := g.streamCli; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt5: not connected") }
	go g.profitRecvLoop(ctx, handler)
	return nil
}

func (g *Gateway) profitRecvLoop(ctx context.Context, handler mdtick.ProfitHandler) {
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

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelProfitSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnOrderProfit(subCtx, &pb.OnOrderProfitRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 profit subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt5: profit stream active")

		// Call AccountSummary once for initial snapshot. MT5 OnOrderProfit
		// only fires when positions change, so without this the frontend
		// would see stale data for idle accounts. AccountSummary is the
		// canonical source (MQL5 AccountInfoDouble).
		g.fetchAndPublish(ctx, sid, nil, handler)

		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 profit recv", zap.Error(err))
				cancel()
				break
			}
			p := resp.GetResult()
			if p == nil { continue }

			// On each stream frame, fetch canonical AccountSummary.
			// Falls back to stream-derived values on RPC failure.
			g.fetchAndPublish(ctx, sid, p, handler)
		}
	}
}

// SubscribeOrderUpdate subscribes to MT5 OnOrderUpdate stream.
// Each event contains account metrics + full OpenedOrders list,
// providing real-time position updates (open/close/modify).
func (g *Gateway) SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error {
	g.mu.RLock(); sc := g.streamCli; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt5: not connected") }
	go g.orderUpdateRecvLoop(ctx, handler)
	return nil
}

func (g *Gateway) orderUpdateRecvLoop(ctx context.Context, handler mdtick.OrderUpdateHandler) {
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

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelOrderUpdateSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnOrderUpdate(subCtx, &pb.OnOrderUpdateRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 order update subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt5: order update stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 order update recv", zap.Error(err))
				cancel()
				break
			}
			s := resp.GetResult()
			if s == nil { continue }

			update := s.GetUpdate()
			var updateTicket int64
			var updateType string
			var updateSymbol string
			var updateVolume float64
			var updateOpenPrice float64
			var updateClosePrice float64
			var updateProfit float64
			var updateSwap float64
			var updateCommission float64
			var updateComment string
			var updateOpenTime int64
			var updateCloseTime int64
			var updateSL float64
			var updateTP float64
			if update != nil {
				o := update.GetOrder()
				updateTicket = o.GetTicket()
				updateType = mt5UpdateTypeLabel(update.GetType())
				updateSymbol = o.GetSymbol()
				updateVolume = o.GetLots()
				updateOpenPrice = o.GetOpenPrice()
				updateClosePrice = o.GetClosePrice()
				updateProfit = o.GetProfit()
				updateSwap = o.GetSwap()
				updateCommission = o.GetCommission()
				updateComment = o.GetComment()
				updateOpenTime = o.GetOpenTime().GetSeconds()
				updateCloseTime = o.GetCloseTime().GetSeconds()
				updateSL = o.GetStopLoss()
				updateTP = o.GetTakeProfit()
			}

			// Convert OpenedOrders to mdtick format.
			positions := make([]mdtick.OrderUpdatePosition, 0, len(s.GetOpenedOrders()))
			for _, o := range s.GetOpenedOrders() {
				positions = append(positions, mdtick.OrderUpdatePosition{
					Ticket:       o.GetTicket(),
					Symbol:       o.GetSymbol(),
					Type:         mt5OrderTypeLabel(o.GetOrderType()),
					Volume:       o.GetLots(),
					OpenPrice:    o.GetOpenPrice(),
					CurrentPrice: o.GetClosePrice(),
					StopLoss:     o.GetStopLoss(),
					TakeProfit:   o.GetTakeProfit(),
					Profit:       o.GetProfit(),
					Swap:         o.GetSwap(),
					Commission:   o.GetCommission(),
					Comment:      o.GetComment(),
					OpenTime:     o.GetOpenTime().GetSeconds(),
				})
			}

			profit := s.GetEquity() - s.GetBalance()
			handler(&mdtick.OrderUpdate{
				AccountID:         g.cfg.AccountID,
				Platform:          "mt5",
				UpdateTicket:      updateTicket,
				UpdateType:        updateType,
				UpdateSymbol:      updateSymbol,
				UpdateVolume:      updateVolume,
				UpdateOpenPrice:   updateOpenPrice,
				UpdateClosePrice:  updateClosePrice,
				UpdateProfit:      updateProfit,
				UpdateSwap:        updateSwap,
				UpdateCommission:  updateCommission,
				UpdateComment:     updateComment,
				UpdateOpenTime:    updateOpenTime,
				UpdateCloseTime:   updateCloseTime,
				UpdateSL:          updateSL,
				UpdateTP:          updateTP,
				Balance:           s.GetBalance(),
				Equity:            s.GetEquity(),
				Margin:            s.GetMargin(),
				FreeMargin:        s.GetFreeMargin(),
				MarginLevel:       s.GetMarginLevel(),
				Profit:            profit,
				Positions:         positions,
			})
		}
	}
}

func mt5UpdateTypeLabel(t pb.UpdateType) string {
	switch t {
	case pb.UpdateType_UpdateType_MarketOpen:
		return "open"
	case pb.UpdateType_UpdateType_MarketClose:
		return "close"
	case pb.UpdateType_UpdateType_PartialClose:
		return "close"
	case pb.UpdateType_UpdateType_PendingOpen:
		return "pending_open"
	case pb.UpdateType_UpdateType_PendingClose:
		return "pending_close"
	case pb.UpdateType_UpdateType_MarketModify:
		return "modify"
	case pb.UpdateType_UpdateType_PendingModify:
		return "modify"
	default:
		return "unknown"
	}
}

func mt5OrderTypeLabel(ot pb.OrderType) string {
	switch ot {
	case pb.OrderType_OrderType_Sell:
		return "sell"
	case pb.OrderType_OrderType_BuyLimit:
		return "buy_limit"
	case pb.OrderType_OrderType_SellLimit:
		return "sell_limit"
	case pb.OrderType_OrderType_BuyStop:
		return "buy_stop"
	case pb.OrderType_OrderType_SellStop:
		return "sell_stop"
	case pb.OrderType_OrderType_BuyStopLimit:
		return "buy_stop_limit"
	case pb.OrderType_OrderType_SellStopLimit:
		return "sell_stop_limit"
	default:
		return "buy"
	}
}

// GetPriceHistory implements backfiller.MTAPIBarSource via MT5 PriceHistory RPC.
func (g *Gateway) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	g.mu.RLock()
	qhCli := g.qhCli
	sid := g.sessionID
	g.mu.RUnlock()

	if qhCli == nil || sid == "" {
		return nil, fmt.Errorf("mt5 GetPriceHistory: not connected")
	}

	tf := mt5PeriodToTimeframe(period)
	fromStr := time.UnixMilli(from).UTC().Format("2006-01-02T15:04:05")
	toStr := time.UnixMilli(to).UTC().Format("2006-01-02T15:04:05")

	resp, err := qhCli.PriceHistory(ctx, &pb.PriceHistoryRequest{
		Id: sid, Symbol: symbolRaw, From: fromStr, To: toStr, TimeFrame: tf,
	})
	if err != nil {
		return nil, fmt.Errorf("mt5 PriceHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 PriceHistory: code=%d msg=%s",
			resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return convertMT5Bars(resp.GetResult(), accountID, period), nil
}

func mt5PeriodToTimeframe(period string) int32 {
	switch period {
	case "1m": return 1
	case "5m": return 5
	case "15m": return 15
	case "30m": return 30
	case "1h": return 60
	case "4h": return 240
	case "1d": return 1440
	default: return 60
	}
}

func convertMT5Bars(bars []*pb.Bar, accountID, period string) []*mdtick.Bar {
	pm := periodMs(period)
	var out []*mdtick.Bar
	for _, b := range bars {
		t := b.GetTime().AsTime()
		out = append(out, &mdtick.Bar{
			AccountID: accountID, Period: period,
			OpenTsUnixMs: t.UnixMilli(), CloseTsUnixMs: t.UnixMilli() + pm,
			Open: decimal.NewFromFloat(b.GetOpenPrice()), High: decimal.NewFromFloat(b.GetHighPrice()),
			Low: decimal.NewFromFloat(b.GetLowPrice()), Close: decimal.NewFromFloat(b.GetClosePrice()),
			Volume: float64(b.GetVolume()), TickCount: uint32(b.GetTickVolume()),
		})
	}
	return out
}

func periodMs(period string) int64 {
	switch period {
	case "1m": return 60_000
	case "5m": return 300_000
	case "15m": return 900_000
	case "1h": return 3_600_000
	case "4h": return 14_400_000
	case "1d": return 86_400_000
	default: return 60_000
	}
}

func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock(); defer g.mu.RUnlock()
	if g.conn == nil { return fmt.Errorf("mt5: not connected") }; return nil
}
func (g *Gateway) SessionID() string { g.mu.RLock(); defer g.mu.RUnlock(); return g.sessionID }
func (g *Gateway) MT5Client() pb.MT5Client { g.mu.RLock(); defer g.mu.RUnlock(); return g.client }

// --- OrderExecutor interface (mthub) ---

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	g.mu.RLock(); tc := g.tradingCli; sid := g.sessionID; g.mu.RUnlock()
	if tc == nil || sid == "" { return 0, fmt.Errorf("mt5 PlaceOrder: not connected") }
	ot := mt5OrderType(req.Side, req.OrderType)
	price := req.Price.InexactFloat64()
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := tc.OrderSend(ctx, &pb.OrderSendRequest{
		Id: sid, Symbol: req.Canonical, Operation: ot,
		Volume: req.Volume.InexactFloat64(),
		Price: &price, Stoploss: pfloat64(req.StopLoss), Takeprofit: pfloat64(req.TakeProfit),
		Comment: &req.Comment, ExpertID: pInt64(int64(req.Magic)),
	})
	if err != nil { return 0, fmt.Errorf("mt5 OrderSend: %w", err) }
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return 0, fmt.Errorf("mt5 OrderSend: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil { return 0, fmt.Errorf("mt5 OrderSend: nil result") }
	return resp.GetResult().GetTicket(), nil
}

func mt5OrderType(side mthub.Side, ot mthub.OrderType) pb.OrderType {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket: return pb.OrderType_OrderType_Buy
	case side == mthub.SideSell && ot == mthub.OrderMarket: return pb.OrderType_OrderType_Sell
	case side == mthub.SideBuy && ot == mthub.OrderLimit: return pb.OrderType_OrderType_BuyLimit
	case side == mthub.SideSell && ot == mthub.OrderLimit: return pb.OrderType_OrderType_SellLimit
	case side == mthub.SideBuy && ot == mthub.OrderStop: return pb.OrderType_OrderType_BuyStop
	case side == mthub.SideSell && ot == mthub.OrderStop: return pb.OrderType_OrderType_SellStop
	case side == mthub.SideBuy && ot == mthub.OrderStopLimit: return pb.OrderType_OrderType_BuyStopLimit
	case side == mthub.SideSell && ot == mthub.OrderStopLimit: return pb.OrderType_OrderType_SellStopLimit
	default: return pb.OrderType_OrderType_Buy
	}
}

func pfloat64(d decimal.Decimal) *float64 {
	if d.IsZero() { return nil }
	v := d.InexactFloat64(); return &v
}

func pInt64(v int64) *int64 { if v == 0 { return nil }; return &v }

func (g *Gateway) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	g.mu.RLock(); tc := g.tradingCli; sid := g.sessionID; g.mu.RUnlock()
	if tc == nil || sid == "" { return fmt.Errorf("mt5 CloseOrder: not connected") }
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	ctx = metadata.NewOutgoingContext(ctx, md)
	l := lots.InexactFloat64()
	resp, err := tc.OrderClose(ctx, &pb.OrderCloseRequest{Id: sid, Ticket: ticket, Lots: &l})
	if err != nil { return fmt.Errorf("mt5 OrderClose: %w", err) }
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("mt5 OrderClose: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func (g *Gateway) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	g.mu.RLock(); tc := g.tradingCli; sid := g.sessionID; g.mu.RUnlock()
	if tc == nil || sid == "" { return fmt.Errorf("mt5 ModifyOrder: not connected") }
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := tc.OrderModify(ctx, &pb.OrderModifyRequest{
		Id: sid, Ticket: ticket, Stoploss: sl.InexactFloat64(), Takeprofit: tp.InexactFloat64(),
	})
	if err != nil { return fmt.Errorf("mt5 OrderModify: %w", err) }
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("mt5 OrderModify: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func (g *Gateway) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 OpenedOrders: %w", err)
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side := mthub.SideBuy
		ot := mthub.OrderMarket
		switch o.GetOrderType() {
		case pb.OrderType_OrderType_Sell:
			side = mthub.SideSell
		case pb.OrderType_OrderType_BuyLimit:
			ot = mthub.OrderLimit
		case pb.OrderType_OrderType_SellLimit:
			side = mthub.SideSell; ot = mthub.OrderLimit
		case pb.OrderType_OrderType_BuyStop:
			ot = mthub.OrderStop
		case pb.OrderType_OrderType_SellStop:
			side = mthub.SideSell; ot = mthub.OrderStop
		case pb.OrderType_OrderType_BuyStopLimit:
			ot = mthub.OrderStopLimit
		case pb.OrderType_OrderType_SellStopLimit:
			side = mthub.SideSell; ot = mthub.OrderStopLimit
		}
		out = append(out, &mthub.OrderRecord{
			Ticket: o.GetTicket(), SymbolRaw: o.GetSymbol(), Canonical: o.GetSymbol(),
			Side: side, OrderType: ot,
			Volume: decimal.NewFromFloat(o.GetLots()),
			OpenPrice: decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime: o.GetOpenTime().AsTime(),
			CloseTime: o.GetCloseTime().AsTime(),
			Profit: decimal.NewFromFloat(o.GetProfit()),
			Swap: decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment: o.GetComment(), Magic: int32(o.GetExpertId()),
			State: mthub.OrderStateOpen,
		})
	}
	return out, nil
}

func (g *Gateway) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*mthub.OrderRecord, error) {
	return nil, nil // TODO: implement via MT5 OrderHistory RPC
}

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock(); client := g.client; sid := g.sessionID; g.mu.RUnlock()
	if client == nil || sid == "" { return nil, fmt.Errorf("mt5 FetchSymbolParams: not connected") }
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	out := make([]*mthub.SymbolParam, 0, len(canonicals))
	for _, c := range canonicals {
		ctx2 := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.SymbolParams(ctx2, &pb.SymbolParamsRequest{Id: sid, Symbol: c})
		if err != nil { return out, fmt.Errorf("mt5 SymbolParams(%s): %w", c, err) }
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return out, fmt.Errorf("mt5 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil { continue }
		si := r.GetSymbolInfo()
		sg := r.GetSymbolGroup()
		out = append(out, &mthub.SymbolParam{
			Canonical: c, SymbolRaw: c,
			Digits:     si.GetDigits(),
			TradeMode:  int32(sg.GetTradeMode()),
			StopLevel:  sg.GetSL(),
			PointValue: decimal.NewFromFloat(si.GetTickValue()),
			LotSize:    decimal.NewFromFloat(si.GetContractSize()),
			LotStep:    decimal.NewFromFloat(sg.GetLotsStep()),
			LotMin:     decimal.NewFromFloat(sg.GetMinLots()),
			LotMax:     decimal.NewFromFloat(sg.GetMaxLots()),
			SpreadFloat: si.GetSpread() > 0,
		})
	}
	return out, nil
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	return fmt.Errorf("mt5: SubscribeOrderEvents not yet implemented")
}

