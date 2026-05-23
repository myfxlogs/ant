package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/ai"
	"anttrader/internal/ai/anthropic"
	"anttrader/internal/ai/deepseek"
	"anttrader/internal/ai/openai"
	"anttrader/internal/ai/zhipu"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
)

// AIConfig 是 service 层向 AI 客户端构造器透传的运行时配置。
// 自 060 起所有「用户当前在用哪个 AI」事实统一来自 system_ai_configs，
// AIConfigService 只负责把行翻译成 AIConfig + 选 provider 的策略。
type AIConfig struct {
	Provider       ai.ProviderType
	APIKey         string
	Model          string
	BaseURL        string
	Enabled        bool
	Temperature    sql.NullFloat64 // 0.0~2.0；零值表示「使用 provider 默认」
	TimeoutSeconds sql.NullInt32   // 1~600
	MaxTokens      sql.NullInt32   // 1~32768
}

// AIProviderInfo 是 ListProviders 给前端的 catalog 条目。
type AIProviderInfo struct {
	Type           ai.ProviderType
	Label          string
	DefaultModel   string
	SupportsStream bool
	Models         []string
	AllowCustom    bool
	RequireBaseURL bool
	BaseURL        string
}

// AIConfigService 聚焦「构造 provider」与「按用户挑当前 provider」。
// 不持久化任何配置——所有写入走 SystemAIService（system_ai_configs）；
// 唯一例外是 users.ai_primary_provider_id / users.ai_primary_model 这一对
// 「默认主模型」标记，由 UserPrimaryStore 接口承载（避免硬依赖整个 UserRepo）。
type AIConfigService struct {
	systemAIRepo *repository.SystemAIConfigRepository
	box          *secretbox.Box
	dynamicCfg   *DynamicConfigService
	primaryStore UserPrimaryStore // 可空；空时退化为「无 primary」纯 fallback 路径
}

// UserPrimaryStore 抽出 user_repo 的 (Get|Set)AIPrimary 两个方法，让 AIConfig
// 服务无需引用整个 UserRepository（避免循环 import 风险）。
type UserPrimaryStore interface {
	GetAIPrimary(ctx context.Context, id uuid.UUID) (providerID, model string, err error)
	SetAIPrimary(ctx context.Context, id uuid.UUID, providerID, model string) error
}

// NewAIConfigService 构造服务。systemAIRepo / box 必填；dynamicCfg / primaryStore 可空。
func NewAIConfigService(
	systemAIRepo *repository.SystemAIConfigRepository,
	box *secretbox.Box,
	dynamicCfg *DynamicConfigService,
) *AIConfigService {
	return &AIConfigService{systemAIRepo: systemAIRepo, box: box, dynamicCfg: dynamicCfg}
}

// WithPrimaryStore 注入用户 primary 读写源（user_repo）。
func (s *AIConfigService) WithPrimaryStore(store UserPrimaryStore) *AIConfigService {
	if s != nil {
		s.primaryStore = store
	}
	return s
}

const cfgAIProviderCatalog = "ai.provider_catalog"

type aiProviderCatalogItem struct {
	Type           string   `json:"type"`
	Label          string   `json:"label"`
	DefaultModel   string   `json:"default_model"`
	SupportsStream bool     `json:"supports_stream"`
	Models         []string `json:"models"`
	AllowCustom    bool     `json:"allow_custom_model"`
	RequireBaseURL bool     `json:"require_base_url"`
	BaseURL        string   `json:"base_url,omitempty"`
}

type aiProviderCatalog struct {
	Providers []aiProviderCatalogItem `json:"providers"`
}

