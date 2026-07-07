// Package ai — locale_agent.go
// Single simple prompt per locale. Aligned with Claude Code: act, don't discuss.

package ai

func PythonAgentPrompt(lang string) string {
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

func PythonAgentDiscipline(_ string) string   { return "" }
func PythonGeneratorPrompt(lang string) string { return PythonAgentPrompt(lang) }
func PythonGeneratorDiscipline(_ string) string { return "" }
