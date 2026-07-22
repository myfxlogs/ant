package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

func (s *MarketplaceServer) GetProviderEarnings(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[antv1.ProviderEarnings], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	result, err := s.svc.GetProviderEarnings(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.ProviderEarnings{
		TotalEarnings:     result.TotalEarnings.String(),
		AvailableBalance:  result.AvailableBalance.String(),
		PendingWithdrawal: result.PendingWithdrawal.String(),
		PendingSettlement: result.PendingSettlement.String(),
		LifetimeWithdrawn: result.LifetimeWithdrawn.String(),
		TotalSales:        result.TotalSales,
		ActiveStrategies:  result.ActiveStrategies,
	}), nil
}

func (s *MarketplaceServer) ListProviderTransactions(
	ctx context.Context,
	req *connect.Request[antv1.ListProviderTransactionsRequest],
) (*connect.Response[antv1.ListProviderTransactionsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	m := req.Msg
	limit := int(m.Limit)
	offset := int(m.Offset)

	rows, total, err := s.svc.ListProviderTransactions(ctx, userID, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.ProviderTransactionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &antv1.ProviderTransactionItem{
			Id:            r.ID,
			TxType:        r.TxType,
			Amount:        r.Amount.String(),
			StrategyTitle: r.StrategyTitle,
			BuyerName:     r.BuyerName,
			CreatedAt:     r.CreatedAt,
		})
	}

	return connect.NewResponse(&antv1.ListProviderTransactionsResponse{
		Transactions: items,
		Total:        int32(total),
	}), nil
}
