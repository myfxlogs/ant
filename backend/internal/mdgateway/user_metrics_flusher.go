// Package mdgateway provides per-user 5-minute metric aggregation (M10-BASE-A4).
//
// UserMetricsFlusher runs a background goroutine that periodically flushes
// per-user aggregated metrics to the ClickHouse user_metrics_5m SummingMergeTree.
// Key constraint: Prometheus metrics do NOT carry user_id label (cardinality).
// CH stores user_id for offline analysis; Prometheus only carries aggregate counters.
package mdgateway

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// UserMetricSample is a single aggregated metric value for a user.
type UserMetricSample struct {
	UserID      string
	MetricName  string
	MetricValue float64
	Count       uint64
}

// UserMetricsCollector accumulates per-user metrics between flush intervals.
type UserMetricsCollector struct {
	mu       sync.Mutex
	buckets  map[string]*userMetricsBucket // key: "user_id:metric_name"
}

type userMetricsBucket struct {
	sum   float64
	count uint64
}

// NewUserMetricsCollector creates a new collector.
func NewUserMetricsCollector() *UserMetricsCollector {
	return &UserMetricsCollector{
		buckets: make(map[string]*userMetricsBucket),
	}
}

// Record adds a metric observation for a user.
func (c *UserMetricsCollector) Record(userID, metricName string, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := userID + ":" + metricName
	b, ok := c.buckets[key]
	if !ok {
		b = &userMetricsBucket{}
		c.buckets[key] = b
	}
	b.sum += value
	b.count++
}

// Flush drains all accumulated metrics and returns them, resetting the collector.
func (c *UserMetricsCollector) Flush() []UserMetricSample {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.buckets) == 0 {
		return nil
	}
	samples := make([]UserMetricSample, 0, len(c.buckets))
	for key, b := range c.buckets {
		// key format: "user_id:metric_name"
		sep := len(key) - 1
		for sep >= 0 && key[sep] != ':' {
			sep--
		}
		if sep <= 0 || sep >= len(key)-1 {
			continue
		}
		uid := key[:sep]
		name := key[sep+1:]
		samples = append(samples, UserMetricSample{
			UserID:      uid,
			MetricName:  name,
			MetricValue: b.sum,
			Count:       b.count,
		})
	}
	c.buckets = make(map[string]*userMetricsBucket)
	return samples
}

// UserMetricsFlusher runs a background goroutine that periodically flushes
// per-user metrics to CH via the provided write callback.
type UserMetricsFlusher struct {
	collector *UserMetricsCollector
	interval  time.Duration
	writeFn   func(ctx context.Context, samples []UserMetricSample) error

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc

	flushedTotal atomic.Int64
	flushErrors  atomic.Int64
}

// NewUserMetricsFlusher creates a flusher with the given interval and CH write callback.
func NewUserMetricsFlusher(interval time.Duration, writeFn func(context.Context, []UserMetricSample) error) *UserMetricsFlusher {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &UserMetricsFlusher{
		collector: NewUserMetricsCollector(),
		interval:  interval,
		writeFn:   writeFn,
	}
}

// Collector returns the underlying collector for recording metrics.
func (f *UserMetricsFlusher) Collector() *UserMetricsCollector {
	return f.collector
}

// Start begins the background flush loop.
func (f *UserMetricsFlusher) Start(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.running {
		return
	}
	f.running = true
	ctx, f.cancel = context.WithCancel(ctx)
	go f.loop(ctx)
}

// Stop halts the background flush loop.
func (f *UserMetricsFlusher) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.running {
		return
	}
	f.running = false
	if f.cancel != nil {
		f.cancel()
	}
}

// FlushedTotal returns the count of successful flushes.
func (f *UserMetricsFlusher) FlushedTotal() int64 { return f.flushedTotal.Load() }

// FlushErrors returns the count of failed flushes.
func (f *UserMetricsFlusher) FlushErrors() int64 { return f.flushErrors.Load() }

func (f *UserMetricsFlusher) loop(ctx context.Context) {
	ticker := Clk.NewTicker(f.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush before exit.
			f.doFlush(context.Background())
			return
		case <-ticker.C():
			f.doFlush(ctx)
		}
	}
}

func (f *UserMetricsFlusher) doFlush(ctx context.Context) {
	samples := f.collector.Flush()
	if len(samples) == 0 {
		return
	}
	if f.writeFn != nil {
		if err := f.writeFn(ctx, samples); err != nil {
			f.flushErrors.Add(1)
			return
		}
	}
	f.flushedTotal.Add(1)
}
