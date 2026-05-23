package mdgateway

import (
	"testing"
	"time"
)

func TestNewAggregator(t *testing.T) {
	a := NewAggregator()
	if a == nil {
		t.Fatal("NewAggregator returned nil")
	}
	if a.Size() != 0 {
		t.Fatal("expected 0 bars initially")
	}
}

func TestAggregator_AddTick(t *testing.T) {
	a := NewAggregator()
	now := time.Now().UnixMilli()

	var completed []Bar
	a.AddTick(&Tick{
		UserID:        "u1",
		Broker:        "demo",
		Symbol:        "EURUSD",
		Canonical:     "EURUSD",
		ArrivedUnixMs: now,
		Bid:           &Money{Value: "1.1000"},
		Ask:           &Money{Value: "1.1005"},
		BidVolume:     1.0,
		AskVolume:     1.0,
	}, func(b Bar) {
		completed = append(completed, b)
	})

	if a.Size() != 6 { // 6 periods
		t.Fatalf("expected 6 open bars, got %d", a.Size())
	}
}

func TestAggregator_FlushAll(t *testing.T) {
	a := NewAggregator()
	now := time.Now().UnixMilli()

	var completed []Bar
	a.AddTick(&Tick{
		UserID:        "u1",
		Broker:        "demo",
		Symbol:        "EURUSD",
		Canonical:     "EURUSD",
		ArrivedUnixMs: now,
		Bid:           &Money{Value: "1.1000"},
		Ask:           &Money{Value: "1.1005"},
		BidVolume:     1.0,
		AskVolume:     1.0,
	}, func(b Bar) {
		completed = append(completed, b)
	})

	a.FlushAll(func(b Bar) {
		completed = append(completed, b)
	})

	if len(completed) != 6 {
		t.Fatalf("expected 6 completed bars after flush, got %d", len(completed))
	}
	if a.Size() != 0 {
		t.Fatal("expected 0 bars after FlushAll")
	}
}

func TestAggregator_OHLC(t *testing.T) {
	a := NewAggregator()
	now := time.Now().UnixMilli()

	var completed []Bar
	// Feed multiple ticks at different prices
	ticks := []float64{1.1000, 1.1010, 1.0990, 1.1005}
	for _, price := range ticks {
		a.AddTick(&Tick{
			UserID:        "u1",
			Broker:        "demo",
			Symbol:        "EURUSD",
			Canonical:     "EURUSD",
			ArrivedUnixMs: now + int64(len(completed)),
			Bid:           &Money{Value: formatFloat(price)},
			Ask:           &Money{Value: formatFloat(price + 0.0005)},
			BidVolume:     1.0,
			AskVolume:     1.0,
		}, func(b Bar) {
			completed = append(completed, b)
		})
	}

	a.FlushAll(func(b Bar) {
		completed = append(completed, b)
	})

	// Find the 1m bar
	for _, b := range completed {
		if b.Period == "1m" {
			if b.Open != 1.1000 {
				t.Errorf("expected Open=1.1000, got %f", b.Open)
			}
			if b.High != 1.1010 {
				t.Errorf("expected High=1.1010, got %f", b.High)
			}
			if b.Low != 1.0990 {
				t.Errorf("expected Low=1.0990, got %f", b.Low)
			}
			if b.Close != 1.1005 {
				t.Errorf("expected Close=1.1005, got %f", b.Close)
			}
			if b.TickCount != 4 {
				t.Errorf("expected TickCount=4, got %d", b.TickCount)
			}
			return
		}
	}
	t.Fatal("1m bar not found in completed bars")
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input  string
		val    float64
		ok     bool
	}{
		{"1.2345", 1.2345, true},
		{"100", 100.0, true},
		{"0.001", 0.001, true},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, tt := range tests {
		v, ok := parseFloat(tt.input)
		if ok != tt.ok {
			t.Errorf("parseFloat(%q) ok=%v, want %v", tt.input, ok, tt.ok)
		}
		if ok && v != tt.val {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.input, v, tt.val)
		}
	}
}

func formatFloat(f float64) string {
	return fastFloatToString(f)
}

func fastFloatToString(f float64) string {
	// Simple formatting for test
	s := int64(f * 1000000)
	intPart := s / 1000000
	fracPart := s % 1000000
	if fracPart < 0 {
		fracPart = -fracPart
	}
	return formatInt(intPart) + "." + padInt(fracPart, 6)
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func padInt(n int64, width int) string {
	s := formatInt(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