// ListProviders 返回支持的 provider 清单（dynamicCfg 优先 → 内置 fallback）。
// 仅作为元数据展示，与具体用户无关。
func (s *AIConfigService) ListProviders(ctx context.Context) []AIProviderInfo {
	if s.dynamicCfg != nil {
		if raw, enabled, _ := s.dynamicCfg.GetString(ctx, cfgAIProviderCatalog, ""); enabled {
			if raw = strings.TrimSpace(raw); raw != "" {
				var cat aiProviderCatalog
				if err := json.Unmarshal([]byte(raw), &cat); err == nil {
					out := make([]AIProviderInfo, 0, len(cat.Providers))
					for _, p := range cat.Providers {
						pt := ai.ProviderType(strings.TrimSpace(p.Type))
						if !pt.IsValid() {
							continue
						}
						out = append(out, AIProviderInfo{
							Type:           pt,
							Label:          p.Label,
							DefaultModel:   p.DefaultModel,
							SupportsStream: p.SupportsStream,
							Models:         p.Models,
							AllowCustom:    p.AllowCustom,
							RequireBaseURL: p.RequireBaseURL,
							BaseURL:        strings.TrimSpace(p.BaseURL),
						})
					}
					if len(out) > 0 {
						return out
					}
				}
			}
		}
	}
	return builtinProviders()
}

// builtinProviders 兜底：当 dynamicConfig 未提供时使用的 7 个内置 provider 描述。
func builtinProviders() []AIProviderInfo {
	return []AIProviderInfo{
		{Type: ai.ProviderOpenAI, Label: "OpenAI", DefaultModel: "gpt-4o-mini", SupportsStream: true, AllowCustom: true},
		{Type: ai.ProviderAnthropic, Label: "Anthropic (Claude)", DefaultModel: "claude-3-5-sonnet-latest", SupportsStream: true, AllowCustom: true},
		{Type: ai.ProviderDeepSeek, Label: "DeepSeek", DefaultModel: "deepseek-chat", SupportsStream: true, AllowCustom: true},
		{Type: ai.ProviderQwen, Label: "通义千问", DefaultModel: "qwen-plus", SupportsStream: true, AllowCustom: true},
		{Type: ai.ProviderMoonshot, Label: "月之暗面 (Kimi)", DefaultModel: "moonshot-v1-8k", SupportsStream: true, AllowCustom: true},
		{Type: ai.ProviderZhipu, Label: "智谱 GLM", DefaultModel: "glm-4-flash", SupportsStream: true, AllowCustom: true},
		{Type: ai.ProviderCustom, Label: "自定义 (OpenAI 兼容)", DefaultModel: "", SupportsStream: true, AllowCustom: true, RequireBaseURL: true},
	}
}

// pickEnabledSystemAIRow 在该用户的 system_ai_configs 中挑一个「启用 + 有密钥 + 配了 default_model」的行。
// 返回 (row, secret, ok)。secret 已解密。
func (s *AIConfigService) pickEnabledSystemAIRow(ctx context.Context, userID uuid.UUID) (*repository.SystemAIConfigRow, string, error) {
	if s.systemAIRepo == nil {
		return nil, "", errors.New("system ai repo not configured")
	}
	rows, err := s.systemAIRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	for _, r := range rows {
		if r == nil || !r.Enabled || !r.HasSecret {
			continue
		}
		if strings.TrimSpace(r.DefaultModel) == "" {
			continue
		}
		secret, sErr := s.decryptSecret(ctx, userID, r.ProviderID)
		if sErr != nil || secret == "" {
			continue
		}
		return r, secret, nil
	}
	return nil, "", errors.New("no enabled & configured AI provider for user")
}

