package mt4

import (
	"context"
	"fmt"
	"io"
	"sync"

	pb "anttrader/mt4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mockMT4Client implements pb.MT4Client for testing data conversion paths.
type mockMT4Client struct {
	openedOrdersRes  *pb.OpenedOrdersReply
	openedOrdersErr  error
	orderHistoryRes  *pb.OrderHistoryReply
	orderHistoryErr  error
	quoteHistoryRes  *pb.QuoteHistoryReply
	quoteHistoryErr  error
}

func (m *mockMT4Client) AccountSummary(ctx context.Context, in *pb.AccountSummaryRequest, opts ...grpc.CallOption) (*pb.AccountSummaryReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) Groups(ctx context.Context, in *pb.GroupsRequest, opts ...grpc.CallOption) (*pb.GroupsReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) Quote(ctx context.Context, in *pb.QuoteRequest, opts ...grpc.CallOption) (*pb.QuoteReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) GetQuoteMany(ctx context.Context, in *pb.GetQuoteManyRequest, opts ...grpc.CallOption) (*pb.GetQuoteManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) OpenedOrders(ctx context.Context, in *pb.OpenedOrdersRequest, opts ...grpc.CallOption) (*pb.OpenedOrdersReply, error) {
	return m.openedOrdersRes, m.openedOrdersErr
}
func (m *mockMT4Client) Symbols(ctx context.Context, in *pb.SymbolsRequest, opts ...grpc.CallOption) (*pb.SymbolsReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) SymbolParams(ctx context.Context, in *pb.SymbolParamsRequest, opts ...grpc.CallOption) (*pb.SymbolParamsReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) ServerTimezone(ctx context.Context, in *pb.ServerTimezoneRequest, opts ...grpc.CallOption) (*pb.ServerTimezoneReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) SymbolParamsMany(ctx context.Context, in *pb.SymbolParamsManyRequest, opts ...grpc.CallOption) (*pb.SymbolParamsManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) OpenedOrder(ctx context.Context, in *pb.OpenedOrderRequest, opts ...grpc.CallOption) (*pb.OpenedOrderReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) OrderHistory(ctx context.Context, in *pb.OrderHistoryRequest, opts ...grpc.CallOption) (*pb.OrderHistoryReply, error) {
	return m.orderHistoryRes, m.orderHistoryErr
}
func (m *mockMT4Client) QuoteHistory(ctx context.Context, in *pb.QuoteHistoryRequest, opts ...grpc.CallOption) (*pb.QuoteHistoryReply, error) {
	return m.quoteHistoryRes, m.quoteHistoryErr
}
func (m *mockMT4Client) QuoteHistoryMany(ctx context.Context, in *pb.QuoteHistoryManyRequest, opts ...grpc.CallOption) (*pb.QuoteHistoryManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) ClosedOrders(ctx context.Context, in *pb.ClosedOrdersRequest, opts ...grpc.CallOption) (*pb.ClosedOrdersReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) RequestQuoteHistory(ctx context.Context, in *pb.RequestQuoteHistoryRequest, opts ...grpc.CallOption) (*pb.RequestQuoteHistoryReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) SetPlacedType(ctx context.Context, in *pb.SetPlacedTypeRequest, opts ...grpc.CallOption) (*pb.SetPlacedTypeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) IsInvestor(ctx context.Context, in *pb.IsInvestorRequest, opts ...grpc.CallOption) (*pb.IsInvestorReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockMT4Client) TickValueWithSize(ctx context.Context, in *pb.TickValueWithSizeRequest, opts ...grpc.CallOption) (*pb.TickValueWithSizeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}

// --- Mock streaming types for recvLoop tests ---

type mockQuoteStream struct {
	mu     sync.Mutex
	quotes []*pb.OnQuoteReply
	idx    int
	ctx    context.Context
}

func (m *mockQuoteStream) Recv() (*pb.OnQuoteReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx < len(m.quotes) {
		q := m.quotes[m.idx]
		m.idx++
		return q, nil
	}
	return nil, io.EOF
}
func (m *mockQuoteStream) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockQuoteStream) Trailer() metadata.MD           { return nil }
func (m *mockQuoteStream) CloseSend() error               { return nil }
func (m *mockQuoteStream) Context() context.Context       { return m.ctx }
func (m *mockQuoteStream) SendMsg(msg any) error          { return nil }
func (m *mockQuoteStream) RecvMsg(msg any) error          { return io.EOF }

type mockProfitStream struct {
	mu       sync.Mutex
	updates  []*pb.OnOrderProfitReply
	idx      int
	ctx      context.Context
}

