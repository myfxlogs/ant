package system

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
)

func (s *AnalyticsServer) GetMonthlyPnL(ctx context.Context, req *connect.Request[antv1.GetMonthlyPnLRequest]) (*connect.Response[antv1.GetMonthlyPnLResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	year := int(req.Msg.Year)
	if year <= 0 {
		year = time.Now().Year()
	}

	monthlyData, err := s.repo.GetMonthlyPnL(ctx, accountID, year)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get monthly pnl: %w", err))
	}

	items := make([]*antv1.MonthlyPnLItem, 0, len(monthlyData))
	for _, m := range monthlyData {
		items = append(items, &antv1.MonthlyPnLItem{
			Month:  int32(m.MonthNum),
			Profit: m.Profit,
			Trades: int64(m.Trades),
		})
	}

	return connect.NewResponse(&antv1.GetMonthlyPnLResponse{
		MonthlyPnl: items,
	}), nil
}

func (s *AnalyticsServer) GetMonthlyAnalysis(ctx context.Context, req *connect.Request[antv1.GetMonthlyAnalysisRequest]) (*connect.Response[antv1.GetMonthlyAnalysisResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	years, err := s.repo.GetMonthlyAnalysisYears(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get monthly analysis years: %w", err))
	}

	points, err := s.repo.GetMonthlyAnalysisRaw(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get monthly analysis raw: %w", err))
	}

	data, err := json.Marshal(points)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal monthly analysis: %w", err))
	}

	protoYears := make([]int32, len(years))
	for i, y := range years {
		protoYears[i] = int32(y)
	}

	return connect.NewResponse(&antv1.GetMonthlyAnalysisResponse{
		Years: protoYears,
		Data:  data,
	}), nil
}
