import { create } from 'zustand';
import type { Position, TradeLog } from '@/types/trading';
import { toCamelCase } from '../adapters/dataAdapter';

/** While profit stream is fresh, keep per-position floating fields from stream when getPositions (Connect RPC) refetches. */
const STREAM_FLOAT_MERGE_MS = 4500;

export type SetPositionsOptions = { preferRpcProfit?: boolean };

export interface AccountInfo {
  balance: number;
  /** Live credit from profit stream (MT5 AccountSummary / gateway ProfitUpdate.Credit). */
  credit: number;
  profit: number;
  profitPercent?: number;
  equity: number;
  margin: number;
  freeMargin: number;
  marginLevel: number;
}

export interface UserSummary {
  totalBalance: number;
  totalEquity: number;
  totalProfit: number;
  accountCount: number;
  connectedCount: number;
  pnlToday: number;
  pnlWeek: number;
  pnlMonth: number;
  tradesToday: number;
  tradesWeek: number;
  tradesMonth: number;
  winRate: number;
  profitFactor: number;
  maxDrawdownPercent: number;
  maxConsecutiveWins: number;
  maxConsecutiveLosses: number;
  updatedAt?: any;
}

interface TradingState {
  positions: Position[];
  positionsMap: Map<string, Position[]>;
  tradeLogs: TradeLog[];
  accountInfo: AccountInfo;
  accountInfoMap: Map<string, AccountInfo>;
  userSummary: UserSummary;
  accountReceivedData: Set<string>;
  /** Last time we applied account-level profit stream batch (ms since epoch). */
  lastStreamProfitAtByAccount: Map<string, number>;
  currentAccountId: string | null;
  loading: boolean;
  setPositions: (_accountId: string, _positions: Position[], _opts?: SetPositionsOptions) => void;
  touchStreamProfitAt: (_accountId: string) => void;
  addPosition: (_accountId: string, _position: Position) => void;
  updatePosition: (_accountId: string, _ticket: number, _updates: Partial<Position>) => void;
  removePosition: (_accountId: string, _ticket: number) => void;
  setTradeLogs: (_logs: TradeLog[]) => void;
  addTradeLog: (_log: TradeLog) => void;
  setAccountInfo: (_info: Partial<AccountInfo>) => void;
  setAccountInfoById: (_accountId: string, _info: Partial<AccountInfo>) => void;
  setUserSummary: (_summary: Partial<UserSummary>) => void;
  getAccountInfoById: (_accountId: string) => AccountInfo | undefined;
  hasReceivedData: (_accountId: string) => boolean;
  setCurrentAccountId: (_accountId: string | null) => void;
  setLoading: (_loading: boolean) => void;
}

const defaultAccountInfo: AccountInfo = {
  balance: 0,
  credit: 0,
  profit: 0,
  profitPercent: 0,
  equity: 0,
  margin: 0,
  freeMargin: 0,
  marginLevel: 0,
};

const defaultUserSummary: UserSummary = {
  totalBalance: 0,
  totalEquity: 0,
  totalProfit: 0,
  accountCount: 0,
  connectedCount: 0,
  pnlToday: 0,
  pnlWeek: 0,
  pnlMonth: 0,
  tradesToday: 0,
  tradesWeek: 0,
  tradesMonth: 0,
  winRate: 0,
  profitFactor: 0,
  maxDrawdownPercent: 0,
  maxConsecutiveWins: 0,
  maxConsecutiveLosses: 0,
};

