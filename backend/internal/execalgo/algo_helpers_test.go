package execalgo

import (
	"time"

	"github.com/shopspring/decimal"
)

func refTime() time.Time { return time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC) }

func closeEnoughAlgo(a, b decimal.Decimal) bool {
	return a.Sub(b).Abs().LessThan(decimal.NewFromFloat(0.001))
}

func decFromFloat(f float64) decimal.Decimal { return decimal.NewFromFloat(f) }

// ---- TWAP ----

type customProfile struct {
	fractions map[int]float64 // hour → fraction
}

func (p customProfile) Fraction(_ string, bucketStart time.Time) float64 {
	if f, ok := p.fractions[bucketStart.Hour()]; ok {
		return f
	}
	return 0.1
}
