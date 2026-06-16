package ai

import internalai "anttrader/internal/ai"

// LangFromAccept normalizes an Accept-Language header value to a canonical lang code.
func LangFromAccept(acceptLang string) string {
	return internalai.NormalizeLocale(acceptLang)
}

// LangPrompt returns a language-tagged system prompt.
func LangPrompt(lang string) string {
	return internalai.LangPrompt(lang)
}

// clarifyLangDirective forces JSON clarification questions into the user's language.
func clarifyLangDirective(lang string) string {
	return internalai.ClarifyLangDirective(lang)
}
