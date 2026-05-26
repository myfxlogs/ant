package mdgateway

import (
	"context"
	"testing"
	"time"
)

func TestUserMetricsCollector_Record(t *testing.T) {
	c := NewUserMetricsCollector()

	c.Record("user-1", "signal_count", 10)
	c.Record("user-1", "signal_count", 20)
	c.Record("user-2", "order_count", 5)

	samples := c.Flush()
	if len(samples) != 2 {
		t.Fatalf("want 2 sample groups, got %d", len(samples))
	}

	// Verify aggregated values.
	for _, s := range samples {
		switch {
		case s.UserID == "user-1" && s.MetricName == "signal_count":
			if s.MetricValue != 30 || s.Count != 2 {
				t.Fatalf("user-1 signal_count: want value=30 count=2, got value=%.0f count=%d", s.MetricValue, s.Count)
			}
		case s.UserID == "user-2" && s.MetricName == "order_count":
			if s.MetricValue != 5 || s.Count != 1 {
				t.Fatalf("user-2 order_count: want value=5 count=1, got value=%.0f count=%d", s.MetricValue, s.Count)
			}
		default:
			t.Fatalf("unexpected sample: user=%s metric=%s", s.UserID, s.MetricName)
		}
	}
}

func TestUserMetricsCollector_FlushResets(t *testing.T) {
	c := NewUserMetricsCollector()
	c.Record("user-1", "ticks", 100)

	samples := c.Flush()
	if len(samples) != 1 {
		t.Fatal("first flush should have 1 sample")
	}

	// Second flush should be empty.
	samples = c.Flush()
	if len(samples) != 0 {
		t.Fatalf("second flush should be empty, got %d", len(samples))
	}
}

func TestUserMetricsCollector_EmptyFlush(t *testing.T) {
	c := NewUserMetricsCollector()
	samples := c.Flush()
	if len(samples) > 0 {
		t.Fatal("empty collector should return nil")
	}
}

func TestUserMetricsFlusher_Lifecycle(t *testing.T) {
	flushed := make(chan []UserMetricSample, 1)
	writeFn := func(_ context.Context, samples []UserMetricSample) error {
		flushed <- samples
		return nil
	}

	f := NewUserMetricsFlusher(50*time.Millisecond, writeFn)
	f.Collector().Record("user-a", "signals", 42)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f.Start(ctx)

	// Wait for flush.
	select {
	case samples := <-flushed:
		if len(samples) != 1 {
			t.Fatalf("want 1 sample, got %d", len(samples))
		}
		if samples[0].UserID != "user-a" || samples[0].MetricName != "signals" {
			t.Fatalf("unexpected sample: %+v", samples[0])
		}
	case <-time.After(time.Second):
		t.Fatal("flush did not happen within timeout")
	}

	if f.FlushedTotal() < 1 {
		t.Fatal("FlushedTotal should be >= 1")
	}

	f.Stop()
}

func TestUserMetricsFlusher_NoWriteFn(t *testing.T) {
	f := NewUserMetricsFlusher(50*time.Millisecond, nil)
	f.Collector().Record("user-x", "metric", 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.Start(ctx)

	// Should not panic — writeFn is nil.
	time.Sleep(100 * time.Millisecond)
	if f.FlushedTotal() < 1 {
		t.Fatal("flushes should still count even when writeFn is nil")
	}
	f.Stop()
}

func TestUserMetricsFlusher_DefaultInterval(t *testing.T) {
	f := NewUserMetricsFlusher(0, nil)
	if f.interval != 5*time.Minute {
		t.Fatalf("default interval: want 5m, got %v", f.interval)
	}
}

func TestUserMetricsFlusher_ConcurrentRecord(t *testing.T) {
	c := NewUserMetricsCollector()
	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func(id int) {
			uid := "user-" + string(rune('A'+id%10))
			c.Record(uid, "ticks", 1)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}

	samples := c.Flush()
	if len(samples) == 0 {
		t.Fatal("should have samples after concurrent records")
	}
	// Verify no duplicates by checking count sums.
	var totalCount uint64
	for _, s := range samples {
		totalCount += s.Count
	}
	if totalCount != 100 {
		t.Fatalf("total count: want 100, got %d", totalCount)
	}
}
