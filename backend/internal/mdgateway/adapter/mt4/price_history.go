package mt4

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 GetPriceHistory: not connected")
	}

	tf, ok := mt4PeriodToTimeframe(period)
	if !ok {
		return nil, fmt.Errorf("mt4 GetPriceHistory: unsupported period %q", period)
	}

	count := int32(((to - from) * 1000) / mdtick.PeriodMs(period))
	if count <= 0 {
		count = 100
	}
	if count > 5000 {
		count = 5000
	}
	fromStr := time.Unix(to, 0).UTC().Format("2006-01-02T15:04:05")

	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	authCtx := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.QuoteHistory(authCtx, &pb.QuoteHistoryRequest{
		Id: sid, Symbol: symbolRaw, Timeframe: tf, From: fromStr, Count: count,
	})
	if err != nil {
		return nil, fmt.Errorf("mt4 QuoteHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 QuoteHistory: code=%d msg=%s",
			resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return convertMT4Bars(resp.GetResult(), accountID, period), nil
}

func mt4PeriodToTimeframe(period string) (pb.Timeframe, bool) {
	switch period {
	case "1m":
		return pb.Timeframe_Timeframe_M1, true
	case "5m":
		return pb.Timeframe_Timeframe_M5, true
	case "15m":
		return pb.Timeframe_Timeframe_M15, true
	case "30m":
		return pb.Timeframe_Timeframe_M30, true
	case "1h":
		return pb.Timeframe_Timeframe_H1, true
	case "4h":
		return pb.Timeframe_Timeframe_H4, true
	case "1d":
		return pb.Timeframe_Timeframe_D1, true
	case "1w":
		return pb.Timeframe_Timeframe_W1, true
	default:
		return 0, false
	}
}

func convertMT4Bars(bars []*pb.Bar, accountID, period string) []*mdtick.Bar {
	var out []*mdtick.Bar
	for _, b := range bars {
		t := b.GetTime().AsTime()
		out = append(out, &mdtick.Bar{
			AccountID:     accountID,
			Period:        period,
			OpenTsUnixMs:  t.UnixMilli(),
			CloseTsUnixMs: t.UnixMilli() + mdtick.PeriodMs(period),
			Open:          decimal.NewFromFloat(b.GetOpen()),
			High:          decimal.NewFromFloat(b.GetHigh()),
			Low:           decimal.NewFromFloat(b.GetLow()),
			Close:         decimal.NewFromFloat(b.GetClose()),
			Volume:        b.GetVolume(),
			IsClosed:      true,
		})
	}
	return out
}
