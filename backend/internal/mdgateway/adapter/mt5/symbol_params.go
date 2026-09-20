package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchSymbolParams: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	out := make([]*mthub.SymbolParam, 0, len(canonicals))
	for _, c := range canonicals {
		ctx2 := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.SymbolParams(ctx2, &pb.SymbolParamsRequest{Id: sid, Symbol: c})
		if err != nil {
			return nil, fmt.Errorf("mt5 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return nil, fmt.Errorf("mt5 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil {
			continue
		}
		si := r.GetSymbolInfo()
		sg := r.GetSymbolGroup()
		// VM-LIVE-PARITY-F2: PointValue is the point SIZE (mt5 pb `points`),
		// not the monetary tick value — the old mapping mislabeled every
		// symbol. Broker facts only; each field keeps its own source.
		pointValue := decimal.NewFromFloat(si.GetPoints())
		if pointValue.IsZero() {
			pointValue = decimal.NewFromFloat(si.GetTickSize())
		}
		// VM-LIVE-VENUE-R2: mt5 pb has no freeze/execution-mode fields → -1 = unknown (0 is a real enum value, not "no data").
		out = append(out, &mthub.SymbolParam{
			Canonical:    c,
			SymbolRaw:    c,
			Digits:       si.GetDigits(),
			TradeMode:    int32(sg.GetTradeMode()),
			TradeExemode: -1,
			StopLevel:    sg.GetSL(),
			FreezeLevel:  -1,
			PointValue:   pointValue,
			ContractSize: decimal.NewFromFloat(si.GetContractSize()),
			LotSize:      decimal.NewFromFloat(si.GetContractSize()),
			LotStep:      decimal.NewFromFloat(sg.GetLotsStep()),
			LotMin:       decimal.NewFromFloat(sg.GetMinLots()),
			LotMax:       decimal.NewFromFloat(sg.GetMaxLots()),
			TickValue:    decimal.NewFromFloat(si.GetTickValue()),
			TickSize:     decimal.NewFromFloat(si.GetTickSize()),
			SpreadFloat:  si.GetSpread() > 0,
		})
	}
	return out, nil
}

// FetchPriceHistory fetches K-line bars via the broker PriceHistory RPC in quotes.go.
func (g *Gateway) FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*mthub.Bar, error) {
	bars, err := g.GetPriceHistory(ctx, "", symbol, period, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]*mthub.Bar, 0, len(bars))
	for _, b := range bars {
		out = append(out, &mthub.Bar{
			Time:   time.UnixMilli(b.OpenTsUnixMs),
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: decimal.NewFromFloat(b.Volume),
		})
	}
	return out, nil
}

// FetchAllSymbols returns all available symbol names from the broker.
func (g *Gateway) FetchAllSymbols(ctx context.Context) ([]string, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchAllSymbols: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx2 := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.SymbolList(ctx2, &pb.SymbolListRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 SymbolList: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 SymbolList: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}
