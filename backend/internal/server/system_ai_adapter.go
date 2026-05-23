package server

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/ai"
	"anttrader/internal/service"
	"anttrader/internal/service/systemai"
)

// systemAIProviderAdapter bridges *systemai.Service into the
// service.SystemAIProviderSource interface used by DebateV2Service. Kept in
// the server package so that the systemai package does not have to import
// the parent service package (which would create a layering loop).
type systemAIProviderAdapter struct {
	svc *systemai.Service
}

func newSystemAIProviderAdapter(svc *systemai.Service) *systemAIProviderAdapter {
	return &systemAIProviderAdapter{svc: svc}
}

// systemProviderTypeAliases normalises the `provider_id` values stored in
// `system_ai_configs` to the canonical ai.ProviderType. Most ids already
// match (zhipu, deepseek, openai, ...) — the explicit mapping covers the
// few that do not, e.g. the OpenAI-compatible custom endpoint.
var systemProviderTypeAliases = map[string]ai.ProviderType{
	"openai_compatible": ai.ProviderCustom,
}

func (a *systemAIProviderAdapter) BuildProviderConfig(ctx context.Context, userID uuid.UUID, providerID string) (*service.AIConfig, error) {
	if a == nil || a.svc == nil {
		return nil, errors.New("system ai service unavailable")
	}
	id := strings.TrimSpace(providerID)
	if id == "" {
		return nil, errors.New("empty provider id")
	}
	row, err := a.svc.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errors.New("system ai provider not found")
	}
	if !row.Enabled {
		return nil, errors.New("system ai provider not enabled")
	}
	if !row.HasSecret {
		return nil, errors.New("system ai provider has no api key configured")
	}
	apiKey, err := a.svc.GetSecret(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	pt, ok := systemProviderTypeAliases[id]
	if !ok {
		pt = ai.ProviderType(id)
	}
	cfg := &service.AIConfig{
		Provider: pt,
		APIKey:   apiKey,
		Model:    row.DefaultModel,
		BaseURL:  row.BaseURL,
		Enabled:  true,
	}
	if row.Temperature != 0 {
		cfg.Temperature = sql.NullFloat64{Float64: row.Temperature, Valid: true}
	}
	if row.TimeoutSeconds > 0 {
		cfg.TimeoutSeconds = sql.NullInt32{Int32: int32(row.TimeoutSeconds), Valid: true}
	}
	if row.MaxTokens > 0 {
		cfg.MaxTokens = sql.NullInt32{Int32: int32(row.MaxTokens), Valid: true}
	}
	return cfg, nil
}
