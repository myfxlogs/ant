package adapter

import (
	"context"
	"time"

	"go.uber.org/zap"

	mt4pb "anttrader/mt4"
	"anttrader/internal/model/mt4"
	"anttrader/internal/mt4client"
)

var logger = zap.L()

type MT4Adapter struct {
	conn *mt4client.MT4Connection
}

func NewMT4Adapter(conn *mt4client.MT4Connection) *MT4Adapter {
	return &MT4Adapter{
		conn: conn,
	}
}

func (a *MT4Adapter) GetAccountSummary(ctx context.Context) (*mt4.Account, error) {
	pbAccount, err := a.conn.AccountSummary(ctx)
	if err != nil {
		return nil, err
	}

	return a.convertToMT4Account(pbAccount), nil
}

func (a *MT4Adapter) GetOpenedOrders(ctx context.Context) ([]*mt4.Position, error) {
	pbOrders, err := a.conn.OpenedOrders(ctx)
	if err != nil {
		return nil, err
	}

	return a.convertToMT4Positions(pbOrders), nil
}

func (a *MT4Adapter) GetQuote(ctx context.Context, symbol string) (*mt4.Quote, error) {
	pbQuote, err := a.conn.Quote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return a.convertToMT4Quote(pbQuote), nil
}

func (a *MT4Adapter) GetQuotes(ctx context.Context, symbols []string) ([]*mt4.Quote, error) {
	var pbQuotes []*mt4pb.QuoteEventArgs
	var err error

	if len(symbols) == 1 && symbols[0] == "ALL" {
		symbolList, err := a.conn.Symbols(ctx)
		if err != nil {
			return nil, err
		}
		
		quotes := make([]*mt4.Quote, 0, len(symbolList))
		for _, symbol := range symbolList {
			quote, err := a.conn.Quote(ctx, symbol)
			if err != nil {
				continue
			}
			quotes = append(quotes, a.convertToMT4Quote(quote))
		}
		return quotes, nil
	}

	pbQuotes, err = a.conn.QuoteMany(ctx, symbols)
	if err != nil {
		return nil, err
	}

	quotes := make([]*mt4.Quote, len(pbQuotes))
	for i, pb := range pbQuotes {
		quotes[i] = a.convertToMT4Quote(pb)
	}

	return quotes, nil
}

func (a *MT4Adapter) convertToMT4Account(pb *mt4pb.AccountSummary) *mt4.Account {
	return &mt4.Account{
		Balance:     pb.Balance,
		Credit:      pb.Credit,
		Profit:      pb.Profit,
		Equity:      pb.Equity,
		Margin:      pb.Margin,
		FreeMargin:  pb.FreeMargin,
		MarginLevel: pb.MarginLevel,
		Leverage:    int32(pb.Leverage),
		Currency:    pb.Currency,
		Type:        mt4.AccountType(pb.Type),
		IsInvestor:  pb.IsInvestor,
	}
}

func (a *MT4Adapter) convertToMT4Positions(pbOrders []*mt4pb.Order) []*mt4.Position {
	positions := make([]*mt4.Position, 0, len(pbOrders))
	for _, pb := range pbOrders {
		closeTime := time.Time{}
		if pb.CloseTime != nil {
			closeTime = pb.CloseTime.AsTime()
		}
		expiration := time.Time{}
		if pb.Expiration != nil {
			expiration = pb.Expiration.AsTime()
		}
		positions = append(positions, &mt4.Position{
			Ticket:       pb.Ticket,
			Symbol:       pb.Symbol,
			OrderType:    mt4.OrderType(pb.Type),
			Volume:       pb.Lots,
			OpenPrice:    pb.OpenPrice,
			ClosePrice:   pb.ClosePrice,
			StopLoss:     pb.StopLoss,
			TakeProfit:   pb.TakeProfit,
			OpenTime:     pb.OpenTime.AsTime(),
			CloseTime:    &closeTime,
			Expiration:   &expiration,
			MagicNumber:  pb.MagicNumber,
			Swap:         pb.Swap,
			Commission:   pb.Commission,
			OrderComment: pb.Comment,
			Profit:       pb.Profit,
			RateOpen:     pb.RateOpen,
			RateClose:    pb.RateClose,
			RateMargin:   pb.RateMargin,
			PlacedType:   mt4.PlacedType(pb.PlacedType),
		})
	}
	return positions
}

func (a *MT4Adapter) convertToMT4Quote(pb *mt4pb.QuoteEventArgs) *mt4.Quote {
	return &mt4.Quote{
		Symbol: pb.Symbol,
		Bid:    pb.Bid,
		Ask:    pb.Ask,
		Time:   pb.Time,
		High:   pb.High,
		Low:    pb.Low,
	}
}
