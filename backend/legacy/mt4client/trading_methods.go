package mt4client

import (
	"context"
	"time"

	pb "anttrader/mt4"
	"anttrader/pkg/faultisol"
)

func (c *MT4Connection) OrderSend(ctx context.Context, symbol string, op pb.Op, volume, price, sl, tp float64, slippage int32, comment string, magic int32) (*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		req := &pb.OrderSendRequest{
			Id:         c.token,
			Symbol:     symbol,
			Operation:  op,
			Volume:     volume,
			Price:      price,
			Slippage:   slippage,
			Stoploss:   sl,
			Takeprofit: tp,
			Comment:    comment,
			Magic:      magic,
		}

		resp, err := c.tradingClient.OrderSend(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_send"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Order), nil
}

func (c *MT4Connection) OrderModify(ctx context.Context, ticket int32, sl, tp, price float64, expiration string) (*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		req := &pb.OrderModifyRequest{
			Id:         c.token,
			Ticket:     ticket,
			Stoploss:   sl,
			Takeprofit: tp,
			Price:      price,
		}

		resp, err := c.tradingClient.OrderModify(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_modify"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Order), nil
}

func (c *MT4Connection) OrderClose(ctx context.Context, ticket int32, volume, price float64, slippage int32) (*pb.Order, error) {
	result, err := c.executor.ExecuteWithValue(ctx, func(ctx context.Context) (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		req := &pb.OrderCloseRequest{
			Id:       c.token,
			Ticket:   ticket,
			Lots:     volume,
			Price:    price,
			Slippage: slippage,
		}

		resp, err := c.tradingClient.OrderClose(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.GetResult(), nil
	}, faultisol.WithOperation("order_close"), faultisol.WithTimeout(30*time.Second))

	if err != nil {
		return nil, err
	}
	return result.(*pb.Order), nil
}

func (c *MT4Connection) OrderDelete(ctx context.Context, ticket int32) error {
	return c.executor.Execute(ctx, func(ctx context.Context) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		ctx = c.contextWithID(ctx)

		_, err := c.tradingClient.OrderDelete(ctx, &pb.OrderDeleteRequest{
			Id:     c.token,
			Ticket: ticket,
		})
		return err
	}, faultisol.WithOperation("order_delete"), faultisol.WithTimeout(30*time.Second))
}
