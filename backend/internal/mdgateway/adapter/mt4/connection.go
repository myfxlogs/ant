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

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

type Gateway struct {
	cfg                      mdtick.AccountConfig
	log                      *zap.Logger
	mu                       sync.RWMutex
	connectMu                sync.Mutex // single-flight guard: serializes concurrent reconnects from the quote/profit/order stream loops
	conn                     *grpc.ClientConn
	client                   pb.MT4Client
	connCli                  pb.ConnectionClient
	streamCli                pb.StreamsClient
	subCli                   pb.SubscriptionsClient
	tradingCli               pb.TradingClient
	serviceCli               pb.ServiceClient
	sessionID                string
	subscribedSymbols        []string // symbols registered via Subscribe/AddSymbols; re-subscribed after reconnect
	cancelSub                context.CancelFunc
	cancelProfitSub          context.CancelFunc
	cancelOrderUpdateSub     context.CancelFunc           // set by orderUpdateRecvLoop (SSE stream)
	cancelHubOrderSub        context.CancelFunc           // set by SubscribeOrderEvents (Hub stream)
	reconnecting             bool                         // true while reconnection is in progress (prevents recvLoop race)
	onStatusChange           func(status, message string) // connection state callback (nil-safe)
	breaker                  mdtick.Breaker
	lastProfitUpdate         *mdtick.ProfitUpdate // last OnOrderProfit frame; used to detect data silence
	lastProfitAt             time.Time            // last time a frame was received
	quoteRecvTimeoutOverride time.Duration        // test-only: overrides the quote stream silence timeout
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
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
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
	// QUOTE-RECONNECT-LOOP S2: drain grace period for in-flight stream
	// Recv() calls to observe the cancelled context before we tear down
	// the conn. Use a ctx-cancellable wait instead of time.Sleep so that
	// a cancelled context returns immediately (<50ms) rather than
	// blocking for 200ms — the old time.Sleep killed all sessions after
	// a fixed delay, amplifying the cascade-disconnect problem.
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
	}

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
		old := g.conn
		g.conn = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					g.log.Warn("mt4: connection Close panic", zap.Any("panic", r))
				}
			}()
			_ = old.Close()
		}()
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
	// Single-flight: the quote, profit and order-update loops all call
	// ensureConnected. Without serialization they race Connect() after a
	// shared-connection teardown, each creating a separate mtapi session;
	// the losers then use a stale/empty sessionID and every SubscribeMany
	// is rejected with "Client with id = ... not found" — silent price starvation.
	if !g.beginConnect() {
		return nil // another loop already restored the connection
	}
	defer g.connectMu.Unlock()
	if err := g.Connect(ctx); err != nil {
		// QUOTE-RECONNECT-LOOP S1: do NOT return error — that would make
		// recvLoop/profitRecvLoop/orderUpdateRecvLoop exit permanently.
		// Instead log.Warn + sleep(backoff) + return nil so the loop
		// continues and retries Connect on the next iteration. Only
		// ctx.Done() should terminate the loop.
		g.log.Warn("mt4 reconnect failed; will retry", zap.Error(err), zap.Duration("backoff", *backoff))
		g.sleep(ctx, *backoff)
		*backoff = minDuration(*backoff*2, maxBackoff)
		return nil
	}
	return nil
}

// beginConnect acquires the single-flight reconnect slot. It returns false
// (without holding connectMu) when another goroutine already restored the
// connection while this one waited, meaning no Connect call is needed.
// Callers that receive true MUST release g.connectMu.
func (g *Gateway) beginConnect() bool {
	g.connectMu.Lock()
	g.mu.RLock()
	conn := g.conn
	g.mu.RUnlock()
	if conn != nil {
		g.connectMu.Unlock()
		return false
	}
	return true
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
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 AccountSummary: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil {
		return nil, fmt.Errorf("mt4 AccountSummary: result nil")
	}

	s := resp.GetResult()
	return &mdtick.BrokerInfo{
		HasAccountSummary: true,
		Balance:           decimal.NewFromFloat(s.GetBalance()),
		Credit:            decimal.NewFromFloat(s.GetCredit()),
		Equity:            decimal.NewFromFloat(s.GetEquity()),
		Margin:            decimal.NewFromFloat(s.GetMargin()),
		FreeMargin:        decimal.NewFromFloat(s.GetFreeMargin()),
		MarginLevel:       decimal.NewFromFloat(s.GetMarginLevel()),
		Profit:            decimal.NewFromFloat(s.GetProfit()),
		Leverage:          int32(s.GetLeverage()),
		CapturedAt:        Clk.Now(),
		AccountType:       mdtick.Mt4AccountTypeToString(int32(s.GetType())), // TRUST-1
	}, nil
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
