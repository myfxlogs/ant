package system

import (
	"context"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// EconomicDataServer implements ant.v1.EconomicDataServiceHandler.
type EconomicDataServer struct{ log *zap.Logger }

var _ antv1c.EconomicDataServiceHandler = (*EconomicDataServer)(nil)

func NewEconomicDataServer(log *zap.Logger) *EconomicDataServer {
	return &EconomicDataServer{log: log}
}

func (s *EconomicDataServer) ListEconomicCalendarEvents(ctx context.Context, req *connect.Request[antv1.ListEconomicCalendarEventsRequest]) (*connect.Response[antv1.ListEconomicCalendarEventsResponse], error) {
	now := time.Now().Unix()
	return connect.NewResponse(&antv1.ListEconomicCalendarEventsResponse{
		Events: []*antv1.EconomicCalendarEvent{
			{
				Date:      "2026-05-25",
				Time:      "20:30",
				Country:   "US",
				Event:     "美国非农就业数据",
				Impact:    "high",
				Actual:    "210K",
				Previous:  "180K",
				Estimate:  "200K",
				Timestamp: now,
			},
			{
				Date:      "2026-05-25",
				Time:      "02:00",
				Country:   "US",
				Event:     "美联储利率决议",
				Impact:    "high",
				Actual:    "5.50%",
				Previous:  "5.50%",
				Estimate:  "5.50%",
				Timestamp: now,
			},
			{
				Date:      "2026-05-25",
				Time:      "09:30",
				Country:   "CN",
				Event:     "中国CPI数据",
				Impact:    "medium",
				Actual:    "0.3%",
				Previous:  "0.2%",
				Estimate:  "0.3%",
				Timestamp: now,
			},
		},
	}), nil
}

func (s *EconomicDataServer) ListEconomicIndicators(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.ListEconomicIndicatorsResponse], error) {
	return connect.NewResponse(&antv1.ListEconomicIndicatorsResponse{
		Indicators: []*antv1.EconomicIndicator{
			{Code: "GDP", Name: "国内生产总值", Units: "%", Frequency: "quarterly"},
			{Code: "CPI", Name: "消费者价格指数", Units: "%", Frequency: "monthly"},
			{Code: "NFP", Name: "非农就业人数", Units: "K", Frequency: "monthly"},
			{Code: "PMI", Name: "制造业采购经理指数", Units: "index", Frequency: "monthly"},
		},
	}), nil
}
