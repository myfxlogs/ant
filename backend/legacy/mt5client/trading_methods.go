package mt5client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pb "anttrader/mt5"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func isExpectedStreamCloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Canceled, codes.DeadlineExceeded:
			return true
		}
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "grpc: the client connection is closing") {
		return true
	}
	if strings.Contains(errStr, "context canceled") {
		return true
	}
	if strings.Contains(errStr, "canceled") {
		return true
	}
	return false
}

func (c *MT5Connection) OrderSend(ctx context.Context, symbol string, op pb.OrderType, volume, price, sl, tp float64, slippage int32, comment string, magic int64) (*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		req := &pb.OrderSendRequest{
			Id:         c.id,
			Symbol:     symbol,
			Operation:  op,
			Volume:     volume,
			Stoploss:   &sl,
			Takeprofit: &tp,
			Comment:    &comment,
			ExpertID:   &magic,
		}

		if price > 0 {
			req.Price = &price
		}
		if slippage > 0 {
			us := uint64(slippage)
			req.Slippage = &us
		}

		resp, err := c.tradingClient.OrderSend(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_send"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Order), nil
}

func (c *MT5Connection) OrderModify(ctx context.Context, ticket int64, sl, tp, price float64, expiration string) (*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		req := &pb.OrderModifyRequest{
			Id:         c.id,
			Ticket:     ticket,
			Stoploss:   sl,
			Takeprofit: tp,
		}

		if price > 0 {
			req.Price = &price
		}

		resp, err := c.tradingClient.OrderModify(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_modify"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Order), nil
}

func (c *MT5Connection) OrderClose(ctx context.Context, ticket int64, volume, price float64, slippage int32) (*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		req := &pb.OrderCloseRequest{
			Id:     c.id,
			Ticket: ticket,
		}

		if volume > 0 {
			req.Lots = &volume
		}
		if price > 0 {
			req.Price = &price
		}
		if slippage > 0 {
			us := uint64(slippage)
			req.Slippage = &us
		}

		resp, err := c.tradingClient.OrderClose(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_close"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Order), nil
}

func (c *MT5Connection) Subscribe(ctx context.Context, symbol string, interval int32) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		req := &pb.SubscribeRequest{
			Id:     c.id,
			Symbol: symbol,
		}

		if interval > 0 {
			req.Interval = &interval
		}

		_, err := c.subscriptionClient.Subscribe(ctx, req)
		return err
	}, faultisol.WithOperation("subscribe"), faultisol.WithTimeout(10*time.Second))
}

func (c *MT5Connection) SubscribeMany(ctx context.Context, symbols []string, interval int32) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		req := &pb.SubscribeManyRequest{
			Id:      c.id,
			Symbols: symbols,
		}

		if interval > 0 {
			req.Interval = &interval
		}

		_, err := c.subscriptionClient.SubscribeMany(ctx, req)
		return err
	}, faultisol.WithOperation("subscribe_many"), faultisol.WithTimeout(10*time.Second))
}

func (c *MT5Connection) Unsubscribe(ctx context.Context, symbol string) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		_, err := c.subscriptionClient.UnSubscribe(ctx, &pb.UnSubscribeRequest{
			Id:     c.id,
			Symbol: symbol,
		})
		return err
	}, faultisol.WithOperation("unsubscribe"), faultisol.WithTimeout(10*time.Second))
}

// SubscribeQuoteStream 订阅实时报价流
func (c *MT5Connection) SubscribeQuoteStream(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.quoteCh != nil {
		return nil // 已经订阅
	}

	ctx = c.contextWithID(ctx)
	stream, err := c.streamClient.OnQuote(ctx, &pb.OnQuoteRequest{Id: c.id})
	if err != nil {
		return fmt.Errorf("failed to subscribe to quote stream: %w", err)
	}

	c.quoteCh = make(chan *pb.Quote, 1000)
	quoteCh := c.quoteCh
	// NOTE: We already hold c.mu.Lock() here. Do NOT call c.MarkSubscribed()
	// — it re-acquires c.mu.Lock() on a non-recursive sync.RWMutex, causing a
	// permanent self-deadlock that blocks every subsequent c.mu.RLock()
	// (e.g. IsConnected, GetQuoteChannel). This manifested as MT5 GetPositions
	// hanging for 2 minutes once the strategy scheduler's retryHFSubscribeMT5
	// path invoked SubscribeQuoteStream.
	c.state = StateSubscribed

	// 启动 goroutine 接收报价
	go func() {
		defer func() {
			close(quoteCh)
			c.mu.Lock()
			if c.quoteCh == quoteCh {
				c.quoteCh = nil
			}
			c.mu.Unlock()
		}()
		for {
			select {
			case <-c.done:
				return
			default:
				reply, err := stream.Recv()
				if err != nil {
					if !isExpectedStreamCloseError(err) {
						c.MarkDegraded()
						logger.Error("MT5 quote stream error",
							zap.String("account_id", c.accountID),
							zap.Error(err),
						)
					}
					return
				}
				if reply.GetError() != nil {
					logger.Warn("OnQuote error",
						zap.String("account_id", c.accountID),
						zap.String("error", reply.GetError().GetMessage()),
					)
					continue
				}
				if quote := reply.GetResult(); quote != nil {
					select {
					case quoteCh <- quote:
					default:
						logger.Warn("MT5 quote channel full, dropping quote",
							zap.String("account_id", c.accountID),
							zap.String("symbol", quote.Symbol),
						)
					}
				}
			}
		}
	}()

	return nil
}

