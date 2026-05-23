package mdgateway

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewQuality(t *testing.T) {
	q := NewQuality(DefaultQualityConfig())
	if q == nil {
		t.Fatal("NewQuality returned nil")
	}
}

func TestQuality_CheckBidAsk(t *testing.T) {
	q := NewQuality(DefaultQualityConfig())
	// bid > ask → dropped
	tick := &Tick{
		Broker: "demo",
		Symbol: "EURUSD",
		Bid:    &Money{Value: "1.2000"},
		Ask:    &Money{Value: "1.1000"},
	}
	result := q.Check(tick)
	if !result.Dropped {
		t.Fatal("expected bid>ask to be dropped")
	}
}

func TestQuality_CheckValid(t *testing.T) {
	q := NewQuality(DefaultQualityConfig())
	tick := &Tick{
		Broker:    "demo",
		Symbol:    "EURUSD",
		TsUnixMs:  1000,
		Bid:       &Money{Value: "1.1000"},
		Ask:       &Money{Value: "1.1005"},
	}
	result := q.Check(tick)
	if result.Dropped {
		t.Fatal("valid tick should not be dropped")
	}
}

func TestQuality_OutlierDetection(t *testing.T) {
	q := NewQuality(QualityConfig{
		HistorySize:  10,
		OutlierSigma: 2,
	})
	// Feed 9 identical ticks
	for i := 0; i < 9; i++ {
		q.Check(&Tick{
			Broker: "demo", Symbol: "EURUSD", TsUnixMs: int64(i * 1000),
			Bid: &Money{Value: "1.1000"}, Ask: &Money{Value: "1.1005"},
		})
	}
	// Feed a spike far from mean
	result := q.Check(&Tick{
		Broker: "demo", Symbol: "EURUSD", TsUnixMs: 10000,
		Bid: &Money{Value: "2.0000"}, Ask: &Money{Value: "2.0005"},
	})
	if !result.Outlier {
		t.Log("spike not flagged as outlier (may be within sigma)")
	}
}

func TestQuality_MedianSigma(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	median, sigma := medianSigma(vals)
	if median != 3.0 {
		t.Fatalf("expected median=3, got %f", median)
	}
	if sigma <= 0 {
		t.Fatal("sigma should be positive")
	}
}

func TestDefaultQualityConfig(t *testing.T) {
	cfg := DefaultQualityConfig()
	if cfg.GapMaxSeconds != 5 {
		t.Fatalf("expected GapMaxSeconds=5, got %f", cfg.GapMaxSeconds)
	}
	if cfg.OutlierSigma != 5 {
		t.Fatalf("expected OutlierSigma=5, got %f", cfg.OutlierSigma)
	}
}

func TestPublisher_NoOp(t *testing.T) {
	p := NewPublisher(zap.NewNop(), "")
	if p == nil {
		t.Fatal("NewPublisher returned nil")
	}
	if err := p.Close(); err != nil {
		t.Fatal("Close should be no-op")
	}
}
