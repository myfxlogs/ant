import { create } from 'zustand';
import type { Position, TradeLog } from '@/types/trading';
import { toCamelCase } from '../adapters/dataAdapter';

type SetPositionsOptions = { preferRpcProfit?: boolean };

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

interface TradingState {
  positions: Position[];
  positionsMap: Map<string, Position[]>;
  tradeLogs: TradeLog[];
  accountInfo: AccountInfo;
  accountInfoMap: Map<string, AccountInfo>;
  accountReceivedData: Set<string>;
  /** Last time we applied account-level profit stream batch (ms since epoch). */
  lastStreamProfitAtByAccount: Map<string, number>;
  currentAccountId: string | null;
  loading: boolean;
  setPositions: (_accountId: string, _positions: readonly Record<string, unknown>[], _opts?: SetPositionsOptions) => void;
  touchStreamProfitAt: (_accountId: string) => void;
  addPosition: (_accountId: string, _position: Position) => void;
  updatePosition: (_accountId: string, _ticket: number, _updates: Partial<Position>) => void;
  removePosition: (_accountId: string, _ticket: number) => void;
  setTradeLogs: (_logs: TradeLog[]) => void;
  addTradeLog: (_log: TradeLog) => void;
  setAccountInfo: (_info: Partial<AccountInfo>) => void;
  setAccountInfoById: (_accountId: string, _info: Partial<AccountInfo>) => void;
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

export const useTradingStore = create<TradingState>((set, get) => createTradingState(set, get));

function createTradingState(
  set: (partial: TradingState | Partial<TradingState> | ((state: TradingState) => TradingState | Partial<TradingState>)) => void,
  get: () => TradingState,
): TradingState {
  return {
  positions: [],
  positionsMap: new Map(),
  tradeLogs: [],
  accountInfo: { ...defaultAccountInfo },
  accountInfoMap: new Map(),
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
  setPositions: (accountId, positions, _opts) => set((state) => {
    const newMap = new Map(state.positionsMap);
    const camelPositions = Array.isArray(positions) ? toCamelCase<Position[]>(positions) : [];
    newMap.set(accountId, camelPositions);

    let newPositions = state.positions;
    if (state.currentAccountId === accountId) {
      newPositions = camelPositions;
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
    // Dedup: skip if a position with the same ticket already exists
    // (e.g. initial stream snapshot may arrive after fetchPositions RPC).
    const ticket = camelPosition.ticket;
    if (accountPositions.some((p) => p.ticket === ticket)) {
      return {};
    }
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
    const updatedPositions = accountPositions.map((p) => (p.ticket === ticket ? { ...p, ...updates } : p));
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
    const exists = accountPositions.some((p) => p.ticket === ticket);
    if (!exists) return {};
    const filteredPositions = accountPositions.filter((p) => p.ticket !== ticket);
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
  };
}
