package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// MarketServer implements ant.v1.MarketServiceHandler.
type MarketServer struct {
	platform  *service.PlatformService
	marketData *repository.MarketDataRepository
	nc        *nats.Conn
	log       *zap.Logger
}

var _ antv1c.MarketServiceHandler = (*MarketServer)(nil)

func NewMarketServer(svc *service.PlatformService, marketData *repository.MarketDataRepository, nc *nats.Conn, log *zap.Logger) *MarketServer {
	return &MarketServer{platform: svc, marketData: marketData, nc: nc, log: log}
}

// GetKlines returns OHLCV kline data from ClickHouse.
func (s *MarketServer) GetKlines(ctx context.Context, req *connect.Request[antv1.GetKlinesRequest]) (*connect.Response[antv1.GetKlinesResponse], error) {
	m := req.Msg
	limit := int32(500)
	if m.Limit > 0 {
		limit = m.Limit
	}
	period := m.Period
	if period == "" {
		period = "M1"
	}

	bars, err := s.marketData.GetKlines(ctx, m.Canonical, period, limit)
	if err != nil {
		s.log.Error("GetKlines", zap.Error(err))
		return connect.NewResponse(&antv1.GetKlinesResponse{}), nil
	}

	var out []*antv1.OHLCV
	for _, b := range bars {
		out = append(out, &antv1.OHLCV{
			OpenTime:  timestamppb.New(b.OpenTime()),
			Open:      decimalFromFloat(b.Open),
			High:      decimalFromFloat(b.High),
			Low:       decimalFromFloat(b.Low),
			Close:     decimalFromFloat(b.Close),
			Volume:    float64(b.TickVolume),
			TickCount: 0,
		})
	}
	if out == nil {
		out = []*antv1.OHLCV{}
	}
	return connect.NewResponse(&antv1.GetKlinesResponse{Bars: out}), nil
}

// GetSymbolStats returns current bid/ask/spread from the latest tick.
func (s *MarketServer) GetSymbolStats(ctx context.Context, req *connect.Request[antv1.GetSymbolStatsRequest]) (*connect.Response[antv1.GetSymbolStatsResponse], error) {
	tick, err := s.marketData.GetLatestTick(ctx, req.Msg.Canonical)
	if err != nil {
		return connect.NewResponse(&antv1.GetSymbolStatsResponse{
			CurrentBid: "0", CurrentAsk: "0", Spread: "0",
		}), nil
	}
	spread := "0"
	bidF, _ := decimalToFloat(tick.Bid)
	askF, _ := decimalToFloat(tick.Ask)
	if bidF > 0 && askF > 0 {
		spread = fmt.Sprintf("%.5f", askF-bidF)
	}
	return connect.NewResponse(&antv1.GetSymbolStatsResponse{
		CurrentBid: tick.Bid,
		CurrentAsk: tick.Ask,
		Spread:     spread,
	}), nil
}

// StreamTicks subscribes to NATS tick.> pattern and forwards TickMsg to the client.
func (s *MarketServer) StreamTicks(ctx context.Context, req *connect.Request[antv1.StreamTicksRequest], stream *connect.ServerStream[antv1.TickMsg]) error {
	m := req.Msg
	subject := "tick.>"
	if len(m.Canonicals) == 1 {
		subject = fmt.Sprintf("tick.%s", m.Canonicals[0])
	}
	sub, err := s.nc.SubscribeSync(subject)
	if err != nil {
		s.log.Error("StreamTicks: subscribe failed", zap.String("subject", subject), zap.Error(err))
		return connect.NewError(connect.CodeInternal, err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		msg, err := sub.NextMsg(500 * time.Millisecond)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		tick := antv1.TickMsg{}
		if err := json.Unmarshal(msg.Data, &tick); err != nil {
			s.log.Warn("SubscribeTicks json unmarshal failed", zap.Error(err))
			continue
		}
		if err := stream.Send(&tick); err != nil {
			return fmt.Errorf("send tick to stream: %w", err)
		}
	}
}

func decimalFromFloat(f float64) string {
	return fmt.Sprintf("%.5f", f)
}

func decimalToFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
