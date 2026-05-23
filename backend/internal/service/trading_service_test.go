package service

import (
	"testing"
)

func TestParseOrderTypeMT4(t *testing.T) {
	tests := []struct {
		input    string
		expected int32
		hasError bool
	}{
		{"buy", 0, false},
		{"sell", 1, false},
		{"buy_limit", 2, false},
		{"sell_limit", 3, false},
		{"buy_stop", 4, false},
		{"sell_stop", 5, false},
		{"BUY", 0, false},
		{"SELL", 1, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := ParseOrderTypeMT4(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseOrderTypeMT4(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseOrderTypeMT4(%s) unexpected error: %v", tt.input, err)
			}
			if int32(result) != tt.expected {
				t.Errorf("ParseOrderTypeMT4(%s) = %d, expected %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestParseOrderTypeMT5(t *testing.T) {
	tests := []struct {
		input    string
		hasError bool
	}{
		{"buy", false},
		{"sell", false},
		{"buy_limit", false},
		{"sell_limit", false},
		{"buy_stop", false},
		{"sell_stop", false},
		{"buy_stop_limit", false},
		{"sell_stop_limit", false},
		{"BUY", false},
		{"SELL", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		_, err := ParseOrderTypeMT5(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseOrderTypeMT5(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseOrderTypeMT5(%s) unexpected error: %v", tt.input, err)
			}
		}
	}
}

func TestOrderTypeToString(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "buy"},
		{1, "sell"},
		{2, "buy_limit"},
		{3, "sell_limit"},
		{4, "buy_stop"},
		{5, "sell_stop"},
		{6, "buy_stop_limit"},
		{7, "sell_stop_limit"},
		{99, "unknown"},
	}

	for _, tt := range tests {
		result := OrderTypeToString(tt.input)
		if result != tt.expected {
			t.Errorf("OrderTypeToString(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
