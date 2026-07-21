package marketplace

import (
	"context"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// GetStrategyPublicInfo returns public strategy data for the share landing page.
// This endpoint does not require authentication — it only returns published strategies.
func (h *MarketplaceServer) GetStrategyPublicInfo(
	ctx context.Context,
	req *connect.Request[antv1.GetStrategyPublicInfoRequest],
) (*connect.Response[antv1.GetStrategyPublicInfoResponse], error) {
	resp, err := h.svc.GetStrategyPublicInfo(ctx, req.Msg.StrategyId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(resp), nil
}
