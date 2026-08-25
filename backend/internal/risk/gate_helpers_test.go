package risk

import (
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ── Test helpers ───────────────────────────────────────────────────────

// newTestGate creates a Gate with all 10 user-configurable rules for testing.
func newTestGate() *Gate {
	return NewGate(
		&MaxLotSize{MaxLots: decimal.NewFromInt(10)},
		&MaxPositionCount{Max: 20},
		&MaxExposure{MaxRatio: decimal.NewFromFloat(0.5)},
		&DailyLossBreaker{MaxDailyLoss: decimal.Zero},
		&DrawdownBreaker{MaxDrawdownPct: decimal.NewFromFloat(0.30)},
		&SymbolWhitelist{Whitelist: nil},
		&LeverageCap{MaxLeverage: 500},
		&OrderFrequencyLimit{MaxOrders: 60, Window: time.Minute},
		&DuplicateProtection{DedupWindow: 5 * time.Second},
		&MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)},
	)
}

func intentBuy(vol string) *antv1.OrderIntent {
	return &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    vol,
		Type:      "market",
		Price:     "1.08500",
		Magic:     1,
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
}

func intentSim(vol string) *antv1.OrderIntent {
	i := intentBuy(vol)
	i.Source = antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM
	return i
}

func defaultState() *AccountState {
	return &AccountState{
		Balance:        decimal.NewFromInt(10000),
		Equity:         decimal.NewFromInt(10050),
		FreeMargin:     decimal.NewFromInt(9550),
		UsedMargin:     decimal.NewFromInt(500),
		OpenPositions:  1,
		DailyPnL:       decimal.NewFromInt(50),
		PeakEquity:     decimal.NewFromInt(10100),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}
}
