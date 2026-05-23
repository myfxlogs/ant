import { create } from 'zustand';
import { message } from 'antd';
import { aiApi } from '@/client/ai';
import { toFriendlyAIChatError } from '@/client/ai';
import type { ConversationSummary } from '@/client/ai';
import i18n from '@/i18n';

const AI_PREFS_KEY = 'ai_user_preferences_v1';

function loadUserPrefs(): string {
	try {
		return localStorage.getItem(AI_PREFS_KEY) || '';
	} catch {
		return '';
	}
}

function saveUserPrefs(next: string) {
	try {
		localStorage.setItem(AI_PREFS_KEY, next);
	} catch {
	}
}

function strategyValidationRulesText(): string {
	return [
		i18n.t('ai.store.strategyRules.title'),
		'',
		i18n.t('ai.store.strategyRules.rules.noImport'),
		i18n.t('ai.store.strategyRules.rules.noGlobal'),
		i18n.t('ai.store.strategyRules.rules.noDunderAccess'),
		i18n.t('ai.store.strategyRules.rules.noDunderName'),
		i18n.t('ai.store.strategyRules.rules.noDangerousCalls'),
		i18n.t('ai.store.strategyRules.rules.runSignature'),
		i18n.t('ai.store.strategyRules.rules.mustDefineEntry'),
		'',
		i18n.t('ai.store.strategyRules.allowedGlobals'),
	].join('\n');
}

function buildChatContext(): string {
	const prefs = loadUserPrefs().trim();
	const parts: string[] = [];
	parts.push(`Locale: ${i18n.language || ''}`.trim());
	parts.push('');
	parts.push(strategyValidationRulesText());
	if (prefs) {
		parts.push('');
		parts.push(i18n.t('ai.store.context.userPrefsTitle'));
		parts.push(prefs);
	}
	parts.push('');
	parts.push(i18n.t('ai.store.context.outputTitle'));
	parts.push(i18n.t('ai.store.context.outputRules.wrapPython'));
	parts.push(i18n.t('ai.store.context.outputRules.validateFirst'));
	parts.push(i18n.t('ai.store.context.outputRules.noImport'));
	return parts.join('\n');
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  isLoading?: boolean;
}

interface Conversation {
	id: string;
	title: string;
	createdAt: Date;
	updatedAt: Date;
	messageCount: number;
}

interface AIState {
	conversations: Conversation[];
	activeConversationId: string;
	messages: Message[];
	loading: boolean;
	sending: boolean;
	conversationsLoaded: boolean;

	loadConversations: () => Promise<void>;
	sendMessage: (_content: string, _accountId?: string) => Promise<void>;
	sendMessageAndGetResponse: (_content: string, _accountId?: string) => Promise<string>;
	clearMessages: () => void;
	newConversation: () => Promise<void>;
	selectConversation: (_id: string) => Promise<void>;
	deleteConversation: (_id: string) => Promise<void>;
	getReports: (_accountId?: string) => Promise<any[]>;
	generateReport: (_accountId: string, _reportType: string, _period: string) => Promise<any>;
	setLoading: (loading: boolean) => void;
}

function toConv(c: ConversationSummary): Conversation {
	return {
		id: c.id,
		title: c.title,
		createdAt: new Date(c.createdAt),
		updatedAt: new Date(c.updatedAt),
		messageCount: c.messageCount,
	};
}

