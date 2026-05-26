package mt4
import (
	"context"; "fmt"; "strings"; "sync"; "time"
	pb "anttrader/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"; "go.uber.org/zap"
	"crypto/tls"
	"google.golang.org/grpc"; "google.golang.org/grpc/credentials"; "google.golang.org/grpc/metadata"
)
type Gateway struct {
	cfg mdtick.AccountConfig; log *zap.Logger
	mu sync.RWMutex; conn *grpc.ClientConn
	client pb.MT4Client; connCli pb.ConnectionClient; streamCli pb.StreamsClient; subCli pb.SubscriptionsClient
	sessionID string; cancelSub context.CancelFunc; cancelProfitSub context.CancelFunc; cancelOrderUpdateSub context.CancelFunc
}
func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway { return &Gateway{cfg: cfg, log: log} }
func (g *Gateway) Platform() string { return "mt4" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }
func (g *Gateway) Connect(ctx context.Context) error {
	// Resolve mtapi gateway: broker config > mtapi.io fallback.
	// mtapi_endpoint is empty by default (admin-configurable via brokers table).
	gateway := g.cfg.MtapiHost
	if gateway == "" || gateway == g.cfg.BrokerHost { gateway = "mt4grpc3.mtapi.io:443" }
	if !strings.Contains(gateway, ":") { gateway += ":443" }
	conn, err := grpc.DialContext(ctx, gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
	)
	if err != nil { return fmt.Errorf("mt4 dial: %w", err) }
	g.mu.Lock(); g.conn=conn; g.client=pb.NewMT4Client(conn); g.connCli=pb.NewConnectionClient(conn); g.streamCli=pb.NewStreamsClient(conn); g.subCli=pb.NewSubscriptionsClient(conn); g.mu.Unlock()
	// Connect to broker via mtapi (alfq pattern: id="mdgw-<login>", no authorization header).
	tempID := "mdgw-" + g.cfg.Login
	md := metadata.New(map[string]string{"id": tempID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	brokerHost := g.cfg.BrokerHost
	if idx := strings.LastIndex(brokerHost, ":"); idx > 0 { brokerHost = brokerHost[:idx] }
	loginResp, err := g.connCli.Connect(loginCtx, &pb.ConnectRequest{Host: brokerHost, Port: 443, User: int32(strToInt(g.cfg.Login)), Password: g.cfg.Password, Id: &tempID})
	if err != nil { g.conn.Close(); g.conn=nil; return fmt.Errorf("mt4 login: %w", err) }
	token := loginResp.GetResult()
	respErr := loginResp.GetError()
	g.log.Info("mt4 connect response", zap.String("token", token), zap.Any("error", respErr), zap.String("host", brokerHost), zap.String("gateway", gateway))
	if token == "" {
		errMsg := "empty token"
		if respErr != nil { errMsg = fmt.Sprintf("code=%d msg=%s", respErr.GetCode(), respErr.GetMessage()) }
		g.conn.Close(); g.conn=nil; return fmt.Errorf("mt4 login: %s", errMsg)
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
	g.client=nil; g.connCli=nil; g.streamCli=nil; g.subCli=nil; g.sessionID=""; return nil
}
func (g *Gateway) Subscribe(ctx context.Context, syms []string, handler mdtick.TickHandler) error {
	g.mu.RLock(); sc := g.streamCli; sub := g.subCli; sid := g.sessionID; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt4: not connected") }
	if sub != nil && len(syms) > 0 {
		subMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
		subCtx := metadata.NewOutgoingContext(ctx, subMd)
		if _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms}); err != nil {
			g.log.Warn("mt4: subscribe symbols failed", zap.Strings("syms", syms), zap.Error(err))
		} else {
			g.log.Info("mt4: subscribed symbols", zap.Strings("syms", syms))
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
			g.log.Warn("mt4 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second // reset on successful connection
		g.log.Info("mt4: quote stream active")
		for {
			quote, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 recv", zap.Error(err))
				cancel()
				break // reconnect
			}
			q := quote.GetResult()
			if q == nil {
				continue
			}
			handler(&mdtick.Tick{
				UserID: g.cfg.UserID, AccountID: g.cfg.AccountID, Broker: g.cfg.Broker, Platform: "mt4",
				SymbolRaw: q.GetSymbol(), Canonical: "", TsUnixMs: q.GetTime().AsTime().UnixMilli(),
				ArrivedUnixMs: Clk.Now().UTC().UnixMilli(),
				Bid: decimal.NewFromFloat(q.GetBid()), Ask: decimal.NewFromFloat(q.GetAsk()),
			})
		}
	}
}

func (g *Gateway) ensureConnected(ctx context.Context, backoff *time.Duration, maxBackoff time.Duration) error {
	g.mu.RLock()
	conn := g.conn
	g.mu.RUnlock()
	if conn != nil {
		return nil
	}
	if err := g.Connect(ctx); err != nil {
		g.log.Warn("mt4 reconnect failed", zap.Error(err), zap.Duration("backoff", *backoff))
		g.sleep(ctx, *backoff)
		*backoff = minDuration(*backoff*2, maxBackoff)
		return fmt.Errorf("mt4 reconnect: %w", err)
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

func (g *Gateway) SubscribeProfit(ctx context.Context, handler mdtick.ProfitHandler) error {
	g.mu.RLock(); sc := g.streamCli; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt4: not connected") }
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
			g.log.Warn("mt4 profit subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt4: profit stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 profit recv", zap.Error(err))
				cancel()
				break
			}
			p := resp.GetResult()
			if p == nil { continue }
			profit := p.GetEquity() - p.GetBalance()
			var profitPercent float64
			if p.GetBalance() > 0 { profitPercent = profit / p.GetBalance() * 100 }
			positions := make([]mdtick.ProfitPosition, 0, len(p.GetOrders()))
			for _, o := range p.GetOrders() {
				positions = append(positions, mdtick.ProfitPosition{
					Ticket: int64(o.GetTicket()), Symbol: o.GetSymbol(),
					Profit: o.GetProfit(), Volume: o.GetLots(),
					CurrentPrice: o.GetClosePrice(),
				})
			}
			handler(&mdtick.ProfitUpdate{
				AccountID: g.cfg.AccountID, Platform: "mt4",
				Balance: p.GetBalance(), Credit: p.GetCredit(), Equity: p.GetEquity(),
				Margin: p.GetMargin(), FreeMargin: p.GetFreeMargin(), MarginLevel: p.GetMarginLevel(),
				Profit: profit, ProfitPercent: profitPercent, Positions: positions,
			})
		}
	}
}

// SubscribeOrderUpdate subscribes to MT4 OnOrderUpdate stream.
// Each event contains account metrics + full OpenedOrders list,
// providing real-time position updates (open/close/modify).
func (g *Gateway) SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error {
	g.mu.RLock(); sc := g.streamCli; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt4: not connected") }
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
			g.log.Warn("mt4 order update subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt4: order update stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 order update recv", zap.Error(err))
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
				updateTicket = int64(o.GetTicket())
				updateType = mt4UpdateActionLabel(update.GetAction())
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
					Ticket:       int64(o.GetTicket()),
					Symbol:       o.GetSymbol(),
					Type:         mt4OrderOpLabel(o.GetType()),
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
				Platform:          "mt4",
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
				Credit:            s.GetCredit(),
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

func mt4UpdateActionLabel(a pb.UpdateAction) string {
	switch a {
	case pb.UpdateAction_UpdateAction_PositionOpen:
		return "open"
	case pb.UpdateAction_UpdateAction_PositionClose:
		return "close"
	case pb.UpdateAction_UpdateAction_PositionModify:
		return "modify"
	case pb.UpdateAction_UpdateAction_PendingOpen:
		return "pending_open"
	case pb.UpdateAction_UpdateAction_PendingClose:
		return "pending_close"
	case pb.UpdateAction_UpdateAction_PendingModify:
		return "pending_modify"
	case pb.UpdateAction_UpdateAction_PendingFill:
		return "open"
	default:
		return "unknown"
	}
}

func mt4OrderOpLabel(op pb.Op) string {
	switch op {
	case pb.Op_Op_Sell:
		return "sell"
	case pb.Op_Op_BuyLimit:
		return "buy_limit"
	case pb.Op_Op_SellLimit:
		return "sell_limit"
	case pb.Op_Op_BuyStop:
		return "buy_stop"
	case pb.Op_Op_SellStop:
		return "sell_stop"
	default:
		return "buy"
	}
}

// GetPriceHistory implements backfiller.MTAPIBarSource via MT4 QuoteHistory RPC.
func (g *Gateway) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 GetPriceHistory: not connected")
	}

	tf, ok := mt4PeriodToTimeframe(period)
	if !ok {
		return nil, fmt.Errorf("mt4 GetPriceHistory: unsupported period %q", period)
	}

	// MT4 QuoteHistory works with from-date + count (bars back from from).
	count := int32((to - from) / periodMs(period))
	if count <= 0 { count = 100 }
	if count > 5000 { count = 5000 }
	fromStr := time.UnixMilli(to).UTC().Format("2006-01-02T15:04:05")

	resp, err := client.QuoteHistory(ctx, &pb.QuoteHistoryRequest{
		Id: sid, Symbol: symbolRaw, Timeframe: tf, From: fromStr, Count: count,
	})
	if err != nil {
		return nil, fmt.Errorf("mt4 QuoteHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 QuoteHistory: code=%d msg=%s",
			resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return convertMT4Bars(resp.GetResult(), accountID, period), nil
}

func mt4PeriodToTimeframe(period string) (pb.Timeframe, bool) {
	switch period {
	case "1m": return pb.Timeframe_Timeframe_M1, true
	case "5m": return pb.Timeframe_Timeframe_M5, true
	case "15m": return pb.Timeframe_Timeframe_M15, true
	case "30m": return pb.Timeframe_Timeframe_M30, true
	case "1h": return pb.Timeframe_Timeframe_H1, true
	case "4h": return pb.Timeframe_Timeframe_H4, true
	case "1d": return pb.Timeframe_Timeframe_D1, true
	default: return 0, false
	}
}

func convertMT4Bars(bars []*pb.Bar, accountID, period string) []*mdtick.Bar {
	var out []*mdtick.Bar
	for _, b := range bars {
		t := b.GetTime().AsTime()
		out = append(out, &mdtick.Bar{
			AccountID: accountID, Period: period,
			OpenTsUnixMs: t.UnixMilli(), CloseTsUnixMs: t.UnixMilli() + periodMs(period),
			Open: decimal.NewFromFloat(b.GetOpen()), High: decimal.NewFromFloat(b.GetHigh()),
			Low: decimal.NewFromFloat(b.GetLow()), Close: decimal.NewFromFloat(b.GetClose()),
			Volume: b.GetVolume(),
		})
	}
	return out
}
func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock(); defer g.mu.RUnlock()
	if g.conn == nil { return fmt.Errorf("mt4: not connected") }; return nil
}
func (g *Gateway) SessionID() string { g.mu.RLock(); defer g.mu.RUnlock(); return g.sessionID }
func (g *Gateway) MT4Client() pb.MT4Client { g.mu.RLock(); defer g.mu.RUnlock(); return g.client }
func strToInt(s string) int {
	v := 0
	for _, c := range s { if c >= '0' && c <= '9' { v = v*10 + int(c-'0') } }
	return v
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

// --- OrderExecutor interface (mthub) ---

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	return 0, fmt.Errorf("mt4: PlaceOrder not yet implemented")
}

func (g *Gateway) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	return fmt.Errorf("mt4: CloseOrder not yet implemented")
}

