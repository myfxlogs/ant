package mt4

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type Gateway struct {
	cfg                 mdtick.AccountConfig
	log                 *zap.Logger
	mu                  sync.RWMutex
	conn                *grpc.ClientConn
	client              pb.MT4Client
	connCli             pb.ConnectionClient
	streamCli           pb.StreamsClient
	subCli              pb.SubscriptionsClient
	tradingCli          pb.TradingClient
	sessionID           string
	cancelSub           context.CancelFunc
	cancelProfitSub     context.CancelFunc
	cancelOrderUpdateSub context.CancelFunc
}

func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway {
	return &Gateway{cfg: cfg, log: log}
}

func (g *Gateway) Platform() string  { return "mt4" }
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
	g.mu.Unlock()

	tempID := "mdgw-" + g.cfg.Login
	md := metadata.New(map[string]string{"id": tempID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	brokerHost := g.cfg.BrokerHost
	if idx := strings.LastIndex(brokerHost, ":"); idx > 0 {
		brokerHost = brokerHost[:idx]
	}
	loginResp, err := g.connCli.Connect(loginCtx, &pb.ConnectRequest{
		Host: brokerHost, Port: 443, User: int32(strToInt(g.cfg.Login)),
		Password: g.cfg.Password, Id: &tempID,
	})
	if err != nil {
		g.conn.Close()
		g.conn = nil
		return fmt.Errorf("mt4 login: %w", err)
	}
	token := loginResp.GetResult()
	respErr := loginResp.GetError()
	g.log.Info("mt4 connect response",
		zap.String("token", token), zap.Any("error", respErr),
		zap.String("host", brokerHost), zap.String("gateway", gateway))
	if token == "" {
		errMsg := "empty token"
		if respErr != nil {
			errMsg = fmt.Sprintf("code=%d msg=%s", respErr.GetCode(), respErr.GetMessage())
		}
		g.conn.Close()
		g.conn = nil
		return fmt.Errorf("mt4 login: %s", errMsg)
	}
	g.mu.Lock()
	g.sessionID = token
	g.mu.Unlock()
	return nil
}

func (g *Gateway) Disconnect(ctx context.Context) error {
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
	if g.conn != nil {
		g.conn.Close()
		g.conn = nil
	}
	g.client = nil
	g.connCli = nil
	g.streamCli = nil
	g.subCli = nil
	g.tradingCli = nil
	g.sessionID = ""
	return nil
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

	resp, err := client.AccountSummary(ctx, &pb.AccountSummaryRequest{Id: sid})
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

func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.conn == nil {
		return fmt.Errorf("mt4: not connected")
	}
	return nil
}

func (g *Gateway) SessionID() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sessionID
}

func (g *Gateway) MT4Client() pb.MT4Client {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.client
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
