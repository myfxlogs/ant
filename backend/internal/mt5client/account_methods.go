package mt5client

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt5"
	"anttrader/pkg/faultisol"
)

func (c *MT5Connection) AccountSummary(ctx context.Context) (*pb.AccountSummary, error) {
	// Do not hold mu during gRPC: executor can return on timeout while the call still runs;
	// holding RLock would block other methods (e.g. GetProfitChannel) that need the lock.
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.AccountSummary(ctx, &pb.AccountSummaryRequest{
			Id: c.id,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("account_summary"), faultisol.WithTimeout(15*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.AccountSummary), nil
}

func (c *MT5Connection) OpenedOrders(ctx context.Context) ([]*pb.Order, error) {
	// Do not hold mu during gRPC: IsolatedExecutor can return on timeout while the call
	// still runs; holding RLock would block every subsequent OpenedOrders on this connection.
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{
			Id: c.id,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("opened_orders"), faultisol.WithTimeout(25*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Order), nil
}

func (c *MT5Connection) OrderHistory(ctx context.Context, from, to string) ([]*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.OrderHistory(ctx, &pb.OrderHistoryRequest{
			Id:     c.id,
			From:   from,
			To:     to,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_history"))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Order), nil
}

func (c *MT5Connection) AccountBalance(ctx context.Context) (*pb.AccountRec, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.Account(ctx, &pb.AccountRequest{
			Id: c.id,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("account_balance"))

	if err != nil {
		return nil, err
	}
	return result.(*pb.AccountRec), nil
}

func (c *MT5Connection) AccountInfo(ctx context.Context) (*pb.AccountSummary, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.AccountSummary(ctx, &pb.AccountSummaryRequest{
			Id: c.id,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("account_info"))

	if err != nil {
		return nil, err
	}
	return result.(*pb.AccountSummary), nil
}

func (c *MT5Connection) SymbolParams(ctx context.Context, symbol string) (*pb.SymbolParams, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.SymbolParams(ctx, &pb.SymbolParamsRequest{
			Id:     c.id,
			Symbol: symbol,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("symbol_params"))

	if err != nil {
		return nil, err
	}
	return result.(*pb.SymbolParams), nil
}

func (c *MT5Connection) PriceHistory(ctx context.Context, symbol string, timeframe int32, from, to string) ([]*pb.Bar, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		quoteHistoryClient := pb.NewQuoteHistoryClient(c.conn)

		resp, err := quoteHistoryClient.PriceHistory(ctx, &pb.PriceHistoryRequest{
			Id:        c.id,
			Symbol:    symbol,
			TimeFrame: timeframe,
			From:      from,
			To:        to,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("price_history"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Bar), nil
}

func (c *MT5Connection) PriceHistoryEx(ctx context.Context, symbol string, timeframe int32, from string, numBars int32) ([]*pb.Bar, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		quoteHistoryClient := pb.NewQuoteHistoryClient(c.conn)

		resp, err := quoteHistoryClient.PriceHistoryEx(ctx, &pb.PriceHistoryExRequest{
			Id:        c.id,
			Symbol:    symbol,
			TimeFrame: timeframe,
			From:      from,
			NumBars:   numBars,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("price_history_ex"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Bar), nil
}

func (c *MT5Connection) Quote(ctx context.Context, symbol string) (*pb.Quote, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		msNotOlder := int32(0)
		resp, err := c.mt5Client.GetQuote(ctx, &pb.GetQuoteRequest{
			Id:         c.id,
			Symbol:     symbol,
			MsNotOlder: &msNotOlder,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("quote"), faultisol.WithTimeout(10*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Quote), nil
}

func (c *MT5Connection) QuoteMany(ctx context.Context, symbols []string) ([]*pb.Quote, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		msNotOlder := int32(0)
		resp, err := c.mt5Client.GetQuoteMany(ctx, &pb.GetQuoteManyRequest{
			Id:         c.id,
			Symbols:    symbols,
			MsNotOlder: &msNotOlder,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("quote_many"), faultisol.WithTimeout(10*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.Quote), nil
}

func (c *MT5Connection) Symbols(ctx context.Context) ([]*pb.SymbolInfo, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.Symbols(ctx, &pb.SymbolsRequest{
			Id: c.id,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("symbols"), faultisol.WithTimeout(10*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]*pb.SymbolInfo), nil
}

func (c *MT5Connection) SymbolList(ctx context.Context) ([]string, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)
		resp, err := c.mt5Client.SymbolList(ctx, &pb.SymbolListRequest{
			Id: c.id,
		})
		if err != nil {
			return nil, err
		}
		if resp.GetError() != nil {
			return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("symbol_list"), faultisol.WithTimeout(10*time.Second))

	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}
