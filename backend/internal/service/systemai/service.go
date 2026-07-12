// Package systemai provides per-user AI provider configuration.
// 自 059 迁移起本表按 user_id 隔离，每个用户首次进 /ai/settings 时
// 由 EnsureSeed 自动 seed 8 个 provider 空行。
package systemai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
)

// defaultProviderSeeds 描述每个用户首次进 /ai/settings 时应自动创建的
// provider 空行。BaseURL 预设值减少用户手动输入。
var defaultProviderSeeds = []struct {
	ProviderID string
	Name       string
	BaseURL    string
}{
	{"openai", "OpenAI", "https://api.openai.com/v1"},
	{"anthropic", "Anthropic (Claude)", "https://api.anthropic.com/v1"},
	{"deepseek", "DeepSeek", "https://api.deepseek.com/v1"},
	{"qwen", "通义千问", "https://dashscope.aliyuncs.com/compatible-mode/v1"},
	{"moonshot", "月之暗面 (Kimi)", "https://api.moonshot.cn/v1"},
	{"zhipu", "智谱 GLM", "https://open.bigmodel.cn/api/paas/v4"},
	{"openai_compatible", "自定义 (OpenAI 兼容)", ""},
}

// ErrInsufficientBalance is returned by the wallet pre-check when the user
// cannot afford an AI call.
var ErrInsufficientBalance = errors.New("insufficient balance for AI — please top up your wallet")

// InsufficientBalanceCode is the stable error code sent to the frontend so it
// can reliably detect balance errors without string-matching.
const InsufficientBalanceCode = "AI_INSUFFICIENT_BALANCE"

// WrapAIError maps AI service errors to ConnectRPC-friendly errors.
// If err wraps ErrInsufficientBalance, returns FailedPrecondition so
// the frontend can show a friendly "insufficient balance" message.
// Other errors are returned as-is.
func WrapAIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInsufficientBalance) {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New(InsufficientBalanceCode))
	}
	return err
}

// PostCallBiller is called after a successful AI call (streaming or non-streaming).
// If it returns an error, the result is discarded — ensuring users cannot use AI without paying.
type PostCallBiller func(ctx context.Context, userID uuid.UUID, providerID, modelName, feature string, inputTokens, outputTokens int) error

// Service exposes high-level operations consumed by the connect handler.
type Service struct {
	repo                *repository.SystemAIConfigRepository
	userRepo            *repository.UserRepository
	box                 *secretbox.Box
	secretCache         sync.Map
	postCallBiller      PostCallBiller
	walletChecker       func(ctx context.Context, userID uuid.UUID) error              // pre-check before API call
	gatewayProviderRepo *repository.SystemAIProviderRepository                         // optional: fallback for AI Gateway
	cbDB                cbExecutor                                                     // optional: PG pool for persistent circuit breaker
	modelFilter         func(ctx context.Context, userID uuid.UUID, model string) bool // optional: ADR-0025 §5.2 model whitelist
}

// SetUserRepo sets the user repository for AI primary model queries.
func (s *Service) SetUserRepo(r *repository.UserRepository) {
	s.userRepo = r
}

// secretCacheEntry holds a decrypted secret with expiry.
type secretCacheEntry struct {
	secret    string
	expiresAt time.Time
}

func NewService(repo *repository.SystemAIConfigRepository, box *secretbox.Box) *Service {
	return &Service{repo: repo, box: box}
}

// SetWalletChecker sets a pre-check called before each AI API call.
// If it returns an error, the API call is aborted before any tokens are consumed.
func (s *Service) SetWalletChecker(fn func(ctx context.Context, userID uuid.UUID) error) {
	s.walletChecker = fn
}

// SetPostCallBiller sets a billing hook called after each successful AI call.
// If it returns an error, the AI result is discarded.
func (s *Service) SetPostCallBiller(fn PostCallBiller) {
	s.postCallBiller = fn
}

// SetGatewayProviderRepo sets an optional fallback provider repo for AI Gateway.
func (s *Service) SetGatewayProviderRepo(repo *repository.SystemAIProviderRepository) {
	s.gatewayProviderRepo = repo
}

// SetModelFilter sets an optional model whitelist filter (ADR-0025 §5.2).
// When set, providers whose model is not in the whitelist are skipped during resolution.
// Returns true if the model is allowed, false to filter it out.
func (s *Service) SetModelFilter(fn func(ctx context.Context, userID uuid.UUID, model string) bool) {
	s.modelFilter = fn
}

// aiFeatureKey is a context key for tagging AI calls with a feature name.
type aiFeatureKey struct{}

// WithAIFeature returns a context tagged with the given AI feature name.
// Callers pass this ctx to ChatCompletion / ChatCompletionStream for billing attribution.
func WithAIFeature(ctx context.Context, feature string) context.Context {
	return context.WithValue(ctx, aiFeatureKey{}, feature)
}

// aiFeatureFromCtx extracts the AI feature tag from context, defaulting to "chat".
func aiFeatureFromCtx(ctx context.Context) string {
	if v := ctx.Value(aiFeatureKey{}); v != nil {
		return v.(string)
	}
	return "chat"
}

