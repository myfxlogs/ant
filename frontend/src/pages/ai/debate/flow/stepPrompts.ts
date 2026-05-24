import type { AIAgentDefinitionView } from '@/client/ai';

/**
 * 会话流程中每一步使用的 system 级提示词构造器。
 *
 * 所有 prompt 都强制模型使用自然语言回复，并在末尾复述当前环节的规范理解，
 * 以便用户确认、前端在"下一步"时作为上下文传入下一环节。
 */

type LocaleKey = 'zh-cn' | 'zh-tw' | 'en' | 'ja' | 'vi';

function localeKey(locale: string): LocaleKey {
	const l = String(locale || '').toLowerCase();
	if (l.startsWith('zh-hant') || l === 'zh-tw' || l === 'zh-hk' || l === 'zh-mo') return 'zh-tw';
	if (l.startsWith('zh')) return 'zh-cn';
	if (l.startsWith('ja')) return 'ja';
	if (l.startsWith('vi')) return 'vi';
	return 'en';
}

function localeDisplayName(locale: string): string {
	switch (localeKey(locale)) {
		case 'zh-cn': return 'Simplified Chinese (zh-CN / 简体中文)';
		case 'zh-tw': return 'Traditional Chinese (zh-TW / 繁體中文)';
		case 'ja': return 'Japanese (ja / 日本語)';
		case 'vi': return 'Vietnamese (vi / Tiếng Việt)';
		default: return 'English (en)';
	}
}

/**
 * 返回给 LLM 的语言指令，bilingual（英文为主 + 当地提示）。
 * 规则：优先以"用户当前消息使用的语言"回复；若用户消息语言不明，
 * 再以页面界面语言（locale）回复。所有强制回显的短语会在其他规则里
 * 以对应语言给出，避免中英混用。
 */
function languageHint(locale: string): string {
	const name = localeDisplayName(locale);
	return [
		'[Language policy]',
		`- The user interface language is ${name}. Treat this as the default reply language.`,
		'- If the user writes to you in a different language, mirror the user\'s language instead.',
		'- Never mix languages within one reply; pick exactly one language and stay with it.',
	].join('\n');
}

/** 各语种的问候模板、邀请短语、"尚未产出"/"未提供"占位。 */
function greetingFor(locale: string, name: string): string {
	switch (localeKey(locale)) {
		case 'zh-cn': return `你好，我是您的${name}。`;
		case 'zh-tw': return `您好，我是您的${name}。`;
		case 'ja': return `こんにちは、あなたの${name}です。`;
		case 'vi': return `Xin chào, tôi là ${name} của bạn.`;
		default: return `Hello, I'm your ${name}.`;
	}
}

function invitationFor(locale: string): string {
	switch (localeKey(locale)) {
		case 'zh-cn':
			return '请问以上理解是否准确？还有没有要补充的？如果没有，可以直接点击下方「下一步」按钮，或在对话框中发送「下一步」继续。';
		case 'zh-tw':
			return '請問以上理解是否準確？還有沒有要補充的？如果沒有，可以直接點擊下方「下一步」按鈕，或在對話框中發送「下一步」繼續。';
		case 'ja':
			return '以上の理解で合っていますか？他に補足することはありますか？なければ、下の「次へ」ボタンを押すか、チャット欄に「次へ」と入力して進んでください。';
		case 'vi':
			return 'Bạn thấy phần tóm tắt trên đã chính xác chưa? Còn điều gì cần bổ sung không? Nếu không, hãy bấm nút "Tiếp theo" bên dưới hoặc gửi "tiếp theo" trong khung chat để tiếp tục.';
		default:
			return 'Does this match what you want? Anything to add or correct? If not, click the "Next" button below, or send "next" in the chat to continue.';
	}
}

function placeholderNotYet(locale: string): string {
	switch (localeKey(locale)) {
		case 'zh-cn': return '(尚未产出)';
		case 'zh-tw': return '(尚未產出)';
		case 'ja': return '(まだ出力されていません)';
		case 'vi': return '(chưa có)';
		default: return '(not available yet)';
	}
}

