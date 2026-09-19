package paper

// POST-2 capacity probe (S3): paper order path latency/throughput.
//
// Measures the PaperEngine in-process order path — fill-price calculation,
// nil-guard skip, repo writes (in-memory stub), balance update, broadcast —
// as a latency distribution under concurrency C ∈ {1,8,32} plus a sequential
// throughput benchmark.
//
// 口径 (honest scope): this is the PAPER path baseline only. The real broker
// path (mtapi gRPC → real broker RTT) is out of scope — recorded in the
// staging list of docs/benchmarks/post2-capacity-baseline-2026-09.md.
// Probe runs DB-free (in-memory repo stub, thread-safe for the concurrent
// levels — engine_test.go's stubPaperRepo is single-goroutine only).

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/repository"
)

// probeRepo is a thread-safe in-memory paperRepository stub for concurrent
// probes. Orders are counted, not retained (no unbounded growth under
// benchmark iteration counts).
type probeRepo struct {
	mu         sync.Mutex
	accounts   map[string]*repository.PaperAccount
	orderCount int64
}

func newProbeRepo() *probeRepo {
	return &probeRepo{accounts: make(map[string]*repository.PaperAccount)}
}

func (r *probeRepo) seed(id string, balance float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[id] = &repository.PaperAccount{
		ID:             id,
		CurrentBalance: decimal.NewFromFloat(balance),
		Equity:         decimal.NewFromFloat(balance),
	}
}

func (r *probeRepo) CreateOrder(_ context.Context, _ *repository.PaperOrder) error {
	atomic.AddInt64(&r.orderCount, 1)
	return nil
}

func (r *probeRepo) GetAccount(_ context.Context, id string) (*repository.PaperAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[id]
	if !ok {
		return nil, nil
	}
	return a, nil
}

func (r *probeRepo) ListAccounts(_ context.Context, _ string) ([]*repository.PaperAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*repository.PaperAccount, 0, len(r.accounts))
	for _, a := range r.accounts {
		out = append(out, a)
	}
	return out, nil
}

func (r *probeRepo) UpdateAccountBalance(_ context.Context, accountID string, balance, equity decimal.Decimal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.accounts[accountID]; ok {
		a.CurrentBalance = balance
		a.Equity = equity
	}
	return nil
}

func (r *probeRepo) GetOrder(_ context.Context, _ string) (*repository.PaperOrder, error) {
	return nil, nil
}

func (r *probeRepo) UpdateOrder(_ context.Context, _ *repository.PaperOrder) error {
	return nil
}

func (r *probeRepo) FindOpenOrder(_ context.Context, _, _ string) (*repository.PaperOrder, error) {
	return nil, nil
}

// TestPlacePaperOrderLatency_Concurrency measures the order-path latency
// distribution at C concurrent workers (own account per worker), 3200 orders
// per level. Throughput and percentiles land in
// docs/benchmarks/post2-capacity-baseline-2026-09.md.
func TestPlacePaperOrderLatency_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("POST-2 probe: skipped in -short")
	}
	ctx := context.Background()
	const totalOrders = 3200
	vol := decimal.NewFromFloat(0.1)
	bid := decimal.NewFromFloat(1.1000)
	ask := decimal.NewFromFloat(1.1005)

	for _, c := range []int{1, 8, 32} {
		repo := newProbeRepo()
		engine := New(repo, nil, testLogger())
		per := totalOrders / c
		for w := 0; w < c; w++ {
			repo.seed("pa-probe-"+strconv.Itoa(c)+"-"+strconv.Itoa(w), 1e9)
		}

		var (
			wg    sync.WaitGroup
			latMu sync.Mutex
			durs  []time.Duration
		)
		start := time.Now()
		for w := 0; w < c; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				acct := "pa-probe-" + strconv.Itoa(c) + "-" + strconv.Itoa(w)
				local := make([]time.Duration, 0, per)
				for i := 0; i < per; i++ {
					t0 := time.Now()
					if err := engine.PlacePaperOrder(ctx, acct, "EURUSD", "buy", vol, bid, ask); err != nil {
						t.Errorf("C=%d worker %d order %d: %v", c, w, i, err)
						return
					}
					local = append(local, time.Since(t0))
				}
				latMu.Lock()
				durs = append(durs, local...)
				latMu.Unlock()
			}(w)
		}
		wg.Wait()
		elapsed := time.Since(start)

		s := append([]time.Duration(nil), durs...)
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		pick := func(p float64) time.Duration { return s[int(float64(len(s)-1)*p)] }
		var sum time.Duration
		for _, d := range s {
			sum += d
		}
		t.Logf("POST-2 paper order C=%d: n=%d ops/s=%.0f mean=%s p50=%s p95=%s p99=%s max=%s",
			c, len(s), float64(len(s))/elapsed.Seconds(),
			sum/time.Duration(len(s)), pick(0.50), pick(0.95), pick(0.99), pick(1.0))
	}
}

// BenchmarkPlacePaperOrder is the sequential paper order-path throughput
// baseline (engine + in-memory repo + empty-fanout broadcast; 0 subscribers).
func BenchmarkPlacePaperOrder(b *testing.B) {
	repo := newProbeRepo()
	repo.seed("pa-bench", 1e9)
	engine := New(repo, nil, testLogger())
	ctx := context.Background()
	vol := decimal.NewFromFloat(0.1)
	bid := decimal.NewFromFloat(1.1000)
	ask := decimal.NewFromFloat(1.1005)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := engine.PlacePaperOrder(ctx, "pa-bench", "EURUSD", "buy", vol, bid, ask); err != nil {
			b.Fatal(err)
		}
	}
}
