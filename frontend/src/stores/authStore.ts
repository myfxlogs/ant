import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { User } from '@/types/auth';

const REMEMBER_ME_KEY = 'auth-remember-me';
const TOKEN_KEY = 'auth-access-token';

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

/**
 * Persist only user + _rememberMe to localStorage.
 * accessToken is managed manually via setTokens/logout to support
 * the "remember me" feature: localStorage (persist across restarts) when
 * checked, sessionStorage (cleared on tab close) when unchecked.
 *
 * _rememberMe flag itself is ALWAYS in localStorage so that on rehydrate
 * we know which storage to read the token from (prevents 4b4564f7
 * "second refresh loses token" regression).
 */
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      _hasHydrated: false,
      _rememberMe: false,
      setUser: (user) => set({ user, isAuthenticated: !!user }),
      setAccessToken: (accessToken) => {
        persistToken(accessToken);
        set({ accessToken, isAuthenticated: true });
      },
      setTokens: (_accessToken, _refreshToken, user, _rememberMe) => {
        const remember = _rememberMe ?? false;
        localStorage.setItem(REMEMBER_ME_KEY, String(remember));
        writeToken(_accessToken, remember);
        set({
          accessToken: _accessToken,
          isAuthenticated: true,
          _hasHydrated: true,
          user: user || null,
          _rememberMe: remember,
        });
      },
      logout: () => {
        clearToken();
        localStorage.removeItem(REMEMBER_ME_KEY);
        set({ user: null, accessToken: null, isAuthenticated: false, _rememberMe: false });
      },
      setHydrated: (hydrated) => set({ _hasHydrated: hydrated }),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        user: state.user,
        _rememberMe: state._rememberMe,
      }),
      onRehydrateStorage: () => {
        return (state, error) => {
          if (error) {
            console.error('[AuthStore] Rehydration error:', error);
          }
          const remember = state?._rememberMe ?? localStorage.getItem(REMEMBER_ME_KEY) === 'true';
          const token = readToken(remember);
          const isAuth = !!state?.user && !!token;
          queueMicrotask(() => {
            useAuthStore.setState({
              _hasHydrated: true,
              isAuthenticated: isAuth,
              accessToken: token,
              _rememberMe: remember,
            });
          });
        };
      },
    }
  )
);

function writeToken(token: string, remember: boolean): void {
  if (remember) {
    sessionStorage.removeItem(TOKEN_KEY);
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
    sessionStorage.setItem(TOKEN_KEY, token);
  }
}

function readToken(remember: boolean): string | null {
  if (remember) {
    return localStorage.getItem(TOKEN_KEY);
  }
  return sessionStorage.getItem(TOKEN_KEY);
}

function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(TOKEN_KEY);
}

function persistToken(token: string): void {
  const remember = useAuthStore.getState()._rememberMe;
  writeToken(token, remember);
}
