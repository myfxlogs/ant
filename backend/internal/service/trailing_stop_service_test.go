package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrailingStopCalculation(t *testing.T) {
	tests := []struct {
		name         string
		entryPrice   float64
		currentPrice float64
		trailingPips float64
		pipSize      float64
		isBuy        bool
		expectedStop float64
		shouldUpdate bool
	}{
		{
			name:         "buy order - price moved up",
			entryPrice:   1.1000,
			currentPrice: 1.1050,
			trailingPips: 30,
			pipSize:      0.0001,
			isBuy:        true,
			expectedStop: 1.1020,
			shouldUpdate: true,
		},
		{
			name:         "buy order - price moved down",
			entryPrice:   1.1000,
			currentPrice: 1.0980,
			trailingPips: 30,
			pipSize:      0.0001,
			isBuy:        true,
			expectedStop: 0,
			shouldUpdate: false,
		},
		{
			name:         "sell order - price moved down",
			entryPrice:   1.1000,
			currentPrice: 1.0950,
			trailingPips: 30,
			pipSize:      0.0001,
			isBuy:        false,
			expectedStop: 1.0980,
			shouldUpdate: true,
		},
		{
			name:         "sell order - price moved up",
			entryPrice:   1.1000,
			currentPrice: 1.1020,
			trailingPips: 30,
			pipSize:      0.0001,
			isBuy:        false,
			expectedStop: 0,
			shouldUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var newStop float64
			shouldUpdate := false

			trailingDistance := tt.trailingPips * tt.pipSize

			if tt.isBuy {
				newStop = tt.currentPrice - trailingDistance
				initialStop := tt.entryPrice - trailingDistance
				if newStop > initialStop {
					shouldUpdate = true
				}
			} else {
				newStop = tt.currentPrice + trailingDistance
				initialStop := tt.entryPrice + trailingDistance
				if newStop < initialStop {
					shouldUpdate = true
				}
			}

			if tt.shouldUpdate {
				assert.InDelta(t, tt.expectedStop, newStop, 0.0001)
			}
			assert.Equal(t, tt.shouldUpdate, shouldUpdate)
		})
	}
}

func TestPipValueCalculation(t *testing.T) {
	tests := []struct {
		name            string
		symbol          string
		accountCurrency string
		expectedPip     float64
	}{
		{"EURUSD standard", "EURUSD", "USD", 0.0001},
		{"USDJPY standard", "USDJPY", "JPY", 0.01},
		{"XAUUSD standard", "XAUUSD", "USD", 0.01},
		{"GBPJPY standard", "GBPJPY", "JPY", 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipSize := getPipSizeForSymbol(tt.symbol)
			assert.Equal(t, tt.expectedPip, pipSize)
		})
	}
}

func getPipSizeForSymbol(symbol string) float64 {
	pipSizes := map[string]float64{
		"EURUSD": 0.0001,
		"USDJPY": 0.01,
		"XAUUSD": 0.01,
		"GBPJPY": 0.01,
	}
	if size, ok := pipSizes[symbol]; ok {
		return size
	}
	return 0.0001
}

func TestStopLossDistance(t *testing.T) {
	tests := []struct {
		name         string
		entryPrice   float64
		stopLoss     float64
		isBuy        bool
		pipSize      float64
		expectedPips float64
	}{
		{"buy with stop loss below", 1.1000, 1.0950, true, 0.0001, 50},
		{"sell with stop loss above", 1.1000, 1.1050, false, 0.0001, 50},
		{"buy with tight stop", 1.1000, 1.0990, true, 0.0001, 10},
		{"sell with tight stop", 1.1000, 1.1010, false, 0.0001, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var distance float64
			if tt.isBuy {
				distance = tt.entryPrice - tt.stopLoss
			} else {
				distance = tt.stopLoss - tt.entryPrice
			}
			pips := distance / tt.pipSize
			assert.InDelta(t, tt.expectedPips, pips, 0.001)
		})
	}
}

func TestTrailingStopActivation(t *testing.T) {
	tests := []struct {
		name            string
		entryPrice      float64
		currentPrice    float64
		activationPips  float64
		isBuy           bool
		expectActivated bool
	}{
		{
			name:            "buy - activated",
			entryPrice:      1.1000,
			currentPrice:    1.1030,
			activationPips:  20,
			isBuy:           true,
			expectActivated: true,
		},
		{
			name:            "buy - not activated",
			entryPrice:      1.1000,
			currentPrice:    1.1015,
			activationPips:  20,
			isBuy:           true,
			expectActivated: false,
		},
		{
			name:            "sell - activated",
			entryPrice:      1.1000,
			currentPrice:    1.0970,
			activationPips:  20,
			isBuy:           false,
			expectActivated: true,
		},
		{
			name:            "sell - not activated",
			entryPrice:      1.1000,
			currentPrice:    1.0985,
			activationPips:  20,
			isBuy:           false,
			expectActivated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var priceDiff float64
			if tt.isBuy {
				priceDiff = tt.currentPrice - tt.entryPrice
			} else {
				priceDiff = tt.entryPrice - tt.currentPrice
			}
			pipsDiff := priceDiff / 0.0001
			activated := pipsDiff >= tt.activationPips

			assert.Equal(t, tt.expectActivated, activated)
		})
	}
}
