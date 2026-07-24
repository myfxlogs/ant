// Package mdgateway metrics — HTTP handler, spill tracking, and DLQ sampling.
package mdgateway

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --- Spill tracking ---

// SpillPendingFiles is the count of unreplayed spill JSONL files.
// Updated every 30s by spill_replay goroutine.
var spillPendingFiles atomic.Int64

// UpdateSpillPendingFiles scans the spill directory and updates the gauge.
func UpdateSpillPendingFiles(spillDir string) {
	if spillDir == "" {
		return
	}
	files, _ := filepath.Glob(spillDir + "/*.jsonl")
	spillPendingFiles.Store(int64(len(files)))
}

// SpillPendingFilesCount returns the current spill backlog count.
func SpillPendingFilesCount() int64 {
	return spillPendingFiles.Load()
}

// --- DLQ sampling ---

// DLQSampled tracks DLQ entries written, by reason.
var dlqSampled = map[string]*atomic.Int64{
	"parse_error":  {},
	"bid_gt_ask":   {},
	"non_positive": {},
	"spill_failed": {},
}

// DLQSampled returns the DLQ sample count for a reason.
func DLQSampled(reason string) int64 {
	if c, ok := dlqSampled[reason]; ok {
		return c.Load()
	}
	return 0
}

// --- Prometheus /metrics HTTP handler ---

type captureWriter struct {
	headers http.Header
	buf     *bytes.Buffer
}

func (w *captureWriter) Header() http.Header         { return w.headers }
func (w *captureWriter) Write(b []byte) (int, error) { return w.buf.Write(b) }
func (w *captureWriter) WriteHeader(statusCode int)   {}

// writeLegacyMetrics writes custom mdgateway metrics in Prometheus exposition format.
func writeLegacyMetrics(b *strings.Builder) {
	// md_e2e_latency_seconds histogram
	b.WriteString("# TYPE md_e2e_latency_seconds histogram\n")
	var total int64
	for i := range e2eLatency.counts {
		total += e2eLatency.counts[i].Load()
	}
	if total > 0 {
		var cum int64
		for i, bucket := range e2eLatency.buckets {
			cum += e2eLatency.counts[i].Load()
			fmt.Fprintf(b, "md_e2e_latency_seconds_bucket{le=\"%g\"} %d\n", bucket, cum)
		}
		fmt.Fprintf(b, "md_e2e_latency_seconds_bucket{le=\"+Inf\"} %d\n", total)
		fmt.Fprintf(b, "md_e2e_latency_seconds_sum %g\n", float64(e2eLatency.sum.Load())/1e9)
		fmt.Fprintf(b, "md_e2e_latency_seconds_count %d\n", total)
	} else {
		fmt.Fprintf(b, "md_e2e_latency_seconds_bucket{le=\"0.01\"} 0\n")
		fmt.Fprintf(b, "md_e2e_latency_seconds_bucket{le=\"+Inf\"} 0\n")
		b.WriteString("md_e2e_latency_seconds_sum 0\n")
		b.WriteString("md_e2e_latency_seconds_count 0\n")
	}

	// md_spill_pending_files gauge
	b.WriteString("# TYPE md_spill_pending_files gauge\n")
	fmt.Fprintf(b, "md_spill_pending_files %d\n", spillPendingFiles.Load())

	// md_dlq_sampled_total counter
	b.WriteString("# TYPE md_dlq_sampled_total counter\n")
	for reason, c := range dlqSampled {
		fmt.Fprintf(b, "md_dlq_sampled_total{reason=\"%s\"} %d\n", reason, c.Load())
	}

	// Additional M10 quality metrics
	b.WriteString("# TYPE md_gap_avg_seconds gauge\n")
	fmt.Fprintf(b, "md_gap_avg_seconds %g\n", GapAvgSeconds())
	b.WriteString("# TYPE md_gap_max_seconds gauge\n")
	fmt.Fprintf(b, "md_gap_max_seconds %g\n", GapMaxSeconds())
	b.WriteString("# TYPE md_gap_exceeded_total counter\n")
	fmt.Fprintf(b, "md_gap_exceeded_total %d\n", GapExceeded())
	b.WriteString("# TYPE md_clock_skew_max_seconds gauge\n")
	fmt.Fprintf(b, "md_clock_skew_max_seconds %g\n", ClockSkewMaxSeconds())
	b.WriteString("# TYPE md_clock_skew_exceeded_total counter\n")
	fmt.Fprintf(b, "md_clock_skew_exceeded_total %d\n", ClockSkewExceeded())
	b.WriteString("# TYPE md_bar_skipped_finalized_total counter\n")
	fmt.Fprintf(b, "md_bar_skipped_finalized_total %d\n", BarSkippedFinalized())
	b.WriteString("# TYPE md_stale_accounts gauge\n")
	fmt.Fprintf(b, "md_stale_accounts %d\n", StaleAccountCount())
	b.WriteString("# TYPE md_dead_accounts gauge\n")
	fmt.Fprintf(b, "md_dead_accounts %d\n", DeadAccountCount())

	// M10-BASE-B6: backpressure metrics
	b.WriteString("# TYPE md_chan_full_total counter\n")
	fmt.Fprintf(b, "md_chan_full_total %d\n", ChanFullTotal())
	b.WriteString("# TYPE md_nats_publish_dropped_total counter\n")
	fmt.Fprintf(b, "md_nats_publish_dropped_total %d\n", NATSPublishDroppedTotal())
	b.WriteString("# TYPE md_consumer_lag gauge\n")
	fmt.Fprintf(b, "md_consumer_lag %d\n", ConsumerLag())
	b.WriteString("# TYPE signal_dropped_total counter\n")
	fmt.Fprintf(b, "signal_dropped_total %d\n", SignalDroppedTotal())

	// M10-BASE-F3/F4/F5: clock skew drop + stuffing + spread anomaly
	b.WriteString("# TYPE md_clock_skew_dropped_total counter\n")
	fmt.Fprintf(b, "md_clock_skew_dropped_total %d\n", ClockSkewDroppedTotal())
	b.WriteString("# TYPE md_stuffing_detected_total counter\n")
	fmt.Fprintf(b, "md_stuffing_detected_total %d\n", StuffingDetectedTotal())
	b.WriteString("# TYPE md_spread_anomaly_total counter\n")
	fmt.Fprintf(b, "md_spread_anomaly_total %d\n", SpreadAnomalyTotal())
}

// MetricsHandler returns an http.Handler that serves mdgateway custom metrics
// plus Go runtime and process collector metrics from promhttp.
func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		var b strings.Builder
		writeLegacyMetrics(&b)

		// Append promhttp metrics (go_gc_duration_seconds, process_cpu_seconds_total, etc.)
		// Strip Accept-Encoding to prevent promhttp from gzip-encoding its output,
		// which would corrupt the combined plain-text metrics response.
		r2 := r.Clone(r.Context())
		r2.Header.Set("Accept-Encoding", "identity")
		var buf bytes.Buffer
		cw := &captureWriter{headers: make(http.Header), buf: &buf}
		promhttp.Handler().ServeHTTP(cw, r2)
		b.Write(buf.Bytes())

		_, _ = w.Write([]byte(b.String()))
	})
}
