import type { AIAgentDefinitionView } from '@/client/ai';
import type { V2Usage } from '@/client/debateV2';

export type ChatMessage = {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  isLoading?: boolean;
  /** 'kickoff' means a hidden system-handoff user turn; UI hides it. */
  kind?: 'kickoff';
};

/**
 * StepKey mirrors the backend step naming with an additional UI-only
 * 'agent_selection' sentinel used before the session exists.
 */
export type StepKey = 'agent_selection' | 'intent' | `agent:${string}` | 'code';

export interface StepState {
  messages: ChatMessage[];
  extractedPrompt: string;
  promptDraft: string;
}

export interface CodeState {
  text: string;
  python: string;
  loading: boolean;
  elapsedSeconds: number;
}

export interface UseDebateFlowResult {
  currentStep: StepKey;
  stepIndex: number;
  stepLabels: Array<{ key: StepKey; label: string }>;
  selectedAgents: AIAgentDefinitionView[];
  sending: boolean;

  /** True while a unary RPC that calls the LLM is in flight (chat / advance / reject). */
  modelWaitActive: boolean;
  /** Monotonic seconds since modelWait started; 0 when inactive. */
  modelWaitElapsedSeconds: number;

  stepState: (key: StepKey) => StepState;
  updatePromptDraft: (key: StepKey, text: string) => void;

  setSelectedAgents: (agents: AIAgentDefinitionView[]) => void;
  startFlow: () => Promise<void>;
  sendMessage: (text: string) => Promise<void>;
  advance: () => Promise<void>;
  back: () => Promise<void>;
  reset: () => void;
  rejectCode: (feedback: string) => Promise<void>;
  retryCodeGeneration: () => Promise<void>;

  code: CodeState;
  advanceStreamPreview: string;
  sessionId: string;
  provider: string;
  model: string;
  usage: V2Usage;
}
