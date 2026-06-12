package mt5

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type Gateway struct {
	cfg                  mdtick.AccountConfig
	log                  *zap.Logger
	mu                   sync.RWMutex
	conn                 *grpc.ClientConn
	client               pb.MT5Client
	connCli              pb.ConnectionClient
	streamCli            pb.StreamsClient
	qhCli                pb.QuoteHistoryClient
	subCli               pb.SubscriptionsClient
	tradingCli           pb.TradingClient
	serviceCli           pb.ServiceClient
	sessionID            string
	subscribedSymbols    []string             // symbols registered via Subscribe/AddSymbols; re-subscribed after reconnect
	cancelSub            context.CancelFunc
	cancelProfitSub      context.CancelFunc
	cancelOrderUpdateSub context.CancelFunc   // set by orderUpdateRecvLoop (SSE stream)
	cancelHubOrderSub    context.CancelFunc   // set by SubscribeOrderEvents (Hub stream)
	reconnecting         bool // true while reconnection is in progress (prevents recvLoop race)
	onStatusChange       func(status, message string) // connection state callback (nil-safe)
}

func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway {
	return &Gateway{cfg: cfg, log: log}
}

func (g *Gateway) Platform() string  { return "mt5" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }

// token returns the sanitized mtapi token (strips \r, \n, and other control chars).
func (g *Gateway) token() string {
	return sanitizeToken(g.cfg.MtapiToken)
}

// sanitizeToken strips control characters that could enable HTTP header injection.
func sanitizeToken(t string) string {
	b := make([]byte, 0, len(t))
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c >= 32 && c != 127 {
			b = append(b, c)
		}
	}
	return string(b)
}

func (g *Gateway) Connect(ctx context.Context) error {
	gateway := g.cfg.MtapiHost
	if gateway == "" || gateway == g.cfg.BrokerHost {
		gateway = "mt5grpc3.mtapi.io:443"
	}
	if !strings.Contains(gateway, ":") {
		gateway += ":443"
	}
	conn, err := grpc.DialContext(ctx, gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
	)
	if err != nil {
		return fmt.Errorf("mt5 dial: %w", err)
	}
	g.mu.Lock()
	g.conn = conn
	g.client = pb.NewMT5Client(conn)
	g.connCli = pb.NewConnectionClient(conn)
	g.streamCli = pb.NewStreamsClient(conn)
	g.qhCli = pb.NewQuoteHistoryClient(conn)
	g.subCli = pb.NewSubscriptionsClient(conn)
	g.tradingCli = pb.NewTradingClient(conn)
	g.serviceCli = pb.NewServiceClient(conn)
	g.mu.Unlock()

	tempID := "mdgw-" + g.cfg.Login
	md := metadata.New(map[string]string{"id": tempID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	brokerHost := g.cfg.BrokerHost
	if idx := strings.LastIndex(brokerHost, ":"); idx > 0 {
		brokerHost = brokerHost[:idx]
	}
	loginResp, err := g.connCli.Connect(loginCtx, &pb.ConnectRequest{
		Host: brokerHost, Port: 443, User: strToUint64(g.cfg.Login),
		Password: g.cfg.Password,
	})
	if err != nil {
		g.mu.Lock()
		if g.conn != nil {
			g.conn.Close()
			g.conn = nil
		}
		g.client = nil
		g.connCli = nil
		g.streamCli = nil
		g.qhCli = nil
		g.subCli = nil
		g.tradingCli = nil
		g.serviceCli = nil
		g.mu.Unlock()
		return fmt.Errorf("mt5 login: %w", err)
	}
	token := loginResp.GetResult()
	respErr := loginResp.GetError()
	g.log.Info("mt5 connect response",
		zap.String("token", token), zap.Any("error", respErr),
		zap.String("host", brokerHost), zap.String("gateway", gateway))
	if token == "" {
		errMsg := "empty token"
		if respErr != nil {
			errMsg = fmt.Sprintf("code=%d msg=%s", respErr.GetCode(), respErr.GetMessage())
		}
		g.mu.Lock()
		if g.conn != nil {
			g.conn.Close()
			g.conn = nil
		}
		g.client = nil
		g.connCli = nil
		g.streamCli = nil
		g.qhCli = nil
		g.subCli = nil
		g.tradingCli = nil
		g.serviceCli = nil
		g.mu.Unlock()
		return fmt.Errorf("mt5 login: %s", errMsg)
	}
	g.mu.Lock()
	g.sessionID = token
	g.mu.Unlock()
	return nil
}

func (g *Gateway) Disconnect(ctx context.Context) error {
	// Drain: brief grace period for in-flight stream Recv() calls
	// to observe the cancelled context before we tear down the conn.
	time.Sleep(200 * time.Millisecond)

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cancelSub != nil {
		g.cancelSub()
		g.cancelSub = nil
	}
	if g.cancelProfitSub != nil {
		g.cancelProfitSub()
		g.cancelProfitSub = nil
	}
	if g.cancelOrderUpdateSub != nil {
		g.cancelOrderUpdateSub()
		g.cancelOrderUpdateSub = nil
	}
	if g.cancelHubOrderSub != nil {
		g.cancelHubOrderSub()
		g.cancelHubOrderSub = nil
	}
	if g.conn != nil {
		g.conn.Close()
		g.conn = nil
	}
	g.client = nil
	g.connCli = nil
	g.streamCli = nil
	g.qhCli = nil
	g.subCli = nil
	g.tradingCli = nil
	g.serviceCli = nil
	g.sessionID = ""
	return nil
}

func (g *Gateway) ensureConnected(ctx context.Context, backoff *time.Duration, maxBackoff time.Duration) error {
	g.mu.RLock()
	conn := g.conn
	reconnecting := g.reconnecting
	g.mu.RUnlock()

	if conn != nil {
		return nil
	}
	// If ReconnectGateway is in progress, wait for it to finish
	// instead of racing a second Connect call.
	if reconnecting {
		g.sleep(ctx, 500*time.Millisecond)
		return nil // recvLoop will retry on next iteration
	}
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
	if a < b {
		return a
	}
	return b
}

// FetchBrokerInfo implements mdtick.BrokerInfoFetcher.
// Calls AccountSummary after Connect to extract broker-level margin settings.
// Current mtapi proto does not expose ACCOUNT_MARGIN_SO_CALL / ACCOUNT_MARGIN_SO_SO;
// returns zero values to signal "use schema DEFAULTs" until the proto is extended.
func (g *Gateway) FetchBrokerInfo(ctx context.Context) (*mdtick.BrokerInfo, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return &mdtick.BrokerInfo{}, nil
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	asCtx := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 AccountSummary: %w", err)
	}
	if resp.GetResult() == nil {
		msg := "no error details"
		if errInfo := resp.GetError(); errInfo != nil {
			msg = errInfo.GetMessage()
		}
		return nil, fmt.Errorf("mt5 AccountSummary: result nil, msg=%s", msg)
	}

	// Proto v2.x AccountSummary does not carry MarginCallLevel / StopOutLevel.
	// When these fields are added to the mtapi proto, uncomment:
	//   summary := resp.GetResult()
	//   return &mdtick.BrokerInfo{
	//       MarginCallPct: summary.GetMarginCallLevel(),
	//       StopOutPct:    summary.GetStopOutLevel(),
	//   }, nil
	return &mdtick.BrokerInfo{}, nil
}

func strToUint64(s string) uint64 {
	var v uint64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + uint64(c-'0')
		}
	}
	return v
}

