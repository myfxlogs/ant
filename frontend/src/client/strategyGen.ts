import { strategyGenClient } from './connect';
import type { GenerateStrategyChunk } from '../gen/ant/v1/strategy_generation_pb';

export interface StrategyGenInput {
  message: string;
  conversationId?: string;
  symbol?: string;
  timeframe?: string;
  templateId?: string;
  clarificationRound?: number;
  // Phase 3: feedback context (optional — absent = fresh mode)
  previousCode?: string;
  backtestMetricsJson?: string;
  feedbackMessage?: string;
}

export interface StrategyGenCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onQuestions: (questions: string[]) => void;
  onCode: (code: string) => void;
  onBacktestId: (runId: string) => void;
  onTemplate: (name: string) => void;
  // Phase 3: structured feedback output
  onAnalysis?: (text: string) => void;
  onAdvice?: (text: string) => void;
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
          // Phase 3: feedback context
          previousCode: input.previousCode || '',
          backtestMetricsJson: input.backtestMetricsJson || '',
          feedbackMessage: input.feedbackMessage || '',
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
  // Phase 3: structured feedback output
  if (chunk.analysis) {
    cbs.onAnalysis?.(chunk.analysis);
  }
  if (chunk.advice) {
    cbs.onAdvice?.(chunk.advice);
  }
  if (chunk.error) {
    cbs.onError(chunk.error);
  }
}
