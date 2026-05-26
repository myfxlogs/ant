package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/service"
)

// MarketServer implements ant.v1.MarketServiceHandler.
type MarketServer struct {
	platform *service.PlatformService
	ch       clickhouse.Conn
	nc       *nats.Conn
	log      *zap.Logger
}

var _ antv1c.MarketServiceHandler = (*MarketServer)(nil)

// NewMarketServer creates a market handler with ClickHouse + NATS for real data.
func NewMarketServer(svc *service.PlatformService, ch clickhouse.Conn, nc *nats.Conn, log *zap.Logger) *MarketServer {
	return &MarketServer{platform: svc, ch: ch, nc: nc, log: log}
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
		period = "M1" // default 1-min bars
	}

	rows, err := s.ch.Query(ctx,
		`SELECT canonical, broker, ts_unix_ms, open, high, low, close, tick_volume
		 FROM kline
		 WHERE canonical = $1 AND period = $2
		 ORDER BY ts_unix_ms DESC
		 LIMIT $3`,
		m.Canonical, period, limit,
	)
	if err != nil {
		s.log.Error("GetKlines", zap.Error(err))
		return connect.NewResponse(&antv1.GetKlinesResponse{}), nil
	}
	defer rows.Close()

	var bars []*antv1.OHLCV
	for rows.Next() {
		var canonical, broker string
		var ts int64
		var open, high, low, close float64
		var vol int64
		if err := rows.Scan(&canonical, &broker, &ts, &open, &high, &low, &close, &vol); err != nil {
			continue
		}
		bars = append(bars, &antv1.OHLCV{
			OpenTime:  timestamppb.New(time.UnixMilli(ts)),
			Open:      decimalFromFloat(open),
			High:      decimalFromFloat(high),
			Low:       decimalFromFloat(low),
			Close:     decimalFromFloat(close),
			Volume:    float64(vol),
			TickCount: 0,
		})
	}
	if err := rows.Err(); err != nil {
		s.log.Warn("GetKlines rows iteration error", zap.Error(err))
	}
	if bars == nil {
		bars = []*antv1.OHLCV{}
	}
	return connect.NewResponse(&antv1.GetKlinesResponse{Bars: bars}), nil
}

// GetSymbolStats returns current bid/ask/spread from the latest tick.
func (s *MarketServer) GetSymbolStats(ctx context.Context, req *connect.Request[antv1.GetSymbolStatsRequest]) (*connect.Response[antv1.GetSymbolStatsResponse], error) {
	var bid, ask string
	row := s.ch.QueryRow(ctx,
		`SELECT bid, ask FROM tick_raw
		 WHERE canonical = $1
		 ORDER BY ts_unix_ms DESC
		 LIMIT 1`,
		req.Msg.Canonical,
	)
	if err := row.Scan(&bid, &ask); err != nil {
		return connect.NewResponse(&antv1.GetSymbolStatsResponse{
			CurrentBid: "0", CurrentAsk: "0", Spread: "0",
		}), nil
	}
	spread := "0"
	bidF, _ := decimalToFloat(bid)
	askF, _ := decimalToFloat(ask)
	if bidF > 0 && askF > 0 {
		spread = fmt.Sprintf("%.5f", askF-bidF)
	}
	return connect.NewResponse(&antv1.GetSymbolStatsResponse{
		CurrentBid: bid,
		CurrentAsk: ask,
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
