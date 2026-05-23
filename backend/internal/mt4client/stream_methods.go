package mt4client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pb "anttrader/mt4"
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

type ProfitUpdateCallback func(update *pb.ProfitUpdate)
type OrderUpdateCallback func(update *pb.OrderUpdateSummary)
type QuoteCallback func(quote *pb.QuoteEventArgs)

func (c *MT4Connection) SubscribeOrderProfitStream(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.profitCh != nil {
		return nil // 已经订阅
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "id", c.token)
	stream, err := c.streamClient.OnOrderProfit(ctx, &pb.OnOrderProfitRequest{
		Id: c.token,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			logger.Warn("MT4 subscribe order profit stream failed",
				zap.String("account_id", c.accountID),
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to subscribe order profit stream: %w", err)
	}

	// latest-wins: profit update is a snapshot-like stream. We keep only the newest value
	// to avoid unbounded backlog when downstream is slower.
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
					fields := []zap.Field{zap.String("account_id", c.accountID), zap.Error(err)}
					if st, ok := status.FromError(err); ok {
						fields = append(fields,
							zap.String("grpc_code", st.Code().String()),
							zap.String("grpc_message", st.Message()),
						)
					}
					logger.Warn("OnOrderProfit stream error", fields...)
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
					// latest-wins: if the channel is full, drop the old value then try again.
					select {
					case profitCh <- profit:
						// sent
					default:
						select {
						case <-profitCh:
						default:
						}
						select {
						case profitCh <- profit:
							// sent after dropping old
						default:
							// still cannot send; warn but throttle to avoid log spam
							if time.Since(lastDropWarnAt) > 5*time.Second {
								logger.Warn("MT4 profit channel full, dropping update",
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

func (c *MT4Connection) SubscribeOrderUpdateStream(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.orderCh != nil {
		return nil // 已经订阅
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "id", c.token)
	stream, err := c.streamClient.OnOrderUpdate(ctx, &pb.OnOrderUpdateRequest{
		Id: c.token,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			logger.Warn("MT4 subscribe order update stream failed",
				zap.String("account_id", c.accountID),
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
				zap.Error(err),
			)
		}
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
					fields := []zap.Field{zap.String("account_id", c.accountID), zap.Error(err)}
					if st, ok := status.FromError(err); ok {
						fields = append(fields,
							zap.String("grpc_code", st.Code().String()),
							zap.String("grpc_message", st.Message()),
						)
					}
					logger.Warn("OnOrderUpdate stream error", fields...)
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
							logger.Warn("MT4 order channel full, dropping update",
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

func (c *MT4Connection) GetProfitChannel() <-chan *pb.ProfitUpdate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profitCh
}

func (c *MT4Connection) GetOrderChannel() <-chan *pb.OrderUpdateSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.orderCh
}

func (c *MT4Connection) SubscribeQuoteStream(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.quoteCh != nil {
		return nil // 已经订阅
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "id", c.token)
	stream, err := c.streamClient.OnQuote(ctx, &pb.OnQuoteRequest{
		Id: c.token,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			logger.Warn("MT4 subscribe quote stream failed",
				zap.String("account_id", c.accountID),
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to subscribe quote stream: %w", err)
	}

	c.quoteCh = make(chan *pb.QuoteEventArgs, 1000)
	c.MarkSubscribed()

	go func() {
		defer func() {
			close(c.quoteCh)
			c.mu.Lock()
			c.quoteCh = nil
			c.mu.Unlock()
		}()
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
					fields := []zap.Field{zap.String("account_id", c.accountID), zap.Error(err)}
					if st, ok := status.FromError(err); ok {
						fields = append(fields,
							zap.String("grpc_code", st.Code().String()),
							zap.String("grpc_message", st.Message()),
						)
					}
					logger.Error("MT4 quote stream error", fields...)
					return
				}
				if resp.GetError() != nil {
					logger.Warn("OnQuote error",
						zap.String("account_id", c.accountID),
						zap.String("error", resp.GetError().GetMessage()))
					continue
				}
				if quote := resp.GetResult(); quote != nil {
					select {
					case c.quoteCh <- quote:
					default:
						logger.Warn("MT4 quote channel full, dropping quote",
							zap.String("account_id", c.accountID),
							zap.String("symbol", quote.Symbol))
					}
				}
			}
		}
	}()

	return nil
}

// subscribePositionSymbols 订阅所有持仓品种的报价
func (c *MT4Connection) subscribePositionSymbols(ctx context.Context) {
	// 等待一小段时间确保连接已建立
	select {
	case <-time.After(2 * time.Second):
	case <-c.done:
		return
	}

	// 获取持仓订单
	orders, err := c.OpenedOrders(ctx)
	if err != nil {
		logger.Error("获取持仓订单失败",
			zap.String("account_id", c.accountID),
			zap.Error(err))
		return
	}

	// 收集所有持仓品种
	symbolSet := make(map[string]bool)
	for _, order := range orders {
		if order.Symbol != "" {
			symbolSet[order.Symbol] = true
		}
	}

	if len(symbolSet) == 0 {
		return
	}

	// 转换为切片
	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}

	// 批量订阅
	if len(symbols) > 0 {
		_, err := c.subscriptionClient.SubscribeMany(ctx, &pb.SubscribeManyRequest{
			Id:       c.token,
			Symbols:  symbols,
			Interval: 0,
		})
		if err != nil {
			logger.Error("订阅持仓品种失败",
				zap.String("account_id", c.accountID),
				zap.Strings("symbols", symbols),
				zap.Error(err))
		}
	}
}

func (c *MT4Connection) GetQuoteChannel() <-chan *pb.QuoteEventArgs {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.quoteCh
}
