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

// TestE2ESmoke (M10.5-11): inject ticks through the mdgateway data pipeline
// using real CH + real NATS, then reconcile three independent sinks:
//
//	(1) NATS  — subscribe to md.tick.<brokerTag>.>
//	(2) CH    — SELECT count() FROM md_ticks FINAL WHERE broker=<brokerTag>
//	(3) DLQ   — SELECT count() FROM md_ticks_dlq WHERE broker=<brokerTag>
//
// Mix: 95 valid (NATS+CH) + 5 parse_error (DLQ 100%) + 500 bid_gt_ask (DLQ 1%).
// Reconciliation requires NATS>=95, CH>=95, DLQ total>=5, DLQ bid_gt_ask>=1.
//
// Build tag: e2e. Required env: CH_USER, CH_PASSWORD, CH_DATABASE.
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

	brokerTag := fmt.Sprintf("E2E_SMOKE_%d", time.Now().UnixNano())
	t.Logf("Broker tag: %s", brokerTag)

	// --- 2. Connect to NATS ---
	natsURL := envOr("E2E_NATS_URL", natsgo.DefaultURL)
	nc, err := natsgo.Connect(natsURL)
	if err != nil {
		t.Fatalf("NATS connect %s: %v", natsURL, err)
	}
	defer nc.Close()
	_ = natscli.MDStreams
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("JetStream context: %v", err)
	}

	var natsCount atomic.Int64
	sub, err := nc.Subscribe("md.tick."+brokerTag+".>", func(_ *natsgo.Msg) {
		natsCount.Add(1)
	})
	if err != nil {
		t.Fatalf("NATS subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	t.Logf("NATS connected: %s, sub=md.tick.%s.>", natsURL, brokerTag)

	// --- 3. Build mdgateway components ---
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

	// --- 4. Inject ticks ---
	// 95 valid (NATS+CH) + 5 parse_error (DLQ 100% sampled) + 500 bid_gt_ask (DLQ 1% sampled)
	const validTicks = 95
	const parseErrorTicks = 5
	const bidGtAskTicks = 500

	now := time.Now().UnixMilli()
	tickIdx := 0

	// Phase A: valid ticks → NATS + CH
	for i := 0; i < validTicks; i++ {
		bid := decimal.NewFromFloat(1.0800 + float64(i)*0.0001)
		ask := bid.Add(decimal.NewFromFloat(0.0002))
		tk := &mdtick.Tick{
			UserID: "e2e-user", AccountID: "e2e-acc", Broker: brokerTag,
			SymbolRaw: fmt.Sprintf("EURUSD-%d", tickIdx),
			Canonical: fmt.Sprintf("EURUSD%d", tickIdx),
			TsUnixMs: now + int64(tickIdx), ArrivedUnixMs: now + int64(tickIdx),
			Bid: bid, Ask: ask, BidVolume: 1000, AskVolume: 1000,
		}
		tickIdx++
		if err := publisher.PublishTick(tk); err != nil {
			t.Errorf("PublishTick valid i=%d: %v", i, err)
		}
		chWriter.EnqueueTick(tk)
	}

	// Phase B: parse_error ticks → DLQ only (100% sampled, guaranteed)
	for i := 0; i < parseErrorTicks; i++ {
		tk := &mdtick.Tick{
			UserID: "e2e-user", AccountID: "e2e-acc", Broker: brokerTag,
			SymbolRaw: fmt.Sprintf("EURUSD-%d", tickIdx),
			Canonical: fmt.Sprintf("EURUSD%d", tickIdx),
			TsUnixMs: now + int64(tickIdx), ArrivedUnixMs: now + int64(tickIdx),
			Bid: decimal.NewFromFloat(1.0800), Ask: decimal.NewFromFloat(1.0802),
			BidVolume: 1000, AskVolume: 1000,
		}
		tickIdx++
		dlqWriter.WriteTick(ctx, tk, "parse_error", `{"phase":"parse_error"}`)
	}

	// Phase C: bid_gt_ask ticks → DLQ only (1% sampled)
	for i := 0; i < bidGtAskTicks; i++ {
		bid := decimal.NewFromFloat(1.0800 + float64(i)*0.0001)
		ask := bid.Sub(decimal.NewFromFloat(0.0001)) // bid > ask
		tk := &mdtick.Tick{
			UserID: "e2e-user", AccountID: "e2e-acc", Broker: brokerTag,
			SymbolRaw: fmt.Sprintf("EURUSD-%d", tickIdx),
			Canonical: fmt.Sprintf("EURUSD%d", tickIdx),
			TsUnixMs: now + int64(tickIdx), ArrivedUnixMs: now + int64(tickIdx),
			Bid: bid, Ask: ask, BidVolume: 1000, AskVolume: 1000,
		}
		tickIdx++
		dlqWriter.WriteTick(ctx, tk, "bid_gt_ask", "")
	}

	t.Logf("Injected: %d valid + %d parse_error + %d bid_gt_ask = %d total",
		validTicks, parseErrorTicks, bidGtAskTicks, tickIdx)

	// --- 5. Wait for flush + propagation ---
	deadline := time.Now().Add(15 * time.Second)
	var chCount, dlqTotal, dlqBidGtAsk uint64
	var natsSeen int64
	for time.Now().Before(deadline) {
		if err := ch.QueryRow(ctx,
			`SELECT count() FROM md_ticks FINAL WHERE broker = ?`, brokerTag,
		).Scan(&chCount); err != nil {
			t.Logf("CH scan md_ticks: %v", err)
		}
		if err := ch.QueryRow(ctx,
			`SELECT count() FROM md_ticks_dlq WHERE broker = ?`, brokerTag,
		).Scan(&dlqTotal); err != nil {
			t.Logf("CH scan md_ticks_dlq: %v", err)
		}
		if err := ch.QueryRow(ctx,
			`SELECT count() FROM md_ticks_dlq WHERE broker = ? AND reason = 'bid_gt_ask'`, brokerTag,
		).Scan(&dlqBidGtAsk); err != nil {
			t.Logf("CH scan md_ticks_dlq(reason=bid_gt_ask): %v", err)
		}
		natsSeen = natsCount.Load()
		if chCount >= validTicks && dlqTotal >= uint64(parseErrorTicks) && dlqBidGtAsk >= 1 && natsSeen >= int64(validTicks) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Logf("Reconciliation: NATS=%d CH=%d DLQ(total)=%d DLQ(bid_gt_ask)=%d (need NATS>=%d CH>=%d DLQ>=%d bid_gt_ask>=1)",
		natsSeen, chCount, dlqTotal, dlqBidGtAsk, validTicks, validTicks, parseErrorTicks)

	// --- 6. Three-way assertions ---
	failures := 0
	if natsSeen < int64(validTicks) {
		t.Errorf("NATS delivered %d, want >= %d", natsSeen, validTicks)
		failures++
	}
	if chCount < uint64(validTicks) {
		t.Errorf("CH md_ticks count %d, want >= %d", chCount, validTicks)
		failures++
	}
	if dlqTotal < uint64(parseErrorTicks) {
		t.Errorf("DLQ total count %d, want >= %d", dlqTotal, parseErrorTicks)
		failures++
	}
	if dlqBidGtAsk < 1 {
		t.Errorf("DLQ bid_gt_ask count %d, want >= 1 (1%% sampling on %d entries)", dlqBidGtAsk, bidGtAskTicks)
		failures++
	}
	if failures == 0 {
		t.Logf("E2E smoke PASS: 3-way reconciliation (NATS=%d CH=%d DLQ=%d/%d bid_gt_ask)",
			natsSeen, chCount, dlqTotal, dlqBidGtAsk)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
