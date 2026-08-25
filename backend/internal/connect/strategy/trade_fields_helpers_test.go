package strategy

import (
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

// helper: build a trade with explicit price, side, times (other fields defaulted to valid).
func makeTradeWithFields(entryPrice, exitPrice decimal.Decimal, side sdk.PositionSide, entryTime, exitTime time.Time) backtest.Trade {
	return backtest.Trade{
		Symbol:     "EURUSD",
		Side:       side,
		EntryTime:  entryTime,
		ExitTime:   exitTime,
		EntryPrice: entryPrice,
		ExitPrice:  exitPrice,
		Volume:     decimal.NewFromFloat(0.1),
		Profit:     decimal.NewFromInt(100),
		Commission: decimal.NewFromInt(5),
		Swap:       decimal.Zero,
		Comment:    "test",
	}
}

// validTrade returns a trade with all fields valid (positive prices, valid side, correct time order).
func validTrade() backtest.Trade {
	return makeTradeWithFields(
		decimal.NewFromFloat(1.1),
		decimal.NewFromFloat(1.11),
		sdk.SideBuy,
		time.UnixMilli(1000),
		time.UnixMilli(2000),
	)
}
