import { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { QueryProvider } from '@/providers/QueryProvider';
import { LocaleProvider } from '@/providers/LocaleProvider';
import { AppRoutes } from '@/routes/AppRoutes';

export default function App() {
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

  return (
    <LocaleProvider>
      <QueryProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </QueryProvider>
    </LocaleProvider>
  );
}