// FetchAccountInfo calls AccountSummary and returns basic account details.
func (g *Gateway) FetchAccountInfo(ctx context.Context) (*mdtick.MTAccountInfo, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	asCtx := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 AccountSummary: %w", err)
	}
	if resp.GetResult() == nil {
		// Investor/read-only accounts may not expose AccountSummary.
		// Return zero-value account info (caller handles gracefully).
		return &mdtick.MTAccountInfo{}, nil
	}

	s := resp.GetResult()
	return &mdtick.MTAccountInfo{
		Balance:    s.GetBalance(),
		Credit:     s.GetCredit(),
		Equity:     s.GetEquity(),
		Margin:     s.GetMargin(),
		FreeMargin: s.GetFreeMargin(),
		Leverage:   int32(s.GetLeverage()),
		Currency:   s.GetCurrency(),
		IsInvestor: s.GetIsInvestor(),
	}, nil
}

func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock()
	client := g.serviceCli
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return fmt.Errorf("mt5: not connected")
	}

	hcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	// MT5 has a dedicated Health RPC (not available in MT4).
	_, err := client.Health(metadata.NewOutgoingContext(hcCtx, md), &pb.HealthRequest{})
	if err != nil {
		return fmt.Errorf("mt5: health check failed: %w", err)
	}
	return nil
}

func (g *Gateway) SessionID() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sessionID
}

// SetReconnecting guards against recvLoop races during managed reconnection.
func (g *Gateway) SetReconnecting(v bool) {
	g.mu.Lock()
	g.reconnecting = v
	g.mu.Unlock()
}

// SetStatusCallback registers a callback for connection state changes.
// The callback must not block — it is called from recvLoop goroutines.
func (g *Gateway) SetStatusCallback(fn func(status, message string)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onStatusChange = fn
}

// reportStatus invokes the registered status callback if set.
// Concurrency-safe: reads the callback field under RLock.
func (g *Gateway) reportStatus(status, message string) {
	g.mu.RLock()
	fn := g.onStatusChange
	g.mu.RUnlock()
	if fn != nil {
		fn(status, message)
	}
}

func (g *Gateway) MT5Client() pb.MT5Client {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.client
}
