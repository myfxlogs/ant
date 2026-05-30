/**
 * Centralized query key factory for TanStack Query.
 * All query keys MUST be defined here to ensure consistent cache invalidation.
 */
export const queryKeys = {
  accounts: {
    all: ['accounts'] as const,
    list: () => [...queryKeys.accounts.all, 'list'] as const,
    detail: (id: string) => [...queryKeys.accounts.all, 'detail', id] as const,
    financials: (id: string) => [...queryKeys.accounts.all, 'financials', id] as const,
  },
  positions: {
    byAccount: (accountId: string) => ['positions', accountId] as const,
  },
  userSummary: {
    all: ['userSummary'] as const,
  },
  analytics: {
    detail: (accountId: string, period: string) =>
      ['analytics', accountId, period] as const,
    recentTrades: (accountId: string) =>
      ['analytics', 'recentTrades', accountId] as const,
    monthlyPnL: (accountId: string, year: number) =>
      ['analytics', 'monthlyPnL', accountId, year] as const,
    monthlyAnalysis: (accountId: string) =>
      ['analytics', 'monthlyAnalysis', accountId] as const,
  },
  templates: {
    all: ['templates'] as const,
    list: () => [...queryKeys.templates.all, 'list'] as const,
  },
  schedules: {
    all: ['schedules'] as const,
    list: () => [...queryKeys.schedules.all, 'list'] as const,
  },
  ai: {
    conversations: {
      all: ['ai', 'conversations'] as const,
      list: () => [...queryKeys.ai.conversations.all, 'list'] as const,
      detail: (id: string) =>
        [...queryKeys.ai.conversations.all, 'detail', id] as const,
    },
    agents: {
      all: ['ai', 'agents'] as const,
      list: () => [...queryKeys.ai.agents.all, 'list'] as const,
    },
  },
  systemAI: {
    configs: ['systemAI', 'configs'] as const,
  },
};
