package system

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// EconomicDataServer implements ant.v1.EconomicDataServiceHandler.
// Currently returns empty results — a real economic calendar API (e.g.,
// Trading Economics, FRED, or Alpha Vantage) has not been integrated yet.
type EconomicDataServer struct{ log *zap.Logger }

var _ antv1c.EconomicDataServiceHandler = (*EconomicDataServer)(nil)

func NewEconomicDataServer(log *zap.Logger) *EconomicDataServer {
	return &EconomicDataServer{log: log}
}

func (s *EconomicDataServer) ListEconomicCalendarEvents(ctx context.Context, req *connect.Request[antv1.ListEconomicCalendarEventsRequest]) (*connect.Response[antv1.ListEconomicCalendarEventsResponse], error) {
	return connect.NewResponse(&antv1.ListEconomicCalendarEventsResponse{
		Events: []*antv1.EconomicCalendarEvent{},
	}), nil
}

func (s *EconomicDataServer) ListEconomicIndicators(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.ListEconomicIndicatorsResponse], error) {
	return connect.NewResponse(&antv1.ListEconomicIndicatorsResponse{
		Indicators: []*antv1.EconomicIndicator{},
	}), nil
}