func (g *Gateway) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	return fmt.Errorf("mt4: ModifyOrder not yet implemented")
}

func (g *Gateway) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt4 OpenedOrders: %w", err)
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side := mthub.SideBuy
		ot := mthub.OrderMarket
		switch o.GetType() {
		case pb.Op_Op_Sell:
			side = mthub.SideSell
		case pb.Op_Op_BuyLimit:
			ot = mthub.OrderLimit
		case pb.Op_Op_SellLimit:
			side = mthub.SideSell; ot = mthub.OrderLimit
		case pb.Op_Op_BuyStop:
			ot = mthub.OrderStop
		case pb.Op_Op_SellStop:
			side = mthub.SideSell; ot = mthub.OrderStop
		}
		out = append(out, &mthub.OrderRecord{
			Ticket: int64(o.GetTicket()), SymbolRaw: o.GetSymbol(), Canonical: o.GetSymbol(),
			Side: side, OrderType: ot,
			Volume: decimal.NewFromFloat(o.GetLots()),
			OpenPrice: decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime: o.GetOpenTime().AsTime(),
			CloseTime: o.GetCloseTime().AsTime(),
			Profit: decimal.NewFromFloat(o.GetProfit()),
			Swap: decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment: o.GetComment(), Magic: o.GetMagicNumber(),
			State: mthub.OrderStateOpen,
		})
	}
	return out, nil
}

