//go:build loadtest
// +build loadtest

package loadtest

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

// Test100AccountsNoSpill (M10.5-12): 100 mock brokers × 250 tick/s × 5min.
// Asserts md_spill_writes_total == 0 and P99 e2e latency < 500ms (ADR-0011 §6).
//
// Build tag: loadtest. Required env: CH_USER, CH_PASSWORD, CH_DATABASE.
func Test100AccountsNoSpill(t *testing.T) {
	if os.Getenv("CH_USER") == "" {
		t.Skip("CH_USER not set; loadtest needs running CH docker stack")
	}

	// Duration can be shortened via env for smoke runs.
	dur := 5 * time.Minute
	if s := os.Getenv("LOADTEST_DURATION"); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			dur = d
			t.Logf("LOADTEST_DURATION=%s (override)", s)
		}
	}
	t.Logf("Load test duration: %s", dur)

	ctx, cancel := context.WithTimeout(context.Background(), dur+30*time.Second)
	defer cancel()

	log := zap.NewNop()

	// --- 1. Connect to ClickHouse ---
	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{envOr("CH_HOST", "localhost") + ":" + envOr("CH_PORT", "9000")},
		Auth: clickhouse.Auth{
			Username: os.Getenv("CH_USER"),
			Password: os.Getenv("CH_PASSWORD"),
			Database: envOr("CH_DATABASE", "ant"),
		},
	})
	if err != nil {
		t.Fatalf("CH connect: %v", err)
	}
	defer ch.Close()
	if err := ch.Ping(ctx); err != nil {
		t.Fatalf("CH ping: %v", err)
	}
	t.Logf("CH connected: %s:%s", envOr("CH_HOST", "localhost"), envOr("CH_PORT", "9000"))

	// --- 2. Build CHWriter with tuned config for 25k tick/s peak ---
	spillCfg := mdgateway.DefaultSpillConfig()
	spillCfg.Dir = t.TempDir()
	spillW, err := mdgateway.NewSpillWriter(spillCfg, log)
	if err != nil {
		t.Fatalf("spill writer: %v", err)
	}
	defer spillW.Close()

	chCfg := mdgateway.DefaultCHWriterConfig()
	chCfg.FlushInterval = 200 * time.Millisecond
	chCfg.MaxBatchSize = 5000
	chCfg.QueueSize = 1000000
	chWriter := mdgateway.NewCHWriter(chCfg, ch, spillW, log)
	go chWriter.Start(ctx)

	// Publisher with nil JetStream — load test focuses on CHWriter throughput.
	publisher := mdgateway.NewPublisher(nil)

	// --- 3. Start 100 mock brokers ---
	const numBrokers = 100
	const ticksPerSec = 250

	var wg sync.WaitGroup
	var enqueueErrors atomicErrCounter

	brokerTag := fmt.Sprintf("LOAD_%d", time.Now().UnixNano()%1000000)
	t.Logf("Broker tag prefix: %s", brokerTag)

	genStart := time.Now()
	for b := 0; b < numBrokers; b++ {
		wg.Add(1)
		go func(brokerID int) {
			defer wg.Done()
			brokerName := fmt.Sprintf("%s_BRK%d", brokerTag, brokerID)
			accountID := fmt.Sprintf("load-acc-%d", brokerID)

			ticker := time.NewTicker(time.Second / ticksPerSec)
			defer ticker.Stop()

			deadline := time.Now().Add(dur)

			var i int
			for time.Now().Before(deadline) {
				select {
				case <-ticker.C:
					now := time.Now().UnixMilli()
					bid := decimal.NewFromFloat(1.0800 + float64(i%1000)*0.0001)
					ask := bid.Add(decimal.NewFromFloat(0.0002))
					tk := &mdtick.Tick{
						UserID:        "load-user",
						AccountID:     accountID,
						Broker:        brokerName,
						SymbolRaw:     fmt.Sprintf("EURUSD-%d", i),
						Canonical:     fmt.Sprintf("EURUSD%d", i%100),
						TsUnixMs:      now,
						ArrivedUnixMs: now,
						Bid:           bid,
						Ask:           ask,
						BidVolume:     1000,
						AskVolume:     1000,
					}
					i++
					_ = publisher.PublishTick(tk)
					chWriter.EnqueueTick(tk)
				case <-ctx.Done():
					return
				}
			}
		}(b)
		// Stagger goroutine creation to avoid initial burst overwhelming the channel.
		time.Sleep(8 * time.Millisecond)
	}

	t.Logf("100 mock brokers started (staggered), generating %d tick/s each (total ~%d tick/s)",
		ticksPerSec, numBrokers*ticksPerSec)

	wg.Wait()
	genEnd := time.Now()
	genDuration := genEnd.Sub(genStart)
	t.Logf("Generation complete in %s", genDuration.Round(time.Second))

	// Let the CHWriter drain remaining ticks.
	drainDeadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(drainDeadline) {
		time.Sleep(500 * time.Millisecond)
		files, _ := filepathGlob(spillCfg.Dir + "/*.jsonl")
		obs := mdgateway.E2eLatencyCount()
		if len(files) == 0 && obs > 0 {
			break
		}
	}

	// --- 4. Assertions ---
	// Spill = 0: no spill files created.
	spillFiles, _ := filepathGlob(spillCfg.Dir + "/*.jsonl")
	spillCount := len(spillFiles)
	if spillCount > 0 {
		for _, f := range spillFiles {
			fi, _ := os.Stat(f)
			t.Logf("Spill file: %s size=%d", f, fi.Size())
		}
		t.Errorf("Spill: %d files found in %s, want 0", spillCount, spillCfg.Dir)
	} else {
		t.Logf("Spill check: 0 files (PASS)")
	}

	// P99 latency < 500ms.
	p99 := mdgateway.E2eLatencyP99()
	obsCount := mdgateway.E2eLatencyCount()
	t.Logf("E2E latency: P99=%.3fs  observations=%d", p99, obsCount)
	if p99 > 0.5 {
		t.Errorf("E2E latency P99 = %.3fs, want < 0.5s", p99)
	} else if obsCount == 0 {
		t.Error("E2E latency: 0 observations recorded (CHWriter may not have flushed)")
	} else {
		t.Logf("E2E latency: P99=%.3fs < 0.5s (PASS)", p99)
	}

	if enqueueErrors.Load() > 0 {
		t.Errorf("Enqueue errors: %d", enqueueErrors.Load())
	}

	if spillCount == 0 && p99 <= 0.5 && p99 > 0 && enqueueErrors.Load() == 0 {
		t.Logf("Load test PASS: 100 brokers × %d tick/s × %s, spill=0, P99=%.3fs",
			ticksPerSec, genDuration.Round(time.Second), p99)
	}
}

// --- helpers ---

type atomicErrCounter struct {
	mu    sync.Mutex
	count int64
}

func (c *atomicErrCounter) Add(n int64) {
	c.mu.Lock()
	c.count += n
	c.mu.Unlock()
}

func (c *atomicErrCounter) Load() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// filepathGlob delegates to filepath.Glob.
func filepathGlob(pattern string) ([]string, error) {
	// Avoid import cycle; use os.ReadDir + manual filter.
	dir := pattern[:len(pattern)-len("/*.jsonl")]
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 6 && e.Name()[len(e.Name())-6:] == ".jsonl" {
			matches = append(matches, dir+"/"+e.Name())
		}
	}
	return matches, nil
}
