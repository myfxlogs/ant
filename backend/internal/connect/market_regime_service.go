package connect

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type MarketRegimeService struct {
	repo     *repository.MarketRegimeRepository
	klineSvc *service.KlineService
}

func NewMarketRegimeService(repo *repository.MarketRegimeRepository, klineSvc *service.KlineService) *MarketRegimeService {
	return &MarketRegimeService{repo: repo, klineSvc: klineSvc}
}

func (s *MarketRegimeService) DetectMarketRegime(ctx context.Context, req *connect.Request[v1.DetectMarketRegimeRequest]) (*connect.Response[v1.MarketRegime], error) {
	if s == nil || s.repo == nil || s.klineSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("market regime service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	accountID, err := parseRequestUUID(req.Msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	count := int(req.Msg.GetCount())
	if count <= 0 {
		count = 120
	}
	if count > 500 {
		count = 500
	}
	from := ""
	to := ""
	var fromTime *time.Time
	var toTime *time.Time
	if req.Msg.GetFrom() != nil {
		v := req.Msg.GetFrom().AsTime()
		fromTime = &v
		from = v.Format(time.RFC3339)
	}
	if req.Msg.GetTo() != nil {
		v := req.Msg.GetTo().AsTime()
		toTime = &v
		to = v.Format(time.RFC3339)
	}
	klines, err := s.klineSvc.GetKlines(ctx, userID, accountID, &service.KlineRequest{AccountID: accountID.String(), Symbol: req.Msg.GetSymbol(), Timeframe: req.Msg.GetTimeframe(), From: from, To: to, Count: count})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(klines) < 2 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("not enough klines to detect market regime"))
	}
	regime, confidence, features := detectRegimeFeatures(klines)
	families := regimeStrategyFamilies(regime)
	featuresJSON, _ := json.Marshal(features)
	segment := map[string]any{"from": klines[0].OpenTime, "to": klines[len(klines)-1].CloseTime, "regime": regime, "confidence": confidence}
	segmentsJSON, _ := json.Marshal([]map[string]any{segment})
	row := &repository.MarketRegime{UserID: userID, AccountID: accountID, Symbol: req.Msg.GetSymbol(), Timeframe: req.Msg.GetTimeframe(), Regime: regime, Confidence: confidence, Features: featuresJSON, Segments: segmentsJSON, StrategyFamilies: pq.StringArray(families), FromTime: fromTime, ToTime: toTime, ModelVersion: "rule-v1"}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(marketRegimeToProto(row)), nil
}

func (s *MarketRegimeService) GetMarketRegime(ctx context.Context, req *connect.Request[v1.GetMarketRegimeRequest]) (*connect.Response[v1.MarketRegime], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("market regime service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseRequestUUID(req.Msg.GetRegimeId())
	if err != nil {
		return nil, err
	}
	row, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, repository.ErrMarketRegimeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(marketRegimeToProto(row)), nil
}

func detectRegimeFeatures(klines []*service.KlineResponse) (string, float64, map[string]any) {
	first := klines[0].ClosePrice
	last := klines[len(klines)-1].ClosePrice
	if first <= 0 {
		first = klines[0].OpenPrice
	}
	trendReturn := 0.0
	if first > 0 {
		trendReturn = (last - first) / first
	}
	var sumAbs, sumSq, avgRange float64
	var prev = klines[0].ClosePrice
	for i := 1; i < len(klines); i++ {
		ret := 0.0
		if prev > 0 {
			ret = (klines[i].ClosePrice - prev) / prev
		}
		sumAbs += math.Abs(ret)
		sumSq += ret * ret
		if klines[i].ClosePrice > 0 {
			avgRange += (klines[i].HighPrice - klines[i].LowPrice) / klines[i].ClosePrice
		}
		prev = klines[i].ClosePrice
	}
	n := float64(len(klines) - 1)
	volatility := math.Sqrt(sumSq / math.Max(n, 1))
	efficiency := math.Abs(trendReturn) / math.Max(sumAbs, 0.000001)
	avgRange = avgRange / math.Max(n, 1)
	regime := "range"
	if efficiency > 0.45 && math.Abs(trendReturn) > 0.01 {
		if trendReturn > 0 {
			regime = "trend_up"
		} else {
			regime = "trend_down"
		}
	} else if volatility > 0.018 || avgRange > 0.025 {
		regime = "high_volatility"
	}
	confidence := math.Min(0.95, 0.5+efficiency*0.35+math.Min(volatility*8, 0.2))
	return regime, confidence, map[string]any{"trend_return": trendReturn, "volatility": volatility, "efficiency_ratio": efficiency, "average_range": avgRange, "sample_size": len(klines)}
}

func regimeStrategyFamilies(regime string) []string {
	switch regime {
	case "trend_up", "trend_down":
		return []string{"trend_following", "breakout"}
	case "high_volatility":
		return []string{"volatility_filter", "mean_reversion_light"}
	default:
		return []string{"mean_reversion", "range_breakout"}
	}
}

func marketRegimeToProto(row *repository.MarketRegime) *v1.MarketRegime {
	if row == nil {
		return nil
	}
	features := &structpb.Struct{}
	_ = json.Unmarshal(row.Features, features)
	out := &v1.MarketRegime{Id: row.ID.String(), AccountId: row.AccountID.String(), Symbol: row.Symbol, Timeframe: row.Timeframe, Regime: row.Regime, Confidence: row.Confidence, Features: features, StrategyFamilies: []string(row.StrategyFamilies), ModelVersion: row.ModelVersion, CreatedAt: timestamppb.New(row.CreatedAt.UTC())}
	if row.FromTime != nil {
		out.From = timestamppb.New(row.FromTime.UTC())
	}
	if row.ToTime != nil {
		out.To = timestamppb.New(row.ToTime.UTC())
	}
	out.Segments = []*v1.MarketRegimeSegment{{From: out.From, To: out.To, Regime: row.Regime, Confidence: row.Confidence}}
	return out
}
