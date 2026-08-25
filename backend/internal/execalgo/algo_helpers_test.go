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