// GetQuoteChannel 获取实时报价通道
func (c *MT5Connection) GetQuoteChannel() <-chan *pb.Quote {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.quoteCh
}

// SubscribeOrderProfitStream 订阅订单利润流
func (c *MT5Connection) SubscribeOrderProfitStream(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.profitCh != nil {
		return nil // 已经订阅
	}

	ctxWithID := metadata.AppendToOutgoingContext(ctx, "id", c.id)
	stream, err := c.streamClient.OnOrderProfit(ctxWithID, &pb.OnOrderProfitRequest{Id: c.id})
	if err != nil {
		return fmt.Errorf("failed to subscribe order profit stream: %w", err)
	}

	// latest-wins: profit updates are snapshot-like; keep only the newest to avoid backlog
	c.profitCh = make(chan *pb.ProfitUpdate, 1)

	profitCh := c.profitCh
	go func() {
		defer func() {
			close(profitCh)
			c.mu.Lock()
			// reset so supervisor can re-subscribe after stream errors
			if c.profitCh == profitCh {
				c.profitCh = nil
			}
			c.mu.Unlock()
		}()
		lastDropWarnAt := time.Time{}
		for {
			select {
			case <-c.done:
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					if isExpectedStreamCloseError(err) {
						return
					}
					c.MarkDegraded()
					logger.Warn("OnOrderProfit stream error",
						zap.String("account_id", c.accountID),
						zap.Error(err))
					return
				}
				if resp.GetError() != nil {
					logger.Warn("OnOrderProfit error",
						zap.String("account_id", c.accountID),
						zap.String("error", resp.GetError().GetMessage()))
					continue
				}
				if profit := resp.GetResult(); profit != nil {
					c.markProfitRecvNow()
					select {
					case profitCh <- profit:
						// sent
					default:
						// overwrite old
						select {
						case <-profitCh:
						default:
						}
						select {
						case profitCh <- profit:
						default:
							if time.Since(lastDropWarnAt) > 5*time.Second {
								logger.Warn("MT5 profit channel full, dropping update",
									zap.String("account_id", c.accountID))
								lastDropWarnAt = time.Now()
							}
						}
					}
				}
			}
		}
	}()

	return nil
}

// SubscribeOrderUpdateStream 订阅订单更新流
func (c *MT5Connection) SubscribeOrderUpdateStream(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.orderCh != nil {
		return nil // 已经订阅
	}

	ctxWithID := c.contextWithID(ctx)
	stream, err := c.streamClient.OnOrderUpdate(ctxWithID, &pb.OnOrderUpdateRequest{Id: c.id})
	if err != nil {
		return fmt.Errorf("failed to subscribe order update stream: %w", err)
	}

	c.orderCh = make(chan *pb.OrderUpdateSummary, 1000)

	orderCh := c.orderCh
	go func() {
		defer func() {
			close(orderCh)
			c.mu.Lock()
			// reset so supervisor can re-subscribe after stream errors
			if c.orderCh == orderCh {
				c.orderCh = nil
			}
			c.mu.Unlock()
		}()
		lastDropWarnAt := time.Time{}
		for {
			select {
			case <-c.done:
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					if isExpectedStreamCloseError(err) {
						return
					}
					c.MarkDegraded()
					logger.Warn("OnOrderUpdate stream error",
						zap.String("account_id", c.accountID),
						zap.Error(err))
					return
				}
				if resp.GetError() != nil {
					logger.Warn("OnOrderUpdate error",
						zap.String("account_id", c.accountID),
						zap.String("error", resp.GetError().GetMessage()))
					continue
				}
				if order := resp.GetResult(); order != nil {
					c.markOrderRecvNow()
					select {
					case orderCh <- order:
					default:
						if time.Since(lastDropWarnAt) > 5*time.Second {
							logger.Warn("MT5 order channel full, dropping update",
								zap.String("account_id", c.accountID))
							lastDropWarnAt = time.Now()
						}
					}
				}
			}
		}
	}()

	return nil
}

// GetProfitChannel 获取利润通道
func (c *MT5Connection) GetProfitChannel() <-chan *pb.ProfitUpdate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profitCh
}

// GetOrderChannel 获取订单通道
func (c *MT5Connection) GetOrderChannel() <-chan *pb.OrderUpdateSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.orderCh
}
