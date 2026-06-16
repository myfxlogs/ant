// Package ai provides centralized language detection and prompt generation.
// All AI-facing language logic lives here so handler packages don't duplicate.
package ai

import "strings"

// NormalizeLocale converts frontend-style locale codes (zh-CN, zh-TW, zh-HK)
// to the canonical form used by system prompts (zh, zh-tw, ja, vi, en).
func NormalizeLocale(raw string) string {
	primary := strings.TrimSpace(raw)
	if idx := strings.IndexByte(primary, ','); idx > 0 {
		primary = primary[:idx]
	}
	if idx := strings.IndexByte(primary, ';'); idx > 0 {
		primary = primary[:idx]
	}
	switch {
	case strings.HasPrefix(primary, "zh-tw") || strings.HasPrefix(primary, "zh-TW") ||
		strings.HasPrefix(primary, "zh-HK") || strings.HasPrefix(primary, "zh_HK") ||
		strings.HasPrefix(primary, "zh-Hant"):
		return "zh-tw"
	case strings.HasPrefix(primary, "zh"):
		return "zh"
	case strings.HasPrefix(primary, "ja"):
		return "ja"
	case strings.HasPrefix(primary, "vi"):
		return "vi"
	default:
		return "en"
	}
}

// LangPrompt returns a system prompt instructing the LLM to respond in the
// given language. lang must be pre-normalized (use NormalizeLocale).
func LangPrompt(lang string) string {
	switch lang {
	case "zh":
		return "你是一个专业的量化交易助手。请始终使用简体中文回复，简洁具体。"
	case "zh-tw":
		return "你是一個專業的量化交易助手。請始終使用繁體中文回覆，簡潔具體。"
	case "ja":
		return "あなたはプロのクオンツトレーディングアシスタントです。常に日本語で簡潔に具体的に回答してください。"
	case "vi":
		return "Bạn là trợ lý giao dịch định lượng chuyên nghiệp. Luôn trả lời bằng tiếng Việt, ngắn gọn và cụ thể."
	default:
		return "You are a professional quantitative trading assistant. You MUST reply in English only. Keep it concise and specific. Do NOT use any other language."
	}
}

// ClarifyLangDirective forces structured JSON output to be in the user's language.
func ClarifyLangDirective(lang string) string {
	switch lang {
	case "zh":
		return "questions 数组中的所有问题必须使用简体中文。"
	case "zh-tw":
		return "questions 陣列中的所有問題必須使用繁體中文。"
	case "ja":
		return "questions 配列内のすべての質問は日本語で記述してください。"
	case "vi":
		return "Tất cả câu hỏi trong mảng 'questions' phải được viết bằng tiếng Việt."
	default:
		return "All questions in the 'questions' array MUST be written in English."
	}
}

// FallbackQuestions returns language-appropriate fallback prompts when AI
// clarification fails (JSON parse error or very low confidence).
func FallbackQuestions(lang string) (detailed, brief string) {
	switch lang {
	case "zh":
		return "请更详细地描述您的策略思路（入场条件、风控偏好等）", "您的策略描述比较简短，能否补充入场条件和风控偏好？"
	case "zh-tw":
		return "請更詳細地描述您的策略思路（入場條件、風控偏好等）", "您的策略描述比較簡短，能否補充入場條件和風控偏好？"
	case "ja":
		return "戦略の考え方をより詳しく説明してください（エントリー条件、リスク管理の好みなど）", "戦略の説明が短いようです。エントリー条件とリスク管理の好みを補足していただけますか？"
	case "vi":
		return "Vui lòng mô tả chi tiết hơn về ý tưởng chiến lược của bạn (điều kiện vào lệnh, sở thích quản lý rủi ro, v.v.)", "Mô tả chiến lược của bạn khá ngắn. Bạn có thể bổ sung điều kiện vào lệnh và sở thích quản lý rủi ro không?"
	default:
		return "Please describe your strategy idea in more detail (entry conditions, risk preferences, etc.)", "Your strategy description is quite brief. Could you add entry conditions and risk preferences?"
	}
}

// LocaleDirective returns a prose language instruction for discuss/explain
// modes. It normalizes the raw locale string internally.
func LocaleDirective(rawLocale string) string {
	lang := NormalizeLocale(rawLocale)
	switch lang {
	case "zh":
		return "\n\n请使用简体中文回复。"
	case "zh-tw":
		return "\n\n請使用繁體中文回覆。"
	case "ja":
		return "\n\n日本語で回答してください。"
	case "vi":
		return "\n\nVui lòng trả lời bằng tiếng Việt."
	case "":
		return ""
	default:
		return "\n\nRespond in English."
	}
}
