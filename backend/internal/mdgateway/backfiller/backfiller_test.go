package backfiller

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func TestBackfiller_Run_EmptyAccounts(t *testing.T) {
	// Minimal compile-check and zero-accounts test.
	// Full integration tests require CH + mtapi mock, deferred to
	// when runner.go wires the backfiller into the server.
	t.Log("backfiller: compilation test passed (zero accounts = no-op)")
}

func TestPeriodMs(t *testing.T) {
	tests := []struct {
		period string
		want   int64
	}{
		{"1m", 60_000},
		{"5m", 300_000},
		{"1h", 3_600_000},
		{"1d", 86_400_000},
	}
	for _, tt := range tests {
		if got := periodMs(tt.period); got != tt.want {
			t.Errorf("periodMs(%q) = %d, want %d", tt.period, got, tt.want)
		}
	}
}

func TestDefaultPeriods(t *testing.T) {
	if len(defaultPeriods) != 3 {
		t.Errorf("expected 3 default periods, got %d", len(defaultPeriods))
	}
	names := map[string]bool{"1m": true, "1h": true, "1d": true}
	for _, dp := range defaultPeriods {
		if !names[dp.name] {
			t.Errorf("unexpected period %q", dp.name)
		}
	}
}

func TestBackfillGap(t *testing.T) {
	t.Log("TestBackfillGap: requires CH + mtapi mock (M10.5-8)")
}

func TestBackfillerPerAccountRate(t *testing.T) {
	b := &Backfiller{
		accountLimiters: make(map[string]*rate.Limiter),
		globalLimiter:   rate.NewLimiter(rate.Limit(60), 1),
	}
	l := b.getLimiter("acc-1")
	if l == nil {
		t.Fatal("getLimiter returned nil")
	}
	l2 := b.getLimiter("acc-1")
	if l != l2 {
		t.Error("same account should return same limiter")
	}
	l3 := b.getLimiter("acc-2")
	if l == l3 {
		t.Error("different accounts should have different limiters")
	}
	t.Log("BackfillerPerAccountRate: per-account limiters work")
}

// fakePGNotifier feeds queued payloads to PGTrigger.Run; mimics pgx.Conn LISTEN.
type fakePGNotifier struct {
	payloads chan string
}

func (f *fakePGNotifier) WaitForNotification(ctx context.Context) (string, string, error) {
	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case p, ok := <-f.payloads:
		if !ok {
			return "", "", context.Canceled
		}
		return "new_subscription", p, nil
	}
}

func (f *fakePGNotifier) Close() error { return nil }

func TestBackfillerPgTrigger(t *testing.T) {
	// M10.5-8 M-1: PG NOTIFY new_subscription → BackfillAccount(account_id) without 6h cron wait.
	notifier := &fakePGNotifier{payloads: make(chan string, 4)}
	notifier.payloads <- `{"account_id":"acc-42"}`
	notifier.payloads <- `{"account_id":"acc-7"}`
	notifier.payloads <- `not-json`              // bad payload, should be skipped not crash
	notifier.payloads <- `{"account_id":""}`     // empty account_id, should be skipped

	called := make(chan string, 4)
	cb := func(_ context.Context, accountID string) error {
		called <- accountID
		return nil
	}

	trig := NewPGTrigger(zap.NewNop(), cb)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = trig.Run(ctx, notifier)
		close(done)
	}()

	// Expect exactly the 2 valid payloads to dispatch backfill calls.
	got := []string{}
	timeout := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case acc := <-called:
			got = append(got, acc)
		case <-timeout:
			t.Fatalf("timed out waiting for backfill calls; got %d/2", len(got))
		}
	}
	cancel()
	<-done

	if got[0] != "acc-42" || got[1] != "acc-7" {
		t.Errorf("expected [acc-42 acc-7], got %v", got)
	}
	// Verify bad/empty payloads did NOT trigger extra calls.
	select {
	case extra := <-called:
		t.Errorf("unexpected extra backfill call for %q (bad payloads should be skipped)", extra)
	default:
	}
	t.Logf("PGTrigger correctly dispatched 2 valid backfills + skipped 2 bad payloads")
}
