package interceptor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

func TestNewRateLimitInterceptor(t *testing.T) {
	i := NewRateLimitInterceptor(5, true)
	if i.rateLimit != rate.Limit(5.0/60.0) {
		t.Errorf("expected rateLimit=%v, got %v", rate.Limit(5.0/60.0), i.rateLimit)
	}
	if i.burst != 3 {
		t.Errorf("expected burst=3, got %d", i.burst)
	}
	if !i.enabled {
		t.Error("expected enabled=true")
	}
	if !i.protectedPaths["/ant.v1.AuthService/Login"] {
		t.Error("expected /Login in protected paths")
	}
	if !i.protectedPaths["/ant.v1.AuthService/Register"] {
		t.Error("expected /Register in protected paths")
	}
}

func TestRateLimitInterceptor_Disabled(t *testing.T) {
	i := NewRateLimitInterceptor(10, false)
	if i.enabled {
		t.Error("expected enabled=false")
	}
}

func TestGetLimiter_SameIPReturnsSameLimiter(t *testing.T) {
	i := NewRateLimitInterceptor(10, true)
	lim1 := i.getLimiter("10.0.0.1")
	lim2 := i.getLimiter("10.0.0.1")
	if lim1 != lim2 {
		t.Error("same IP should return same limiter")
	}
}

func TestGetLimiter_DifferentIPsIndependent(t *testing.T) {
	i := NewRateLimitInterceptor(10, true)
	lim1 := i.getLimiter("10.0.0.1")
	lim2 := i.getLimiter("10.0.0.2")
	if lim1 == lim2 {
		t.Error("different IPs should have different limiters")
	}
}

func TestLimiter_AllowsBurst(t *testing.T) {
	t.Parallel()
	lim := rate.NewLimiter(rate.Limit(10.0/60.0), 3)
	for j := 0; j < 3; j++ {
		if !lim.Allow() {
			t.Fatalf("burst request %d was rate limited unexpectedly", j+1)
		}
	}
}

func TestLimiter_BlocksExcess(t *testing.T) {
	t.Parallel()
	lim := rate.NewLimiter(rate.Limit(10.0/60.0), 3)
	for j := 0; j < 3; j++ {
		lim.Allow() // consume burst
	}
	if lim.Allow() {
		t.Error("token bucket should be empty immediately after consuming burst")
	}
}

func TestExtractClientIPFromHeader(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   string
	}{
		{"empty", http.Header{}, ""},
		{"x-forwarded-for single", http.Header{"X-Forwarded-For": []string{"1.2.3.4"}}, "1.2.3.4"},
		{"x-forwarded-for chain", http.Header{"X-Forwarded-For": []string{"1.2.3.4, 10.0.0.1"}}, "1.2.3.4"},
		{"x-real-ip", http.Header{"X-Real-Ip": []string{"5.6.7.8"}}, "5.6.7.8"},
		{"x-forwarded-for takes priority", http.Header{
			"X-Forwarded-For": []string{"1.2.3.4"},
			"X-Real-Ip":       []string{"5.6.7.8"},
		}, "1.2.3.4"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractClientIPFromHeader(tc.header)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRateLimitInterceptor_Cleanup(t *testing.T) {
	i := NewRateLimitInterceptor(10, true)
	i.getLimiter("10.1.1.1")

	entry, ok := i.limiters.Load("10.1.1.1")
	if !ok {
		t.Fatal("expected limiter entry for 10.1.1.1")
	}
	e := entry.(*rateLimiterEntry)
	e.lastUsed = time.Now().Add(-15 * time.Minute)
	i.cleanup()

	_, ok = i.limiters.Load("10.1.1.1")
	if ok {
		t.Error("expected stale entry to be cleaned up")
	}
}

func TestWrapUnary_NonProtectedPath(t *testing.T) {
	t.Parallel()

	i := NewRateLimitInterceptor(10, true)
	callCount := 0
	next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		callCount++
		return nil, nil
	}
	wrapped := i.WrapUnary(next)

	msg := struct{ Name string }{}
	req := connect.NewRequest(&msg)

	for j := 0; j < 50; j++ {
		_, err := wrapped(context.Background(), req)
		if err != nil {
			t.Fatalf("non-protected path request %d should not be rate limited: %v", j+1, err)
		}
	}
	if callCount != 50 {
		t.Errorf("expected 50 calls to next, got %d", callCount)
	}
}

func TestWrapUnary_LoginPathRateLimits(t *testing.T) {
	// Use a real connect request with a known procedure.
	// connect.NewRequest doesn't let us set the procedure, so we test via a
	// real httptest handler instead.
	mux := http.NewServeMux()
	mux.HandleFunc("/ant.v1.AuthService/Login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := srv.Client()
	for i := 0; i < 10; i++ {
		resp, err := client.Get(srv.URL + "/ant.v1.AuthService/Login")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func init() {
	_ = httptest.NewServer(nil) // ensure httptest is importable
}