export const useAIStore = create<AIState>((set, get) => ({
	conversations: [],
	activeConversationId: '',
	messages: [],
	loading: false,
	sending: false,
	conversationsLoaded: false,

	loadConversations: async () => {
		try {
			const list = await aiApi.listConversations();
			const convs = list.map(toConv);
			set({ conversations: convs, conversationsLoaded: true });
		} catch {
			set({ conversationsLoaded: true });
		}
	},

	sendMessageAndGetResponse: async (content, accountId) => {
		if (get().sending) {
			return '';
		}
		let { activeConversationId } = get();
		let convReady = !!activeConversationId;

		const rememberPrefix = i18n.t('ai.store.prefs.rememberPrefix');
		if (content.trim().startsWith(rememberPrefix)) {
			const next = content.trim().slice(rememberPrefix.length).trim();
			saveUserPrefs(next);
			message.success(i18n.t('ai.store.prefs.rememberedToast'));
			return i18n.t('ai.store.prefs.savedReply');
		}

		if (!activeConversationId) {
			try {
				const created = await aiApi.createConversation(i18n.t('ai.store.conversations.newConversationTitle'));
				const conv = toConv(created);
				const cur = get().conversations;
				activeConversationId = conv.id;
				convReady = true;
				set({
					conversations: [conv, ...cur],
					activeConversationId: conv.id,
					messages: [],
				});
			} catch {
				// Fallback: still allow chatting without conversation persistence.
				convReady = false;
			}
		}

		const { messages } = get();
		const userMessage: Message = {
			id: `user-${Date.now()}`,
			role: 'user',
			content,
			timestamp: new Date(),
		};
		const aiMessageId = `ai-${Date.now()}`;
		const aiMessage: Message = {
			id: aiMessageId,
			role: 'assistant',
			content: '',
			timestamp: new Date(),
			isLoading: true,
		};

		set({ messages: [...messages, userMessage, aiMessage], sending: true });

		try {
			let response: { message: string; suggestions: string[] };
			try {
				let acc = '';
				response = await aiApi.chatStreaming(
					{
						message: content,
						context: buildChatContext(),
						accountId,
						conversationId: convReady ? activeConversationId : '',
					},
					(delta) => {
						acc += delta;
						set((state) => ({
							messages: state.messages.map((m) =>
								m.id === aiMessageId ? { ...m, content: acc, isLoading: true } : m,
							),
						}));
					},
				);
			} catch {
				response = await aiApi.chat({
					message: content,
					context: buildChatContext(),
					accountId,
					conversationId: convReady ? activeConversationId : '',
				});
			}

			const curMsgs = get().messages.map((m) =>
				m.id === aiMessageId ? { ...m, content: response.message, isLoading: false } : m,
			);
			set({ messages: curMsgs });

			if (convReady) {
				const list = await aiApi.listConversations();
				set({ conversations: list.map(toConv) });
			}
			return response.message || '';
		} catch (e: unknown) {
			const curMsgs = get().messages.map((m) =>
				m.id === aiMessageId ? { ...m, content: i18n.t('ai.store.messages.sendFailedInline'), isLoading: false } : m,
			);
			set({ messages: curMsgs });
			message.error(toFriendlyAIChatError(e) || i18n.t('ai.store.messages.sendFailedToast'));
			return '';
		} finally {
			set({ sending: false });
		}
	},

	newConversation: async () => {
		try {
			const created = await aiApi.createConversation(i18n.t('ai.store.conversations.newConversationTitle'));
			const conv = toConv(created);
			const cur = get().conversations;
			set({
				conversations: [conv, ...cur],
				activeConversationId: conv.id,
				messages: [],
			});
		} catch {
			message.error(i18n.t('ai.store.messages.createConversationFailed'));
		}
	},

	selectConversation: async (id: string) => {
		set({ activeConversationId: id, messages: [], loading: true });
		try {
			const detail = await aiApi.getConversation(id);
			const msgs: Message[] = detail.messages.map((m) => ({
				id: m.id,
				role: m.role as 'user' | 'assistant',
				content: m.content,
				timestamp: new Date(m.createdAt),
			}));
			set({ messages: msgs, loading: false });
		} catch {
			set({ loading: false });
			message.error(i18n.t('ai.store.messages.loadConversationFailed'));
		}
	},

	deleteConversation: async (id: string) => {
		try {
			await aiApi.deleteConversation(id);
			const cur = get().conversations.filter((c) => c.id !== id);
			const activeId = get().activeConversationId;
			if (activeId === id) {
				set({
					conversations: cur,
					activeConversationId: cur[0]?.id || '',
					messages: [],
				});
				if (cur[0]) {
					get().selectConversation(cur[0].id);
				}
			} else {
				set({ conversations: cur });
			}
		} catch {
			message.error(i18n.t('ai.store.messages.deleteConversationFailed'));
		}
	},

	sendMessage: async (content, accountId) => {
		if (get().sending) {
			return;
		}
		let { activeConversationId } = get();
		let convReady = !!activeConversationId;

		const rememberPrefix = i18n.t('ai.store.prefs.rememberPrefix');
		if (content.trim().startsWith(rememberPrefix)) {
			const next = content.trim().slice(rememberPrefix.length).trim();
			saveUserPrefs(next);
			message.success(i18n.t('ai.store.prefs.rememberedToast'));
			return;
		}

		if (!activeConversationId) {
			try {
				const created = await aiApi.createConversation(i18n.t('ai.store.conversations.newConversationTitle'));
				const conv = toConv(created);
				const cur = get().conversations;
				activeConversationId = conv.id;
				convReady = true;
				set({
					conversations: [conv, ...cur],
					activeConversationId: conv.id,
					messages: [],
				});
			} catch {
				// Fallback: still allow chatting without conversation persistence.
				convReady = false;
			}
		}

		const { messages } = get();
		const userMessage: Message = {
			id: `user-${Date.now()}`,
			role: 'user',
			content,
			timestamp: new Date(),
		};
		const aiMessageId = `ai-${Date.now()}`;
		const aiMessage: Message = {
			id: aiMessageId,
			role: 'assistant',
			content: '',
			timestamp: new Date(),
			isLoading: true,
		};

		set({ messages: [...messages, userMessage, aiMessage], sending: true });

		try {
			let response: { message: string; suggestions: string[] };
			try {
				let acc = '';
				response = await aiApi.chatStreaming(
					{
						message: content,
						context: buildChatContext(),
						accountId,
						conversationId: convReady ? activeConversationId : '',
					},
					(delta) => {
						acc += delta;
						set((state) => ({
							messages: state.messages.map((m) =>
								m.id === aiMessageId ? { ...m, content: acc, isLoading: true } : m,
							),
						}));
					},
				);
			} catch {
				response = await aiApi.chat({
					message: content,
					context: buildChatContext(),
					accountId,
					conversationId: convReady ? activeConversationId : '',
				});
			}

			const cur = get().messages.map((m) =>
				m.id === aiMessageId ? { ...m, content: response.message, isLoading: false } : m,
			);
			set({ messages: cur });

			// Refresh conversation list to update title / message count
			if (convReady) {
				const list = await aiApi.listConversations();
				set({ conversations: list.map(toConv) });
			}
		} catch (e: unknown) {
			const cur = get().messages.map((m) =>
				m.id === aiMessageId ? { ...m, content: i18n.t('ai.store.messages.sendFailedInline'), isLoading: false } : m,
			);
			set({ messages: cur });
			message.error(toFriendlyAIChatError(e) || i18n.t('ai.store.messages.sendFailedToast'));
		} finally {
			set({ sending: false });
		}
	},

	clearMessages: () => {
		set({ messages: [] });
		message.success(i18n.t('ai.store.messages.clearedLocalOnly'));
	},

	getReports: async (accountId) => {
		try {
			const reports = await aiApi.getReports({ accountId });
			return reports;
		} catch {
			message.error(i18n.t('ai.store.messages.getReportsFailed'));
			return [];
		}
	},

	generateReport: async (accountId, reportType, period) => {
		try {
			const report = await aiApi.generateReport({
				accountId,
				reportType,
				period,
			});
			message.success(i18n.t('ai.store.messages.generateReportSuccess'));
			return report;
		} catch {
			message.error(i18n.t('ai.store.messages.generateReportFailed'));
			return null;
		}
	},

	setLoading: (loading) => set({ loading }),
}));
