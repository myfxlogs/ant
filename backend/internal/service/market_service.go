package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"anttrader/internal/config"
	"anttrader/internal/model"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
	"anttrader/internal/repository"

	v1 "anttrader/gen/proto"
)

type MarketService struct {
	accountRepo *repository.AccountRepository
	mt4Config   *config.MT4Config
	mt5Config   *config.MT5Config
	quoteCache  interface {
		GetQuote(ctx context.Context, accountID uuid.UUID, symbol string) (*v1.Quote, bool)
		SetQuote(ctx context.Context, accountID uuid.UUID, quote *v1.Quote, ttl time.Duration)
	}
}

func NewMarketService(
	accountRepo *repository.AccountRepository,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
) *MarketService {
	return &MarketService{
		accountRepo: accountRepo,
		mt4Config:   mt4Config,
		mt5Config:   mt5Config,
	}
}

func (s *MarketService) SetQuoteCache(cache interface {
	GetQuote(ctx context.Context, accountID uuid.UUID, symbol string) (*v1.Quote, bool)
	SetQuote(ctx context.Context, accountID uuid.UUID, quote *v1.Quote, ttl time.Duration)
}) {
	s.quoteCache = cache
}

type QuoteResponse struct {
	Symbol string  `json:"symbol"`
	Bid    float64 `json:"bid"`
	Ask    float64 `json:"ask"`
	Time   string  `json:"time"`
	Last   float64 `json:"last,omitempty"`
	Volume int64   `json:"volume,omitempty"`
}

type SymbolResponse struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Digits         int32   `json:"digits"`
	Point          float64 `json:"point"`
	Spread         int32   `json:"spread"`
	MinLot         float64 `json:"min_lot"`
	MaxLot         float64 `json:"max_lot"`
	LotStep        float64 `json:"lot_step"`
	ContractSize   float64 `json:"contract_size"`
	MarginRequired float64 `json:"margin_required"`
	Currency       string  `json:"currency"`
	CurrencyProfit string  `json:"currency_profit"`
	CurrencyMargin string  `json:"currency_margin"`
}

func (s *MarketService) GetQuote(ctx context.Context, userID, accountID uuid.UUID, symbol string) (*QuoteResponse, error) {
	account, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	if s.quoteCache != nil {
		if q, ok := s.quoteCache.GetQuote(ctx, accountID, symbol); ok && q != nil {
			// cached quote is protobuf; convert to REST response shape
			return &QuoteResponse{
				Symbol: q.Symbol,
				Bid:    q.Bid,
				Ask:    q.Ask,
				Time:   q.Time.AsTime().Format("2006-01-02T15:04:05Z"),
				Last:   q.Last,
				Volume: int64(q.Volume),
			}, nil
		}
	}

	host, port := s.parseHostPort(account.BrokerHost)
	loginInt, _ := strconv.ParseInt(account.Login, 10, 64)

	if account.MTType == "MT4" {
		resp, err := s.getQuoteMT4(ctx, account, loginInt, host, port, symbol)
		if err != nil {
			return nil, err
		}
		s.maybeCacheQuote(ctx, accountID, resp)
		return resp, nil
	}
	resp, err := s.getQuoteMT5(ctx, account, loginInt, host, port, symbol)
	if err != nil {
		return nil, err
	}
	s.maybeCacheQuote(ctx, accountID, resp)
	return resp, nil
}

func (s *MarketService) maybeCacheQuote(ctx context.Context, accountID uuid.UUID, resp *QuoteResponse) {
	if s.quoteCache == nil || resp == nil {
		return
	}
	// short TTL to smooth burst requests; actual streaming quotes should remain source of truth.
	ttl := 1 * time.Second

	q := &v1.Quote{
		Symbol: resp.Symbol,
		Bid:    resp.Bid,
		Ask:    resp.Ask,
		Last:   resp.Last,
		Volume: uint64(resp.Volume),
		Time:   timestamppb.Now(),
	}
	s.quoteCache.SetQuote(ctx, accountID, q, ttl)
}

