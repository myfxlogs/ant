// Package systemai provides per-user AI provider configuration.
// 自 059 迁移起本表按 user_id 隔离，每个用户首次进 /ai/settings 时
// 由 EnsureSeed 自动 seed 8 个 provider 空行。
package systemai

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
)

// defaultProviderSeeds 描述每个用户首次进 /ai/settings 时应自动创建的
// provider 空行。
var defaultProviderSeeds = []struct {
	ProviderID string
	Name       string
}{
	{"openai", "OpenAI"},
	{"anthropic", "Anthropic (Claude)"},
	{"deepseek", "DeepSeek"},
	{"qwen", "通义千问"},
	{"moonshot", "月之暗面 (Kimi)"},
	{"zhipu", "智谱 GLM"},
	{"openai_compatible", "自定义 (OpenAI 兼容)"},
}

// Service exposes high-level operations consumed by the connect handler.
type Service struct {
	repo *repository.SystemAIConfigRepository
	box  *secretbox.Box
}

func NewService(repo *repository.SystemAIConfigRepository, box *secretbox.Box) *Service {
	return &Service{repo: repo, box: box}
}

// EnsureSeed 为用户补齐缺失的默认 provider 空行（幂等）。
// 已存在的行不会被覆盖；只为缺失的 provider 插入空 stub。
// 这样既能保证新用户首次进 /ai/settings 看到全部 7 个 provider 卡片，
// 也不会在新增默认 provider（升级版本）时丢失老用户已配置的行。
func (s *Service) EnsureSeed(ctx context.Context, userID uuid.UUID) error {
	rows, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return err
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
		}
		if err := s.repo.Upsert(ctx, row, tag); err != nil {
			return err
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

// UpdateSecret encrypts and stores a provider's API key. Empty secret clears it.
func (s *Service) UpdateSecret(ctx context.Context, userID uuid.UUID, providerID, secret, updatedBy string) error {
	if strings.TrimSpace(secret) == "" {
		if strings.HasPrefix(providerID, "openai_compatible_") {
			if err := s.repo.Delete(ctx, userID, providerID); err != nil {
				if errors.Is(err, repository.ErrSystemAIConfigNotFound) {
					return nil
				}
				return err
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
		return err
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
		return "", err
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
	secret, _ := s.GetSecret(ctx, userID, providerID)

	// Zhipu uses non-standard pagination; try its dedicated path first.
	if providerID == "zhipu" {
		if all, derr := fetchZhipuModels(ctx, base, secret); derr == nil && len(all) > 0 {
			return all, nil
		}
	}
	return fetchOpenAIModels(ctx, base, secret)
}

// FriendlyError maps internal errors to user-readable Chinese messages so the
// connect handler can return clean error strings to the frontend.
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
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
		return "请求超时：请检查网络连通性或稍后重试。"
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
	default:
		return "拉取模型失败：" + msg
	}
}
