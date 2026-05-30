package config

import (
	"time"
)

// AIConfig AI配置结构
type AIConfig struct {
	Provider    ProviderType  `yaml:"provider" json:"provider"`
	// APIKey is stored in plaintext (legacy). New code should use system_ai_configs.secret_ciphertext via systemai.Service.
	APIKey      string        `yaml:"api_key" json:"api_key"`
	ModelName   string        `yaml:"model_name" json:"model_name"`
	MaxTokens   int           `yaml:"max_tokens" json:"max_tokens"`
	Temperature float64       `yaml:"temperature" json:"temperature"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
}

// ProviderType AI提供商类型
type ProviderType string

const (
	ProviderZhipu    ProviderType = "zhipu"
	ProviderDeepSeek ProviderType = "deepseek"
)

// DefaultAIConfigs is read-only after package init. Do not mutate at runtime.
var DefaultAIConfigs = map[ProviderType]AIConfig{
	ProviderZhipu: {
		Provider:    ProviderZhipu,
		ModelName:   "glm-4-flash",
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     300 * time.Second,
	},
	ProviderDeepSeek: {
		Provider:    ProviderDeepSeek,
		ModelName:   "deepseek-chat",
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     300 * time.Second,
	},
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig(provider ProviderType) AIConfig {
	if cfg, ok := DefaultAIConfigs[provider]; ok {
		return cfg
	}
	return AIConfig{}
}

// IsValid 检查配置是否有效
func (c *AIConfig) IsValid() bool {
	if c.Provider == "" || c.APIKey == "" {
		return false
	}
	return c.Provider.IsValid()
}

// ProviderType的IsValid方法
func (p ProviderType) IsValid() bool {
	return p != "" // any non-empty provider is valid (validation is at the systemai layer)
}
