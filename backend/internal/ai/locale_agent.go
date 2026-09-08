// Package ai — locale_agent.go
// Single simple prompt per locale. Aligned with Claude Code: act, don't discuss.

package ai

// mqlImportDirective instructs the agent to run analyze_mql coverage analysis
// before judging imported MQL code — the platform's 盲区桥接 (blind-spot
// bridge) flow: compilable parts stay MQL, unsupported parts get translated
// into the Python subset with an explicit note to the user.
func mqlImportDirective(lang string) string {
	switch lang {
	case "zh", "zh-tw":
		return "\n\n## 匯入 MQL 程式碼\n使用者貼上/匯入 MQL4/MQL5 程式碼時：先呼叫 analyze_mql 工具做覆蓋度分析（編譯狀態/覆蓋度/盲區清單）。可編譯的部分保留原邏輯；不支援的盲區按平台「盲區橋接」機制翻譯為 Python 子集實作，並向使用者說明哪些部分是翻譯實作。禁止未經 analyze_mql 就斷言平台是否支援。"
	case "ja":
		return "\n\n## MQLコードの取り込み\nユーザーがMQL4/MQL5コードを貼り付け・インポートした場合は、まず analyze_mql ツールでカバレッジ分析を行うこと（コンパイル状態/カバレッジ/ブラインドスポット一覧）。コンパイル可能な部分は元のロジックを保持し、未対応部分はプラットフォームの「ブラインドスポットブリッジ」に従い Python サブセットへ翻訳して実装し、どの部分が翻訳かをユーザーに説明すること。analyze_mql を実行せずに対応可否を断定しないこと。"
	case "vi":
		return "\n\n## Nhập mã MQL\nKhi người dùng dán/nhập mã MQL4/MQL5: hãy gọi công cụ analyze_mql để phân tích độ phủ trước (trạng thái biên dịch/độ phủ/danh sách điểm mù). Phần biên dịch được giữ nguyên logic; phần không hỗ trợ được dịch sang tập con Python theo cơ chế \"blind-spot bridge\" và giải thích cho người dùng phần nào là bản dịch. Không kết luận khả năng hỗ trợ khi chưa gọi analyze_mql."
	default:
		return "\n\n## Imported MQL code\nWhen the user pastes or imports MQL4/MQL5 code: always run the analyze_mql tool first for a coverage analysis (compile status/coverage/blind spots). Keep compilable parts as-is; translate unsupported blind spots into the Python subset per the platform's blind-spot bridge, and tell the user which parts were translated. Never assert platform support without running analyze_mql."
	}
}

func PythonAgentPrompt(lang string) string {
	base := pythonAgentBase(lang)
	if base == "" {
		return ""
	}
	return base + mqlImportDirective(lang)
}

func pythonAgentBase(lang string) string {
	switch lang {
	case "zh":
		return agentSystemPrompt_ZH
	case "zh-tw":
		return agentSystemPrompt_ZHTW
	case "ja":
		return agentSystemPrompt_JA
	case "vi":
		return agentSystemPrompt_VI
	default:
		return agentSystemPrompt
	}
}

func PythonAgentDiscipline(_ string) string     { return "" }
func PythonGeneratorPrompt(lang string) string  { return PythonAgentPrompt(lang) }
func PythonGeneratorDiscipline(_ string) string { return "" }
