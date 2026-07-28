
// Shared mapper that converts a raw provider/gateway error message into a
// localized, human-friendly hint. Covers the most common failure modes we see
// across OpenAI-family, Anthropic, and Chinese cloud providers (DeepSeek,
// Zhipu, Qwen, Doubao, Moonshot, ...).
//
// Intentionally pattern-matches on substrings rather than HTTP status alone
// because most providers wrap their JSON inside a transport error and we only
// see the pre-formatted body string by the time it gets here.

export { unwrapProviderMessage, pickErrorText } from './errorPatterns';
export { toFriendlyAIError, toFriendlyAIChatError } from './errorPatterns';
