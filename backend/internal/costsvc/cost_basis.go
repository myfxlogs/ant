package costsvc

import (
	"math"
	"sort"
	"time"
)

// CostBasisMethod defines the matching strategy for closing positions.
type CostBasisMethod string

const (
	FIFO CostBasisMethod = "fifo" // first-in-first-out: oldest opens matched first
	LIFO CostBasisMethod = "lifo" // last-in-first-out: newest opens matched first
	HIFO CostBasisMethod = "hifo" // highest-in-first-out: highest cost basis matched first
)

// OpeningPosition represents an opening trade with remaining volume to close.
type OpeningPosition struct {
	Ticket          string
	Timestamp       time.Time
	Volume          float64
	Price           float64
	Side            string // "buy" (long open) or "sell" (short open)
	RemainingVolume float64
}

// ClosedLot records a matched portion of an opening position closed by a closing trade.
type ClosedLot struct {
	OpeningTicket string
	ClosingTicket string
	Volume        float64
	OpenPrice     float64
	ClosePrice    float64
	RealizedPnL   float64
}

// CostBasisTracker manages opening positions and matches closing trades
// against them using the configured cost basis method.
type CostBasisTracker struct {
	method   CostBasisMethod
	openings []*OpeningPosition
}

// NewCostBasisTracker creates a tracker with the given cost basis method.
func NewCostBasisTracker(method CostBasisMethod) *CostBasisTracker {
	return &CostBasisTracker{method: method}
}

// AddOpening registers a new opening position.
func (t *CostBasisTracker) AddOpening(ticket string, ts time.Time, volume, price float64, side string) {
	t.openings = append(t.openings, &OpeningPosition{
		Ticket:          ticket,
		Timestamp:       ts,
		Volume:          volume,
		Price:           price,
		Side:            side,
		RemainingVolume: volume,
	})
}

// Match matches a closing trade against open positions and returns closed lots.
// closeSide is the side of the closing trade: "sell" closes longs, "buy" closes shorts.
func (t *CostBasisTracker) Match(closeTicket string, closeVolume float64, closePrice float64, closeSide string) []ClosedLot {
	candidates := t.eligibleOpenings(closeSide)
	t.sortByMethod(candidates, closeSide)

	remaining := closeVolume
	var lots []ClosedLot

	for _, pos := range candidates {
		if remaining <= 0 {
			break
		}
		if pos.RemainingVolume <= 0 {
			continue
		}
		matched := math.Min(pos.RemainingVolume, remaining)
		pos.RemainingVolume -= matched
		remaining -= matched
		pnl := realizedPnL(pos.Side, pos.Price, closePrice, matched)
		lots = append(lots, ClosedLot{
			OpeningTicket: pos.Ticket,
			ClosingTicket: closeTicket,
			Volume:        matched,
			OpenPrice:     pos.Price,
			ClosePrice:    closePrice,
			RealizedPnL:   pnl,
		})
	}

	return lots
}

// RemainingVolume returns the total remaining volume across all opening positions.
func (t *CostBasisTracker) RemainingVolume() float64 {
	var total float64
	for _, pos := range t.openings {
		total += pos.RemainingVolume
	}
	return total
}

// OpenPositionCount returns the number of positions with remaining volume > 0.
func (t *CostBasisTracker) OpenPositionCount() int {
	count := 0
	for _, pos := range t.openings {
		if pos.RemainingVolume > 0 {
			count++
		}
	}
	return count
}

func (t *CostBasisTracker) eligibleOpenings(closeSide string) []*OpeningPosition {
	// closeSide "sell" closes longs (openSide "buy")
	// closeSide "buy" closes shorts (openSide "sell")
	openSide := "buy"
	if closeSide == "buy" {
		openSide = "sell"
	}
	var result []*OpeningPosition
	for _, pos := range t.openings {
		if pos.Side == openSide && pos.RemainingVolume > 0 {
			result = append(result, pos)
		}
	}
	return result
}

func (t *CostBasisTracker) sortByMethod(positions []*OpeningPosition, closeSide string) {
	switch t.method {
	case FIFO:
		sort.Slice(positions, func(i, j int) bool {
			return positions[i].Timestamp.Before(positions[j].Timestamp)
		})
	case LIFO:
		sort.Slice(positions, func(i, j int) bool {
			return positions[i].Timestamp.After(positions[j].Timestamp)
		})
	case HIFO:
		// For longs (closeSide="sell"): higher open price = higher cost basis → first
		// For shorts (closeSide="buy"): lower open price = higher cost basis → first
		if closeSide == "sell" {
			sort.Slice(positions, func(i, j int) bool {
				return positions[i].Price > positions[j].Price
			})
		} else {
			sort.Slice(positions, func(i, j int) bool {
				return positions[i].Price < positions[j].Price
			})
		}
	}
}

func realizedPnL(openSide string, openPrice, closePrice, volume float64) float64 {
	if openSide == "buy" {
		return (closePrice - openPrice) * volume
	}
	return (openPrice - closePrice) * volume
}
