package ai

import "strings"

// LangFromAccept detects the user's language from an Accept-Language header value.
func LangFromAccept(acceptLang string) string {
	if acceptLang == "" {
		return "en"
	}
	primary := acceptLang
	if idx := strings.IndexByte(acceptLang, ','); idx > 0 {
		primary = acceptLang[:idx]
	}
	if idx := strings.IndexByte(primary, ';'); idx > 0 {
		primary = primary[:idx]
	}
	primary = strings.TrimSpace(primary)
	switch {
	case strings.HasPrefix(primary, "zh-tw") || strings.HasPrefix(primary, "zh-TW") || strings.HasPrefix(primary, "zh_HK"):
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

// clarifyLangDirective returns an instruction forcing clarification questions
// (JSON "questions" values) to be written in the user's UI language.
func clarifyLangDirective(lang string) string {
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

// LangPrompt returns a language-tagged system prompt for AI interactions.
// The prompt instructs the LLM to respond in the detected language.
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
