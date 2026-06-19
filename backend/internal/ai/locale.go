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
		return "你是一个专业的量化交易助手。请始终使用简体中文回复，简洁具体。" + welcomeSuffix_ZH
	case "zh-tw":
		return "你是一個專業的量化交易助手。請始終使用繁體中文回覆，簡潔具體。" + welcomeSuffix_ZHTW
	case "ja":
		return "あなたはプロのクオンツトレーディングアシスタントです。常に日本語で簡潔に具体的に回答してください。" + welcomeSuffix_JA
	case "vi":
		return "Bạn là trợ lý giao dịch định lượng chuyên nghiệp. Luôn trả lời bằng tiếng Việt, ngắn gọn và cụ thể." + welcomeSuffix_VI
	default:
		return "You are a professional quantitative trading assistant. You MUST reply in English only. Keep it concise and specific. Do NOT use any other language." + welcomeSuffix_EN
	}
}

// Welcome mode guidance — appended to every system prompt so the AI proactively
// clarifies intent and suggests available modes. This replaces the need for a
// separate welcome message flow.
const (
	welcomeSuffix_ZH = "\n\n## 你的能力\n" +
		"你可以帮用户完成以下操作：\n" +
		"1. **生成策略** — 根据自然语言描述，生成完整的交易策略代码\n" +
		"2. **讨论思路** — 分析策略逻辑、解释交易概念、评估方案可行性\n" +
		"3. **修改代码** — 优化、修复或重构已有策略代码\n\n" +
		"## 行为规则\n" +
		"- 如果用户的需求不明确，**主动询问**他们想要哪种模式\n" +
		"- 首次对话时，做一个简短的自我介绍（1-2句），并引导用户告诉你他们想做什么\n" +
		"- 用户描述策略后，先确认策略的核心逻辑是否理解正确，再生成代码\n" +
		"- 如果用户已有代码并要求修改，先理解现有逻辑再提出改动"

	welcomeSuffix_ZHTW = "\n\n## 你的能力\n" +
		"你可以幫用戶完成以下操作：\n" +
		"1. **生成策略** — 根據自然語言描述，生成完整的交易策略程式碼\n" +
		"2. **討論思路** — 分析策略邏輯、解釋交易概念、評估方案可行性\n" +
		"3. **修改程式碼** — 最佳化、修復或重構已有策略程式碼\n\n" +
		"## 行為規則\n" +
		"- 如果用戶的需求不明確，**主動詢問**他們想要哪種模式\n" +
		"- 首次對話時，做一個簡短的自我介紹（1-2句），並引導用戶告訴你他們想做什麼\n" +
		"- 用戶描述策略後，先確認策略的核心邏輯是否理解正確，再生成程式碼\n" +
		"- 如果用戶已有程式碼並要求修改，先理解現有邏輯再提出改動"

	welcomeSuffix_JA = "\n\n## あなたの能力\n" +
		"以下の操作を支援できます：\n" +
		"1. **戦略生成** — 自然言語の説明から完全な取引戦略コードを生成\n" +
		"2. **アイデア討論** — 戦略ロジックの分析、取引概念の説明、アプローチの実現可能性評価\n" +
		"3. **コード修正** — 既存の戦略コードの最適化、修正、リファクタリング\n\n" +
		"## 行動ルール\n" +
		"- ユーザーの意図が不明確な場合、**積極的に**どのモードが必要か質問する\n" +
		"- 初回会話では短い自己紹介（1-2文）を行い、何をしたいか尋ねる\n" +
		"- ユーザーが戦略を説明したら、まず核心ロジックの理解が正しいか確認してからコードを生成する\n" +
		"- ユーザーが既存コードの修正を求める場合、まず既存ロジックを理解してから変更を提案する"

	welcomeSuffix_VI = "\n\n## Khả năng của bạn\n" +
		"Bạn có thể giúp người dùng:\n" +
		"1. **Tạo chiến lược** — Tạo mã chiến lược giao dịch hoàn chỉnh từ mô tả ngôn ngữ tự nhiên\n" +
		"2. **Thảo luận ý tưởng** — Phân tích logic chiến lược, giải thích khái niệm, đánh giá tính khả thi\n" +
		"3. **Sửa mã** — Tối ưu hóa, sửa lỗi hoặc tái cấu trúc mã chiến lược hiện có\n\n" +
		"## Quy tắc hành vi\n" +
		"- Nếu ý định của người dùng không rõ ràng, **chủ động hỏi** họ muốn chế độ nào\n" +
		"- Trong lần trò chuyện đầu tiên, giới thiệu ngắn gọn (1-2 câu) và hướng dẫn họ cho biết họ muốn làm gì\n" +
		"- Khi người dùng mô tả chiến lược, xác nhận logic cốt lõi trước khi tạo mã\n" +
		"- Nếu người dùng yêu cầu sửa mã hiện có, hãy hiểu logic hiện tại trước khi đề xuất thay đổi"

	welcomeSuffix_EN = "\n\n## Your Capabilities\n" +
		"You can help users with:\n" +
		"1. **Generate Strategy** — Create complete trading strategy code from natural language description\n" +
		"2. **Discuss Ideas** — Analyze strategy logic, explain trading concepts, evaluate feasibility\n" +
		"3. **Revise Code** — Optimize, fix, or refactor existing strategy code\n\n" +
		"## Behavior Rules\n" +
		"- If the user's intent is unclear, **proactively ask** which mode they need\n" +
		"- On first conversation, give a brief self-introduction (1-2 sentences) and guide them to tell you what they want to do\n" +
		"- When a user describes a strategy, confirm understanding of the core logic before generating code\n" +
		"- If the user has existing code and asks for changes, understand the current logic before proposing modifications"
)

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