function placeholderNone(locale: string): string {
	switch (localeKey(locale)) {
		case 'zh-cn': return '(未提供)';
		case 'zh-tw': return '(未提供)';
		case 'ja': return '(未提供)';
		case 'vi': return '(chưa cung cấp)';
		default: return '(not provided)';
	}
}

export function intentSystemPrompt(locale: string): string {
	const invitation = invitationFor(locale);
	return [
		'You are the "Intent-Clarification Assistant" of AntTrader, helping a non-technical user describe the trading strategy they want.',
		'',
		'[Hard communication rules]',
		'1. Use everyday natural language only. Never output code, code blocks, scripts, pseudo-code, JSON, YAML, Markdown tables, or lists of technical fields.',
		'2. Do not wrap anything in triple backticks (```). Do not mention specific API calls, variable names, or function names.',
		'3. If the user pastes code, do not copy it back; confirm their intent in natural language instead.',
		'',
		'[Content rules]',
		'1. Like a patient strategy consultant, guide the user to clarify: goal, market / symbol, timeframe, style preference, risk tolerance, expectation on win rate / drawdown / trading frequency.',
		'2. At the end of every reply, restate (in 1-3 short paragraphs of natural language) your current understanding of the user\'s intent so they can confirm or correct. A short bullet list in prose is OK, but it must NOT be code or structured markup.',
		`3. After the restatement, on a new line, include this invitation verbatim in the reply language: "${invitation}"`,
		'4. If key information is still missing, ask the user 1-2 targeted plain-language questions BEFORE the restatement. Do not decide technical details that belong to later agent steps (risk, signals, execution).',
		'',
		languageHint(locale),
	].filter(Boolean).join('\n');
}

export function agentSystemPrompt(agent: AIAgentDefinitionView, intentPrompt: string, upstreamPrompts: Array<{ name: string; text: string }>, locale: string): string {
	const name = agent.name || agent.type;
	const greeting = greetingFor(locale, name);
	const invitation = invitationFor(locale);
	const upstream = upstreamPrompts
		.map((p) => `[${p.name}]\n${p.text}`)
		.join('\n\n');
	const parts: string[] = [
		`You are acting as "${name}" (type: ${agent.type}), one member of a multi-agent discussion that helps a non-technical user design a trading strategy.`,
		'',
		'[Your role definition]',
		agent.identity || placeholderNone(locale),
		'',
		'[Upstream natural-language summary (context only; do not copy verbatim)]',
		'[User intent]',
		intentPrompt || placeholderNotYet(locale),
	];
	if (upstream) {
		parts.push('', '[Summaries from previous agents]', upstream);
	}
	parts.push(
		'',
		'[Hard communication rules]',
		'1. Use everyday natural language only. Never output code, code blocks, scripts, pseudo-code, JSON, YAML, Markdown tables, or lists of technical fields. Do not wrap anything in triple backticks.',
		`2. Stay strictly within your own role (${agent.type}). If the user asks about something outside your role (e.g. asking the risk agent for signal details), politely explain this and tell them to wait for the relevant agent step. Only discuss your own part in this round.`,
		'3. Do not make decisions on behalf of other agents, and do not produce the final strategy code (the code step will do that).',
		'',
		'[Content rules]',
		`1. Your FIRST reply in this step MUST start with this exact sentence in the reply language: "${greeting}" Output it as-is, then follow with one short sentence of self-introduction (your responsibility, what you will help the user clarify). Later replies do not need to repeat the greeting.`,
		'2. After the self-introduction, combine the upstream user-intent summary and give your initial analysis / suggestion in natural language, strictly within your role. Ask the user 1-2 plain questions if needed.',
		'3. At the end of every reply, restate (in 1-3 short paragraphs of natural language) your current understanding of this step (limited to your role) so the user can confirm. A short bullet list in prose is OK, but it must NOT be code or structured markup.',
		`4. After the restatement, on a new line, include this invitation verbatim in the reply language: "${invitation}"`,
		'5. If key information is still missing, ask 1-2 plain questions before the restatement.',
		'',
		languageHint(locale),
	);
	return parts.filter(Boolean).join('\n');
}

