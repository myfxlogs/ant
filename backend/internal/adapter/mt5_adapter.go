package adapter

import (
	"context"

	mt5pb "anttrader/mt5"
	"anttrader/internal/model/mt5"
	"anttrader/internal/mt5client"
)

type MT5Adapter struct {
	conn *mt5client.MT5Connection
}

func NewMT5Adapter(conn *mt5client.MT5Connection) *MT5Adapter {
	return &MT5Adapter{
		conn: conn,
	}
}

func (a *MT5Adapter) GetAccountSummary(ctx context.Context) (*mt5.Account, error) {
	pbAccount, err := a.conn.AccountSummary(ctx)
	if err != nil {
		return nil, err
	}

	return a.convertToMT5Account(pbAccount), nil
}

func (a *MT5Adapter) GetOpenedOrders(ctx context.Context) ([]*mt5.Position, error) {
	pbOrders, err := a.conn.OpenedOrders(ctx)
	if err != nil {
		return nil, err
	}

	return a.convertToMT5Positions(pbOrders), nil
}

func (a *MT5Adapter) GetQuote(ctx context.Context, symbol string) (*mt5.Quote, error) {
	pbQuote, err := a.conn.Quote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return a.convertToMT5Quote(pbQuote), nil
}

func (a *MT5Adapter) GetQuotes(ctx context.Context, symbols []string) ([]*mt5.Quote, error) {
	var pbQuotes []*mt5pb.Quote
	var err error

	if len(symbols) == 1 && symbols[0] == "ALL" {
		symbolList, err := a.conn.SymbolList(ctx)
		if err != nil {
			return nil, err
		}
		
		quotes := make([]*mt5.Quote, 0, len(symbolList))
		for _, symbol := range symbolList {
			quote, err := a.conn.Quote(ctx, symbol)
			if err != nil {
				continue
			}
			quotes = append(quotes, a.convertToMT5Quote(quote))
		}
		return quotes, nil
	}

	pbQuotes, err = a.conn.QuoteMany(ctx, symbols)
	if err != nil {
		return nil, err
	}

	quotes := make([]*mt5.Quote, len(pbQuotes))
	for i, pb := range pbQuotes {
		quotes[i] = a.convertToMT5Quote(pb)
	}

	return quotes, nil
}

func (a *MT5Adapter) convertToMT5Account(pb *mt5pb.AccountSummary) *mt5.Account {
	return &mt5.Account{
		Balance:     pb.Balance,
		Credit:      pb.Credit,
		Profit:      pb.Profit,
		Equity:      pb.Equity,
		Margin:      pb.Margin,
		FreeMargin:  pb.FreeMargin,
		MarginLevel: pb.MarginLevel,
		Leverage:    int32(pb.Leverage),
		Currency:    pb.Currency,
		Method:      mt5.AccMethod(pb.Method),
		Type:        pb.Type,
		IsInvestor:  pb.IsInvestor,
	}
}

func (a *MT5Adapter) convertToMT5Positions(pbOrders []*mt5pb.Order) []*mt5.Position {
	positions := make([]*mt5.Position, 0, len(pbOrders))
	for _, pb := range pbOrders {
		closeTime := pb.CloseTime.AsTime()
		expirationTime := pb.ExpirationTime.AsTime()
		positions = append(positions, &mt5.Position{
			Ticket:            pb.Ticket,
			Symbol:            pb.Symbol,
			OrderType:         mt5.OrderType(pb.OrderType),
			DealType:          mt5.DealType(pb.DealType),
			Volume:            pb.Lots,
			OpenPrice:         pb.OpenPrice,
			ClosePrice:        pb.ClosePrice,
			StopLoss:          pb.StopLoss,
			TakeProfit:        pb.TakeProfit,
			OpenTime:          pb.OpenTime.AsTime(),
			CloseTime:         &closeTime,
			ExpirationTime:    &expirationTime,
			MagicNumber:       pb.ExpertId,
			Swap:              pb.Swap,
			Commission:        pb.Commission,
			Fee:               pb.Fee,
			OrderComment:      pb.Comment,
			Profit:            pb.Profit,
			ProfitRate:        pb.ProfitRate,
			PlacedType:        mt5.PlacedType(pb.PlacedType),
			State:             mt5.OrderState(pb.State),
			ContractSize:      pb.ContractSize,
			CloseVolume:       float64(pb.CloseVolume),
			CloseLots:         pb.CloseLots,
			CloseComment:      pb.CloseComment,
			StopLimitPrice:    pb.StopLimitPrice,
			ExpertId:          pb.ExpertId,
		})
	}
	return positions
}

func (a *MT5Adapter) convertToMT5Quote(pb *mt5pb.Quote) *mt5.Quote {
	return &mt5.Quote{
		Symbol: pb.Symbol,
		Bid:    pb.Bid,
		Ask:    pb.Ask,
		Time:   pb.Time,
		Last:   pb.Last,
		Volume: pb.Volume,
	}
}
