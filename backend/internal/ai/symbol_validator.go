// Package ai — SymbolValidator: extract + resolve canonical symbols (M4-2, M4-3).
// Combines SymbolExtractor (NL→canonical) with symbol.Resolver (canonical→broker check).
package ai

import (
	"context"
	"fmt"

	"anttrader/internal/symbol"
)

// SymbolValidator extracts canonical symbols from user text and validates them
// against the broker_symbols table via SymbolResolver.
type SymbolValidator struct {
	extractor *SymbolExtractor
	resolver  *symbol.Resolver
}

// NewSymbolValidator creates a validator backed by a symbol resolver.
// resolver may be nil (validation skipped).
func NewSymbolValidator(resolver *symbol.Resolver) *SymbolValidator {
	return &SymbolValidator{
		extractor: NewSymbolExtractor(),
		resolver:  resolver,
	}
}

// ValidateResult is the output of extracting + resolving symbols.
type ValidateResult struct {
	Extracted  []ExtractedSymbol    `json:"extracted"`
	Resolved   []ResolvedSymbol     `json:"resolved"`
	Warnings   []string             `json:"warnings,omitempty"`
}

// ResolvedSymbol is an extracted canonical that has been verified against a broker.
type ResolvedSymbol struct {
	Canonical     string `json:"canonical"`
	SymbolRaw     string `json:"symbol_raw,omitempty"`
	TradeMode     int32  `json:"trade_mode"` // 0=disabled 1=long 2=short 3=full
	IsTradeable   bool   `json:"is_tradeable"`
	Confidence    float64 `json:"confidence"`
}

// ExtractAndValidate extracts canonical symbols from user text and validates them
// against the given account's broker. Returns a structured result for the frontend.
func (v *SymbolValidator) ExtractAndValidate(ctx context.Context, text string, accountID string) *ValidateResult {
	extracted := v.extractor.Extract(ctx, text)
	result := &ValidateResult{Extracted: extracted}

	if v.resolver == nil || accountID == "" {
		if len(extracted) == 0 {
			result.Warnings = append(result.Warnings, "no canonical symbols detected in input")
		}
		return result
	}

	for _, sym := range extracted {
		info, tradeable, err := v.resolver.ResolveCanonical(ctx, accountID, sym.Canonical)
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("symbol %s: not found on broker for account %s", sym.Canonical, accountID))
			result.Resolved = append(result.Resolved, ResolvedSymbol{
				Canonical:   sym.Canonical,
				IsTradeable: false,
				Confidence:  sym.Confidence,
			})
			continue
		}
		result.Resolved = append(result.Resolved, ResolvedSymbol{
			Canonical:   sym.Canonical,
			SymbolRaw:   info.SymbolRaw,
			TradeMode:   info.TradeMode,
			IsTradeable: tradeable,
			Confidence:  sym.Confidence,
		})
	}

	return result
}
