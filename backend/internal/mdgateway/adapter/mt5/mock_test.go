package mt5

import (
	"context"
	"fmt"
	"io"
	"sync"

	pb "anttrader/mt5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mockMT5Client implements pb.MT5Client for testing data conversion paths.
type mockMT5Client struct {
	openedOrdersRes    *pb.OpenedOrdersReply
	openedOrdersErr    error
	accountSummaryRes  *pb.AccountSummaryReply
	accountSummaryErr  error
	symbolParamsRes    *pb.SymbolParamsReply
	symbolParamsErr    error
}

func (m *mockMT5Client) getErr() error        { return fmt.Errorf("mock: not implemented") }

func (m *mockMT5Client) Account(ctx context.Context, in *pb.AccountRequest, opts ...grpc.CallOption) (*pb.AccountReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) AccountSummary(ctx context.Context, in *pb.AccountSummaryRequest, opts ...grpc.CallOption) (*pb.AccountSummaryReply, error) {
	return m.accountSummaryRes, m.accountSummaryErr
}
func (m *mockMT5Client) OpenedOrders(ctx context.Context, in *pb.OpenedOrdersRequest, opts ...grpc.CallOption) (*pb.OpenedOrdersReply, error) {
	return m.openedOrdersRes, m.openedOrdersErr
}
func (m *mockMT5Client) OpenedOrder(ctx context.Context, in *pb.OpenedOrderRequest, opts ...grpc.CallOption) (*pb.OpenedOrderReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) OpenedOrdersTickets(ctx context.Context, in *pb.OpenedOrdersTicketsRequest, opts ...grpc.CallOption) (*pb.OpenedOrdersTicketsReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) OrderHistory(ctx context.Context, in *pb.OrderHistoryRequest, opts ...grpc.CallOption) (*pb.OrderHistoryReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) PendingOrderHistory(ctx context.Context, in *pb.PendingOrderHistoryRequest, opts ...grpc.CallOption) (*pb.PendingOrderHistoryReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) OrderHistoryPagination(ctx context.Context, in *pb.OrderHistoryPaginationRequest, opts ...grpc.CallOption) (*pb.OrderHistoryPaginationReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) Symbols(ctx context.Context, in *pb.SymbolsRequest, opts ...grpc.CallOption) (*pb.SymbolsReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) SymbolList(ctx context.Context, in *pb.SymbolListRequest, opts ...grpc.CallOption) (*pb.SymbolListReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) GetQuote(ctx context.Context, in *pb.GetQuoteRequest, opts ...grpc.CallOption) (*pb.GetQuoteReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) GetQuoteMany(ctx context.Context, in *pb.GetQuoteManyRequest, opts ...grpc.CallOption) (*pb.GetQuoteManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) MarketWatchMany(ctx context.Context, in *pb.MarketWatchManyRequest, opts ...grpc.CallOption) (*pb.MarketWatchManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) SymbolParams(ctx context.Context, in *pb.SymbolParamsRequest, opts ...grpc.CallOption) (*pb.SymbolParamsReply, error) {
	return m.symbolParamsRes, m.symbolParamsErr
}
func (m *mockMT5Client) SymbolParamsMany(ctx context.Context, in *pb.SymbolParamsManyRequest, opts ...grpc.CallOption) (*pb.SymbolParamsManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) SymbolSessionsEx(ctx context.Context, in *pb.SymbolSessionsExRequest, opts ...grpc.CallOption) (*pb.SymbolSessionsExReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) SymbolSessionsExMany(ctx context.Context, in *pb.SymbolSessionsExManyRequest, opts ...grpc.CallOption) (*pb.SymbolSessionsExManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) ServerTimezone(ctx context.Context, in *pb.ServerTimezoneRequest, opts ...grpc.CallOption) (*pb.ServerTimezoneReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) IsTradeSession(ctx context.Context, in *pb.IsTradeSessionRequest, opts ...grpc.CallOption) (*pb.IsTradeSessionReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) IsTradeSessionMany(ctx context.Context, in *pb.IsTradeSessionManyRequest, opts ...grpc.CallOption) (*pb.IsTradeSessionManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) IsQuoteSession(ctx context.Context, in *pb.IsQuoteSessionRequest, opts ...grpc.CallOption) (*pb.IsQuoteSessionReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) IsQuoteSessionMany(ctx context.Context, in *pb.IsQuoteSessionManyRequest, opts ...grpc.CallOption) (*pb.IsQuoteSessionManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) GetTickValueMany(ctx context.Context, in *pb.GetTickValueManyRequest, opts ...grpc.CallOption) (*pb.GetTickValueManyReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) TickValueWithSize(ctx context.Context, in *pb.TickValueWithSizeRequest, opts ...grpc.CallOption) (*pb.TickValueWithSizeReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) ChangePassword(ctx context.Context, in *pb.ChangePasswordRequest, opts ...grpc.CallOption) (*pb.ChangePasswordReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) Mails(ctx context.Context, in *pb.MailsRequest, opts ...grpc.CallOption) (*pb.MailsReply, error) {
	return nil, m.getErr()
}
func (m *mockMT5Client) RequiredMargin(ctx context.Context, in *pb.RequiredMarginRequest, opts ...grpc.CallOption) (*pb.RequiredMarginReply, error) {
	return nil, m.getErr()
}

// mockTradingClient implements pb.TradingClient for testing order paths.
type mockTradingClient struct {
	orderSendRes   *pb.OrderSendReply
	orderSendErr   error
	orderCloseRes  *pb.OrderCloseReply
	orderCloseErr  error
	orderModifyRes *pb.OrderModifyReply
	orderModifyErr error
}

func (m *mockTradingClient) OrderSend(ctx context.Context, in *pb.OrderSendRequest, opts ...grpc.CallOption) (*pb.OrderSendReply, error) {
	return m.orderSendRes, m.orderSendErr
}
func (m *mockTradingClient) OrderClose(ctx context.Context, in *pb.OrderCloseRequest, opts ...grpc.CallOption) (*pb.OrderCloseReply, error) {
	return m.orderCloseRes, m.orderCloseErr
}
func (m *mockTradingClient) OrderModify(ctx context.Context, in *pb.OrderModifyRequest, opts ...grpc.CallOption) (*pb.OrderModifyReply, error) {
	return m.orderModifyRes, m.orderModifyErr
}

// mockQHClient implements pb.QuoteHistoryClient for testing GetPriceHistory paths.
type mockQHClient struct{}

func (m *mockQHClient) PriceHistoryMonth(ctx context.Context, in *pb.PriceHistoryMonthRequest, opts ...grpc.CallOption) (*pb.PriceHistoryMonthReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryMonthMany(ctx context.Context, in *pb.PriceHistoryMonthManyRequest, opts ...grpc.CallOption) (*pb.PriceHistoryMonthManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryToday(ctx context.Context, in *pb.PriceHistoryTodayRequest, opts ...grpc.CallOption) (*pb.PriceHistoryTodayReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryTodayMany(ctx context.Context, in *pb.PriceHistoryTodayManyRequest, opts ...grpc.CallOption) (*pb.PriceHistoryTodayManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistory(ctx context.Context, in *pb.PriceHistoryRequest, opts ...grpc.CallOption) (*pb.PriceHistoryReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryMany(ctx context.Context, in *pb.PriceHistoryManyRequest, opts ...grpc.CallOption) (*pb.PriceHistoryManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryHighLow(ctx context.Context, in *pb.PriceHistoryHighLowRequest, opts ...grpc.CallOption) (*pb.PriceHistoryHighLowReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryEx(ctx context.Context, in *pb.PriceHistoryExRequest, opts ...grpc.CallOption) (*pb.PriceHistoryExReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockQHClient) PriceHistoryExMany(ctx context.Context, in *pb.PriceHistoryExManyRequest, opts ...grpc.CallOption) (*pb.PriceHistoryExManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}

// --- Mock streaming types for recvLoop tests ---

type mt5MockQuoteStream struct {
	mu     sync.Mutex
	quotes []*pb.OnQuoteReply
	idx    int
	ctx    context.Context
}

func (m *mt5MockQuoteStream) Recv() (*pb.OnQuoteReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx < len(m.quotes) {
		q := m.quotes[m.idx]
		m.idx++
		return q, nil
	}
	return nil, io.EOF
}
func (m *mt5MockQuoteStream) Header() (metadata.MD, error)  { return nil, nil }
func (m *mt5MockQuoteStream) Trailer() metadata.MD           { return nil }
func (m *mt5MockQuoteStream) CloseSend() error               { return nil }
func (m *mt5MockQuoteStream) Context() context.Context       { return m.ctx }
func (m *mt5MockQuoteStream) SendMsg(msg any) error          { return nil }
func (m *mt5MockQuoteStream) RecvMsg(msg any) error          { return io.EOF }

type mt5MockProfitStream struct {
	mu      sync.Mutex
	updates []*pb.OnOrderProfitReply
	idx     int
	ctx     context.Context
}

func (m *mt5MockProfitStream) Recv() (*pb.OnOrderProfitReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx < len(m.updates) {
		u := m.updates[m.idx]
		m.idx++
		return u, nil
	}
	return nil, io.EOF
}
func (m *mt5MockProfitStream) Header() (metadata.MD, error)  { return nil, nil }
func (m *mt5MockProfitStream) Trailer() metadata.MD           { return nil }
func (m *mt5MockProfitStream) CloseSend() error               { return nil }
func (m *mt5MockProfitStream) Context() context.Context       { return m.ctx }
func (m *mt5MockProfitStream) SendMsg(msg any) error          { return nil }
func (m *mt5MockProfitStream) RecvMsg(msg any) error          { return io.EOF }

type mt5MockOrderUpdateStream struct {
	mu      sync.Mutex
	updates []*pb.OnOrderUpdateReply
	idx     int
	ctx     context.Context
}

func (m *mt5MockOrderUpdateStream) Recv() (*pb.OnOrderUpdateReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx < len(m.updates) {
		u := m.updates[m.idx]
		m.idx++
		return u, nil
	}
	return nil, io.EOF
}
func (m *mt5MockOrderUpdateStream) Header() (metadata.MD, error)  { return nil, nil }
func (m *mt5MockOrderUpdateStream) Trailer() metadata.MD           { return nil }
func (m *mt5MockOrderUpdateStream) CloseSend() error               { return nil }
func (m *mt5MockOrderUpdateStream) Context() context.Context       { return m.ctx }
func (m *mt5MockOrderUpdateStream) SendMsg(msg any) error          { return nil }
func (m *mt5MockOrderUpdateStream) RecvMsg(msg any) error          { return io.EOF }

type mt5MockStreamsClient struct {
	quoteStream       *mt5MockQuoteStream
	quoteErr          error
	profitStream      *mt5MockProfitStream
	profitErr         error
	orderUpdateStream *mt5MockOrderUpdateStream
	orderUpdateErr    error
}

func (m *mt5MockStreamsClient) OnQuote(ctx context.Context, in *pb.OnQuoteRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnQuoteReply], error) {
	if m.quoteErr != nil {
		return nil, m.quoteErr
	}
	return m.quoteStream, nil
}
func (m *mt5MockStreamsClient) OnOrderProfit(ctx context.Context, in *pb.OnOrderProfitRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnOrderProfitReply], error) {
	if m.profitErr != nil {
		return nil, m.profitErr
	}
	return m.profitStream, nil
}
func (m *mt5MockStreamsClient) OnOrderUpdate(ctx context.Context, in *pb.OnOrderUpdateRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnOrderUpdateReply], error) {
	if m.orderUpdateErr != nil {
		return nil, m.orderUpdateErr
	}
	return m.orderUpdateStream, nil
}
func (m *mt5MockStreamsClient) Events(ctx context.Context, in *pb.EventsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.EventsReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mt5MockStreamsClient) OnTickValue(ctx context.Context, in *pb.OnTickValueRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnTickValueReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mt5MockStreamsClient) OnMarketWatch(ctx context.Context, in *pb.OnMarketWatchRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnMarketWatchReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mt5MockStreamsClient) OnTickHistory(ctx context.Context, in *pb.OnTickHistoryRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnTickHistoryReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mt5MockStreamsClient) OnMail(ctx context.Context, in *pb.OnMailRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnMailReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mt5MockStreamsClient) OnOpenedOrdersTickets(ctx context.Context, in *pb.OnOpenedOrdersTicketsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnOpenedOrdersTicketsReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}

type mockMT5ConnCli struct{}

func (m *mockMT5ConnCli) Connect(ctx context.Context, in *pb.ConnectRequest, opts ...grpc.CallOption) (*pb.ConnectReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5ConnCli) ConnectEx(ctx context.Context, in *pb.ConnectExRequest, opts ...grpc.CallOption) (*pb.ConnectExReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5ConnCli) ConnectProxy(ctx context.Context, in *pb.ConnectProxyRequest, opts ...grpc.CallOption) (*pb.ConnectProxyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5ConnCli) CheckConnect(ctx context.Context, in *pb.CheckConnectRequest, opts ...grpc.CallOption) (*pb.CheckConnectReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5ConnCli) Disconnect(ctx context.Context, in *pb.DisconnectRequest, opts ...grpc.CallOption) (*pb.DisconnectReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}

type mockMT5SubCli struct{}

func (m *mockMT5SubCli) Subscribe(ctx context.Context, in *pb.SubscribeRequest, opts ...grpc.CallOption) (*pb.SubscribeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) SubscribeMany(ctx context.Context, in *pb.SubscribeManyRequest, opts ...grpc.CallOption) (*pb.SubscribeManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) UnSubscribe(ctx context.Context, in *pb.UnSubscribeRequest, opts ...grpc.CallOption) (*pb.UnSubscribeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) UnSubscribeMany(ctx context.Context, in *pb.UnSubscribeManyRequest, opts ...grpc.CallOption) (*pb.UnSubscribeManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) SubscribeOrderProfit(ctx context.Context, in *pb.SubscribeOrderProfitRequest, opts ...grpc.CallOption) (*pb.SubscribeOrderProfitReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) SubscribeTickValue(ctx context.Context, in *pb.SubscribeTickValueRequest, opts ...grpc.CallOption) (*pb.SubscribeTickValueReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) SubscribeOrderUpdate(ctx context.Context, in *pb.SubscribeOrderUpdateRequest, opts ...grpc.CallOption) (*pb.SubscribeOrderUpdateReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) SubscribeMarketWatch(ctx context.Context, in *pb.SubscribeMarketWatchRequest, opts ...grpc.CallOption) (*pb.SubscribeMarketWatchReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT5SubCli) SubscribeOpenedOrdersTickets(ctx context.Context, in *pb.SubscribeOpenedOrdersTicketsRequest, opts ...grpc.CallOption) (*pb.SubscribeOpenedOrdersTicketsReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
