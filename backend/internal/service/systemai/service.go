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

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
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

// TokenRecord is passed to the recorder after each successful AI call.
type TokenRecord struct {
	UserID        uuid.UUID
	ProviderID    string
	Model         string
	Feature       string
	InputTokens   int
	OutputTokens  int
}

// TokenRecorder is called after each successful ChatCompletion / ChatCompletionStream.
type TokenRecorder func(ctx context.Context, r TokenRecord)

// Service exposes high-level operations consumed by the connect handler.
type Service struct {
	repo                *repository.SystemAIConfigRepository
	userRepo            *repository.UserRepository
	box                 *secretbox.Box
	secretCache         sync.Map
	tokenRecorder       TokenRecorder
	walletChecker       func(ctx context.Context, userID uuid.UUID) error // pre-check before API call
	gatewayProviderRepo *repository.SystemAIProviderRepository // optional: fallback for AI Gateway
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

// SetTokenRecorder sets a callback invoked after each successful AI call.
func (s *Service) SetTokenRecorder(fn TokenRecorder) {
	s.tokenRecorder = fn
}

// SetWalletChecker sets a pre-check called before each AI API call.
// If it returns an error, the API call is aborted before any tokens are consumed.
func (s *Service) SetWalletChecker(fn func(ctx context.Context, userID uuid.UUID) error) {
	s.walletChecker = fn
}

// SetGatewayProviderRepo sets an optional fallback provider repo for AI Gateway.
func (s *Service) SetGatewayProviderRepo(repo *repository.SystemAIProviderRepository) {
	s.gatewayProviderRepo = repo
}

// aiFeatureKey is a context key for tagging AI calls with a feature name.
type aiFeatureKey struct{}

// WithAIFeature returns a context tagged with the given AI feature name.
// Callers pass this ctx to ChatCompletion / ChatCompletionStream for billing attribution.
func WithAIFeature(ctx context.Context, feature string) context.Context {
	return context.WithValue(ctx, aiFeatureKey{}, feature)
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

// RowToProto converts a repository row into the protobuf config message.
func RowToProto(r *repository.SystemAIConfigRow) *antv1.SystemAIConfig {
	if r == nil {
		return &antv1.SystemAIConfig{}
	}
	return &antv1.SystemAIConfig{
		ProviderId:     r.ProviderID,
		Name:           r.Name,
		BaseUrl:        r.BaseURL,
		Organization:   r.Organization,
		Models:         r.Models,
		DefaultModel:   r.DefaultModel,
		Temperature:    r.Temperature,
		TimeoutSeconds: int32(r.TimeoutSeconds),
		MaxTokens:      int32(r.MaxTokens),
		Purposes:       r.Purposes,
		PrimaryFor:     r.PrimaryFor,
		Enabled:        r.Enabled,
		HasSecret:      r.HasSecret,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      r.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:      r.UpdatedBy,
	}
}

// FriendlyError maps internal errors to user-readable Chinese messages so the
// connect handler can return clean error strings to the frontend.
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrInsufficientBalance) {
		return "余额不足，请先充值。"
	}
	msg := err.Error()
	// Pass through i18n keys without wrapping.
	if strings.HasPrefix(msg, "errors.") {
		return msg
	}
	low := strings.ToLower(msg)
	switch {
	case errors.Is(err, errBaseURLEmpty):
		return "请先填写 Base URL（模型服务地址）。"
	case strings.Contains(low, "free-tier") || strings.Contains(low, "free tier"):
		return "免费额度已耗尽：请在厂商控制台关闭「仅使用免费档」或更换付费 Key。"
	case strings.Contains(low, "quota") || strings.Contains(low, "rate limit") || strings.Contains(low, "too many requests") || strings.Contains(low, "status 429"):
		return "配额受限或被限流：厂商已拒绝调用。请检查计费/速率限制或稍后重试。"
	case strings.Contains(low, "unauthorized"):
		return "鉴权失败：请检查 API Key 是否正确，或确认网关是否需要密钥。"
	case strings.Contains(low, "endpoint not found") || strings.Contains(low, "status 404"):
		return "模型端点不存在：请确认 Base URL 与服务协议匹配（部分服务需要 /v1）。"
	case strings.Contains(low, "timeout"):
		return "请求超时：模型服务未在 60s 内响应，请检查网络或切换到其他厂商（如 DeepSeek）"
	case strings.Contains(low, "unreachable"):
		return "无法连接到模型服务：请检查 Base URL、网络或网关。"
	case strings.Contains(low, "invalid /models response") || strings.Contains(low, "cannot parse json"):
		return "模型服务返回格式不兼容 /models 协议。"
	case strings.Contains(low, "no models"):
		return "模型服务未返回可用模型，请检查账号权限或服务配置。"
	case strings.Contains(low, "user location is not supported"):
		// English only: frontend i18n maps this to user locale (zh-CN/zh-TW/ja/vi/…).
		return "User location is not supported for the API use. The upstream may block this region (egress IP); try a supported network, proxy, or another provider."
	case strings.Contains(low, "base url"):
		return "Base URL 格式无效：请填写完整地址，例如 https://api.example.com/v1。"
	// ── 400 sub-types (read from provider error body) ──
	case strings.Contains(low, "status 400") && strings.Contains(low, "model not found") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "does not exist") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "invalid_model") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "no such model") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "unknown model"):
		return "模型名称不存在于当前供应商，请检查模型名称或切换到其他厂商。"
	case strings.Contains(low, "status 400") && strings.Contains(low, "context") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "too long") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "token limit") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "max tokens"):
		return "输入内容超出模型上下文限制，请缩短消息或清除对话历史后重试。"
	case strings.Contains(low, "status 400") && strings.Contains(low, "invalid api key") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "incorrect api key"):
		return "API Key 格式无效：请检查密钥是否正确，或确认已启用该供应商。"
	case strings.Contains(low, "status 400") && strings.Contains(low, "stream") ||
		strings.Contains(low, "status 400") && strings.Contains(low, "does not support"):
		return "当前模型不支持流式输出，请在设置中更换模型或关闭流式传输。"
	case strings.Contains(low, "status 400"):
		// Generic 400 — try to extract the provider error message for context.
		return "模型服务拒绝了请求，可能是参数不兼容或模型暂时不可用。详情：" + msg
	default:
		return "拉取模型失败：" + msg
	}
}
