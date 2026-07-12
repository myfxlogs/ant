package systemai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/repository"
)

// ── Circuit breaker (PG-backed, shared across instances) ──

const (
	cbFailThreshold = 3
	cbCooldown      = 30 * time.Second
)

// SetCircuitBreakerDB wires a pgxpool-like executor for persistent circuit
// breaker state. Use pgxpool.NewCBAdapter(pool) from the handlers package.
// Without it, the circuit breaker silently degrades to a no-op.
func (s *Service) SetCircuitBreakerDB(db cbExecutor) {
	s.cbDB = db
}

// cbExecutor is the minimal interface needed for circuit breaker queries.
type cbExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (any, error)
	QueryRow(ctx context.Context, sql string, args ...any) interface{ Scan(dest ...any) error }
}

func (s *Service) recordProviderFailure(ctx context.Context, userID uuid.UUID, providerID string) {
	if s.cbDB == nil {
		return
	}
	s.cbDB.Exec(ctx,
		`INSERT INTO ai_circuit_breaker (user_id, provider_id, consecutive_fails, opened_at)
		 VALUES ($1, $2, 1, CASE WHEN 1 >= $3 THEN NOW() ELSE NULL END)
		 ON CONFLICT (user_id, provider_id) DO UPDATE SET
		   consecutive_fails = ai_circuit_breaker.consecutive_fails + 1,
		   opened_at = CASE WHEN ai_circuit_breaker.consecutive_fails + 1 >= $3 THEN NOW() ELSE ai_circuit_breaker.opened_at END`,
		userID, providerID, cbFailThreshold)
}

func (s *Service) recordProviderSuccess(ctx context.Context, userID uuid.UUID, providerID string) {
	if s.cbDB == nil {
		return
	}
	s.cbDB.Exec(ctx, `DELETE FROM ai_circuit_breaker WHERE user_id=$1 AND provider_id=$2`, userID, providerID)
}

func (s *Service) isCircuitOpen(ctx context.Context, userID uuid.UUID, providerID string) bool {
	if s.cbDB == nil {
		return false
	}
	var fails int
	var openedAt *time.Time
	row := s.cbDB.QueryRow(ctx,
		`SELECT consecutive_fails, opened_at FROM ai_circuit_breaker WHERE user_id=$1 AND provider_id=$2`,
		userID, providerID)
	if err := row.Scan(&fails, &openedAt); err != nil {
		return false // not found = closed
	}
	if fails < cbFailThreshold || openedAt == nil {
		return false
	}
	return time.Since(*openedAt) < cbCooldown
}

// ── Failover ──

// failoverErr wraps an error with a transient flag for provider failover decisions.
type failoverErr struct {
	msg       string
	transient bool
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

// chatEndpoint constructs the chat completion API endpoint from a provider's base URL.
func chatEndpoint(providerID, baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if providerID == "zhipu" {
		return base + "/chat/completions"
	}
	base = strings.TrimSuffix(base, "/v1")
	return base + "/v1/chat/completions"
}

// chatProvider holds resolved provider info for a single candidate.
type chatProvider struct {
	userID     uuid.UUID
	providerID string
	model      string
	baseURL    string
	secret     string
	maxTokens  int // from DB config; 0 = use default
}

// resolveAllChatProviders returns all enabled providers with valid secrets,
// ordered by the user's saved primary preference (via SetAIPrimary → users table).
// The primary provider comes first; then other user-configured providers; finally
// Gateway system providers as a fallback when the user has no configs at all.
func (s *Service) resolveAllChatProviders(ctx context.Context, userID uuid.UUID) ([]chatProvider, error) {
	// ── Determine the user's explicit primary choice ──
	primaryPID, primaryModel := s.getAIPrimaryGateway(ctx, userID)

	// ── Collect user-configured providers ──
	rows, err := s.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list AI providers: %w", err)
	}

	var primary, rest []chatProvider
	seenPID := map[string]bool{}

	for _, row := range rows {
		if row == nil || !row.Enabled {
			continue
		}
		sec, secErr := s.getCachedSecret(ctx, userID, row.ProviderID)
		if secErr != nil || sec == "" {
			continue
		}
		base := strings.TrimRight(strings.TrimSpace(row.BaseURL), "/")
		if base == "" {
			continue
		}
		m := resolveModel(row.DefaultModel, row.Models, row.ProviderID, primaryPID, primaryModel)
		if m == "" {
			continue
		}
		// ADR-0025 §5.2: model whitelist filter — skip providers whose model is not allowed.
		if s.modelFilter != nil && !s.modelFilter(ctx, userID, m) {
			continue
		}
		if s.isCircuitOpen(ctx, userID, row.ProviderID) {
			continue
		}
		cp := chatProvider{
			userID: userID, providerID: row.ProviderID,
			model: m, baseURL: base, secret: sec,
			maxTokens: row.MaxTokens,
		}
		seenPID[row.ProviderID] = true
		if row.ProviderID == primaryPID {
			primary = append(primary, cp)
		} else {
			rest = append(rest, cp)
		}
	}
	out := append(primary, rest...)

	// ── Gateway system providers (fallback when user has no own configs) ──
	if len(out) == 0 && s.gatewayProviderRepo != nil {
		sysProviders, sysErr := s.gatewayProviderRepo.ListEnabled(ctx)
		if sysErr == nil {
			for _, sp := range sysProviders {
				if seenPID[sp.ProviderID] {
					continue // user already has a config for this provider
				}
				if len(sp.APIKeyEncrypted) == 0 || len(sp.Models) == 0 {
					continue
				}
				pt, openErr := repository.OpenAPIKey(sp.APIKeyEncrypted, s.box)
				if openErr != nil || pt == "" {
					continue
				}
				base := strings.TrimRight(strings.TrimSpace(sp.BaseURL), "/")
				if base == "" {
					continue
				}
				m := resolveModel(sp.DefaultModel, sp.Models, sp.ProviderID, primaryPID, primaryModel)
				if s.modelFilter != nil && !s.modelFilter(ctx, userID, m) {
					continue
				}
				cp := chatProvider{
					userID: userID, providerID: sp.ProviderID,
					model: m, baseURL: base, secret: pt,
					// gateway providers have no per-config maxTokens; use default
				}
				if sp.ProviderID == primaryPID {
					out = append([]chatProvider{cp}, out...)
				} else {
					out = append(out, cp)
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("errors.ai.not_configured")
	}
	return out, nil
}

// getAIPrimaryGateway reads the user's saved Gateway model preference.
// Returns ("", "") when not set (user hasn't picked a Gateway model yet).
func (s *Service) getAIPrimaryGateway(ctx context.Context, userID uuid.UUID) (providerID, model string) {
	id, m, err := s.GetAIPrimary(ctx, userID)
	if err != nil || id == "" {
		return "", ""
	}
	return id, m
}

// resolveModel picks the best model for a provider given the user's preferences.
// Priority: primaryModel (explicit user pick) > defaultModel > models[0].
func resolveModel(defaultModel string, models []string, providerID, primaryPID, primaryModel string) string {
	m := strings.TrimSpace(defaultModel)
	if m == "" && len(models) > 0 {
		m = strings.TrimSpace(models[0])
	}
	if providerID == primaryPID && primaryModel != "" {
		m = primaryModel
	}
	return m
}
