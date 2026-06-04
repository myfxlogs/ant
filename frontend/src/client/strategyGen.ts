import { strategyGenClient } from './connect';
import type { GenerateStrategyChunk } from '../gen/ant/v1/strategy_generation_pb';

export interface StrategyGenInput {
  message: string;
  conversationId?: string;
  symbol?: string;
  timeframe?: string;
  templateId?: string;
  clarificationRound?: number;
}

export interface StrategyGenCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onQuestions: (questions: string[]) => void;
  onCode: (code: string) => void;
  onBacktestId: (runId: string) => void;
  onTemplate: (name: string) => void;
  onError: (error: string) => void;
  onDone: () => void;
}

/** Streaming strategy generation — shows phases in real-time. Returns abort function. */
export function generateStrategyStream(
  input: StrategyGenInput,
  callbacks: StrategyGenCallbacks,
): () => void {
  const abortController = new AbortController();

  (async () => {
    try {
      const stream = strategyGenClient.generateStrategy(
        {
          conversationId: input.conversationId || '',
          message: input.message,
          symbol: input.symbol || '',
          timeframe: input.timeframe || '',
          templateId: input.templateId || '',
          clarificationRound: input.clarificationRound || 0,
        },
        { signal: abortController.signal },
      );

      for await (const chunk of stream) {
        handleChunk(chunk, callbacks);
      }
      callbacks.onDone();
    } catch (e: unknown) {
      const s = String(e);
      if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
      callbacks.onError(s);
    }
  })();

  return () => abortController.abort();
}

function handleChunk(chunk: GenerateStrategyChunk, cbs: StrategyGenCallbacks): void {
  cbs.onPhase(chunk.phase);

  if (chunk.questions?.length) {
    cbs.onQuestions(chunk.questions);
  }
  if (chunk.delta) {
    cbs.onDelta(chunk.delta);
  }
  if (chunk.templateName) {
    cbs.onTemplate(chunk.templateName);
  }
  if (chunk.code) {
    cbs.onCode(chunk.code);
  }
  if (chunk.backtestRunId) {
    cbs.onBacktestId(chunk.backtestRunId);
  }
  if (chunk.error) {
    cbs.onError(chunk.error);
  }
}
