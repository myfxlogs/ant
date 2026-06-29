package systemai

import (
	"errors"
	"strings"
	"time"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
)

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
	if strings.HasPrefix(msg, "errors.") {
		return msg
	}
	low := strings.ToLower(msg)
	errType := ""
	if strings.HasPrefix(msg, "[") {
		if end := strings.IndexByte(msg, ']'); end > 1 {
			errType = strings.ToLower(msg[1:end])
		}
	}

	switch {
	case errors.Is(err, errBaseURLEmpty):
		return "请先填写 Base URL（模型服务地址）。"

	case errType == "insufficient_quota" || errType == "rate_limit_error" ||
		strings.Contains(low, "status 429"):
		return "配额受限或被限流：厂商已拒绝调用。请检查计费/速率限制或稍后重试。"
	case errType == "authentication_error" || errType == "invalid_api_key" ||
		strings.Contains(low, "status 401"):
		return "鉴权失败：请检查 API Key 是否正确，或确认网关是否需要密钥。"
	case errType == "invalid_request_error":
		switch {
		case strings.Contains(low, "model") && (strings.Contains(low, "not found") || strings.Contains(low, "not exist") || strings.Contains(low, "unknown")):
			return "模型名称不存在于当前供应商，请检查模型名称或切换到其他厂商。"
		case strings.Contains(low, "context") || strings.Contains(low, "token") || strings.Contains(low, "too long"):
			return "输入内容超出模型上下文限制，请缩短消息或清除对话历史后重试。"
		case strings.Contains(low, "stream"):
			return "当前模型不支持流式输出，正在自动切换非流式模式，请稍候重试。"
		default:
			return "请求参数有误：" + extractMessage(msg)
		}
	case errType == "server_error":
		return "模型服务内部错误，请稍后重试或切换到其他厂商。"

	case strings.Contains(low, "free-tier") || strings.Contains(low, "free tier"):
		return "免费额度已耗尽：请在厂商控制台关闭「仅使用免费档」或更换付费 Key。"
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
		return "User location is not supported for the API use. The upstream may block this region (egress IP); try a supported network, proxy, or another provider."
	case strings.Contains(low, "base url"):
		return "Base URL 格式无效：请填写完整地址，例如 https://api.example.com/v1。"
	case strings.Contains(low, "api key") && (strings.Contains(low, "expired") || strings.Contains(low, "revoked") || strings.Contains(low, "suspended") || strings.Contains(low, "disabled") || strings.Contains(low, "billing") || strings.Contains(low, "forbidden")):
		return "API Key 已失效（过期/被撤销/欠费），请更新密钥或联系供应商。"
	case strings.Contains(low, "status 400"):
		return "模型服务拒绝了请求，可能是参数不兼容或模型暂时不可用。详情：" + msg
	default:
		return "拉取模型失败：" + msg
	}
}

// extractMessage strips the error type prefix and returns the human-readable part.
func extractMessage(msg string) string {
	if idx := strings.IndexByte(msg, ']'); idx > 0 && idx < len(msg)-1 {
		return strings.TrimSpace(msg[idx+1:])
	}
	return msg
}
