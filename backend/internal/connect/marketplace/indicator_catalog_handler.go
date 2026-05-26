package marketplace

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/service"
)

// IndicatorCatalogServer implements ant.v1.IndicatorCatalogServiceHandler.
type IndicatorCatalogServer struct{ log *zap.Logger }

var _ antv1c.IndicatorCatalogServiceHandler = (*IndicatorCatalogServer)(nil)

func NewIndicatorCatalogServer(log *zap.Logger) *IndicatorCatalogServer {
	return &IndicatorCatalogServer{log: log}
}

func (s *IndicatorCatalogServer) GetIndicatorCatalog(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.IndicatorCatalogResponse], error) {
	cat := service.GetIndicatorCatalog()

	indicators := make([]*antv1.IndicatorCatalogItem, len(cat.Indicators))
	for i, ind := range cat.Indicators {
		params := make([]*antv1.IndicatorCatalogParam, len(ind.ParamKeys))
		for j, p := range ind.ParamKeys {
			params[j] = &antv1.IndicatorCatalogParam{
				Key:         p.Key,
				Label:       p.Label,
				Type:        p.Type,
				Default:     p.Default,
				Min:         p.Min,
				Max:         p.Max,
				Description: p.Description,
			}
		}
		indicators[i] = &antv1.IndicatorCatalogItem{
			Name:          ind.Name,
			CallSignature: ind.CallSignature,
			Description:   ind.Description,
			ParamKeys:     params,
		}
	}

	riskParams := make([]*antv1.IndicatorCatalogParam, len(cat.RiskParams))
	for i, p := range cat.RiskParams {
		riskParams[i] = &antv1.IndicatorCatalogParam{
			Key:         p.Key,
			Label:       p.Label,
			Type:        p.Type,
			Default:     p.Default,
			Min:         p.Min,
			Max:         p.Max,
			Description: p.Description,
		}
	}

	return connect.NewResponse(&antv1.IndicatorCatalogResponse{
		Indicators: indicators,
		RiskParams: riskParams,
	}), nil
}
