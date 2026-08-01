package interceptor

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

// RateLimitInterceptor applies token-bucket rate limiting to selected RPC paths.
// Currently protects /login and /register against brute-force by IP.
// In-memory only — sufficient for single-process monolith deployment.
// To scale horizontally, swap to a Redis-based counter (INCR + EXPIRE).
type RateLimitInterceptor struct {
	limiters      sync.Map // string IP -> *rateLimiterEntry
	rateLimit     rate.Limit
	burst         int
	enabled       bool
	protectedPaths map[string]bool
}

// NewRateLimitInterceptor creates a rate limiter for login/register endpoints.
// requestsPerMinute: max requests per minute per IP (default 10).
// enabled: if false, all requests pass through without limiting.
func NewRateLimitInterceptor(requestsPerMinute int, enabled bool) *RateLimitInterceptor {
	i := &RateLimitInterceptor{
		rateLimit: rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:     3,
		enabled:   enabled,
		protectedPaths: map[string]bool{
			"/ant.v1.AuthService/Login":            true,
			"/ant.v1.AuthService/Register":         true,
			"/ant.v1.AuthService/ForgotPassword":   true,
			"/ant.v1.AuthService/VerifyMTIdentity": true,
			"/ant.v1.AIService/GenerateStrategy":   true,
			"/ant.v1.StrategyPlanService/Plan":     true,
			"/ant.v1.CodeAssistService/Analyze":    true,
			"/ant.v1.AIGatewayService/Complete":    true,
		},
	}
	go i.cleanupLoop()
	return i
}

func (i *RateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if !i.enabled || !i.protectedPaths[req.Spec().Procedure] {
			return next(ctx, req)
		}

		ip := extractClientIPFromHeader(req.Header())
		if ip == "" {
			ip = "unknown"
		}

		limiter := i.getLimiter(ip)
		if !limiter.Allow() {
			return nil, connect.NewError(
				connect.CodeResourceExhausted,
				fmt.Errorf("rate limit exceeded: max %d requests per minute", int(i.rateLimit*60)),
			)
		}
		return next(ctx, req)
	}
}

func (i *RateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *RateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (i *RateLimitInterceptor) getLimiter(ip string) *rate.Limiter {
	entry, _ := i.limiters.LoadOrStore(ip, &rateLimiterEntry{
		limiter:  rate.NewLimiter(i.rateLimit, i.burst),
		lastUsed: time.Now(),
	})
	e := entry.(*rateLimiterEntry)
	e.lastUsed = time.Now()
	return e.limiter
}

func (i *RateLimitInterceptor) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		i.cleanup()
	}
}

func (i *RateLimitInterceptor) cleanup() {
	now := time.Now()
	i.limiters.Range(func(key, value interface{}) bool {
		e := value.(*rateLimiterEntry)
		if now.Sub(e.lastUsed) > 10*time.Minute {
			i.limiters.Delete(key)
		}
		return true
	})
}

// extractClientIPFromHeader extracts the client IP from HTTP headers.
// X-Real-IP is preferred over X-Forwarded-For because nginx sets it to
// $remote_addr (the actual TCP peer IP), overwriting any client-supplied
// value. X-Forwarded-For can contain client-supplied spoofed IPs since
// $proxy_add_x_forwarded_for appends rather than replaces.
func extractClientIPFromHeader(header http.Header) string {
	if xri := header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if xff := header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}
	return ""
}
