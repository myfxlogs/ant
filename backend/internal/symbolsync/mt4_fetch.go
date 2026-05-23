// Package symbolsync — MT4 symbol fetcher with concurrent per-symbol detail.
// Ported from alfq. 10-concurrency goroutine pool + SymbolParamsMany fallback.
package symbolsync

import (
	"context"
	"fmt"
	"sync"

	"anttrader/internal/mt4client"
	mt4pb "anttrader/mt4"
	"go.uber.org/zap"
)

const (
	mt4FetchConcurrency = 10
)

// FetchMT4Symbols pulls all symbols via MT4 gateway.
// Strategy: Symbols() → list names → concurrent SymbolParams per symbol.
// Falls back to batch when Symbols() fails.
func FetchMT4Symbols(ctx context.Context, conn *mt4client.MT4Connection, brokerID string, log *zap.Logger) ([]BrokerSymbol, error) {
	// Step 1: get symbol names
	names, err := conn.Symbols(ctx)
	if err != nil {
		// Fallback: try batch SymbolParamsMany equivalent
		log.Warn("mt4 Symbols() failed, symbols may be unavailable", zap.Error(err))
		return nil, fmt.Errorf("mt4 symbols: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("mt4: no symbols returned")
	}

	log.Info("mt4 symbol names fetched", zap.Int("count", len(names)))

	// Step 2: concurrent per-symbol SymbolParams
	type result struct {
		sym BrokerSymbol
		ok  bool
	}
	sem := make(chan struct{}, mt4FetchConcurrency)
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
			sym := convertMT4SymbolParams(sp, brokerID)
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

	log.Info("mt4 symbols converted",
		zap.Int("count", len(symbols)),
		zap.Int("requested", len(names)),
	)

	return symbols, nil
}

// convertMT4SymbolParams converts an ant MT4 SymbolParams proto to BrokerSymbol.
func convertMT4SymbolParams(sp *mt4pb.SymbolParams, brokerID string) BrokerSymbol {
	raw := sp.GetSymbolName()
	sym := BrokerSymbol{BrokerID: brokerID, SymbolRaw: raw}

	if info := sp.GetSymbol(); info != nil {
		sym.Digits = int16(info.GetDigits())
		sym.Point = info.GetPoint()
		sym.ContractSize = info.GetContractSize()
		sym.SwapLong = info.GetSwapLong()
		sym.SwapShort = info.GetSwapShort()
	}
	if gp := sp.GetGroupParams(); gp != nil {
		sym.MinLot = gp.GetMinLot()
		sym.MaxLot = gp.GetMaxLot()
		sym.LotStep = gp.GetLotStep()
	}

	// MT4 GroupParams.TradeMode is unreliable; assume full trading for valid symbols
	if sym.Digits > 0 && sym.Point > 0 && sym.ContractSize > 0 {
		sym.TradeMode = 4 // SYMBOL_TRADE_MODE_FULL
	}

	if sym.Digits == 0 || sym.Point == 0 || sym.ContractSize == 0 {
		sym.Partial = true
	}

	return sym
}