func (g *Gateway) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 FetchOrderHistory: not connected")
	}
	fromStr := from.UTC().Format("2006-01-02T15:04:05")
	toStr := to.UTC().Format("2006-01-02T15:04:05")
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.cfg.MtapiToken})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OrderHistory(ctx, &pb.OrderHistoryRequest{Id: sid, From: fromStr, To: toStr})
	if err != nil {
		return nil, fmt.Errorf("mt4 OrderHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 OrderHistory: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side := mthub.SideBuy
		ot := mthub.OrderMarket
		switch o.GetType() {
		case pb.Op_Op_Sell:
			side = mthub.SideSell
		case pb.Op_Op_BuyLimit:
			ot = mthub.OrderLimit
		case pb.Op_Op_SellLimit:
			side = mthub.SideSell; ot = mthub.OrderLimit
		case pb.Op_Op_BuyStop:
			ot = mthub.OrderStop
		case pb.Op_Op_SellStop:
			side = mthub.SideSell; ot = mthub.OrderStop
		}
		state := mthub.OrderStateClosed
		if o.GetCloseTime().GetSeconds() == 0 { state = mthub.OrderStateOpen }
		out = append(out, &mthub.OrderRecord{
			Ticket: int64(o.GetTicket()), SymbolRaw: o.GetSymbol(), Canonical: o.GetSymbol(),
			Side: side, OrderType: ot,
			Volume: decimal.NewFromFloat(o.GetLots()),
			OpenPrice: decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime: o.GetOpenTime().AsTime(),
			CloseTime: o.GetCloseTime().AsTime(),
			Profit: decimal.NewFromFloat(o.GetProfit()),
			Swap: decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment: o.GetComment(), Magic: o.GetMagicNumber(),
			State: state,
		})
	}
	return out, nil
}

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	return nil, fmt.Errorf("mt4: FetchSymbolParams not yet implemented")
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	return fmt.Errorf("mt4: SubscribeOrderEvents not yet implemented")
}

