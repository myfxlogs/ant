import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { User } from '@/types/auth';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  _hasHydrated: boolean;
  setUser: (_user: User | null) => void;
  setAccessToken: (_token: string) => void;
  setTokens: (_accessToken: string, _refreshToken: string, _user?: User) => void;
  logout: () => void;
  setHydrated: (_hydrated: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      _hasHydrated: false,
      setUser: (user) => set({ user, isAuthenticated: !!user }),
      setAccessToken: (accessToken) => set({ accessToken, isAuthenticated: true }),
      setTokens: (_accessToken, _refreshToken, user) => {
        set({
          accessToken: _accessToken,
          isAuthenticated: true,
          _hasHydrated: true,
          user: user || null,
        });
      },
      logout: () => set({ user: null, accessToken: null, isAuthenticated: false }),
      setHydrated: (hydrated) => set({ _hasHydrated: hydrated }),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
      }),
      onRehydrateStorage: () => {
        return (state, error) => {
          if (error) {
            console.error('[AuthStore] Rehydration error:', error);
          }
          if (state) {
            state._hasHydrated = true;
            state.isAuthenticated = !!state.user;
          }
        };
      },
    }
  )
);