// EnsureSeed 为用户补齐缺失的默认 provider 空行（幂等）。
// 已存在的行不会被覆盖；只为缺失的 provider 插入空 stub。
// 这样既能保证新用户首次进 /ai/settings 看到全部 7 个 provider 卡片，
// 也不会在新增默认 provider（升级版本）时丢失老用户已配置的行。
func (s *Service) EnsureSeed(ctx context.Context, userID uuid.UUID) error {
	rows, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list AI configs by user: %w", err)
	}
	existing := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		if r != nil {
			existing[r.ProviderID] = struct{}{}
		}
	}
	tag := userID.String()
	for _, p := range defaultProviderSeeds {
		if _, ok := existing[p.ProviderID]; ok {
			continue
		}
		row := &repository.SystemAIConfigRow{
			UserID:     userID,
			ProviderID: p.ProviderID,
			Name:       p.Name,
			BaseURL:    p.BaseURL,
			Enabled:    p.BaseURL != "",
		}
		if err := s.repo.Upsert(ctx, row, tag); err != nil {
			return fmt.Errorf("upsert AI config seed row: %w", err)
		}
	}
	return nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]*repository.SystemAIConfigRow, error) {
	if err := s.EnsureSeed(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID, providerID string) (*repository.SystemAIConfigRow, error) {
	return s.repo.Get(ctx, userID, providerID)
}

func (s *Service) UpdateConfig(ctx context.Context, row *repository.SystemAIConfigRow, updatedBy string) error {
	return s.repo.Upsert(ctx, row, updatedBy)
}

// SetAIPrimary atomically assigns the "chat" primary role to providerID and
// saves the default model name. All row mutations execute inside a single
// database transaction.
func (s *Service) GetAIPrimary(ctx context.Context, userID uuid.UUID) (providerID, model string, err error) {
	if s.userRepo != nil {
		return s.userRepo.GetAIPrimary(ctx, userID)
	}
	return "", "", fmt.Errorf("user repository not available")
}

func (s *Service) SetAIPrimary(ctx context.Context, userID uuid.UUID, providerID, defaultModel string) error {
	// Authoritative store: the users table is what GetAIPrimary reads first, and it
	// works for AI Gateway models too (which have no owning system_ai_configs row).
	if s.userRepo == nil {
		return fmt.Errorf("user repository not available")
	}
	return s.userRepo.SetAIPrimary(ctx, userID, providerID, defaultModel)
}

// UpdateSecret encrypts and stores a provider's API key. Empty secret clears it.
func (s *Service) UpdateSecret(ctx context.Context, userID uuid.UUID, providerID, secret, updatedBy string) error {
	cacheKey := userID.String() + "|" + providerID
	s.secretCache.Delete(cacheKey) // invalidate cache on any change

	if strings.TrimSpace(secret) == "" {
		if strings.HasPrefix(providerID, "openai_compatible_") {
			if err := s.repo.Delete(ctx, userID, providerID); err != nil {
				if errors.Is(err, repository.ErrSystemAIConfigNotFound) {
					return nil
				}
				return fmt.Errorf("delete AI config: %w", err)
			}
			return nil
		}
		return s.repo.SetSecret(ctx, userID, providerID, nil, updatedBy)
	}
	if s.box == nil {
		return errors.New("secret encryption is not initialized; set jwt secret to enable")
	}
	ct, salt, nonce, err := s.box.Seal([]byte(secret))
	if err != nil {
		return fmt.Errorf("encrypt API secret: %w", err)
	}
	return s.repo.SetSecret(ctx, userID, providerID, &repository.SystemAISecret{
		Ciphertext: ct, Salt: salt, Nonce: nonce,
	}, updatedBy)
}

// GetSecret returns the decrypted secret. Empty string when none configured
// or when decryption is unavailable; only the connect handler uses this.
func (s *Service) GetSecret(ctx context.Context, userID uuid.UUID, providerID string) (string, error) {
	rec, err := s.repo.GetSecret(ctx, userID, providerID)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", nil
	}
	if s.box == nil {
		return "", errors.New("secret encryption is not initialized")
	}
	pt, err := s.box.Open(rec.Ciphertext, rec.Salt, rec.Nonce)
	if err != nil {
		return "", fmt.Errorf("decrypt secret for provider %s: %w", providerID, err)
	}
	return string(pt), nil
}

// DiscoverModels calls the provider's /models endpoint and returns deduplicated
// model IDs. The configured base_url and stored secret are used. The result is
// NOT persisted here; the caller decides whether to write it back.
func (s *Service) DiscoverModels(ctx context.Context, userID uuid.UUID, providerID string) ([]string, error) {
	cfg, err := s.repo.Get(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, errBaseURLEmpty
	}
	if perr := validateBaseURL(base); perr != nil {
		return nil, perr
	}
	secret, secErr := s.GetSecret(ctx, userID, providerID)
	if secErr != nil {
		return nil, fmt.Errorf("get secret: %w", secErr)
	}

	// Zhipu uses non-standard pagination; try its dedicated path first.
	if providerID == "zhipu" {
		if all, derr := fetchZhipuModels(ctx, base, secret); derr == nil && len(all) > 0 {
			return all, nil
		}
	}
	return fetchOpenAIModels(ctx, base, secret)
}
