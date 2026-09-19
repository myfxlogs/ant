package interceptor

// Pin for sseOwnerKey (G-POST2-1 S4): the key must come from request
// headers — X-Real-IP preferred, first XFF entry as fallback — because auth
// context values are never populated at HTTP middleware layer. The old
// context-based lookup collapsed every client onto "anon".

import (
	"net/http"
	"testing"
)

func TestSSEOwnerKey_XRealIPPreferredOverXFF(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://x/stream", nil)
	r.Header.Set("X-Real-IP", "203.0.113.7")
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.1")
	if got := sseOwnerKey(r); got != "203.0.113.7" {
		t.Fatalf("X-Real-IP must win: expected 203.0.113.7, got %q", got)
	}
}

func TestSSEOwnerKey_XFFFallbackFirstEntry(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://x/stream", nil)
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.1")
	if got := sseOwnerKey(r); got != "198.51.100.9" {
		t.Fatalf("expected first XFF entry, got %q", got)
	}
}

func TestSSEOwnerKey_NoHeadersFallsBackToAnon(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://x/stream", nil)
	if got := sseOwnerKey(r); got != "anon" {
		t.Fatalf("expected anon, got %q", got)
	}
}
