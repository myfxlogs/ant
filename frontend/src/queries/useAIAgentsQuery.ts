import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { aiApi, type AIAgentDefinitionView } from '@/client/ai';

export function useAIAgentsQuery() {
  return useRpcQuery<AIAgentDefinitionView[]>(
    queryKeys.ai.agents.list(),
    () => aiApi.listAgents(),
    { staleTime: 5 * 60_000 },
  );
}
