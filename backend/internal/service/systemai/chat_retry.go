package systemai

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ── Failover ──

// failoverErr wraps an error with a transient flag for provider failover decisions.
type failoverErr struct {
	msg       string
	transient bool
	// retryAfter is the vendor-advised wait (Retry-After header), 0 when absent.
	retryAfter time.Duration
}

func (e *failoverErr) Error() string { return e.msg }

func isFailoverErr(err error) bool {
	if fe, ok := err.(*failoverErr); ok {
		return fe.transient
	}
	return false
}

// isFailoverStatus returns true for HTTP status codes that indicate a
// provider-level issue (try next) rather than a request-level issue (stop).
//
// For 400 errors (which can be either request-level or provider-level),
// the caller should additionally inspect the response body — model-not-found
// errors are safe to failover because another provider may host the model.
func isFailoverStatus(code int) bool {
	switch code {
	case 401:
		return false // bad API key — don't retry
	case 400:
		return true // may be model-specific → try next; caller checks body for auth errors
	case 403, 429:
		return true // quota/rate-limit/region-block → try next
	case 502, 503, 504:
		return true // provider infrastructure down → try next
	default:
		return code >= 500 // other server errors → try next
	}
}

// isAuthErrorBody checks whether an error body indicates an auth problem.
// When true, failover is pointless — the key itself is invalid.
func isAuthErrorBody(body string) bool {
	low := strings.ToLower(body)
	return strings.Contains(low, "invalid api key") ||
		strings.Contains(low, "invalid key") ||
		strings.Contains(low, "invalid authentication") ||
		strings.Contains(low, "authorization header") ||
		strings.Contains(low, "incorrect api key") ||
		strings.Contains(low, "api key not valid")
}

// apiError holds a parsed OpenAI-compatible error response.
type apiError struct {
	Type    string // e.g. "invalid_request_error", "authentication_error"
	Message string // human-readable description
	Raw     string // original text if JSON parse fails
}

// readAPIErrorBody reads up to 8 KiB of a non-2xx response body and parses it.
func readAPIErrorBody(resp *http.Response) apiError {
	if resp == nil || resp.Body == nil {
		return apiError{}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	return parseAPIError(body)
}

// extractJSONField extracts a string value for the given key from a JSON object.
// Manual string parsing — avoids encoding/json (CLAUDE.md §0 prohibition).
func extractJSONField(raw []byte, key string) string {
	s := string(raw)
	// Look for "key":"value"
	search := `"` + key + `":"`
	idx := strings.Index(s, search)
	if idx < 0 {
		search2 := `"` + key + `": "`
		idx = strings.Index(s, search2)
		if idx < 0 {
			return ""
		}
		start := idx + len(search2)
		end := strings.Index(s[start:], `"`)
		if end < 0 {
			return ""
		}
		return s[start : start+end]
	}
	start := idx + len(search)
	end := strings.Index(s[start:], `"`)
	if end < 0 {
		return ""
	}
	return s[start : start+end]
}

// readAPIErrorBodyFromBytes parses an already-read response body.
func readAPIErrorBodyFromBytes(body []byte) apiError {
	return parseAPIError(body)
}

// parseAPIError extracts error information from provider API responses.
// Uses manual string parsing to avoid encoding/json (CLAUDE.md §0 prohibition).
func parseAPIError(body []byte) apiError {
	msg := extractJSONField(body, "message")
	if msg != "" {
		return apiError{Type: extractJSONField(body, "type"), Message: msg}
	}
	raw := strings.TrimSpace(string(body))
	if len(raw) > 500 {
		raw = raw[:500]
	}
	return apiError{Raw: raw}
}

// String returns a compact representation for error messages.
func (e apiError) String() string {
	if e.Type != "" {
		return e.Type + ": " + e.Message
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Raw
}

// normalizeAPIBase tolerates users pasting a full endpoint URL instead of the
// API root (e.g. NOVA docs give https://host/v1/chat/completions): strip a
// trailing "/chat/completions" or "/models" so endpoint construction never
// doubles the suffix.
func normalizeAPIBase(base string) string {
	for _, suffix := range []string{"/chat/completions", "/models"} {
		if strings.HasSuffix(base, suffix) {
			return strings.TrimSuffix(base, suffix)
		}
	}
	return base
}

// transientRetryBackoff is the wait before each transient retry (indexed by
// retry number). Var so tests can shorten it.
var transientRetryBackoff = []time.Duration{2 * time.Second, 6 * time.Second}

const maxRetryAfter = 15 * time.Second

// retryWait computes the sleep before a transient retry: the vendor's
// Retry-After when present (capped), otherwise the staged backoff table.
func retryWait(retry int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxRetryAfter {
			return maxRetryAfter
		}
		return retryAfter
	}
	if retry-1 >= 0 && retry-1 < len(transientRetryBackoff) {
		return transientRetryBackoff[retry-1]
	}
	return transientRetryBackoff[len(transientRetryBackoff)-1]
}

// parseRetryAfter reads a numeric Retry-After header (seconds), 0 when absent.
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}
