package marketplace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/ai"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

type MarketRegimeServer struct {
	repo           *repository.MarketRegimeRepository
	marketDataRepo repository.MarketDataStore
	platformSvc    *service.PlatformService
	log            *zap.Logger
}

var _ antv1c.MarketRegimeServiceHandler = (*MarketRegimeServer)(nil)

func NewMarketRegimeServer(
	repo *repository.MarketRegimeRepository,
	marketDataRepo repository.MarketDataStore,
	platformSvc *service.PlatformService,
	log *zap.Logger,
) *MarketRegimeServer {
	return &MarketRegimeServer{
		repo:           repo,
		marketDataRepo: marketDataRepo,
		platformSvc:    platformSvc,
		log:            log,
	}
}

// detectResult holds the outcome of regime detection from kline data.
type detectResult struct {
	regime     string
	confidence float64
	features   *structpb.Struct
	segments   []string
}

func (s *MarketRegimeServer) DetectMarketRegime(ctx context.Context, req *connect.Request[antv1.DetectMarketRegimeRequest]) (*connect.Response[antv1.DetectMarketRegimeResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}

	userID, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user_id: %w", err))
	}

	if req.Msg.Symbol == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("symbol is required"))
	}
	if req.Msg.Timeframe == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("timeframe is required"))
	}

	// Broker is required for deterministic kline queries.
	broker, err := s.platformSvc.GetAccountBroker(ctx, accountID.String())
	if err != nil {
		s.log.Error("market regime: failed to resolve broker", zap.String("account", accountID.String()), zap.Error(err))
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("cannot resolve broker for account %s", accountID))
	}
	if broker == "" {
		s.log.Error("market regime: broker not found for account", zap.String("account", accountID.String()))
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("no broker configured for account %s", accountID))
	}

	// Resolve symbol to canonical form so the kline query matches stored data.
	canonical := s.platformSvc.ResolveSymbol(ctx, accountID.String(), req.Msg.Symbol)

	// Extract optional time bounds once — used by both detection and row.
	var fromTime, toTime *time.Time
	if req.Msg.From != nil {
		t := req.Msg.From.AsTime()
		fromTime = &t
	}
	if req.Msg.To != nil {
		t := req.Msg.To.AsTime()
		toTime = &t
	}

	det, err := s.detectRegime(ctx, canonical, broker, req.Msg.Timeframe, fromTime, toTime)
	if err != nil {
		return nil, err
	}

	row := &repository.MarketRegime{
		UserID:           userID,
		AccountID:        accountID,
		Symbol:           canonical,
		Timeframe:        req.Msg.Timeframe,
		Regime:           det.regime,
		Confidence:       det.confidence,
		Features:         det.features,
		Segments:         det.segments,
		StrategyFamilies: []string{},
		FromTime:         fromTime,
		ToTime:           toTime,
	}

	if err := s.repo.Create(ctx, row); err != nil {
		return nil, fmt.Errorf("create market regime: %w", err)
	}

	// Return the created row directly — no redundant Get round-trip.
	// Create already set row.ID and row.CreatedAt, so row is complete.
	return connect.NewResponse(&antv1.DetectMarketRegimeResponse{
		Regime: marketRegimeToProto(row),
	}), nil
}

func (s *MarketRegimeServer) GetMarketRegime(ctx context.Context, req *connect.Request[antv1.GetMarketRegimeRequest]) (*connect.Response[antv1.GetMarketRegimeResponse], error) {
	uid, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	regimeID, err := uuid.Parse(req.Msg.RegimeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid regime_id: %w", err))
	}

	row, err := s.repo.Get(ctx, uid, regimeID)
	if err != nil {
		if errors.Is(err, repository.ErrMarketRegimeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, fmt.Errorf("get market regime: %w", err)
	}

	return connect.NewResponse(&antv1.GetMarketRegimeResponse{
		Regime: marketRegimeToProto(row),
	}), nil
}

// detectRegime fetches klines and runs regime detection.
// Returns an error when detection cannot be performed (no klines, insufficient bars, etc.)
// so the caller never inserts a meaningless empty row.
func (s *MarketRegimeServer) detectRegime(ctx context.Context, symbol, broker, timeframe string, fromTime, toTime *time.Time) (*detectResult, error) {
	if s.marketDataRepo == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("market data store not available"))
	}

	bars, err := s.marketDataRepo.GetKlines(ctx, symbol, broker, timeframe, fromTime, toTime, 2000)
	if err != nil {
		return nil, fmt.Errorf("fetch klines for %s/%s: %w", symbol, timeframe, err)
	}
	if len(bars) < 30 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("insufficient kline data for %s/%s: need at least 30 bars, got %d", symbol, timeframe, len(bars)))
	}

	// Reverse bars so oldest is first — DetectRegime expects chronological order.
	ohlc := make([]ai.OHLCBar, len(bars))
	for i := 0; i < len(bars); i++ {
		b := bars[len(bars)-1-i]
		ohlc[i] = ai.OHLCBar{Open: b.Open.InexactFloat64(), High: b.High.InexactFloat64(), Low: b.Low.InexactFloat64(), Close: b.Close.InexactFloat64(), Volume: b.Volume}
	}
	result := ai.DetectRegime(ohlc)

	featMap := map[string]any{
		"price_change_pct":        result.Features.PriceChangePct,
		"ema_gap_pct":             result.Features.EMAGapPct,
		"realized_vol_pct":        result.Features.RealizedVolPct,
		"atr_pct":                 result.Features.ATRPct,
		"directional_efficiency":  result.Features.DirectionalEfficiency,
		"volume_ratio":            result.Features.VolumeRatio,
	}
	features, _ := structpb.NewStruct(featMap)

	return &detectResult{
		regime:     result.Regime.String(),
		confidence: result.Confidence,
		features:   features,
		segments:   []string{},
	}, nil
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
		Features:         r.Features,
		Segments:         r.Segments,
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
