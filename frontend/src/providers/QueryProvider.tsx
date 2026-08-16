import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useAuthStore } from '@/stores/authStore';

export function QueryProvider({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            retry: 2,
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <QueryCacheGuard>{children}</QueryCacheGuard>
    </QueryClientProvider>
  );
}

export function QueryCacheGuard({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const userId = useAuthStore((s) => s.user?.id);
  const prevUserId = useRef<string | undefined>(userId);

  useEffect(() => {
    if (prevUserId.current !== userId) {
      if (prevUserId.current !== undefined) {
        queryClient.clear();
      }
      prevUserId.current = userId;
    }
  }, [userId, queryClient]);

  return <>{children}</>;
}
