package mt4

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 FetchSymbolParams: not connected")
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
			return nil, fmt.Errorf("mt4 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return nil, fmt.Errorf("mt4 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil {
			continue
		}
		si := r.GetSymbol()
		gp := r.GetGroupParams()
		param := &mthub.SymbolParam{
			Canonical:   c,
			SymbolRaw:   c,
			SpreadFloat: si.GetSpread() > 0,
		}
		if si != nil {
			param.Digits = si.GetDigits()
			param.StopLevel = si.GetStopsLevel()
			param.PointValue = decimal.NewFromFloat(si.GetPoint())
			param.ContractSize = decimal.NewFromFloat(si.GetContractSize())
			param.LotSize = param.ContractSize
		}
		if gp != nil {
			param.LotMin = decimal.NewFromFloat(gp.GetMinLot())
			param.LotMax = decimal.NewFromFloat(gp.GetMaxLot())
			param.LotStep = decimal.NewFromFloat(gp.GetLotStep())
			param.TradeMode = gp.GetExecution()
		}
		// Do not default ContractSize to 1; zero means "unknown" and triggers
		// fail-closed margin checks in the risk gate.
		out = append(out, param)
	}
	return out, nil
}

// FetchPriceHistory fetches K-line bars from the broker (MT4 QuoteHistory RPC).
// Delegates to GetPriceHistory to avoid duplicating the RPC call and auth logic.
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

// FetchAllSymbols returns all available symbol names from the broker (MT4 Symbols RPC).
func (g *Gateway) FetchAllSymbols(ctx context.Context) ([]string, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 FetchAllSymbols: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx2 := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.Symbols(ctx2, &pb.SymbolsRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt4 Symbols: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 Symbols: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}
