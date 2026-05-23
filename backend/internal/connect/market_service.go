package connect

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type MarketService struct {
	accountRepo *repository.AccountRepository
	marketSvc   *service.MarketService
	klineSvc    *service.KlineService
}

func NewMarketService(accountRepo *repository.AccountRepository, marketSvc *service.MarketService, klineSvc *service.KlineService) *MarketService {
	return &MarketService{
		accountRepo: accountRepo,
		marketSvc:   marketSvc,
		klineSvc:    klineSvc,
	}
}

func (s *MarketService) GetQuote(ctx context.Context, req *connect.Request[v1.GetQuoteRequest]) (*connect.Response[v1.Quote], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeMarketRead); err != nil {
		return nil, err
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	quote, err := s.marketSvc.GetQuote(ctx, userID, accountID, req.Msg.Symbol)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.Quote{
		Symbol: quote.Symbol,
		Bid:    quote.Bid,
		Ask:    quote.Ask,
		Time:   parseTimeToProto(quote.Time),
		Last:   quote.Last,
		Volume: uint64(quote.Volume),
	}), nil
}

func (s *MarketService) GetQuotes(ctx context.Context, req *connect.Request[v1.GetQuotesRequest]) (*connect.Response[v1.GetQuotesResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeMarketRead); err != nil {
		return nil, err
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	quotes := make([]*v1.Quote, len(req.Msg.Symbols))
	for i, symbol := range req.Msg.Symbols {
		quote, err := s.marketSvc.GetQuote(ctx, userID, accountID, symbol)
		if err != nil {
			continue
		}
		quotes[i] = &v1.Quote{
			Symbol: quote.Symbol,
			Bid:    quote.Bid,
			Ask:    quote.Ask,
			Time:   parseTimeToProto(quote.Time),
			Last:   quote.Last,
			Volume: uint64(quote.Volume),
		}
	}

	return connect.NewResponse(&v1.GetQuotesResponse{
		Quotes: quotes,
	}), nil
}

func (s *MarketService) GetSymbols(ctx context.Context, req *connect.Request[v1.GetSymbolsRequest]) (*connect.Response[v1.GetSymbolsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	symbols, err := s.marketSvc.GetSymbols(ctx, userID, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetSymbolsResponse{
		Symbols: make([]*v1.SymbolInfo, len(symbols)),
	}

	for i, sym := range symbols {
		response.Symbols[i] = &v1.SymbolInfo{
			Symbol:       sym.Name,
			Description:  sym.Description,
			Currency:     sym.Currency,
			Digits:       sym.Digits,
			TickSize:     sym.Point,
			TickValue:    sym.Point,
			ContractSize: sym.ContractSize,
			MinLot:       sym.MinLot,
			MaxLot:       sym.MaxLot,
			LotStep:      sym.LotStep,
		}
	}

	return connect.NewResponse(response), nil
}

func (s *MarketService) GetSymbolParams(ctx context.Context, req *connect.Request[v1.GetSymbolParamsRequest]) (*connect.Response[v1.SymbolParamsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params, err := s.marketSvc.GetSymbolParams(ctx, userID, accountID, req.Msg.Symbol)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.SymbolParamsResponse{
		Symbol: &v1.SymbolInfo{
			Symbol:       params.Name,
			Description:  params.Description,
			Currency:     params.Currency,
			Digits:       params.Digits,
			TickSize:     params.Point,
			TickValue:    params.Point,
			ContractSize: params.ContractSize,
			MinLot:       params.MinLot,
			MaxLot:       params.MaxLot,
			LotStep:      params.LotStep,
		},
		Group: &v1.SymbolGroup{
			Name:      "default",
			MinLot:    params.MinLot,
			MaxLot:    params.MaxLot,
			LotStep:   params.LotStep,
			TradeMode: "full",
		},
	}), nil
}

func (s *MarketService) GetKlines(ctx context.Context, req *connect.Request[v1.GetKlinesRequest]) (*connect.Response[v1.GetKlinesResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	klineReq := &service.KlineRequest{
		AccountID: req.Msg.AccountId,
		Symbol:    req.Msg.Symbol,
		Timeframe: req.Msg.Timeframe,
		From:      req.Msg.From,
		To:        req.Msg.To,
		Count:     int(req.Msg.Count),
	}

	klines, err := s.klineSvc.GetKlines(ctx, userID, accountID, klineReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetKlinesResponse{
		Klines: make([]*v1.Kline, len(klines)),
	}

	for i, k := range klines {
		openTime, _ := time.Parse(time.RFC3339, k.OpenTime)
		response.Klines[i] = &v1.Kline{
			Time:   timestamppb.New(openTime),
			Open:   k.OpenPrice,
			High:   k.HighPrice,
			Low:    k.LowPrice,
			Close:  k.ClosePrice,
			Volume: uint64(k.Volume),
		}
	}

	return connect.NewResponse(response), nil
}

func (s *MarketService) GetMarketWatch(ctx context.Context, req *connect.Request[v1.GetMarketWatchRequest]) (*connect.Response[v1.GetMarketWatchResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	items := make([]*v1.MarketWatchItem, len(req.Msg.Symbols))
	for i, symbol := range req.Msg.Symbols {
		quote, err := s.marketSvc.GetQuote(ctx, userID, accountID, symbol)
		if err != nil {
			continue
		}
		items[i] = &v1.MarketWatchItem{
			Symbol: quote.Symbol,
			Bid:    quote.Bid,
			Ask:    quote.Ask,
			Spread: int32(quote.Ask - quote.Bid),
		}
	}

	return connect.NewResponse(&v1.GetMarketWatchResponse{
		Items: items,
	}), nil
}

func parseTimeToProto(s string) *timestamppb.Timestamp {
	if s == "" {
		return timestamppb.Now()
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		return timestamppb.Now()
	}
	return timestamppb.New(t)
}
