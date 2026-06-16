package systemai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

// ── Circuit breaker ──

const (
	cbFailThreshold = 3           // consecutive failures before opening
	cbCooldown      = 30 * time.Second // skip provider for this long after threshold
)

var cbState sync.Map // key: "userID|providerID" → *circuitBreaker

type circuitBreaker struct {
	consecutiveFails int
	openedAt         time.Time
}

func (cb *circuitBreaker) isOpen() bool {
	if cb.consecutiveFails < cbFailThreshold {
		return false
	}
	return time.Since(cb.openedAt) < cbCooldown
}

func recordProviderSuccess(userID uuid.UUID, providerID string) {
	key := userID.String() + "|" + providerID
	cbState.Delete(key)
}

func recordProviderFailure(userID uuid.UUID, providerID string) {
	key := userID.String() + "|" + providerID
	v, _ := cbState.LoadOrStore(key, &circuitBreaker{})
	cb := v.(*circuitBreaker)
	cb.consecutiveFails++
	if cb.consecutiveFails >= cbFailThreshold {
		cb.openedAt = time.Now()
	}
	cbState.Store(key, cb)
}

func isCircuitOpen(userID uuid.UUID, providerID string) bool {
	v, ok := cbState.Load(userID.String() + "|" + providerID)
	if !ok {
		return false
	}
	return v.(*circuitBreaker).isOpen()
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
func isFailoverStatus(code int) bool {
	switch code {
	case 401:
		return false // bad API key — don't retry
	case 403, 429:
		return true // quota/rate-limit/region-block → try next
	case 502, 503, 504:
		return true // provider infrastructure down → try next
	default:
		return code >= 500 // other server errors → try next
	}
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
}

// resolveAllChatProviders returns all enabled providers with valid secrets,
// ordered by preference (primary first, then others). Used for multi-provider
// failover: if the first provider fails with a transient error, the caller
// can try the next one without re-resolving.
func (s *Service) resolveAllChatProviders(ctx context.Context, userID uuid.UUID, modelHint string) ([]chatProvider, error) {
	rows, err := s.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list AI providers: %w", err)
	}
	var out []chatProvider
	for _, preferPrimary := range []bool{true, false} {
		for _, row := range rows {
			if row == nil || !row.Enabled {
				continue
			}
			if preferPrimary && !hasPrimaryChat(row.PrimaryFor) {
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
			m := strings.TrimSpace(row.DefaultModel)
			if m == "" {
				m = modelHint
			}
			if m == "" && len(row.Models) > 0 {
				m = strings.TrimSpace(row.Models[0])
			}
			if m == "" {
				continue
			}
			if isCircuitOpen(userID, row.ProviderID) {
				continue // skip provider in cooldown
			}
			out = append(out, chatProvider{
				userID:     userID,
				providerID: row.ProviderID,
				model:      m,
				baseURL:    base,
				secret:     sec,
			})
		}
	}
	if len(out) == 0 && s.gatewayProviderRepo != nil {
		// Fallback: use system providers from AI Gateway.
		sysProviders, sysErr := s.gatewayProviderRepo.ListEnabled(ctx)
		if sysErr == nil {
			for _, sp := range sysProviders {
				if len(sp.APIKeyEncrypted) == 0 {
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
				m := modelHint
				if m == "" && len(sp.Models) > 0 {
					m = strings.TrimSpace(sp.Models[0])
				}
				if m == "" {
					continue
				}
				out = append(out, chatProvider{
					userID: userID, providerID: sp.ProviderID, model: m,
					baseURL: base, secret: pt,
				})
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("errors.ai.not_configured")
	}
	return out, nil
}

// resolveChatProvider picks a provider, model, base URL, and secret for the given user.
