import { create } from 'zustand';
import type { AccountWithUser } from '@/client/admin';

interface CachedAccountData {
  accounts: AccountWithUser[];
  total: number;
  timestamp: number;
}

interface AccountManagementState {
  cachedData: Map<string, CachedAccountData>;
  loading: boolean;
  getCachedData: (_paramsKey: string, _cacheTTL?: number) => CachedAccountData | null;
  setCachedData: (_paramsKey: string, _data: AccountWithUser[], _total: number) => void;
  invalidateCache: () => void;
  setLoading: (loading: boolean) => void;
}

const CACHE_TTL = 5 * 60 * 1000;

export const useAdminAccountStore = create<AccountManagementState>((set, get) => ({
  cachedData: new Map(),
  loading: false,
  
  getCachedData: (paramsKey: string, cacheTTL = CACHE_TTL) => {
    const { cachedData } = get();
    const cached = cachedData.get(paramsKey);
    if (!cached) return null;
    
    const now = Date.now();
    if (now - cached.timestamp > cacheTTL) {
      return null;
    }
    return cached;
  },
  
  setCachedData: (paramsKey: string, data: AccountWithUser[], total: number) => {
    set((state) => {
      const newCache = new Map(state.cachedData);
      newCache.set(paramsKey, {
        accounts: data,
        total,
        timestamp: Date.now(),
      });
      return { cachedData: newCache };
    });
  },
  
  invalidateCache: () => {
    set({ cachedData: new Map() });
  },
  
  setLoading: (loading) => set({ loading }),
}));