func (s *MarketService) getQuoteMT4(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32, symbol string) (*QuoteResponse, error) {
	client := mt4client.NewMT4Client(s.mt4Config)
	conn, err := client.Connect(ctx, int32(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	quote, err := conn.Quote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	timeStr := ""
	if quote.GetTime() != nil {
		timeStr = quote.GetTime().AsTime().Format("2006-01-02T15:04:05Z")
	}

	return &QuoteResponse{
		Symbol: quote.GetSymbol(),
		Bid:    quote.GetBid(),
		Ask:    quote.GetAsk(),
		Time:   timeStr,
	}, nil
}

func (s *MarketService) getQuoteMT5(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32, symbol string) (*QuoteResponse, error) {
	client := mt5client.NewMT5Client(s.mt5Config)
	conn, err := client.Connect(ctx, uint64(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	quote, err := conn.Quote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	timeStr := ""
	if quote.GetTime() != nil {
		timeStr = quote.GetTime().AsTime().Format("2006-01-02T15:04:05Z")
	}

	return &QuoteResponse{
		Symbol: quote.GetSymbol(),
		Bid:    quote.GetBid(),
		Ask:    quote.GetAsk(),
		Time:   timeStr,
		Last:   quote.GetLast(),
		Volume: int64(quote.GetVolume()),
	}, nil
}

func (s *MarketService) GetQuotes(ctx context.Context, userID, accountID uuid.UUID, symbols []string) ([]*QuoteResponse, error) {
	account, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	host, port := s.parseHostPort(account.BrokerHost)
	loginInt, _ := strconv.ParseInt(account.Login, 10, 64)

	if account.MTType == "MT4" {
		return s.getQuotesMT4(ctx, account, loginInt, host, port, symbols)
	}
	return s.getQuotesMT5(ctx, account, loginInt, host, port, symbols)
}

func (s *MarketService) getQuotesMT4(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32, symbols []string) ([]*QuoteResponse, error) {
	client := mt4client.NewMT4Client(s.mt4Config)
	conn, err := client.Connect(ctx, int32(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	quotes, err := conn.QuoteMany(ctx, symbols)
	if err != nil {
		return nil, err
	}

	var result []*QuoteResponse
	for _, q := range quotes {
		timeStr := ""
		if q.GetTime() != nil {
			timeStr = q.GetTime().AsTime().Format("2006-01-02T15:04:05Z")
		}
		result = append(result, &QuoteResponse{
			Symbol: q.GetSymbol(),
			Bid:    q.GetBid(),
			Ask:    q.GetAsk(),
			Time:   timeStr,
		})
	}

	return result, nil
}

func (s *MarketService) getQuotesMT5(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32, symbols []string) ([]*QuoteResponse, error) {
	client := mt5client.NewMT5Client(s.mt5Config)
	conn, err := client.Connect(ctx, uint64(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	quotes, err := conn.QuoteMany(ctx, symbols)
	if err != nil {
		return nil, err
	}

	var result []*QuoteResponse
	for _, q := range quotes {
		timeStr := ""
		if q.GetTime() != nil {
			timeStr = q.GetTime().AsTime().Format("2006-01-02T15:04:05Z")
		}
		result = append(result, &QuoteResponse{
			Symbol: q.GetSymbol(),
			Bid:    q.GetBid(),
			Ask:    q.GetAsk(),
			Time:   timeStr,
			Last:   q.GetLast(),
			Volume: int64(q.GetVolume()),
		})
	}

	return result, nil
}

func (s *MarketService) GetSymbols(ctx context.Context, userID, accountID uuid.UUID) ([]*SymbolResponse, error) {
	account, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	host, port := s.parseHostPort(account.BrokerHost)
	loginInt, _ := strconv.ParseInt(account.Login, 10, 64)

	if account.MTType == "MT4" {
		return s.getSymbolsMT4(ctx, account, loginInt, host, port)
	}
	return s.getSymbolsMT5(ctx, account, loginInt, host, port)
}

func (s *MarketService) getSymbolsMT4(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32) ([]*SymbolResponse, error) {
	client := mt4client.NewMT4Client(s.mt4Config)
	conn, err := client.Connect(ctx, int32(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	symbols, err := conn.Symbols(ctx)
	if err != nil {
		return nil, err
	}

	var result []*SymbolResponse
	for _, sym := range symbols {
		result = append(result, &SymbolResponse{
			Name: sym,
		})
	}

	return result, nil
}

func (s *MarketService) getSymbolsMT5(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32) ([]*SymbolResponse, error) {
	client := mt5client.NewMT5Client(s.mt5Config)
	conn, err := client.Connect(ctx, uint64(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	symbolNames, err := conn.SymbolList(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*SymbolResponse, 0, len(symbolNames))
	for _, name := range symbolNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		result = append(result, &SymbolResponse{Name: name})
	}

	return result, nil
}

func (s *MarketService) GetSymbolParams(ctx context.Context, userID, accountID uuid.UUID, symbol string) (*SymbolResponse, error) {
	account, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	host, port := s.parseHostPort(account.BrokerHost)
	loginInt, _ := strconv.ParseInt(account.Login, 10, 64)

	if account.MTType == "MT4" {
		return s.getSymbolParamsMT4(ctx, account, loginInt, host, port, symbol)
	}
	return s.getSymbolParamsMT5(ctx, account, loginInt, host, port, symbol)
}

func (s *MarketService) getSymbolParamsMT4(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32, symbol string) (*SymbolResponse, error) {
	client := mt4client.NewMT4Client(s.mt4Config)
	conn, err := client.Connect(ctx, int32(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	params, err := conn.SymbolParams(ctx, symbol)
	if err != nil {
		return nil, err
	}

	info := params.GetSymbol()
	return &SymbolResponse{
		Name:           params.GetSymbolName(),
		Digits:         info.GetDigits(),
		Point:          info.GetPoint(),
		Spread:         info.GetSpread(),
		Currency:       info.GetCurrency(),
		CurrencyMargin: info.GetMarginCurrency(),
	}, nil
}

func (s *MarketService) getSymbolParamsMT5(ctx context.Context, account *model.MTAccount, loginInt int64, host string, port int32, symbol string) (*SymbolResponse, error) {
	client := mt5client.NewMT5Client(s.mt5Config)
	conn, err := client.Connect(ctx, uint64(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	params, err := conn.SymbolParams(ctx, symbol)
	if err != nil {
		return nil, err
	}

	info := params.GetSymbolInfo()
	return &SymbolResponse{
		Name:           params.GetSymbol(),
		Description:    info.GetDescription(),
		Digits:         info.GetDigits(),
		Spread:         info.GetSpread(),
		Currency:       info.GetCurrency(),
		CurrencyProfit: info.GetProfitCurrency(),
		CurrencyMargin: info.GetMarginCurrency(),
	}, nil
}

func (s *MarketService) getAccountAndVerify(ctx context.Context, userID, accountID uuid.UUID) (*model.MTAccount, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	if account.IsDisabled {
		return nil, errors.New("account is disabled")
	}
	return account, nil
}

func (s *MarketService) parseHostPort(hostPort string) (string, int32) {
	parts := strings.Split(hostPort, ":")
	if len(parts) == 2 {
		host := parts[0]
		port, _ := strconv.ParseInt(parts[1], 10, 32)
		return host, int32(port)
	}
	return hostPort, 443
}
