package strategy

import (
	"fmt"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// validateFirstBarContext validates the first bar context before Init runs.
// VM-TRADE-CONTEXT-6 S5: invalid OHLCV lengths or financial fields must be
// rejected before OnInit executes — otherwise the strategy runs with corrupt
// data and g_init is set to 1, making it impossible to distinguish a valid
// init from a corrupt-data init.
//
// This is a one-time pre-Init check. Per-event validation (OHLCV lengths,
// strict decimal parse, nil positions) is done in vmHandleBar/vmHandleTick/
// vmHandleTrade. The two layers are complementary, not redundant:
//   - validateFirstBarContext: catches corrupt data before Init side effects
//   - vmHandle*: catches corrupt data on every subsequent event
func validateFirstBarContext(bctx *antv1.LiveStrategyContext) error {
	if bctx == nil {
		return fmt.Errorf("bar_context is nil")
	}

	// OHLCV array length consistency.
	if err := validateOHLCVLengths(bctx.Open, bctx.High, bctx.Low, bctx.Close, bctx.Volume, bctx.BarTimesMs); err != nil {
		return fmt.Errorf("OHLCV array length mismatch: %w", err)
	}

	// Financial fields must be valid decimals (Balance/Equity/Margin/FreeMargin).
	// Empty strings are allowed (field not provided) — only unparseable values
	// are rejected. This catches corrupt/placeholder data before Init.
	if bctx.Balance != "" {
		if _, err := parseDecimalStrict(bctx.Balance); err != nil {
			return fmt.Errorf("invalid Balance: %w", err)
		}
	}
	if bctx.Equity != "" {
		if _, err := parseDecimalStrict(bctx.Equity); err != nil {
			return fmt.Errorf("invalid Equity: %w", err)
		}
	}
	if bctx.Margin != "" {
		if _, err := parseDecimalStrict(bctx.Margin); err != nil {
			return fmt.Errorf("invalid Margin: %w", err)
		}
	}
	if bctx.FreeMargin != "" {
		if _, err := parseDecimalStrict(bctx.FreeMargin); err != nil {
			return fmt.Errorf("invalid FreeMargin: %w", err)
		}
	}

	// Validate each bar's OHLCV values are parseable decimals.
	for i := 0; i < len(bctx.Close); i++ {
		if _, err := parseDecimalStrict(bctx.Close[i]); err != nil {
			return fmt.Errorf("invalid decimal in Close[%d]: %w", i, err)
		}
		if _, err := parseDecimalStrict(bctx.Open[i]); err != nil {
			return fmt.Errorf("invalid decimal in Open[%d]: %w", i, err)
		}
		if _, err := parseDecimalStrict(bctx.High[i]); err != nil {
			return fmt.Errorf("invalid decimal in High[%d]: %w", i, err)
		}
		if _, err := parseDecimalStrict(bctx.Low[i]); err != nil {
			return fmt.Errorf("invalid decimal in Low[%d]: %w", i, err)
		}
	}

	return nil
}
