package mt4client

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt4"
	"anttrader/pkg/faultisol"
)

func (c *MT4Connection) Subscribe(ctx context.Context, symbol string) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		_, err := c.subscriptionClient.Subscribe(ctx, &pb.SubscribeRequest{
			Id:     c.token,
			Symbol: symbol,
		})
		return err
	}, faultisol.WithOperation("subscribe"), faultisol.WithTimeout(10*time.Second))
}

func (c *MT4Connection) Unsubscribe(ctx context.Context, symbol string) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		_, err := c.subscriptionClient.UnSubscribe(ctx, &pb.UnSubscribeRequest{
			Id:     c.token,
			Symbol: symbol,
		})
		return err
	}, faultisol.WithOperation("unsubscribe"), faultisol.WithTimeout(10*time.Second))
}

func (c *MT4Connection) UnsubscribeMany(ctx context.Context, symbols []string) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		_, err := c.subscriptionClient.UnSubscribeMany(ctx, &pb.UnSubscribeManyRequest{
			Id:      c.token,
			Symbols: symbols,
		})
		return err
	}, faultisol.WithOperation("unsubscribe_many"), faultisol.WithTimeout(10*time.Second))
}

func (c *MT4Connection) SubscribeQuoteHistory(ctx context.Context) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		_, err := c.subscriptionClient.SubscribeQuoteHistory(ctx, &pb.SubscribeQuoteHistoryRequest{
			Id: c.token,
		})
		return err
	}, faultisol.WithOperation("subscribe_quote_history"), faultisol.WithTimeout(10*time.Second))
}

func (c *MT4Connection) QuoteHistory(ctx context.Context, symbol string, timeframe pb.Timeframe, from string, count int32) ([]*pb.Bar, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.QuoteHistory(ctx, &pb.QuoteHistoryRequest{
			Id:        c.token,
			Symbol:    symbol,
			Timeframe: timeframe,
			From:      from,
			Count:     count,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("quote_history"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Bar), nil
}

func (c *MT4Connection) QuoteHistoryMany(ctx context.Context, symbols []string, timeframe pb.Timeframe, from string, count int32) ([]*pb.BarsForSymbol, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.QuoteHistoryMany(ctx, &pb.QuoteHistoryManyRequest{
			Id:        c.token,
			Symbol:    symbols,
			Timeframe: timeframe,
			From:      from,
			Count:     count,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("quote_history_many"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.BarsForSymbol), nil
}

func (c *MT4Connection) OrderHistory(ctx context.Context, from, to string) ([]*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.OrderHistory(ctx, &pb.OrderHistoryRequest{
			Id:   c.token,
			From: from,
			To:   to,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_history"), faultisol.WithTimeout(60*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Order), nil
}
