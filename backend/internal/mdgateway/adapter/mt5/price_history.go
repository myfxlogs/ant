package mt5

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// GetPriceHistory implements backfiller.MTAPIBarSource via MT5 PriceHistory RPC.
// Requires gRPC metadata with session token for authorization on the mtapi proxy.
func (g *Gateway) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	g.mu.RLock()
	qhCli := g.qhCli
	sid := g.sessionID
	g.mu.RUnlock()

	if qhCli == nil || sid == "" {
		return nil, fmt.Errorf("mt5 GetPriceHistory: not connected")
	}

	// Build gRPC metadata with session token — required by mtapi for authorization.
	authMd := make(map[string]string, 2)
	authMd["id"] = sid
	if tok := g.token(); tok != "" {
		authMd["authorization"] = "Bearer " + tok
	}
	authCtx := metadata.NewOutgoingContext(ctx, metadata.New(authMd))

	tf := mt5PeriodToTimeframe(period)
	fromStr := time.Unix(from, 0).UTC().Format("2006-01-02T15:04:05")
	toStr := time.Unix(to, 0).UTC().Format("2006-01-02T15:04:05")

	resp, err := qhCli.PriceHistory(authCtx, &pb.PriceHistoryRequest{
		Id: sid, Symbol: symbolRaw, From: fromStr, To: toStr, TimeFrame: tf,
	})
	if err != nil {
		g.log.Error("mt5 GetPriceHistory: RPC failed",
			zap.String("symbol", symbolRaw), zap.Error(err))
		return nil, fmt.Errorf("mt5 PriceHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt5 GetPriceHistory: broker error",
			zap.String("symbol", symbolRaw), zap.Int32("code", int32(resp.GetError().GetCode())),
			zap.String("msg", resp.GetError().GetMessage()))
		return nil, fmt.Errorf("mt5 PriceHistory: code=%d msg=%s",
			resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	bars := convertMT5Bars(resp.GetResult(), accountID, period)
	g.log.Info("mt5 GetPriceHistory: response",
		zap.String("symbol", symbolRaw), zap.Int("bars", len(bars)),
		zap.String("from", fromStr), zap.String("to", toStr), zap.Int32("tf", tf))
	return bars, nil
}

func mt5PeriodToTimeframe(period string) int32 {
	switch period {
	case "1m":
		return 1
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	case "1w":
		return 10080 // MT5 PERIOD_W1
	default:
		return 60
	}
}

func convertMT5Bars(bars []*pb.Bar, accountID, period string) []*mdtick.Bar {
	pm := mdtick.PeriodMs(period)
	var out []*mdtick.Bar
	for _, b := range bars {
		t := b.GetTime().AsTime()
		out = append(out, &mdtick.Bar{
			AccountID:     accountID,
			Period:        period,
			OpenTsUnixMs:  t.UnixMilli(),
			CloseTsUnixMs: t.UnixMilli() + pm,
			Open:          decimal.NewFromFloat(b.GetOpenPrice()),
			High:          decimal.NewFromFloat(b.GetHighPrice()),
			Low:           decimal.NewFromFloat(b.GetLowPrice()),
			Close:         decimal.NewFromFloat(b.GetClosePrice()),
			Volume:        float64(b.GetVolume()),
			TickCount:     uint32(b.GetTickVolume()),
			IsClosed:      true,
		})
	}
	return out
}
