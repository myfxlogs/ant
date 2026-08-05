import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { User } from '@/types/auth';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  _hasHydrated: boolean;
  _rememberMe: boolean;
  setUser: (_user: User | null) => void;
  setAccessToken: (_token: string) => void;
  setTokens: (_accessToken: string, _refreshToken: string, _user?: User, _rememberMe?: boolean) => void;
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
      _rememberMe: false,
      setUser: (user) => set({ user, isAuthenticated: !!user }),
      setAccessToken: (accessToken) => set({ accessToken, isAuthenticated: true }),
      setTokens: (_accessToken, _refreshToken, user, _rememberMe) => {
        set({
          accessToken: _accessToken,
          isAuthenticated: true,
          _hasHydrated: true,
          user: user || null,
          _rememberMe: _rememberMe ?? false,
        });
      },
      logout: () => set({ user: null, accessToken: null, isAuthenticated: false, _rememberMe: false }),
      setHydrated: (hydrated) => set({ _hasHydrated: hydrated }),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken || undefined,
        _rememberMe: state._rememberMe,
      }),
      onRehydrateStorage: () => {
        return (state, error) => {
          if (error) {
            console.error('[AuthStore] Rehydration error:', error);
          }
          const isAuth = !!(state as AuthState | undefined)?.user && !!(state as AuthState | undefined)?.accessToken;
          queueMicrotask(() => {
            useAuthStore.setState({ _hasHydrated: true, isAuthenticated: isAuth });
          });
        };
      },
    }
  )
);
