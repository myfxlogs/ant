package admin

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalToFloat(t *testing.T) {
	tests := []struct {
		name  string
		input decimal.Decimal
		want  float64
	}{
		{"zero", decimal.Zero, 0.0},
		{"positive", decimal.NewFromFloat(3.14), 3.14},
		{"negative", decimal.NewFromFloat(-1.5), -1.5},
		{"large", decimal.NewFromInt(1000000), 1000000.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimalToFloat(tt.input)
			if got != tt.want {
				t.Errorf("decimalToFloat(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