export const useTradingStore = create<TradingState>((set, get) => ({
  positions: [],
  positionsMap: new Map(),
  tradeLogs: [],
  accountInfo: { ...defaultAccountInfo },
  accountInfoMap: new Map(),
  userSummary: { ...defaultUserSummary },
  accountReceivedData: new Set<string>(),
  lastStreamProfitAtByAccount: new Map<string, number>(),
  currentAccountId: null,
  loading: false,
  hasReceivedData: (accountId) => get().accountReceivedData.has(accountId),
  touchStreamProfitAt: (accountId) => set((state) => {
    const m = new Map(state.lastStreamProfitAtByAccount);
    m.set(accountId, Date.now());
    return { lastStreamProfitAtByAccount: m };
  }),
  setPositions: (accountId, positions, opts) => set((state) => {
    const newMap = new Map(state.positionsMap);
    const camelPositions = Array.isArray(positions) ? toCamelCase<Position[]>(positions) : [];
    const lastAt = state.lastStreamProfitAtByAccount.get(accountId) ?? 0;
    const streamFresh =
      state.accountReceivedData.has(accountId) && Date.now() - lastAt < STREAM_FLOAT_MERGE_MS;
    const prev = state.positionsMap.get(accountId) || [];
    const prevByTicket = new Map(prev.map((p) => [Number((p as Position).ticket), p]));
    const partialShrink =
      streamFresh && prev.length > 0 && camelPositions.length > 0 && camelPositions.length < prev.length;
    let merged = camelPositions;
    // Guard against transient partial getPositions snapshots while stream is actively updating.
    // Keep rows not present in this RPC frame for a short window to avoid list count flapping.
    if (partialShrink) {
      const incomingByTicket = new Map(
        camelPositions.map((p) => [Number((p as Position).ticket), p] as const),
      );
      merged = prev.map((old) => {
        const t = Number((old as Position).ticket);
        const inc = incomingByTicket.get(t);
        if (!inc) return old;
        return {
          ...inc,
          profit: old.profit,
          currentPrice: old.currentPrice || inc.currentPrice,
          closePrice: old.closePrice ?? inc.closePrice,
        };
      });
      for (const p of camelPositions) {
        const t = Number((p as Position).ticket);
        if (!prevByTicket.has(t)) merged.push(p);
      }
    }
    if (streamFresh && prev.length > 0 && opts?.preferRpcProfit !== true && !partialShrink) {
      merged = camelPositions.map((p) => {
        const t = Number((p as Position).ticket);
        const old = prevByTicket.get(t);
        if (!old) return p;
        return {
          ...p,
          profit: old.profit,
          currentPrice: old.currentPrice || (p as Position).currentPrice,
          closePrice: old.closePrice ?? (p as Position).closePrice,
        };
      });
    }
    newMap.set(accountId, merged);

    let newPositions = state.positions;
    if (state.currentAccountId === accountId) {
      newPositions = merged;
    }

    return {
      positionsMap: newMap,
      positions: newPositions,
    };
  }),
  addPosition: (accountId, position) => set((state) => {
    const newMap = new Map(state.positionsMap);
    const accountPositions = newMap.get(accountId) || [];
    const camelPosition = toCamelCase<Position>(position);
    newMap.set(accountId, [...accountPositions, camelPosition]);
    
    let newPositions = state.positions;
    if (state.currentAccountId === accountId) {
      newPositions = [...accountPositions, camelPosition];
    }
    
    return {
      positionsMap: newMap,
      positions: newPositions
    };
  }),
  updatePosition: (accountId, ticket, updates) => set((state) => {
    const newMap = new Map(state.positionsMap);
    const accountPositions = newMap.get(accountId) || [];
    const updatedPositions = accountPositions.map((p) => (Number((p as any).ticket) === ticket ? { ...p, ...updates } : p));
    newMap.set(accountId, updatedPositions);
    
    let newPositions = state.positions;
    if (state.currentAccountId === accountId) {
      newPositions = updatedPositions;
    }
    
    return {
      positionsMap: newMap,
      positions: newPositions
    };
  }),
  removePosition: (accountId, ticket) => set((state) => {
    const newMap = new Map(state.positionsMap);
    const accountPositions = newMap.get(accountId) || [];
    const filteredPositions = accountPositions.filter((p) => Number((p as any).ticket) !== ticket);
    newMap.set(accountId, filteredPositions);
    
    let newPositions = state.positions;
    if (state.currentAccountId === accountId) {
      newPositions = filteredPositions;
    }
    
    return {
      positionsMap: newMap,
      positions: newPositions
    };
  }),
  setTradeLogs: (logs) => set({ tradeLogs: logs }),
  addTradeLog: (log) => set((state) => ({ tradeLogs: [log, ...state.tradeLogs] })),
  setAccountInfo: (info) => {
    const state = get();
    const newInfo = { ...state.accountInfo, ...info };
    
    if (state.currentAccountId) {
      const newMap = new Map(state.accountInfoMap);
      newMap.set(state.currentAccountId, newInfo);
      set({ 
        accountInfo: newInfo,
        accountInfoMap: newMap,
      });
    } else {
      set({ accountInfo: newInfo });
    }
  },
  setAccountInfoById: (accountId, info) => set((state) => {
    const existingInfo = state.accountInfoMap.get(accountId);
    const newInfo = { ...(existingInfo || { ...defaultAccountInfo }), ...info };
    
    const newMap = new Map(state.accountInfoMap);
    newMap.set(accountId, newInfo);
    
    const newReceivedData = new Set(state.accountReceivedData);
    newReceivedData.add(accountId);
    
    if (state.currentAccountId === accountId) {
      return { 
        accountInfo: newInfo,
        accountInfoMap: newMap,
        accountReceivedData: newReceivedData,
      };
    }
    return { accountInfoMap: newMap, accountReceivedData: newReceivedData };
  }),
  setUserSummary: (summary) => set((state) => ({ userSummary: { ...state.userSummary, ...summary } })),
  getAccountInfoById: (accountId) => {
    return get().accountInfoMap.get(accountId);
  },
  setCurrentAccountId: (accountId) => set((state) => {
    let newPositions: Position[] = [];
    if (accountId) {
      newPositions = state.positionsMap.get(accountId) || [];
    }
    
    if (accountId && state.accountInfoMap.has(accountId)) {
      return { 
        currentAccountId: accountId,
        accountInfo: state.accountInfoMap.get(accountId) || { ...defaultAccountInfo },
        positions: newPositions
      };
    }
    return { 
      currentAccountId: accountId,
      accountInfo: { ...defaultAccountInfo },
      positions: newPositions
    };
  }),
  setLoading: (loading) => set({ loading: loading }),
}));
