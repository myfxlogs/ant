import { useQuery, type UseQueryOptions, type QueryKey } from '@tanstack/react-query';

export function useRpcQuery<TData, TError = Error>(
  queryKey: QueryKey,
  queryFn: () => Promise<TData>,
  options?: Omit<UseQueryOptions<TData, TError, TData, QueryKey>, 'queryKey' | 'queryFn'>,
) {
  return useQuery<TData, TError, TData, QueryKey>({
    queryKey,
    queryFn,
    staleTime: 30_000,
    retry: 2,
    refetchOnWindowFocus: false,
    ...options,
  });
}
