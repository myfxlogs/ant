package mt4

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

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
	client               pb.MT4Client
	connCli              pb.ConnectionClient
	streamCli            pb.StreamsClient
	subCli               pb.SubscriptionsClient
	tradingCli           pb.TradingClient
	serviceCli           pb.ServiceClient
	sessionID            string
	subscribedSymbols    []string // symbols registered via Subscribe/AddSymbols; re-subscribed after reconnect
	cancelSub            context.CancelFunc
	cancelProfitSub      context.CancelFunc
	cancelOrderUpdateSub context.CancelFunc           // set by orderUpdateRecvLoop (SSE stream)
	cancelHubOrderSub    context.CancelFunc           // set by SubscribeOrderEvents (Hub stream)
	reconnecting         bool                         // true while reconnection is in progress (prevents recvLoop race)
	onStatusChange       func(status, message string) // connection state callback (nil-safe)
	breaker              mdtick.Breaker
	quoteTimeout         time.Duration // no-data timeout for quote recvLoop (default 90s, injectable for tests)
	orderUpdateTimeout   time.Duration // no-data timeout for orderUpdate recvLoop (default 90s, injectable for tests)
}

func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway {
	return &Gateway{cfg: cfg, log: log}
}

// SetBreaker injects the shared per-broker circuit breaker.
func (g *Gateway) SetBreaker(b mdtick.Breaker) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.breaker = b
}

func (g *Gateway) Platform() string  { return "mt4" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }

// Config returns a copy of the gateway's account config.
func (g *Gateway) Config() mdtick.AccountConfig {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cfg
}

// SetBrokerHost updates the broker host in the gateway's config (§0 rediscovery).
func (g *Gateway) SetBrokerHost(host string) {
	g.mu.Lock()
	g.cfg.BrokerHost = host
	g.mu.Unlock()
}

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
		gateway = "mt4grpc3.mtapi.io:443"
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
		return fmt.Errorf("mt4 dial: %w", err)
	}
	g.mu.Lock()
	g.conn = conn
	g.client = pb.NewMT4Client(conn)
	g.connCli = pb.NewConnectionClient(conn)
	g.streamCli = pb.NewStreamsClient(conn)
	g.subCli = pb.NewSubscriptionsClient(conn)
	g.tradingCli = pb.NewTradingClient(conn)
	g.serviceCli = pb.NewServiceClient(conn)
	g.mu.Unlock()

	tempID := "mdgw-" + g.cfg.Login + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	md := metadata.New(map[string]string{"id": tempID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	brokerHost := g.cfg.BrokerHost
	brokerPort := int32(443)
	if idx := strings.LastIndex(brokerHost, ":"); idx > 0 {
		if p, err := strconv.ParseInt(brokerHost[idx+1:], 10, 32); err == nil && p > 0 && p <= 65535 {
			brokerPort = int32(p)
		}
		brokerHost = brokerHost[:idx]
	}
	connCli := g.connCli
	loginResp, err := connCli.Connect(loginCtx, &pb.ConnectRequest{
		Host: brokerHost, Port: brokerPort, User: int32(strToInt(g.cfg.Login)),
		Password: g.cfg.Password, Id: &tempID,
	})
	if err != nil {
		g.mu.Lock()
		if g.conn != nil {
			_ = g.conn.Close()
			g.conn = nil
		}
		g.client = nil
		g.connCli = nil
		g.streamCli = nil
		g.subCli = nil
		g.tradingCli = nil
		g.serviceCli = nil
		g.mu.Unlock()
		return fmt.Errorf("mt4 login: %w", err)
	}
	token := loginResp.GetResult()
	respErr := loginResp.GetError()
	g.log.Info("mt4 connect response",
		zap.String("host", brokerHost), zap.String("gateway", gateway),
		zap.Bool("has_token", token != ""), zap.Any("error", respErr))
	if token == "" {
		errMsg := "empty token"
		if respErr != nil {
			errMsg = fmt.Sprintf("code=%d msg=%s", respErr.GetCode(), respErr.GetMessage())
		}
		g.mu.Lock()
		if g.conn != nil {
			_ = g.conn.Close()
			g.conn = nil
		}
		g.client = nil
		g.connCli = nil
		g.streamCli = nil
		g.subCli = nil
		g.tradingCli = nil
		g.serviceCli = nil
		g.mu.Unlock()
		return fmt.Errorf("mt4 login: %s", errMsg)
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
		_ = g.conn.Close()
		g.conn = nil
	}
	g.client = nil
	g.connCli = nil
	g.streamCli = nil
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
		g.log.Warn("mt4 reconnect failed", zap.Error(err), zap.Duration("backoff", *backoff))
		g.sleep(ctx, *backoff)
		*backoff = minDuration(*backoff*2, maxBackoff)
		return fmt.Errorf("mt4 reconnect: %w", err)
	}
	return nil
}

func (g *Gateway) sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
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
		return nil, fmt.Errorf("mt4 AccountSummary: %w", err)
	}
	if resp.GetResult() == nil {
		return &mdtick.BrokerInfo{}, nil
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

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func strToInt(s string) int {
	v := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		}
	}
	return v
}
