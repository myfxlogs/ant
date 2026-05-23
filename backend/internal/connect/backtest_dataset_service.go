package connect

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

func timeframeToMinutesForDataset(tf string) int {
	tf = strings.ToUpper(strings.TrimSpace(tf))
	if tf == "" {
		return 60
	}
	unit := tf[len(tf)-1:]
	numStr := tf[:len(tf)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 60
	}
	switch unit {
	case "M":
		return n
	case "H":
		return n * 60
	case "D":
		return n * 60 * 24
	default:
		return 60
	}
}

type BacktestDatasetService struct {
	dsSvc   *service.BacktestDatasetService
	klineSvc *service.KlineService
	dynamicCfg *service.DynamicConfigService
}

func NewBacktestDatasetService(dsSvc *service.BacktestDatasetService, klineSvc *service.KlineService, dynamicCfg *service.DynamicConfigService) *BacktestDatasetService {
	return &BacktestDatasetService{dsSvc: dsSvc, klineSvc: klineSvc, dynamicCfg: dynamicCfg}
}

func (s *BacktestDatasetService) ListBacktestDatasets(ctx context.Context, req *connect.Request[v1.ListBacktestDatasetsRequest]) (*connect.Response[v1.ListBacktestDatasetsResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.dsSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dataset service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var accountID *uuid.UUID
	if req.Msg.GetAccountId() != "" {
		id, perr := uuid.Parse(req.Msg.GetAccountId())
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, perr)
		}
		accountID = &id
	}
	var symbol *string
	if req.Msg.GetSymbol() != "" {
		s := strings.TrimSpace(req.Msg.GetSymbol())
		symbol = &s
	}
	var timeframe *string
	if req.Msg.GetTimeframe() != "" {
		t := strings.TrimSpace(req.Msg.GetTimeframe())
		timeframe = &t
	}

	limit := int(req.Msg.GetLimit())
	offset := int(req.Msg.GetOffset())
	items, err := s.dsSvc.ListDatasets(ctx, userID, accountID, symbol, timeframe, limit, offset)
	if err != nil {
		es := err.Error()
		if strings.Contains(es, "relation \"backtest_datasets\" does not exist") || strings.Contains(es, "column \"") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("数据库迁移未完成：缺少 backtest_datasets 表或字段，请先执行 backend/migrations 下的 .up.sql"))
		}
		return nil, err
	}

	out := make([]*v1.BacktestDataset, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		msg := &v1.BacktestDataset{
			Id: it.ID.String(),
			AccountId: it.AccountID.String(),
			Symbol: it.Symbol,
			Timeframe: it.Timeframe,
			Count: int32(it.Count),
			Frozen: it.Frozen,
			CreatedAt: timestamppb.New(it.CreatedAt.UTC()),
		}
		if it.FromTime != nil {
			msg.From = timestamppb.New(it.FromTime.UTC())
		}
		if it.ToTime != nil {
			msg.To = timestamppb.New(it.ToTime.UTC())
		}
		out = append(out, msg)
	}

	return connect.NewResponse(&v1.ListBacktestDatasetsResponse{Datasets: out}), nil
}

func (s *BacktestDatasetService) CreateFrozenBacktestDataset(ctx context.Context, req *connect.Request[v1.CreateFrozenBacktestDatasetRequest]) (*connect.Response[v1.CreateFrozenBacktestDatasetResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.dsSvc == nil || s.klineSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dataset service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	accountID, err := uuid.Parse(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetSymbol() == "" || req.Msg.GetTimeframe() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("symbol/timeframe required"))
	}
	if req.Msg.From == nil || req.Msg.To == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("from/to required"))
	}
	fromT := req.Msg.From.AsTime().UTC()
	toT := req.Msg.To.AsTime().UTC()
	if !fromT.Before(toT) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("from must be < to"))
	}
	if toT.After(time.Now().UTC()) {
		toT = time.Now().UTC()
	}

	// Fetch klines using the same service used by backtest runs.
	// Estimate bars roughly to cap payload.
	tfMin := timeframeToMinutesForDataset(req.Msg.GetTimeframe())
	if tfMin <= 0 {
		tfMin = 60
	}
	durMin := int(toT.Sub(fromT).Minutes())
	estBars := durMin/tfMin + 50
	if estBars < 200 {
		estBars = 200
	}
	if estBars > 20000 {
		estBars = 20000
	}

	klines, err := s.klineSvc.GetKlines(ctx, userID, accountID, &service.KlineRequest{
		AccountID: req.Msg.GetAccountId(),
		Symbol: req.Msg.GetSymbol(),
		Timeframe: req.Msg.GetTimeframe(),
		From: fromT.Format(time.RFC3339),
		To: toT.Format(time.RFC3339),
		Count: estBars,
	})
	if err != nil {
		return nil, err
	}
	if len(klines) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no kline data returned for the given range"))
	}

	cost := service.ResolveBacktestCostModel(ctx, s.dynamicCfg)
	dsid, err := s.dsSvc.CreateFrozenDatasetFromKlines(ctx, userID, accountID, req.Msg.GetSymbol(), req.Msg.GetTimeframe(), &fromT, &toT, estBars, klines, &cost)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateFrozenBacktestDatasetResponse{DatasetId: dsid.String()}), nil
}

func (s *BacktestDatasetService) DeleteBacktestDataset(ctx context.Context, req *connect.Request[v1.DeleteBacktestDatasetRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.dsSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dataset service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(strings.TrimSpace(req.Msg.GetDatasetId()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	deleted, err := s.dsSvc.DeleteDataset(ctx, userID, id)
	if err != nil {
		if err == service.ErrUnauthorized {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, err
	}
	if !deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("dataset not found"))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}
