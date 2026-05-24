package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/service"
)

// MarketServer implements ant.v1.MarketServiceHandler.
type MarketServer struct {
	platform *service.PlatformService
}

var _ antv1c.MarketServiceHandler = (*MarketServer)(nil)

// NewMarketServer creates a market handler backed by the platform service.
func NewMarketServer(svc *service.PlatformService) *MarketServer {
	return &MarketServer{platform: svc}
}

func (s *MarketServer) GetKlines(ctx context.Context, req *connect.Request[antv1.GetKlinesRequest]) (*connect.Response[antv1.GetKlinesResponse], error) {
	return connect.NewResponse(&antv1.GetKlinesResponse{}), nil
}

func (s *MarketServer) GetSymbolStats(ctx context.Context, req *connect.Request[antv1.GetSymbolStatsRequest]) (*connect.Response[antv1.GetSymbolStatsResponse], error) {
	return connect.NewResponse(&antv1.GetSymbolStatsResponse{
		CurrentBid: "0", CurrentAsk: "0", Spread: "0",
	}), nil
}

func (s *MarketServer) StreamTicks(ctx context.Context, req *connect.Request[antv1.StreamTicksRequest], stream *connect.ServerStream[antv1.TickMsg]) error {
	<-ctx.Done()
	return nil
}

// Ensure timestamp import is used.
var _ = timestamppb.Now
