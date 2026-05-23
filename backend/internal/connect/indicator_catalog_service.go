package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

type IndicatorCatalogService struct{}

func NewIndicatorCatalogService() *IndicatorCatalogService {
	return &IndicatorCatalogService{}
}

func (s *IndicatorCatalogService) GetIndicatorCatalog(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.IndicatorCatalogResponse], error) {
	_ = ctx
	_ = req
	catalog := service.GetIndicatorCatalog()
	return connect.NewResponse(&v1.IndicatorCatalogResponse{
		Indicators: indicatorCatalogItems(catalog.Indicators),
		RiskParams: indicatorCatalogParams(catalog.RiskParams),
	}), nil
}

func indicatorCatalogItems(items []service.StrategyIndicator) []*v1.IndicatorCatalogItem {
	out := make([]*v1.IndicatorCatalogItem, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.IndicatorCatalogItem{
			Name:          item.Name,
			CallSignature: item.CallSignature,
			Description:   item.Description,
			ParamKeys:     indicatorCatalogParams(item.ParamKeys),
		})
	}
	return out
}

func indicatorCatalogParams(params []service.StrategyParamSpec) []*v1.IndicatorCatalogParam {
	out := make([]*v1.IndicatorCatalogParam, 0, len(params))
	for _, p := range params {
		out = append(out, &v1.IndicatorCatalogParam{
			Key:         p.Key,
			Label:       p.Label,
			Type:        p.Type,
			Default:     p.Default,
			Min:         p.Min,
			Max:         p.Max,
			Description: p.Description,
		})
	}
	return out
}
