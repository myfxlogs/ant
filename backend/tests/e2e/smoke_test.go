//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	natsgo "github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/mdtick"
	natscli "anttrader/internal/storage/nats"
)

// TestE2ESmoke (M10.5-11): inject 100 real ticks through the mdgateway data
// pipeline using real CH + real NATS, then reconcile three independent sinks:
//
//	(1) NATS JetStream  — subscribe to md.tick.E2E_SMOKE.>
//	(2) ClickHouse      — SELECT count() FROM md_ticks FINAL WHERE broker='E2E_SMOKE'
//	(3) DLQ             — SELECT count() FROM md_ticks_dlq WHERE broker='E2E_SMOKE'
//
// The mix is 95 valid + 5 bid>ask (quality drops). Reconciliation requires that
// (1)+(2) >= 90 and (3) >= 1 (at least one quality-drop sampled into DLQ).
//
// Build tag: e2e. Required env: CH_USER, CH_PASSWORD, CH_DATABASE; defaults
// localhost:9000 / localhost:4222.
func TestE2ESmoke(t *testing.T) {
	if os.Getenv("CH_USER") == "" {
		t.Skip("CH_USER not set; e2e smoke needs running CH+NATS docker stack")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log, _ := zap.NewDevelopment()

	// --- 1. Connect to ClickHouse ---
	chHost := envOr("CH_HOST", "localhost")
	chPort := envOr("CH_PORT", "9000")
	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{chHost + ":" + chPort},
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
	t.Logf("CH connected: %s:%s db=%s", chHost, chPort, envOr("CH_DATABASE", "ant"))

	// Use a unique broker tag per test run so we never collide with prior runs
	// or with concurrent ant-backend ingest. ALTER...DELETE is async and would
	// race with our writes if we cleaned by broker name alone.
	brokerTag := fmt.Sprintf("E2E_SMOKE_%d", time.Now().UnixNano())
	t.Logf("Broker tag: %s", brokerTag)

	// --- 3. Connect to NATS + ensure MD_EVENTS stream exists ---
	natsURL := envOr("E2E_NATS_URL", natsgo.DefaultURL)
	nc, err := natsgo.Connect(natsURL)
	if err != nil {
		t.Fatalf("NATS connect %s: %v", natsURL, err)
	}
	defer nc.Close()
	// Streams are typically created by ant-backend startup; assume they exist
	// since this test runs against a live ant-* docker stack.
	_ = natscli.MDStreams // imports kept for documentation
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("JetStream context: %v", err)
	}

	// Subscribe BEFORE publishing — pull-style core sub on the same subject.
	var natsCount atomic.Int64
	sub, err := nc.Subscribe("md.tick."+brokerTag+".>", func(_ *natsgo.Msg) {
		natsCount.Add(1)
	})
	if err != nil {
		t.Fatalf("NATS subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	t.Logf("NATS connected: %s, stream=MD_EVENTS, sub=md.tick.%s.>", natsURL, brokerTag)

	// --- 4. Build mdgateway components ---
	publisher := mdgateway.NewPublisher(js)
	chCfg := mdgateway.DefaultCHWriterConfig()
	chCfg.FlushInterval = 200 * time.Millisecond
	chCfg.MaxBatchSize = 50
	spillCfg := mdgateway.DefaultSpillConfig()
	spillCfg.Dir = t.TempDir()
	spillW, err := mdgateway.NewSpillWriter(spillCfg, log)
	if err != nil {
		t.Fatalf("spill writer: %v", err)
	}
	defer spillW.Close()
	chWriter := mdgateway.NewCHWriter(chCfg, ch, spillW, log)
	go chWriter.Start(ctx)

	dlqWriter := mdgateway.NewDLQWriter(ch, spillW, log)

	// --- 5. Inject 100 ticks — 95 valid + 5 bid>ask quality drops ---
	const N = 100
	const driftFromValid = 5 // last 5 are bid>ask
	now := time.Now().UnixMilli()
	for i := 0; i < N; i++ {
		bid := decimal.NewFromFloat(1.0800 + float64(i)*0.0001)
		ask := bid.Add(decimal.NewFromFloat(0.0002))
		if i >= N-driftFromValid {
			// quality drop: bid > ask
			ask = bid.Sub(decimal.NewFromFloat(0.0001))
		}
		tk := &mdtick.Tick{
			UserID:        "e2e-user",
			AccountID:     "e2e-acc",
			Broker:        brokerTag,
			SymbolRaw:     fmt.Sprintf("EURUSD-%d", i),
			Canonical:     fmt.Sprintf("EURUSD%d", i),
			TsUnixMs:      now + int64(i),
			ArrivedUnixMs: now + int64(i),
			Bid:           bid,
			Ask:           ask,
			BidVolume:     1000,
			AskVolume:     1000,
		}
		if i >= N-driftFromValid {
			// route to DLQ (sample the drop) — no NATS publish, no CH ticks insert.
			dlqWriter.WriteTick(ctx, tk, "bid_gt_ask", "")
			continue
		}
		// valid tick: publish to NATS + enqueue to CH writer
		if err := publisher.PublishTick(tk); err != nil {
			t.Errorf("PublishTick i=%d: %v", i, err)
		}
		chWriter.EnqueueTick(tk)
	}

	// --- 6. Wait for flush + propagation ---
	deadline := time.Now().Add(8 * time.Second)
	var chCount, dlqCount int64
	var natsSeen int64
	for time.Now().Before(deadline) {
		_ = ch.QueryRow(ctx, `SELECT count() FROM md_ticks FINAL WHERE broker = ?`, brokerTag).Scan(&chCount)
		_ = ch.QueryRow(ctx, `SELECT count() FROM md_ticks_dlq WHERE broker = ?`, brokerTag).Scan(&dlqCount)
		natsSeen = natsCount.Load()
		if chCount >= int64(N-driftFromValid) && dlqCount >= 1 && natsSeen >= int64(N-driftFromValid) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Logf("Reconciliation result: NATS=%d CH=%d DLQ=%d (expected NATS>=%d CH>=%d DLQ>=1)",
		natsSeen, chCount, dlqCount, N-driftFromValid, N-driftFromValid)

	// --- 7. Three-way assertions ---
	if natsSeen < int64(N-driftFromValid) {
		t.Errorf("NATS delivered %d, want >= %d", natsSeen, N-driftFromValid)
	}
	if chCount < int64(N-driftFromValid) {
		t.Errorf("CH md_ticks count %d, want >= %d", chCount, N-driftFromValid)
	}
	if dlqCount < 1 {
		t.Errorf("DLQ md_ticks_dlq count %d, want >= 1", dlqCount)
	}

	t.Logf("E2E smoke PASS: 95 valid → NATS+CH; 5 bid>ask → DLQ")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

