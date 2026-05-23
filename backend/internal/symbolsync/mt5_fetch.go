// Package symbolsync — MT5 symbol fetcher.
// Ported from alfq. Uses ant's mt5client for symbol discovery.
package symbolsync

import (
	"context"
	"fmt"
	"sync"

	"anttrader/internal/mt5client"
	mt5pb "anttrader/mt5"
	"go.uber.org/zap"
)

const (
	mt5FetchConcurrency = 10
)

// FetchMT5Symbols pulls all symbols via MT5 gateway.
// Strategy: SymbolList() → names → concurrent SymbolParams per symbol for full metadata.
func FetchMT5Symbols(ctx context.Context, conn *mt5client.MT5Connection, brokerID string, log *zap.Logger) ([]BrokerSymbol, error) {
	names, err := conn.SymbolList(ctx)
	if err != nil {
		return nil, fmt.Errorf("mt5 symbol list: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("mt5: no symbols returned")
	}

	log.Info("mt5 symbol names fetched", zap.Int("count", len(names)))

	type result struct {
		sym BrokerSymbol
		ok  bool
	}
	sem := make(chan struct{}, mt5FetchConcurrency)
	results := make(chan result, len(names))
	var wg sync.WaitGroup

	for _, name := range names {
		wg.Add(1)
		go func(symbolName string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			sp, err := conn.SymbolParams(ctx, symbolName)
			if err != nil || sp == nil {
				return
			}
			sym := convertMT5SymbolParams(sp, brokerID)
			results <- result{sym: sym, ok: true}
		}(name)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var symbols []BrokerSymbol
	for r := range results {
		if r.ok {
			symbols = append(symbols, r.sym)
		}
	}

	log.Info("mt5 symbols converted",
		zap.Int("count", len(symbols)),
		zap.Int("requested", len(names)),
	)
	return symbols, nil
}

func convertMT5SymbolParams(sp *mt5pb.SymbolParams, brokerID string) BrokerSymbol {
	raw := sp.GetSymbol()
	sym := BrokerSymbol{BrokerID: brokerID, SymbolRaw: raw}

	if info := sp.GetSymbolInfo(); info != nil {
		sym.Digits = int16(info.GetDigits())
		sym.Point = info.GetPoints()
		sym.TickSize = info.GetTickSize()
		sym.TickValue = info.GetTickValue()
		sym.ContractSize = info.GetContractSize()
		sym.Description = info.GetDescription()
		sym.MarginCurrency = info.GetMarginCurrency()
		sym.ProfitCurrency = info.GetProfitCurrency()
	}
	if grp := sp.GetSymbolGroup(); grp != nil {
		sym.MinLot = grp.GetMinLots()
		sym.MaxLot = grp.GetMaxLots()
		sym.LotStep = grp.GetLotsStep()
		sym.MarginInitial = grp.GetInitialMargin()
		sym.SwapLong = grp.GetSwapLong()
		sym.SwapShort = grp.GetSwapShort()
		sym.TradeMode = int16(grp.GetTradeMode())
		sym.SwapMode = int16(grp.GetSwapType())
		sym.SwapRolloverDay = int16(grp.GetThreeDaysSwap())
	}

	if sym.Digits == 0 || sym.Point == 0 || sym.ContractSize == 0 {
		sym.Partial = true
	}

	return sym
}
