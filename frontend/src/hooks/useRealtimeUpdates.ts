import { useConnect } from '@/providers/useConnect';

export function useRealtimeUpdates(accountId: string | undefined) {
  const { isConnected, connectionState } = useConnect();

  return {
    streamConnected: isConnected,
    connectionState,
    accountId,
  };
}
