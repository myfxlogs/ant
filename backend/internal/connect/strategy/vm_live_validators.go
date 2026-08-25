package strategy

import (
	"fmt"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// This file contains validation helpers for live VM execution.
// VM-TRADE-CONTEXT-6 round 4/5: strict validation of bar context, financial
// fields, and mode-aware fail-closed semantics.

// validateFinancialFields strict-parses account financial string fields.
// VM-TRADE-CONTEXT-6 round 4: Balance/Equity/Margin/FreeMargin must be valid
// decimals — mustDecimal in runner converts invalid to -1 sentinel (fail-open),
// so we validate here at the handler boundary to fail-closed.
// VM-TRADE-CONTEXT-6 round 5: in live mode, empty financial fields are rejected
// (authoritative broker data must be present). In paper/backtest mode, empty
// strings are allowed (simulation may not have real account data).
func validateFinancialFields(balance, equity, margin, freeMargin string) error {
	if err := validateOptionalDecimal(balance, "balance"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(equity, "equity"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(margin, "margin"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(freeMargin, "free_margin"); err != nil {
		return err
	}
	return nil
}

// validateLiveFinancialFields requires all financial fields to be non-empty
// and valid decimals. VM-TRADE-CONTEXT-6 round 5: live mode must have
// authoritative broker financial data — empty fields mean the strategy
// runs without knowing the real account balance/equity, which is fail-open.
func validateLiveFinancialFields(balance, equity, margin, freeMargin string) error {
	if balance == "" {
		return fmt.Errorf("balance: required in live mode (authoritative broker data missing)")
	}
	if equity == "" {
		return fmt.Errorf("equity: required in live mode (authoritative broker data missing)")
	}
	if margin == "" {
		return fmt.Errorf("margin: required in live mode (authoritative broker data missing)")
	}
	if freeMargin == "" {
		return fmt.Errorf("free_margin: required in live mode (authoritative broker data missing)")
	}
	return validateFinancialFields(balance, equity, margin, freeMargin)
}

// validateFinancialFieldsForMode dispatches to the mode-aware validator.
// VM-TRADE-CONTEXT-6 round 5: live mode requires non-empty authoritative
// financial fields; paper/backtest allows empty (simulation may not have
// real account data).
func validateFinancialFieldsForMode(balance, equity, margin, freeMargin, mode string) error {
	if mode == "live" {
		return validateLiveFinancialFields(balance, equity, margin, freeMargin)
	}
	return validateFinancialFields(balance, equity, margin, freeMargin)
}

// validateOptionalDecimal rejects non-empty invalid decimal strings.
// Empty string is allowed (field not provided).
func validateOptionalDecimal(s, name string) error {
	if s == "" {
		return nil
	}
	if _, err := parseDecimalStrict(s); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// validateBarContext is the shared first-request validator used by both
// VMLiveSession.Start and dispatchVMLive. It validates ALL fields before
// any runner.Init/OnInit call so that a bad request cannot execute OnInit
// and then fail mid-strategy.
// VM-TRADE-CONTEXT-6 round 4: unified validation — main OHLCV, extra Symbols
// OHLCV, nil repeated messages, strict decimal/integer parsing, financial
// fields, positions/orders domain and enum validation.
// VM-TRADE-CONTEXT-6 round 5: mode-aware financial validation — live mode
// requires non-empty authoritative financial fields; paper/backtest allows
// empty (simulation may not have real account data).
func validateBarContext(bctx *antv1.LiveStrategyContext) error {
	return validateBarContextWithMode(bctx, bctx.GetMode())
}

// validateBarContextWithMode is the mode-aware variant. In live mode,
// financial fields must be non-empty (authoritative broker data required).
func validateBarContextWithMode(bctx *antv1.LiveStrategyContext, mode string) error {
	n := len(bctx.Close)
	if n == 0 {
		return fmt.Errorf("OHLCV arrays are empty (need at least 1 bar for OnInit)")
	}
	if len(bctx.Open) != n || len(bctx.High) != n || len(bctx.Low) != n ||
		len(bctx.Volume) != n || len(bctx.BarTimesMs) != n {
		return fmt.Errorf("OHLCV array length mismatch: close=%d open=%d high=%d low=%d volume=%d times=%d",
			n, len(bctx.Open), len(bctx.High), len(bctx.Low), len(bctx.Volume), len(bctx.BarTimesMs))
	}
	// Nil repeated messages.
	for i, lp := range bctx.Positions {
		if lp == nil {
			return fmt.Errorf("positions[%d] is nil", i)
		}
	}
	for i, lo := range bctx.PendingOrders {
		if lo == nil {
			return fmt.Errorf("pending_orders[%d] is nil", i)
		}
	}
	for i, ss := range bctx.Symbols {
		if ss == nil {
			return fmt.Errorf("symbols[%d] is nil", i)
		}
	}
	// Strict-parse main OHLCV.
	for i := 0; i < n; i++ {
		if _, err := parseDecimalStrict(bctx.Open[i]); err != nil {
			return fmt.Errorf("bar[%d].open: %w", i, err)
		}
		if _, err := parseDecimalStrict(bctx.High[i]); err != nil {
			return fmt.Errorf("bar[%d].high: %w", i, err)
		}
		if _, err := parseDecimalStrict(bctx.Low[i]); err != nil {
			return fmt.Errorf("bar[%d].low: %w", i, err)
		}
		if _, err := parseDecimalStrict(bctx.Close[i]); err != nil {
			return fmt.Errorf("bar[%d].close: %w", i, err)
		}
		vol, err := parseInt64Strict(bctx.Volume[i])
		if err != nil {
			return fmt.Errorf("bar[%d].volume: %w", i, err)
		}
		if vol < 0 {
			return fmt.Errorf("bar[%d].volume: negative value %d", i, vol)
		}
	}
	// Validate extra symbol OHLCV (round 4: was missing from validateFirstBarContext).
	for _, ss := range bctx.Symbols {
		sn := len(ss.Close)
		if sn == 0 {
			continue
		}
		if len(ss.Open) != sn || len(ss.High) != sn || len(ss.Low) != sn || len(ss.Volume) != sn {
			return fmt.Errorf("symbol %s: OHLCV array length mismatch: close=%d open=%d high=%d low=%d volume=%d",
				ss.Symbol, sn, len(ss.Open), len(ss.High), len(ss.Low), len(ss.Volume))
		}
		for i := 0; i < sn; i++ {
			if _, err := parseDecimalStrict(ss.Open[i]); err != nil {
				return fmt.Errorf("symbol %s bar[%d].open: %w", ss.Symbol, i, err)
			}
			if _, err := parseDecimalStrict(ss.High[i]); err != nil {
				return fmt.Errorf("symbol %s bar[%d].high: %w", ss.Symbol, i, err)
			}
			if _, err := parseDecimalStrict(ss.Low[i]); err != nil {
				return fmt.Errorf("symbol %s bar[%d].low: %w", ss.Symbol, i, err)
			}
			if _, err := parseDecimalStrict(ss.Close[i]); err != nil {
				return fmt.Errorf("symbol %s bar[%d].close: %w", ss.Symbol, i, err)
			}
			sv, err := parseInt64Strict(ss.Volume[i])
			if err != nil {
				return fmt.Errorf("symbol %s bar[%d].volume: %w", ss.Symbol, i, err)
			}
			if sv < 0 {
				return fmt.Errorf("symbol %s bar[%d].volume: negative value %d", ss.Symbol, i, sv)
			}
		}
	}
	// Validate positions/orders (strict parse + domain + enum).
	if _, err := vmPositionsToSdk(bctx.Positions); err != nil {
		return err
	}
	if _, err := vmPendingOrdersToSdk(bctx.PendingOrders); err != nil {
		return err
	}
	// Validate financial fields (round 4: was missing — mustDecimal converts
	// invalid to -1 sentinel, which is fail-open).
	// VM-TRADE-CONTEXT-6 round 5: live mode requires non-empty authoritative
	// financial fields; paper/backtest allows empty (simulation may not have
	// real account data).
	if mode == "live" {
		if err := validateLiveFinancialFields(bctx.Balance, bctx.Equity, bctx.Margin, bctx.FreeMargin); err != nil {
			return err
		}
	} else {
		if err := validateFinancialFields(bctx.Balance, bctx.Equity, bctx.Margin, bctx.FreeMargin); err != nil {
			return err
		}
	}
	return nil
}

// parseOptionalDecimalStrict parses a decimal string that may be empty.
// Returns decimal.Zero and nil error for empty string (field not provided).
// VM-TRADE-CONTEXT-6 round 4: for financial fields where empty means "not set"
// but non-empty invalid must fail-closed.
func parseOptionalDecimalStrict(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return parseDecimalStrict(s)
}
