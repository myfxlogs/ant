package mt4client

import (
	"context"
	"fmt"

	pb "anttrader/mt4"
	"anttrader/pkg/faultisol"
)

func (c *MT4Connection) AccountSummary(ctx context.Context) (*pb.AccountSummary, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.AccountSummary(ctx, &pb.AccountSummaryRequest{
			Id: c.token,
		})
		if err != nil {
			return nil, err
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("account_summary"))

	if err != nil {
		return nil, err
	}
	return result.(*pb.AccountSummary), nil
}

func (c *MT4Connection) OpenedOrders(ctx context.Context) ([]*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{
			Id: c.token,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("opened_orders"))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Order), nil
}

func (c *MT4Connection) Quote(ctx context.Context, symbol string) (*pb.QuoteEventArgs, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.Quote(ctx, &pb.QuoteRequest{
			Id:     c.token,
			Symbol: symbol,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("quote"))

	if err != nil {
		return nil, err
	}
	return result.(*pb.QuoteEventArgs), nil
}

func (c *MT4Connection) Symbols(ctx context.Context) ([]string, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.Symbols(ctx, &pb.SymbolsRequest{
			Id: c.token,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("symbols"))

	if err != nil {
		return nil, err
	}

	symbols := result.([]string)
	return symbols, nil
}

func (c *MT4Connection) QuoteMany(ctx context.Context, symbols []string) ([]*pb.QuoteEventArgs, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.GetQuoteMany(ctx, &pb.GetQuoteManyRequest{
			Id:      c.token,
			Symbols: symbols,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT4 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("quote_many"))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.QuoteEventArgs), nil
}

func (c *MT4Connection) SymbolParams(ctx context.Context, symbol string) (*pb.SymbolParams, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		resp, err := c.mt4Client.SymbolParams(ctx, &pb.SymbolParamsRequest{
			Id:     c.token,
			Symbol: symbol,
		})
		if err != nil {
			return nil, err
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("symbol_params"))

	if err != nil {
		return nil, err
	}
	return result.(*pb.SymbolParams), nil
}
