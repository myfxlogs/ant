package secrets_test

import (
	"os"
	"sync/atomic"
	"testing"

	"anttrader/internal/secrets"
)

// deterministicRand is a concurrency-safe io.Reader that never fails, producing
// a reproducible byte stream. It eliminates crypto/rand.Reader entropy
// exhaustion flakiness in CI environments.
type deterministicRand struct{ counter atomic.Uint64 }

func (r *deterministicRand) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(r.counter.Add(1))
	}
	return len(p), nil
}

// TestMain sets a deterministic random source so entropy-dependent tests
// never fail with "unexpected EOF" in CI environments.
func TestMain(m *testing.M) {
	secrets.SetRandReader(&deterministicRand{})
	os.Exit(m.Run())
}
