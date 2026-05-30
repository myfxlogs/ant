import type React from 'react';
import { toFriendlyAIError } from '@/client/ai';
import type { AIAgentDefinitionView } from '@/client/ai';
import type { StepKey, StepState } from './types';

export function emptyStep(): StepState {
  return { messages: [], extractedPrompt: '', promptDraft: '' };
}

export function stepKeyForAgent(agent: AIAgentDefinitionView): StepKey {
  return `agent:${agent.agentKey || agent.type}`;
}

export function mkId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
}

export function friendlyError(e: unknown): string {
  return toFriendlyAIError(e);
}

export function beginModelWait(
  setStartedAt: (n: number) => void,
  setElapsed: (n: number) => void,
) {
  setStartedAt(Date.now());
  setElapsed(0);
}

export function endModelWait(
  setStartedAt: (v: null) => void,
  setElapsed: (n: number) => void,
) {
  setStartedAt(null);
  setElapsed(0);
}

/**
 * Create a stream chunk merger that handles deduplication between
 * catch-up SSE (sends full prefix) and live SSE (sends deltas).
 */
export function createStreamMerger(
  setter: React.Dispatch<React.SetStateAction<string>>,
): (delta: string) => void {
  return (delta: string) => {
    if (!delta) return;
    setter((prev) => {
      if (prev === delta) return prev;
      if (prev === '') return delta;
      if (delta.startsWith(prev)) return delta;
      return prev + delta;
    });
  };
}
