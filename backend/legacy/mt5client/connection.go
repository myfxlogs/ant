package mt5client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"anttrader/internal/config"
	pb "anttrader/mt5"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func NewMT5Client(cfg *config.MT5Config) *MT5Client {
	return &MT5Client{
		cfg:         cfg,
		connections: make(map[string]*MT5Connection),
	}
}

func (c *MT5Client) getSearchConn(ctx context.Context) (*grpc.ClientConn, error) {
	var connErr error
	c.searchConnMu.Do(func() {
		dialOpts := []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTimeout(c.cfg.Timeout),
		}
		if c.cfg.UseTLS {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		} else {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}
		addr := fmt.Sprintf("%s:%d", c.cfg.GatewayHost, c.cfg.GatewayPort)
		c.searchConn, connErr = grpc.DialContext(ctx, addr, dialOpts...)
	})
	return c.searchConn, connErr
}

func (c *MT5Client) Connect(ctx context.Context, user uint64, password, host string, port int32) (*MT5Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	accountID := uuid.New().String()

	cb := faultisol.NewCircuitBreaker(faultisol.CircuitBreakerConfig{
		Name:             "mt5-" + accountID[:8],
		FailureThreshold: 5,
		SuccessThreshold: 2,
		CooldownPeriod:   30 * time.Second,
	})

	executor := faultisol.NewIsolatedExecutor("mt5-"+accountID[:8],
		faultisol.WithCircuitBreaker(cb),
		faultisol.WithTimeoutConfig(faultisol.TimeoutConfig{
			Connect:   60 * time.Second,
			Query:     30 * time.Second,
			Trading:   30 * time.Second,
			Subscribe: 10 * time.Second,
			Default:   30 * time.Second,
		}),
	)

	conn := &MT5Connection{
		accountID:      accountID,
		state:          StateConnecting,
		done:           make(chan struct{}),
		circuitBreaker: cb,
		executor:       executor,
	}

	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTimeout(c.cfg.Timeout),
	}

	if c.cfg.UseTLS {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.GatewayHost, c.cfg.GatewayPort)
	grpcConn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		conn.state = StateDisconnected
		return nil, fmt.Errorf("failed to connect to MT5 gateway: %w", err)
	}

	conn.conn = grpcConn
	conn.connectionClient = pb.NewConnectionClient(grpcConn)
	conn.mt5Client = pb.NewMT5Client(grpcConn)
	conn.tradingClient = pb.NewTradingClient(grpcConn)
	conn.subscriptionClient = pb.NewSubscriptionsClient(grpcConn)
	conn.streamClient = pb.NewStreamsClient(grpcConn)

	tempID := uuid.New().String()
	ctxWithID := metadata.AppendToOutgoingContext(ctx, "id", tempID)

	connectReq := &pb.ConnectRequest{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
	}

	resp, err := conn.connectionClient.Connect(ctxWithID, connectReq)
	if err != nil {
		grpcConn.Close()
		conn.state = StateDisconnected
		return nil, fmt.Errorf("failed to connect to MT5 server: %w", err)
	}

	if resp.GetError() != nil {
		grpcConn.Close()
		conn.state = StateDisconnected
		return nil, fmt.Errorf("MT5 connection error: %s", resp.GetError().GetMessage())
	}

	conn.id = resp.GetResult()
	conn.state = StateReady
	conn.createdAt = time.Now()
	conn.lastActive = time.Now()

	c.connections[accountID] = conn

	go conn.heartbeatLoop()

	// 订阅实时流
	go func() {
		if err := conn.SubscribeOrderProfitStream(context.Background()); err != nil {
			logger.Error("MT5 订阅订单利润流失败",
				zap.String("account_id", accountID),
				zap.Error(err),
			)
		}
		if err := conn.SubscribeOrderUpdateStream(context.Background()); err != nil {
			logger.Error("MT5 订阅订单更新流失败",
				zap.String("account_id", accountID),
				zap.Error(err),
			)
		}
	}()

	return conn, nil
}

func (c *MT5Client) ConnectWithRetry(ctx context.Context, user uint64, password, host string, port int32) (*MT5Connection, error) {
	var lastErr error

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		connectCtx, cancel := context.WithTimeout(ctx, ConnectTimeout)
		conn, err := c.Connect(connectCtx, user, password, host, port)
		cancel()

		if err == nil {
			return conn, nil
		}

		lastErr = err
		logger.Warn("MT5 connection attempt failed, retrying...",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", MaxRetries),
			zap.Error(err),
		)

		if attempt < MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RetryDelay):
			}
		}
	}

	return nil, fmt.Errorf("MT5 connection failed after %d attempts: %w", MaxRetries, lastErr)
}

func (c *MT5Client) GetConnection(accountID string) (*MT5Connection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, exists := c.connections[accountID]
	return conn, exists
}

func (c *MT5Client) Disconnect(ctx context.Context, accountID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, exists := c.connections[accountID]
	if !exists {
		return nil
	}

	close(conn.done)

	if conn.conn != nil {
		conn.conn.Close()
	}

	if conn.IsConnected() {
		_, _ = conn.connectionClient.Disconnect(ctx, &pb.DisconnectRequest{
			Id: conn.id,
		})
	}

	delete(c.connections, accountID)
	conn.state = StateClosed

	return nil
}