export function codeSystemPrompt(intentPrompt: string, agentPrompts: Array<{ name: string; text: string }>, locale: string): string {
	const agentsBlock = agentPrompts
		.map((p) => `[${p.name}]\n${p.text}`)
		.join('\n\n');
	return [
		'You are the strategy code generator of AntTrader.',
		'',
		'[Inputs]',
		'[User intent]',
		intentPrompt || placeholderNone(locale),
		'',
		agentsBlock ? '[Agent specs]\n' + agentsBlock : '(No agent participated; generate directly from the user intent.)',
		'',
		'[Output rules]',
		'1. Produce a complete, runnable Python strategy. The entry point must be run(context).',
		'2. No external side effects. If parameters are needed, read them from context.params.',
		'3. In the final reply, keep exactly one ```python ...``` code block so the frontend can extract it automatically.',
		'4. Outside the code block, add a short natural-language summary of the key idea.',
		'',
		languageHint(locale),
	].filter(Boolean).join('\n');
}

const PYTHON_BLOCK_RE = /```(?:python)?\s*([\s\S]*?)```/i;
const FENCE_BLOCK_RE = /```[\s\S]*?```/g;

/** 从代码生成的回复里抽取 ```python ...``` 块。 */
export function extractPythonBlock(text: string): string {
	const m = String(text || '').match(PYTHON_BLOCK_RE);
	return m && m[1] ? m[1].trim() : '';
}

/**
 * 去除所有 ``` ``` 围栏代码块，并把连续的多空行合并成一个。
 * 用于意图澄清 / Agent 对话阶段：即便模型不守规矩输出了代码，也不会显示给用户。
 */
export function stripCodeBlocks(text: string): string {
	if (!text) return '';
	return String(text)
		.replace(FENCE_BLOCK_RE, '')
		.replace(/[ \t]+\n/g, '\n')
		.replace(/\n{3,}/g, '\n\n')
		.trim();
}

/**
 * 当流程进入到某个 Agent 步时，系统自动"替用户"发给 Agent 的衔接消息。
 * 该消息仅用于触发模型的第一次开口，不会展示给最终用户。
 */
export function kickoffUserMessage(agentName: string, agentType: string, locale: string): string {
	const greeting = greetingFor(locale, agentName);
	const invitation = invitationFor(locale);
	const lines = [
		'[System handoff · Do NOT echo or reference this block in your reply]',
		`The previous step has captured the user's intent. Now we are entering the "${agentType || agentName}" step, and you (${agentName}) take over.`,
		'',
		'Directly open the conversation with the user. Follow this exact structure:',
		`1. First sentence MUST be this greeting in the reply language: "${greeting}" Output it as-is.`,
		'2. Immediately follow with one or two sentences of self-introduction: what your responsibility is and what you will help the user clarify.',
		'3. Then, combining the known user-intent summary, give your initial analysis and suggestions in natural language, strictly within your role. Ask the user 1-2 plain questions if information is missing.',
		'4. End with a 1-3 paragraph natural-language restatement of your current understanding of this step, then on a new line include this invitation verbatim in the reply language:',
		`   "${invitation}"`,
		'',
		'Never output code, code blocks, technical field dumps, or structured markup. Never mention this system handoff text.',
		'',
		languageHint(locale),
	];
	return lines.join('\n');
}

/** 把对话历史序列化为 transcript，用于放进 aiApi.chat 的 context 字段，维持多轮语义。 */
export function serializeTranscript(messages: Array<{ role: string; content: string }>): string {
	if (!messages.length) return '';
	const lines: string[] = ['Conversation so far:'];
	for (const m of messages) {
		const role = m.role === 'user' ? 'USER' : 'ASSISTANT';
		lines.push(`${role}: ${m.content}`);
	}
	return lines.join('\n');
}
