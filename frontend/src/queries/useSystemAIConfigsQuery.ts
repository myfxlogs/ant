import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { listSystemAIConfigs } from '@/pages/ai/systemai/api';

export function useSystemAIConfigsQuery() {
  return useRpcQuery(queryKeys.systemAI.configs, () => listSystemAIConfigs(), {
    staleTime: 0, // always refetch on mount so AI Settings shows latest providers
  });
}
