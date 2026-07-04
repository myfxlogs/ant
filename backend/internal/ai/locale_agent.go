// Package ai — locale_agent.go
// i18n prompts for the Python strategy Agent (chat) and Generator.
// 5 languages: en, zh, zh-tw, ja, vi — all equally detailed.
// PythonSubsetRules (python_rules.go) is language-neutral and kept in English.

package ai

// ── PythonAgentPrompt: Chat system prompt ──

func PythonAgentPrompt(lang string) string {
	switch lang {
	case "zh":
		return pythonAgentPrompt_ZH
	case "zh-tw":
		return pythonAgentPrompt_ZHTW
	case "ja":
		return pythonAgentPrompt_JA
	case "vi":
		return pythonAgentPrompt_VI
	default:
		return pythonAgentPrompt_EN
	}
}

// ── PythonAgentDiscipline: Thinking discipline + checklist + error memory ──

func PythonAgentDiscipline(lang string) string {
	switch lang {
	case "zh":
		return pythonAgentDiscipline_ZH
	case "zh-tw":
		return pythonAgentDiscipline_ZHTW
	case "ja":
		return pythonAgentDiscipline_JA
	case "vi":
		return pythonAgentDiscipline_VI
	default:
		return pythonAgentDiscipline_EN
	}
}

// ── PythonGeneratorPrompt: Generator system prompt (includes discipline) ──

func PythonGeneratorPrompt(lang string) string {
	switch lang {
	case "zh":
		return pythonGeneratorPrompt_ZH
	case "zh-tw":
		return pythonGeneratorPrompt_ZHTW
	case "ja":
		return pythonGeneratorPrompt_JA
	case "vi":
		return pythonGeneratorPrompt_VI
	default:
		return pythonGeneratorPrompt_EN
	}
}
