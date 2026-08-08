import { useEffect, useRef } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { HelmetProvider } from 'react-helmet-async';
import { QueryProvider } from '@/providers/QueryProvider';
import { LocaleProvider } from '@/providers/LocaleProvider';
import { AppRoutes } from '@/routes/AppRoutes';
import { useAuthStore } from '@/stores/authStore';
import { resetAllStores } from '@/stores/resetAllStores';

export default function App() {
  const userId = useAuthStore((s) => s.user?.id);
  const prevUserId = useRef<string | undefined>(userId);

  // Proactive token-lifecycle: background refresh on user activity + tab visibility.
  // The transport interceptor also calls ensureFreshToken() per request.
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const { startTokenScheduler, ensureFreshToken } = await import('@/utils/tokenLifecycle');
      if (cancelled) return;
      await ensureFreshToken();
      startTokenScheduler();
    })();
    return () => { cancelled = true; };
  }, []);

  // Defensive guard: when user.id changes (including logout → null),
  // reset all user-scoped stores to prevent data leakage between users.
  useEffect(() => {
    if (prevUserId.current !== userId) {
      if (prevUserId.current !== undefined) {
        resetAllStores();
      }
      prevUserId.current = userId;
    }
  }, [userId]);

  return (
    <HelmetProvider>
      <LocaleProvider>
        <QueryProvider>
          <BrowserRouter>
            <AppRoutes />
          </BrowserRouter>
        </QueryProvider>
      </LocaleProvider>
    </HelmetProvider>
  );
}
