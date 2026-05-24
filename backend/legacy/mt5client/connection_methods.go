package mt5client

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"

	pb "anttrader/mt5"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (c *MT5Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateReady || c.state == StateSubscribed || c.state == StateDegraded
}

func (c *MT5Connection) GetID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

func (c *MT5Connection) GetAccountID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountID
}

func (c *MT5Connection) GetStreamClient() pb.StreamsClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.streamClient
}

func (c *MT5Connection) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *MT5Connection) MarkDisconnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateDisconnected
	logger.Warn("Connection marked as disconnected",
		zap.String("account_id", c.accountID))
}

func (c *MT5Connection) MarkReady() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateReady
}

func (c *MT5Connection) MarkSubscribed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateSubscribed
}

func (c *MT5Connection) MarkDegraded() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != StateDisconnected && c.state != StateClosed {
		c.state = StateDegraded
	}
}

func (c *MT5Connection) CircuitBreakerState() faultisol.CircuitState {
	return c.circuitBreaker.State()
}

func (c *MT5Connection) markProfitRecvNow() {
	if c == nil {
		return
	}
	c.profitRecvAt.Store(time.Now().UnixNano())
}

func (c *MT5Connection) markOrderRecvNow() {
	if c == nil {
		return
	}
	c.orderRecvAt.Store(time.Now().UnixNano())
}

func (c *MT5Connection) LastProfitRecvAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	if v := c.profitRecvAt.Load(); v > 0 {
		return time.Unix(0, v)
	}
	return time.Time{}
}

func (c *MT5Connection) LastOrderRecvAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	if v := c.orderRecvAt.Load(); v > 0 {
		return time.Unix(0, v)
	}
	return time.Time{}
}

func (c *MT5Connection) heartbeatLoop() {
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

func (c *MT5Connection) checkConnection() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.executor.Execute(ctx, func(ctx context.Context) error {
		_, err := c.connectionClient.CheckConnect(ctx, &pb.CheckConnectRequest{
			Id: c.id,
		})
		return err
	}, faultisol.WithOperation("heartbeat"), faultisol.WithTimeout(5*time.Second))

	if err != nil {
		logger.Warn("MT5 heartbeat failed",
			zap.String("account_id", c.accountID),
			zap.Error(err),
		)
		return false
	}
	return true
}

func (c *MT5Connection) contextWithID(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "id", c.id)
}
