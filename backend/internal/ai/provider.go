package ai

import (
	"context"
	"errors"
)

var (
	ErrProviderNotFound    = errors.New("ai provider not found")
	ErrInvalidAPIKey       = errors.New("invalid api key")
	ErrRequestFailed       = errors.New("request failed")
	ErrStreamNotSupported  = errors.New("stream not supported")
	ErrInvalidResponse     = errors.New("invalid response")
	ErrConfigNotFound      = errors.New("ai config not found")
	ErrConfigAlreadyExists = errors.New("ai config already exists")
)

// Message AI消息结构
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// Usage Token使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Response AI响应结构
type Response struct {
	Content string `json:"content"`
	Usage   Usage  `json:"usage"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
	Error   error  `json:"error,omitempty"`
}

// AIProvider AI模型提供商接口
type AIProvider interface {
	// Chat 同步对话
	Chat(ctx context.Context, messages []Message) (*Response, error)

	// StreamChat 流式对话
	StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error)

	// ValidateConfig 验证配置
	ValidateConfig() error

	// GetModelName 获取模型名称
	GetModelName() string

	// GetProviderName 获取提供商名称
	GetProviderName() string
}

// ProviderType 提供商类型
type ProviderType string

const (
	ProviderZhipu        ProviderType = "zhipu"
	ProviderDeepSeek     ProviderType = "deepseek"
	ProviderCustom       ProviderType = "custom"
	ProviderOpenAI       ProviderType = "openai"
	ProviderMoonshot     ProviderType = "moonshot"
	ProviderQwen         ProviderType = "qwen"
	ProviderDoubao       ProviderType = "doubao"
	ProviderOpenRouter   ProviderType = "openrouter"
	ProviderMistral      ProviderType = "mistral"
	ProviderGroq         ProviderType = "groq"
	ProviderSiliconFlow  ProviderType = "siliconflow"
	ProviderAnthropic    ProviderType = "anthropic"
)

// IsValid 检查提供商类型是否有效
func (p ProviderType) IsValid() bool {
	switch p {
	case ProviderZhipu, ProviderDeepSeek, ProviderCustom,
		ProviderOpenAI, ProviderMoonshot, ProviderQwen, ProviderDoubao,
		ProviderOpenRouter, ProviderMistral, ProviderGroq,
		ProviderSiliconFlow, ProviderAnthropic:
		return true
	default:
		return false
	}
}

// IsOpenAICompatible reports whether the provider speaks the OpenAI
// /v1/chat/completions + /v1/models protocol, so it can be served by the
// generic openai client.
func (p ProviderType) IsOpenAICompatible() bool {
	switch p {
	case ProviderCustom, ProviderOpenAI, ProviderDeepSeek, ProviderMoonshot,
		ProviderQwen, ProviderDoubao, ProviderOpenRouter,
		ProviderMistral, ProviderGroq, ProviderSiliconFlow:
		return true
	default:
		return false
	}
}

// PresetBaseURL returns the canonical OpenAI-compatible base URL for a
// provider. Empty string means the user must provide one (e.g. custom).
func (p ProviderType) PresetBaseURL() string {
	switch p {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderDeepSeek:
		return "https://api.deepseek.com/v1"
	case ProviderMoonshot:
		return "https://api.moonshot.cn/v1"
	case ProviderQwen:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case ProviderDoubao:
		return "https://ark.cn-beijing.volces.com/api/v3"
	case ProviderOpenRouter:
		return "https://openrouter.ai/api/v1"
	case ProviderMistral:
		return "https://api.mistral.ai/v1"
	case ProviderGroq:
		return "https://api.groq.com/openai/v1"
	case ProviderSiliconFlow:
		return "https://api.siliconflow.cn/v1"
	case ProviderZhipu:
		// Zhipu has its own SDK path, so this URL is used only for /models probe.
		return "https://open.bigmodel.cn/api/paas/v4"
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	default:
		return ""
	}
}

// String 返回提供商类型字符串
func (p ProviderType) String() string {
	return string(p)
}
