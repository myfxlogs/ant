package systemai

import (
	"context"
	"fmt"
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
	_, _ = s.cbDB.Exec(ctx,
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
	_, _ = s.cbDB.Exec(ctx, `DELETE FROM ai_circuit_breaker WHERE user_id=$1 AND provider_id=$2`, userID, providerID)
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

// chatEndpoint constructs the chat completion API endpoint from a provider's base URL.
func chatEndpoint(providerID, baseURL string) string {
	base := normalizeAPIBase(strings.TrimRight(baseURL, "/"))
	if providerID == "zhipu" {
		return base + "/chat/completions"
	}
	base = strings.TrimSuffix(base, "/v1")
	return base + "/v1/chat/completions"
}

// chatProvider holds resolved provider info for a single candidate.
type chatProvider struct {
	userID          uuid.UUID
	providerID      string
	model           string
	baseURL         string
	secret          string
	maxTokens       int     // from DB config; 0 = use default
	temperature     float64 // from DB config; <=0 resolved to defaultTemperature
	timeoutSeconds  int     // from DB config; 0 = platform default (150s / 120s first-byte)
	reasoningEffort string  // from DB config; "" = omit (vendor default)
	organization    string  // from DB config; "" = no OpenAI-Organization header
	gateway         bool    // true = platform-paid Gateway candidate (quota/wallet gating + system billing apply)
}

// systemPaidCall reports whether every candidate is platform-paid. Candidates
// are never mixed (gateway providers are only resolved when the user has no
// own keyed providers), so the first candidate decides.
func systemPaidCall(providers []chatProvider) bool {
	return len(providers) > 0 && providers[0].gateway
}

// defaultTemperature is the platform sampling temperature applied when the user
// hasn't configured one (row value 0 = unset). Reasoning models that mandate
// temperature=1 are handled by the tryChatCompletion 400-retry, not here.
func defaultTemperature(t float64) float64 {
	if t > 0 {
		return t
	}
	return 0.3
}

// normalizeReasoningEffort passes through recognized effort levels, "" otherwise.
func normalizeReasoningEffort(s string) string {
	switch v := strings.ToLower(strings.TrimSpace(s)); v {
	case "low", "medium", "high":
		return v
	}
	return ""
}

// effectiveTimeout clamps a user-configured timeout (seconds) into a safe
// range; 0/negative falls back to the platform default.
func effectiveTimeout(seconds int, def time.Duration) time.Duration {
	if seconds <= 0 {
		return def
	}
	d := time.Duration(seconds) * time.Second
	const minD, maxD = 5 * time.Second, 10 * time.Minute
	if d < minD {
		return minD
	}
	if d > maxD {
		return maxD
	}
	return d
}

// isTemperatureErrorBody reports whether an error body indicates the model
// rejects the requested sampling temperature (e.g. "field Temperature invalid,
// only 1 is allowed for this model" from kimi-k3/reasoning models).
func isTemperatureErrorBody(body []byte) bool {
	return strings.Contains(strings.ToLower(string(body)), "temperature")
}

// isReasoningEffortErrorBody reports whether an error body indicates the
// vendor rejects the reasoning_effort param — the self-heal then drops it.
func isReasoningEffortErrorBody(body []byte) bool {
	return strings.Contains(strings.ToLower(string(body)), "reasoning_effort")
}

// resolveAllChatProviders returns all enabled providers with valid secrets,
// ordered by the user's saved primary preference (via SetAIPrimary → users table).
// The primary provider comes first; then other user-configured providers; finally
// Gateway system providers as a fallback when the user has no configs at all.
func (s *Service) resolveAllChatProviders(ctx context.Context, userID uuid.UUID) ([]chatProvider, error) {
	primaryPID, primaryModel := s.getAIPrimaryGateway(ctx, userID)

	rows, err := s.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list AI providers: %w", err)
	}

	primary, rest, seenPID := s.resolveUserProviders(ctx, userID, rows, primaryPID, primaryModel)
	out := append(primary, rest...)

	// Cost breaker: when tripped, block system-paid Gateway fallback.
	// BYO-key users (out > 0) continue to work; only users with no own providers are blocked.
	if len(out) == 0 && s.gatewayProviderRepo != nil {
		if s.costBreaker != nil && s.costBreaker.IsTripped(ctx) {
			return nil, fmt.Errorf("platform daily cost limit reached — configure your own API key to continue")
		}
		out = s.resolveGatewayProviders(ctx, userID, seenPID, primaryPID, primaryModel, out)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("errors.ai.not_configured")
	}
	return out, nil
}

func (s *Service) resolveUserProviders(ctx context.Context, userID uuid.UUID, rows []*repository.SystemAIConfigRow, primaryPID, primaryModel string) (primary, rest []chatProvider, seenPID map[string]bool) {
	seenPID = map[string]bool{}
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
		if ValidateBaseURL(base) != nil {
			continue
		}
		m := resolveModel(row.DefaultModel, row.Models, row.ProviderID, primaryPID, primaryModel)
		if m == "" {
			continue
		}
		if s.modelFilter != nil && !s.modelFilter(ctx, userID, m) {
			continue
		}
		if s.isCircuitOpen(ctx, userID, row.ProviderID) {
			continue
		}
		cp := chatProvider{
			userID: userID, providerID: row.ProviderID,
			model: m, baseURL: base, secret: sec,
			maxTokens: row.MaxTokens, temperature: defaultTemperature(row.Temperature),
			timeoutSeconds: row.TimeoutSeconds, reasoningEffort: normalizeReasoningEffort(row.ReasoningEffort),
			organization: row.Organization,
		}
		seenPID[row.ProviderID] = true
		if row.ProviderID == primaryPID {
			primary = append(primary, cp)
		} else {
			rest = append(rest, cp)
		}
	}
	return
}

func (s *Service) resolveGatewayProviders(ctx context.Context, userID uuid.UUID, seenPID map[string]bool, primaryPID, primaryModel string, out []chatProvider) []chatProvider {
	sysProviders, sysErr := s.gatewayProviderRepo.ListEnabled(ctx)
	if sysErr != nil {
		return out
	}
	for _, sp := range sysProviders {
		if seenPID[sp.ProviderID] {
			continue
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
			temperature: defaultTemperature(0),
			gateway:     true,
		}
		if sp.ProviderID == primaryPID {
			out = append([]chatProvider{cp}, out...)
		} else {
			out = append(out, cp)
		}
	}
	return out
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
