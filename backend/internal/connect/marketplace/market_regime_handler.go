package connect

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
)

type MarketRegimeServer struct {
	repo *repository.MarketRegimeRepository
	log  *zap.Logger
}

var _ antv1c.MarketRegimeServiceHandler = (*MarketRegimeServer)(nil)

func NewMarketRegimeServer(repo *repository.MarketRegimeRepository, log *zap.Logger) *MarketRegimeServer {
	return &MarketRegimeServer{repo: repo, log: log}
}

func (s *MarketRegimeServer) DetectMarketRegime(ctx context.Context, req *connect.Request[antv1.DetectMarketRegimeRequest]) (*connect.Response[antv1.DetectMarketRegimeResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}

	row := &repository.MarketRegime{
		AccountID:  accountID,
		Symbol:     req.Msg.Symbol,
		Timeframe:  req.Msg.Timeframe,
		Confidence: 0,
		Features:   []byte(`{}`),
		Segments:   []byte(`[]`),
	}

	if req.Msg.From != nil {
		t := req.Msg.From.AsTime()
		row.FromTime = &t
	}
	if req.Msg.To != nil {
		t := req.Msg.To.AsTime()
		row.ToTime = &t
	}

	if err := s.repo.Create(ctx, row); err != nil {
		return nil, fmt.Errorf("create market regime: %w", err)
	}

	regime, err := s.repo.Get(ctx, row.UserID, row.ID)
	if err != nil {
		return nil, fmt.Errorf("get market regime after create: %w", err)
	}

	return connect.NewResponse(&antv1.DetectMarketRegimeResponse{
		Regime: marketRegimeToProto(regime),
	}), nil
}

func (s *MarketRegimeServer) GetMarketRegime(ctx context.Context, req *connect.Request[antv1.GetMarketRegimeRequest]) (*connect.Response[antv1.GetMarketRegimeResponse], error) {
	regimeID, err := uuid.Parse(req.Msg.RegimeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid regime_id: %w", err))
	}

	row, err := s.repo.GetByID(ctx, regimeID)
	if err != nil {
		return nil, fmt.Errorf("get market regime: %w", err)
	}

	return connect.NewResponse(&antv1.GetMarketRegimeResponse{
		Regime: marketRegimeToProto(row),
	}), nil
}

func marketRegimeToProto(r *repository.MarketRegime) *antv1.MarketRegime {
	p := &antv1.MarketRegime{
		Id:               r.ID.String(),
		UserId:           r.UserID.String(),
		AccountId:        r.AccountID.String(),
		Symbol:           r.Symbol,
		Timeframe:        r.Timeframe,
		Regime:           r.Regime,
		Confidence:       r.Confidence,
		Features:         string(r.Features),
		Segments:         string(r.Segments),
		StrategyFamilies: r.StrategyFamilies,
		ModelVersion:     r.ModelVersion,
		CreatedAt:        timestamppb.New(r.CreatedAt),
	}
	if r.FromTime != nil {
		p.FromTime = timestamppb.New(*r.FromTime)
	}
	if r.ToTime != nil {
		p.ToTime = timestamppb.New(*r.ToTime)
	}
	return p
}

