package backtest

import (
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// checkSLTPPath checks SL/TP using OHLC path simulation.
// Path: bullish (Close >= Open) → O→H→L→C; bearish (Close < Open) → O→L→H→C.
// 3 monotonic segments, each checked for SL/TP triggers.
// SL and TP in different segments: earlier segment triggers first.
// SL and TP in same segment: nearer to segment start triggers first (defensive, unreachable with valid SL/TP).
// Gap-at-open: if Open already crosses SL/TP, fill at Open price.
func (e *Engine) checkSLTPPath(bar sdk.Bar) {
	for i := 0; i < len(e.broker.positions); i++ {
		pos := e.broker.positions[i]
		var closed bool
		var closePrice decimal.Decimal

		if pos.Side == sdk.SideBuy {
			closed, closePrice = checkBuySLTPPath(bar.Open, bar.High, bar.Low, bar.Close, pos.StopLoss, pos.TakeProfit)
		} else {
			closed, closePrice = checkSellSLTPPath(bar.Open, bar.High, bar.Low, bar.Close, pos.StopLoss, pos.TakeProfit)
		}

		if closed {
			if closePrice.IsZero() {
				closePrice = bar.Close
			}
			pos.ClosePrice = closePrice
			heldDuration := time.UnixMilli(bar.Timestamp).Sub(pos.OpenTime)
			days := int64(heldDuration.Hours() / 24)
			if days < 0 {
				days = 0
			}
			e.broker.applySwap(pos, pos.Volume, int(days))
			contractSize := e.broker.config.ContractSize
			if contractSize.IsZero() {
				contractSize = decimal.NewFromInt(100000)
			}
			pos.Profit = pos.ClosePrice.Sub(pos.Price).Mul(pos.Volume).Mul(contractSize)
			if pos.Side == sdk.SideSell {
				pos.Profit = pos.Profit.Neg()
			}
			pos.State = OrderClosed
			pos.CloseTime = time.UnixMilli(bar.Timestamp)
			e.broker.history = append(e.broker.history, pos)
			e.broker.recordDeal(pos, pos.Volume, pos.Profit, pos.CloseTime)
			e.broker.recordTrade(pos)
			e.broker.equity = e.broker.equity.Add(pos.Profit)
			e.broker.balance = e.broker.balance.Add(pos.Profit)
			e.broker.positions = append(e.broker.positions[:i], e.broker.positions[i+1:]...)
			i--
		}
	}
}

// segment represents a monotonic price move from start to end.
type segment struct{ start, end decimal.Decimal }

// buildPath constructs the 3-segment OHLC path for a bar.
// Bullish (Close >= Open): O→H→L→C → segments [O,H], [H,L], [L,C]
// Bearish (Close < Open): O→L→H→C → segments [O,L], [L,H], [H,C]
func buildPath(o, h, l, c decimal.Decimal) []segment {
	if c.GreaterThanOrEqual(o) {
		return []segment{{o, h}, {h, l}, {l, c}}
	}
	return []segment{{o, l}, {l, h}, {h, c}}
}

// checkBuySLTPPath checks SL/TP for a buy position using OHLC path.
// Buy SL triggers when price moves down through SL; Buy TP triggers when price moves up through TP.
func checkBuySLTPPath(o, h, l, c, sl, tp decimal.Decimal) (bool, decimal.Decimal) {
	if !sl.IsPositive() && !tp.IsPositive() {
		return false, decimal.Zero
	}
	// Gap-at-open: if Open already at or below SL → SL hit at open.
	if sl.IsPositive() && o.LessThanOrEqual(sl) {
		return true, o
	}
	// Gap-at-open: if Open already at or above TP → TP hit at open.
	if tp.IsPositive() && o.GreaterThanOrEqual(tp) {
		return true, o
	}
	segs := buildPath(o, h, l, c)
	for _, seg := range segs {
		ascending := seg.end.GreaterThan(seg.start)
		// Buy SL: price moves down through SL → triggers in descending segment when start > SL >= end
		// Buy TP: price moves up through TP → triggers in ascending segment when start < TP <= end
		var slHit, tpHit bool
		if ascending {
			if tp.IsPositive() && seg.start.LessThan(tp) && seg.end.GreaterThanOrEqual(tp) {
				tpHit = true
			}
			if sl.IsPositive() && seg.start.LessThanOrEqual(sl) && seg.end.GreaterThanOrEqual(sl) {
				// Defensive: SL in ascending segment means SL < start (invalid for buy with SL < entry)
				slHit = true
			}
		} else {
			if sl.IsPositive() && seg.start.GreaterThan(sl) && seg.end.LessThanOrEqual(sl) {
				slHit = true
			}
			if tp.IsPositive() && seg.start.GreaterThanOrEqual(tp) && seg.end.LessThanOrEqual(tp) {
				// Defensive: TP in descending segment means TP > start (invalid for buy with TP > entry)
				tpHit = true
			}
		}
		if slHit && tpHit {
			// Both in same segment (defensive, unreachable with valid SL/TP).
			// Nearer to segment start triggers first.
			slDist := seg.start.Sub(sl).Abs()
			tpDist := seg.start.Sub(tp).Abs()
			if slDist.LessThan(tpDist) {
				return true, sl
			}
			return true, tp
		}
		if slHit {
			return true, sl
		}
		if tpHit {
			return true, tp
		}
	}
	return false, decimal.Zero
}

// checkSellSLTPPath checks SL/TP for a sell position using OHLC path.
// Sell SL triggers when price moves up through SL; Sell TP triggers when price moves down through TP.
func checkSellSLTPPath(o, h, l, c, sl, tp decimal.Decimal) (bool, decimal.Decimal) {
	if !sl.IsPositive() && !tp.IsPositive() {
		return false, decimal.Zero
	}
	// Gap-at-open: if Open already at or above SL → SL hit at open.
	if sl.IsPositive() && o.GreaterThanOrEqual(sl) {
		return true, o
	}
	// Gap-at-open: if Open already at or below TP → TP hit at open.
	if tp.IsPositive() && o.LessThanOrEqual(tp) {
		return true, o
	}
	segs := buildPath(o, h, l, c)
	for _, seg := range segs {
		ascending := seg.end.GreaterThan(seg.start)
		// Sell SL: price moves up through SL → triggers in ascending segment when start < SL <= end
		// Sell TP: price moves down through TP → triggers in descending segment when start > TP >= end
		var slHit, tpHit bool
		if ascending {
			if sl.IsPositive() && seg.start.LessThan(sl) && seg.end.GreaterThanOrEqual(sl) {
				slHit = true
			}
			if tp.IsPositive() && seg.start.LessThanOrEqual(tp) && seg.end.GreaterThanOrEqual(tp) {
				// Defensive: TP in ascending segment means TP < start (invalid for sell with TP < entry)
				tpHit = true
			}
		} else {
			if tp.IsPositive() && seg.start.GreaterThan(tp) && seg.end.LessThanOrEqual(tp) {
				tpHit = true
			}
			if sl.IsPositive() && seg.start.GreaterThanOrEqual(sl) && seg.end.LessThanOrEqual(sl) {
				// Defensive: SL in descending segment means SL > start (invalid for sell with SL > entry)
				slHit = true
			}
		}
		if slHit && tpHit {
			// Both in same segment (defensive, unreachable with valid SL/TP).
			// Nearer to segment start triggers first.
			slDist := seg.start.Sub(sl).Abs()
			tpDist := seg.start.Sub(tp).Abs()
			if slDist.LessThan(tpDist) {
				return true, sl
			}
			return true, tp
		}
		if slHit {
			return true, sl
		}
		if tpHit {
			return true, tp
		}
	}
	return false, decimal.Zero
}
