package mt4client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"anttrader/internal/config"
	pb "anttrader/mt4"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func NewMT4Client(cfg *config.MT4Config) *MT4Client {
	return &MT4Client{
		cfg:         cfg,
		connections: make(map[string]*MT4Connection),
	}
}

func (c *MT4Client) getSearchConn(ctx context.Context) (*grpc.ClientConn, error) {
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

func (c *MT4Client) Connect(ctx context.Context, user int32, password, host string, port int32) (*MT4Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	accountID := uuid.New().String()

	cb := faultisol.NewCircuitBreaker(faultisol.CircuitBreakerConfig{
		Name:             "mt4-" + accountID[:8],
		FailureThreshold: 5,
		SuccessThreshold: 2,
		CooldownPeriod:   30 * time.Second,
	})

	executor := faultisol.NewIsolatedExecutor("mt4-"+accountID[:8],
		faultisol.WithCircuitBreaker(cb),
		faultisol.WithTimeoutConfig(faultisol.TimeoutConfig{
			Connect:   60 * time.Second,
			Query:     30 * time.Second,
			Trading:   30 * time.Second,
			Subscribe: 10 * time.Second,
			Default:   30 * time.Second,
		}),
	)

	conn := &MT4Connection{
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
		if st, ok := status.FromError(err); ok {
			logger.Error("MT4 dial gateway failed",
				zap.String("gateway_addr", addr),
				zap.Bool("use_tls", c.cfg.UseTLS),
				zap.Duration("timeout", c.cfg.Timeout),
				zap.Int32("login", user),
				zap.String("mt_host", host),
				zap.Int32("mt_port", port),
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
			)
		}
		return nil, fmt.Errorf("failed to connect to MT4 gateway: %w", err)
	}

	conn.conn = grpcConn
	conn.connectionClient = pb.NewConnectionClient(grpcConn)
	conn.mt4Client = pb.NewMT4Client(grpcConn)
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
		Id:       &tempID,
	}

	resp, err := conn.connectionClient.Connect(ctxWithID, connectReq)
	if err != nil {
		grpcConn.Close()
		conn.state = StateDisconnected
		if st, ok := status.FromError(err); ok {
			logger.Error("MT4 Connect RPC failed",
				zap.String("gateway_addr", addr),
				zap.Bool("use_tls", c.cfg.UseTLS),
				zap.Duration("timeout", c.cfg.Timeout),
				zap.Int32("login", user),
				zap.String("mt_host", host),
				zap.Int32("mt_port", port),
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
			)
			if st.Code() == codes.InvalidArgument {
				return nil, fmt.Errorf("failed to connect to MT4 server: %s", st.Message())
			}
			return nil, fmt.Errorf("failed to connect to MT4 server: %s", st.Message())
		}
		return nil, fmt.Errorf("failed to connect to MT4 server: %w", err)
	}

	if resp.GetError() != nil {
		grpcConn.Close()
		conn.state = StateDisconnected
		logger.Error("MT4 Connect returned application error",
			zap.String("gateway_addr", addr),
			zap.Bool("use_tls", c.cfg.UseTLS),
			zap.Duration("timeout", c.cfg.Timeout),
			zap.Int32("login", user),
			zap.String("mt_host", host),
			zap.Int32("mt_port", port),
			zap.String("error", resp.GetError().GetMessage()),
		)
		return nil, fmt.Errorf("MT4 connection error: %s", resp.GetError().GetMessage())
	}

	conn.token = resp.GetResult()
	if conn.token == "" {
		grpcConn.Close()
		conn.state = StateDisconnected
		return nil, fmt.Errorf("MT4 connection returned empty token")
	}
	conn.state = StateReady
	conn.createdAt = time.Now()
	conn.lastActive = time.Now()

	c.connections[accountID] = conn

	go conn.heartbeatLoop()

	// 订阅实时流
	go func() {
		if err := conn.SubscribeOrderProfitStream(context.Background()); err != nil {
			fields := []zap.Field{zap.String("account_id", accountID), zap.Error(err)}
			if st, ok := status.FromError(err); ok {
				fields = append(fields,
					zap.String("grpc_code", st.Code().String()),
					zap.String("grpc_message", st.Message()),
				)
			}
			logger.Error("MT4 订阅订单利润流失败", fields...)
		}
		if err := conn.SubscribeOrderUpdateStream(context.Background()); err != nil {
			fields := []zap.Field{zap.String("account_id", accountID), zap.Error(err)}
			if st, ok := status.FromError(err); ok {
				fields = append(fields,
					zap.String("grpc_code", st.Code().String()),
					zap.String("grpc_message", st.Message()),
				)
			}
			logger.Error("MT4 订阅订单更新流失败", fields...)
		}
	}()

	return conn, nil
}

func (c *MT4Client) GetConnection(accountID string) (*MT4Connection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, exists := c.connections[accountID]
	return conn, exists
}

func (c *MT4Client) Disconnect(ctx context.Context, accountID string) error {
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
			Id: conn.token,
		})
	}

	delete(c.connections, accountID)
	conn.state = StateClosed

	return nil
}
