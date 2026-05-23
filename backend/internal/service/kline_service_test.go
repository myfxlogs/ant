package service

import (
	"testing"

	mt4pb "anttrader/mt4"
)

func TestKlineService_ParseTimeframeMT4(t *testing.T) {
	svc := &KlineService{}

	tests := []struct {
		name     string
		input    string
		expected int32
	}{
		{"M1", "m1", int32(mt4pb.Timeframe_Timeframe_M1)},
		{"M5", "m5", int32(mt4pb.Timeframe_Timeframe_M5)},
		{"M15", "m15", int32(mt4pb.Timeframe_Timeframe_M15)},
		{"M30", "m30", int32(mt4pb.Timeframe_Timeframe_M30)},
		{"H1", "h1", int32(mt4pb.Timeframe_Timeframe_H1)},
		{"H4", "h4", int32(mt4pb.Timeframe_Timeframe_H4)},
		{"D1", "d1", int32(mt4pb.Timeframe_Timeframe_D1)},
		{"W1", "w1", int32(mt4pb.Timeframe_Timeframe_W1)},
		{"MN", "mn", int32(mt4pb.Timeframe_Timeframe_MN1)},
		{"1m format", "1m", int32(mt4pb.Timeframe_Timeframe_M1)},
		{"5m format", "5m", int32(mt4pb.Timeframe_Timeframe_M5)},
		{"15m format", "15m", int32(mt4pb.Timeframe_Timeframe_M15)},
		{"30m format", "30m", int32(mt4pb.Timeframe_Timeframe_M30)},
		{"1h format", "1h", int32(mt4pb.Timeframe_Timeframe_H1)},
		{"4h format", "4h", int32(mt4pb.Timeframe_Timeframe_H4)},
		{"1d format", "1d", int32(mt4pb.Timeframe_Timeframe_D1)},
		{"1w format", "1w", int32(mt4pb.Timeframe_Timeframe_W1)},
		{"mn1 format", "mn1", int32(mt4pb.Timeframe_Timeframe_MN1)},
		{"uppercase H1", "H1", int32(mt4pb.Timeframe_Timeframe_H1)},
		{"default unknown", "unknown", int32(mt4pb.Timeframe_Timeframe_H1)},
		{"empty", "", int32(mt4pb.Timeframe_Timeframe_H1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.parseTimeframeMT4(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimeframeMT4(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestKlineService_ParseTimeframeMT5(t *testing.T) {
	svc := &KlineService{}

	tests := []struct {
		name     string
		input    string
		expected int32
	}{
		{"M1", "m1", 1},
		{"M5", "m5", 5},
		{"M15", "m15", 15},
		{"M30", "m30", 30},
		{"H1", "h1", 60},
		{"H4", "h4", 240},
		{"D1", "d1", 1440},
		{"W1", "w1", 10080},
		{"MN", "mn", 43200},
		{"1m format", "1m", 1},
		{"5m format", "5m", 5},
		{"15m format", "15m", 15},
		{"30m format", "30m", 30},
		{"1h format", "1h", 60},
		{"4h format", "4h", 240},
		{"1d format", "1d", 1440},
		{"1w format", "1w", 10080},
		{"mn1 format", "mn1", 43200},
		{"uppercase H1", "H1", 60},
		{"default unknown", "unknown", 60},
		{"empty", "", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.parseTimeframeMT5(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimeframeMT5(%s) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestKlineService_ParseHostPort(t *testing.T) {
	svc := &KlineService{}

	tests := []struct {
		name         string
		input        string
		expectedHost string
		expectedPort int32
	}{
		{"with port", "192.168.1.1:443", "192.168.1.1", 443},
		{"with port 444", "example.com:444", "example.com", 444},
		{"without port", "example.com", "example.com", 443},
		{"empty", "", "", 443},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := svc.parseHostPort(tt.input)
			if host != tt.expectedHost {
				t.Errorf("parseHostPort(%s) host = %s, expected %s", tt.input, host, tt.expectedHost)
			}
			if port != tt.expectedPort {
				t.Errorf("parseHostPort(%s) port = %d, expected %d", tt.input, port, tt.expectedPort)
			}
		})
	}
}
