import { create } from 'zustand';
import { aiApi, type AIAgentDefinitionView } from '@/client/ai';

/**
 * AI Agent 共享 store。
 *
 * 自 060 起 Agent 不再依附「ai_config_profiles 档」，每个用户拥有唯一一组
 * Agent，因此本 store 只跟 userID 相关；preload() 拉到的就是当前用户的全部
 * 8 个 Agent。`profileId` 字段被彻底移除。
 *
 * - preload(): 幂等加载，已加载或在途时复用 Promise，避免页面切换时重复请求。
 * - refresh(): 强制刷新（保存后调用）。
 * - setAgents(): 写入路径，比如 AgentsPage 保存成功后直接 hot-update。
 */
export interface AgentStoreState {
	agentDefs: AIAgentDefinitionView[];
	loading: boolean;
	loadedAt: number;
	error: string;
	inflight: Promise<void> | null;

	preload: (opts?: { force?: boolean }) => Promise<void>;
	refresh: () => Promise<void>;
	setAgents: (list: AIAgentDefinitionView[]) => void;
	reset: () => void;
}

export const useAgentStore = create<AgentStoreState>((set, get) => ({
	agentDefs: [],
	loading: false,
	loadedAt: 0,
	error: '',
	inflight: null,

	preload: async (opts) => {
		const force = !!opts?.force;
		const state = get();
		if (!force && state.loadedAt > 0) return;
		if (state.inflight) return state.inflight;

		const promise = (async () => {
			set({ loading: true, error: '' });
			try {
				const list = await aiApi.listAgents();
				set({
					agentDefs: list,
					loading: false,
					loadedAt: Date.now(),
					error: '',
					inflight: null,
				});
			} catch (e: any) {
				set({
					loading: false,
					error: String(e?.message || e || 'load failed'),
					inflight: null,
				});
			}
		})();
		set({ inflight: promise });
		return promise;
	},

	refresh: async () => {
		return get().preload({ force: true });
	},

	setAgents: (list) => {
		set({
			agentDefs: list,
			loadedAt: Date.now(),
		});
	},

	reset: () => set({
		agentDefs: [],
		loading: false,
		loadedAt: 0,
		error: '',
		inflight: null,
	}),
}));