func (s *AIConfigService) decryptSecret(ctx context.Context, userID uuid.UUID, providerID string) (string, error) {
	rec, err := s.systemAIRepo.GetSecret(ctx, userID, providerID)
	if err != nil || rec == nil {
		return "", err
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

// rowToAIConfig 将 system_ai_configs 行 + 已解密 secret 包成 AIConfig。
func rowToAIConfig(row *repository.SystemAIConfigRow, secret string) *AIConfig {
	pt := ai.ProviderType(row.ProviderID)
	if row.ProviderID == "openai_compatible" || strings.HasPrefix(row.ProviderID, "openai_compatible_") {
		pt = ai.ProviderCustom
	}
	cfg := &AIConfig{
		Provider: pt,
		APIKey:   secret,
		Model:    row.DefaultModel,
		BaseURL:  row.BaseURL,
		Enabled:  row.Enabled,
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
	return cfg
}

// GetConfig 返回该用户「当前默认在用」的 AIConfig。
// 调用方常用于 chat 兜底；找不到合适 provider 时返回 (nil, false, nil)。
func (s *AIConfigService) GetConfig(ctx context.Context, userID uuid.UUID) (*AIConfig, bool, error) {
	row, secret, err := s.pickEnabledSystemAIRow(ctx, userID)
	if err != nil {
		return nil, false, nil
	}
	return rowToAIConfig(row, secret), true, nil
}

// GetProviderByRole 拿「用户默认主模型」provider：
//  1. 若用户在 /ai/settings 里显式选了 primary（users.ai_primary_*），按那条
//     行 + override model 构造 provider；
//  2. 否则 fallback 到 pickEnabledSystemAIRow 首行（保持 060 行为）。
//
// role 参数保留以兼容旧调用方，自 060 起忽略。
func (s *AIConfigService) GetProviderByRole(ctx context.Context, userID uuid.UUID, _ string) (ai.AIProvider, error) {
	if cfg, ok, _ := s.primaryConfig(ctx, userID); ok {
		return s.buildProvider(cfg)
	}
	row, secret, err := s.pickEnabledSystemAIRow(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no AI provider available: %w", err)
	}
	return s.buildProvider(rowToAIConfig(row, secret))
}

// primaryConfig 把 users.ai_primary_* 转成可构造 provider 的 AIConfig。
// providerID 为空、行不存在、密钥缺失等情况均返回 (nil, false, nil)，让调用方
// 自然 fallback；只有 DB 真正报错才返回 err。
func (s *AIConfigService) primaryConfig(ctx context.Context, userID uuid.UUID) (*AIConfig, bool, error) {
	if s == nil || s.primaryStore == nil || s.systemAIRepo == nil {
		return nil, false, nil
	}
	pid, model, err := s.primaryStore.GetAIPrimary(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	pid = strings.TrimSpace(pid)
	if pid == "" {
		return nil, false, nil
	}
	row, err := s.systemAIRepo.Get(ctx, userID, pid)
	if err != nil || row == nil || !row.Enabled || !row.HasSecret {
		return nil, false, nil
	}
	secret, err := s.decryptSecret(ctx, userID, pid)
	if err != nil || secret == "" {
		return nil, false, nil
	}
	cfg := rowToAIConfig(row, secret)
	if m := strings.TrimSpace(model); m != "" {
		cfg.Model = m
	}
	if strings.TrimSpace(cfg.Model) == "" {
		// row 没 default_model 又没 override → 没法用，回落 fallback。
		return nil, false, nil
	}
	return cfg, true, nil
}

// GetPrimary 给 HTTP handler 用：返回用户当前 primary 选择，未设置 = 空字符串。
func (s *AIConfigService) GetPrimary(ctx context.Context, userID uuid.UUID) (providerID, model string, err error) {
	if s == nil || s.primaryStore == nil {
		return "", "", nil
	}
	return s.primaryStore.GetAIPrimary(ctx, userID)
}

// SetPrimary 给 HTTP handler 用。空 providerID = 清除选择。非空时校验该
// (provider_id, model) 在用户的 system_ai_configs 中真实存在且 enabled+has_secret，
// 失败则拒绝写入，避免存进一行「永远跑不起来」的死配置。
func (s *AIConfigService) SetPrimary(ctx context.Context, userID uuid.UUID, providerID, model string) error {
	providerID = strings.TrimSpace(providerID)
	model = strings.TrimSpace(model)
	if s == nil || s.primaryStore == nil {
		return errors.New("primary store not configured")
	}
	if providerID == "" {
		return s.primaryStore.SetAIPrimary(ctx, userID, "", "")
	}
	if s.systemAIRepo == nil {
		return errors.New("system ai repo not configured")
	}
	row, err := s.systemAIRepo.Get(ctx, userID, providerID)
	if err != nil {
		return err
	}
	if row == nil || !row.Enabled || !row.HasSecret {
		return errors.New("provider is not enabled or has no secret")
	}
	if model == "" {
		model = row.DefaultModel
	}
	if model == "" {
		return errors.New("model is required")
	}
	// 仅当 row 已显式列出 models 时才严格校验；空列表的 row 容许任意 model
	// （某些 provider 我们并没有把所有可用模型枚举完整）。
	if len(row.Models) > 0 {
		hit := false
		for _, m := range row.Models {
			if strings.EqualFold(strings.TrimSpace(m), model) {
				hit = true
				break
			}
		}
		if !hit {
			return fmt.Errorf("model %q not in provider's models list", model)
		}
	}
	return s.primaryStore.SetAIPrimary(ctx, userID, providerID, model)
}

// ValidateConfig 实际 ping 一下 provider 用最小消息。
func (s *AIConfigService) ValidateConfig(ctx context.Context, cfg *AIConfig) error {
	provider, err := s.buildProvider(cfg)
	if err != nil {
		return err
	}
	if err := provider.ValidateConfig(); err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = provider.Chat(pingCtx, []ai.Message{
		{Role: "system", Content: "You are a health-check endpoint. Reply with a single word: OK."},
		{Role: "user", Content: "OK"},
	})
	return err
}

// BuildProvider 是 buildProvider 的导出别名，给 debate_v2 等模块使用。
func (s *AIConfigService) BuildProvider(cfg *AIConfig) (ai.AIProvider, error) {
	return s.buildProvider(cfg)
}

func (s *AIConfigService) buildProvider(cfg *AIConfig) (ai.AIProvider, error) {
	const (
		defaultChatTimeout = 90 * time.Second
		defaultMaxTokens   = 2048
	)
	if cfg == nil {
		return nil, fmt.Errorf("nil ai config")
	}
	if !cfg.Provider.IsValid() {
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}
	build := func(provider ai.AIProvider) ai.AIProvider {
		temperature := nullFloatPtr(cfg.Temperature)
		maxTokens := nullIntPtr(cfg.MaxTokens)
		if maxTokens == nil {
			n := defaultMaxTokens
			maxTokens = &n
		}
		if setter, ok := provider.(interface{ SetSamplingParams(*float64, *int) }); ok {
			setter.SetSamplingParams(temperature, maxTokens)
		}
		if setter, ok := provider.(interface{ SetTimeout(time.Duration) }); ok {
			if cfg.TimeoutSeconds.Valid && cfg.TimeoutSeconds.Int32 > 0 {
				setter.SetTimeout(time.Duration(cfg.TimeoutSeconds.Int32) * time.Second)
			} else {
				setter.SetTimeout(defaultChatTimeout)
			}
		}
		return provider
	}
	switch cfg.Provider {
	case ai.ProviderZhipu:
		return build(zhipu.NewClient(cfg.APIKey, cfg.Model)), nil
	case ai.ProviderAnthropic:
		baseURL := strings.TrimSpace(cfg.BaseURL)
		if baseURL == "" {
			baseURL = ai.ProviderAnthropic.PresetBaseURL()
		}
		return build(anthropic.NewClient(cfg.APIKey, cfg.Model, baseURL)), nil
	}
	if cfg.Provider.IsOpenAICompatible() {
		baseURL := strings.TrimSpace(cfg.BaseURL)
		if baseURL == "" {
			baseURL = cfg.Provider.PresetBaseURL()
		}
		if cfg.Provider == ai.ProviderDeepSeek {
			if cfg.Model != "" {
				return build(deepseek.NewClientWithModel(cfg.APIKey, cfg.Model)), nil
			}
			return build(deepseek.NewClient(cfg.APIKey)), nil
		}
		return build(openai.NewClient(cfg.APIKey, cfg.Model, baseURL)), nil
	}
	return nil, fmt.Errorf("provider not supported: %s", cfg.Provider)
}

func nullFloatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func nullIntPtr(v sql.NullInt32) *int {
	if !v.Valid || v.Int32 <= 0 {
		return nil
	}
	n := int(v.Int32)
	return &n
}

// MaskAPIKey 给前端展示用：前 4 + **** + 后 4。
func MaskAPIKey(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ""
	}
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}
