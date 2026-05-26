package connect

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// IndicatorCatalogServer implements ant.v1.IndicatorCatalogServiceHandler.
type IndicatorCatalogServer struct{ log *zap.Logger }

var _ antv1c.IndicatorCatalogServiceHandler = (*IndicatorCatalogServer)(nil)

func NewIndicatorCatalogServer(log *zap.Logger) *IndicatorCatalogServer {
	return &IndicatorCatalogServer{log: log}
}

func (s *IndicatorCatalogServer) GetIndicatorCatalog(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.IndicatorCatalogResponse], error) {
	return connect.NewResponse(&antv1.IndicatorCatalogResponse{
		Indicators: []*antv1.IndicatorCatalogItem{
			{
				Name:          "sma",
				CallSignature: "sma(period)",
				Description:   "计算指定周期内的算术平均值，用于识别趋势方向",
				ParamKeys: []*antv1.IndicatorCatalogParam{
					{Key: "period", Label: "周期", Type: "int", Default: 14},
				},
			},
			{
				Name:          "ema",
				CallSignature: "ema(period)",
				Description:   "对近期价格赋予更高权重，反应更灵敏",
				ParamKeys: []*antv1.IndicatorCatalogParam{
					{Key: "period", Label: "周期", Type: "int", Default: 14},
				},
			},
			{
				Name:          "rsi",
				CallSignature: "rsi(period)",
				Description:   "衡量价格变动的速度和幅度，判断超买超卖",
				ParamKeys: []*antv1.IndicatorCatalogParam{
					{Key: "period", Label: "周期", Type: "int", Default: 14},
					{Key: "overbought", Label: "超买阈值", Type: "float", Default: 70},
					{Key: "oversold", Label: "超卖阈值", Type: "float", Default: 30},
				},
			},
			{
				Name:          "macd",
				CallSignature: "macd(fast, slow, signal)",
				Description:   "指数平滑异同移动平均线，用于判断趋势和背离",
				ParamKeys: []*antv1.IndicatorCatalogParam{
					{Key: "fast", Label: "快线周期", Type: "int", Default: 12},
					{Key: "slow", Label: "慢线周期", Type: "int", Default: 26},
					{Key: "signal", Label: "信号线周期", Type: "int", Default: 9},
				},
			},
			{
				Name:          "bollinger",
				CallSignature: "bollinger(period, stddev)",
				Description:   "基于标准差的价格通道，用于判断波动性和突破",
				ParamKeys: []*antv1.IndicatorCatalogParam{
					{Key: "period", Label: "周期", Type: "int", Default: 20},
					{Key: "stddev", Label: "标准差倍数", Type: "float", Default: 2},
				},
			},
		},
	}), nil
}
