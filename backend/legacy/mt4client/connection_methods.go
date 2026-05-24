package mt4client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "anttrader/mt4"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (c *MT4Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateReady || c.state == StateSubscribed || c.state == StateDegraded
}

func (c *MT4Connection) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

func (c *MT4Connection) GetAccountID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountID
}

func (c *MT4Connection) GetStreamClient() pb.StreamsClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.streamClient
}

func (c *MT4Connection) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *MT4Connection) MarkDisconnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateDisconnected
	logger.Warn("Connection marked as disconnected",
		zap.String("account_id", c.accountID))
}

func (c *MT4Connection) MarkReady() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateReady
}

func (c *MT4Connection) MarkSubscribed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateSubscribed
}

func (c *MT4Connection) MarkDegraded() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != StateDisconnected && c.state != StateClosed {
		c.state = StateDegraded
	}
}

func (c *MT4Connection) CircuitBreakerState() faultisol.CircuitState {
	return c.circuitBreaker.State()
}

func (c *MT4Connection) markProfitRecvNow() {
	if c == nil {
		return
	}
	c.profitRecvAt.Store(time.Now().UnixNano())
}

func (c *MT4Connection) markOrderRecvNow() {
	if c == nil {
		return
	}
	c.orderRecvAt.Store(time.Now().UnixNano())
}

func (c *MT4Connection) LastProfitRecvAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	if v := c.profitRecvAt.Load(); v > 0 {
		return time.Unix(0, v)
	}
	return time.Time{}
}

func (c *MT4Connection) LastOrderRecvAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	if v := c.orderRecvAt.Load(); v > 0 {
		return time.Unix(0, v)
	}
	return time.Time{}
}

func (c *MT4Connection) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !c.checkConnection() {
				c.MarkDisconnected()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *MT4Connection) checkConnection() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.executor.Execute(ctx, func(ctx context.Context) error {
		ctx = c.contextWithID(ctx)
		_, err := c.connectionClient.CheckConnect(ctx, &pb.CheckConnectRequest{
			Id: c.token,
		})
		return err
	}, faultisol.WithOperation("heartbeat"))

	if err != nil {
		fields := []zap.Field{zap.String("account_id", c.accountID), zap.Error(err)}
		if st, ok := status.FromError(err); ok {
			fields = append(fields,
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
			)
		}
		logger.Warn("MT4 heartbeat failed", fields...)
		return false
	}
	return true
}

func (c *MT4Connection) contextWithID(ctx context.Context) context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return metadata.AppendToOutgoingContext(ctx, "id", c.token)
}

func (c *MT4Connection) safeCall(fn func() (interface{}, error)) (interface{}, error) {
	result := faultisol.SafeCallWithLog(fn,
		zap.String("account_id", c.accountID),
		zap.String("connection_state", fmt.Sprintf("%d", c.state)),
	)

	if result.Panic != nil {
		c.MarkDisconnected()
	}
	return result.Value, result.Error
}