func (m *mockProfitStream) Recv() (*pb.OnOrderProfitReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx < len(m.updates) {
		u := m.updates[m.idx]
		m.idx++
		return u, nil
	}
	return nil, io.EOF
}
func (m *mockProfitStream) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockProfitStream) Trailer() metadata.MD           { return nil }
func (m *mockProfitStream) CloseSend() error               { return nil }
func (m *mockProfitStream) Context() context.Context       { return m.ctx }
func (m *mockProfitStream) SendMsg(msg any) error          { return nil }
func (m *mockProfitStream) RecvMsg(msg any) error          { return io.EOF }

type mockOrderUpdateStream struct {
	mu      sync.Mutex
	updates []*pb.OnOrderUpdateReply
	idx     int
	ctx     context.Context
}

func (m *mockOrderUpdateStream) Recv() (*pb.OnOrderUpdateReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx < len(m.updates) {
		u := m.updates[m.idx]
		m.idx++
		return u, nil
	}
	return nil, io.EOF
}
func (m *mockOrderUpdateStream) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockOrderUpdateStream) Trailer() metadata.MD           { return nil }
func (m *mockOrderUpdateStream) CloseSend() error               { return nil }
func (m *mockOrderUpdateStream) Context() context.Context       { return m.ctx }
func (m *mockOrderUpdateStream) SendMsg(msg any) error          { return nil }
func (m *mockOrderUpdateStream) RecvMsg(msg any) error          { return io.EOF }

// mockStreamsClient implements pb.StreamsClient for testing recv loops.
type mockStreamsClient struct {
	quoteStream       *mockQuoteStream
	quoteErr          error
	profitStream      *mockProfitStream
	profitErr         error
	orderUpdateStream *mockOrderUpdateStream
	orderUpdateErr    error
}

func (m *mockStreamsClient) OnQuote(ctx context.Context, in *pb.OnQuoteRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnQuoteReply], error) {
	if m.quoteErr != nil {
		return nil, m.quoteErr
	}
	return m.quoteStream, nil
}
func (m *mockStreamsClient) OnOrderProfit(ctx context.Context, in *pb.OnOrderProfitRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnOrderProfitReply], error) {
	if m.profitErr != nil {
		return nil, m.profitErr
	}
	return m.profitStream, nil
}
func (m *mockStreamsClient) OnOrderUpdate(ctx context.Context, in *pb.OnOrderUpdateRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnOrderUpdateReply], error) {
	if m.orderUpdateErr != nil {
		return nil, m.orderUpdateErr
	}
	return m.orderUpdateStream, nil
}
func (m *mockStreamsClient) OnTickValue(ctx context.Context, in *pb.OnTickValueRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.OnTickValueReply], error) {
	return nil, fmt.Errorf("mock: not implemented")
}

type mockConnCli struct{}

func (m *mockConnCli) Connect(ctx context.Context, in *pb.ConnectRequest, opts ...grpc.CallOption) (*pb.ConnectReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockConnCli) ConnectProxy(ctx context.Context, in *pb.ConnectProxyRequest, opts ...grpc.CallOption) (*pb.ConnectProxyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockConnCli) CheckConnect(ctx context.Context, in *pb.CheckConnectRequest, opts ...grpc.CallOption) (*pb.CheckConnectReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockConnCli) Disconnect(ctx context.Context, in *pb.DisconnectRequest, opts ...grpc.CallOption) (*pb.DisconnectReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}

type mockSubCli struct{}

func (m *mockSubCli) Subscribe(ctx context.Context, in *pb.SubscribeRequest, opts ...grpc.CallOption) (*pb.SubscribeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) SubscribeMany(ctx context.Context, in *pb.SubscribeManyRequest, opts ...grpc.CallOption) (*pb.SubscribeManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) UnSubscribe(ctx context.Context, in *pb.UnSubscribeRequest, opts ...grpc.CallOption) (*pb.UnSubscribeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) UnSubscribeMany(ctx context.Context, in *pb.UnSubscribeManyRequest, opts ...grpc.CallOption) (*pb.UnSubscribeManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) SubscribeOrderProfit(ctx context.Context, in *pb.SubscribeOrderProfitRequest, opts ...grpc.CallOption) (*pb.SubscribeOrderProfitReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) SubscribeTickValue(ctx context.Context, in *pb.SubscribeTickValueRequest, opts ...grpc.CallOption) (*pb.SubscribeTickValueReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) SubscribeOrderUpdate(ctx context.Context, in *pb.SubscribeOrderUpdateRequest, opts ...grpc.CallOption) (*pb.SubscribeOrderUpdateReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCli) SubscribeQuoteHistory(ctx context.Context, in *pb.SubscribeQuoteHistoryRequest, opts ...grpc.CallOption) (*pb.SubscribeQuoteHistoryReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
