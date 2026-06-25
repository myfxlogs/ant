import { codeAssistClient } from './connect';
import { create } from '@bufbuild/protobuf';
import {
  ReviseCodeRequestSchema,
  type ReviseCodeStreamChunk,
} from '../gen/ant/v1/code_assist_pb';

// Client for the lightweight code-assist ConnectRPC service.

export interface CodeChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface ReviseCodeInput {
  code: string;
  instruction: string;
  history?: CodeChatMessage[];
  locale?: string;
  sessionId?: string;
}

export interface ReviseCodeResult {
  text: string;
  python: string;
}

export interface ExplainCodeInput {
  code: string;
  locale?: string;
}

export interface RequiredParamSpec {
  key: string;
  required: boolean;
  default?: unknown;
  type?: 'int' | 'float' | 'str' | 'bool';
  suggested?: unknown;
}

export interface CodeQualityHint {
  category: string;   // "FUTURE_DATA_LEAK" | "MISSING_PARAM" | "UNREAD_PARAM"
  severity: string;   // "error" | "warn" | "info"
  message: string;
  line: number;
  snippet: string;
}

/** Backend-provided sweep dimension from @param annotations (zero-trust). */
export interface BackendSweepDim {
  key: string;
  type: 'int' | 'float';
  default: number;
  min: number;
  max: number;
  step: number;
  hasRange: boolean;
}

export interface ValidateExtendedResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  parameters: RequiredParamSpec[];
  parametersJson: string;  // JSON array of {name, type, default, label} from self.ctx.param()
  qualityHints: CodeQualityHint[];
  sweepDimensions: BackendSweepDim[];
  strategyDirectives: { key: string; value: string }[];
  strategyType: string;  // "run_dataframe" | "run_context"
}

const parseParamValue = (value: string, type?: RequiredParamSpec['type']) => {
  if (!value) return undefined;
  if (type === 'bool') return value === 'true';
  if (type === 'int' || type === 'float') {
    const n = Number(value);
    return Number.isFinite(n) ? n : undefined;
  }
  return value;
};

export interface ReviseStreamCallbacks {
  onDelta: (delta: string, done: boolean) => void;
  onResult: (python: string) => void;
  onError?: (err: unknown) => void;
}

export const codeAssistApi = {
  revise: async (input: ReviseCodeInput): Promise<ReviseCodeResult> => {
    const data = await codeAssistClient.reviseCode({
      code: input.code,
      instruction: input.instruction,
      history: input.history || [],
      locale: input.locale || '',
    });
    return { text: data.text || '', python: data.python || '' };
  },

  /** Streaming revise — shows LLM output in real-time. Returns an abort function. */
  reviseStream: (
    input: ReviseCodeInput,
    callbacks: ReviseStreamCallbacks,
  ): (() => void) => {
    const abortController = new AbortController();
    (async () => {
      try {
        const msg = create(ReviseCodeRequestSchema, {
          code: input.code,
          instruction: input.instruction,
          history: input.history || [],
          locale: input.locale || '',
          sessionId: input.sessionId || '',
        });
        const stream = codeAssistClient.reviseCodeStream(msg, { signal: abortController.signal });
        for await (const chunk of stream) {
          if (chunk.python) {
            callbacks.onDelta(chunk.delta, true);
            callbacks.onResult(chunk.python);
            break;
          }
          callbacks.onDelta(chunk.delta || '', chunk.done || false);
        }
      } catch (e: unknown) {
        const s = String(e);
        if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
        callbacks.onError?.(e);
      }
    })();
    return () => abortController.abort();
  },

  explain: async (input: ExplainCodeInput): Promise<string> => {
    const data = await codeAssistClient.explainCode({
      code: input.code,
      locale: input.locale || '',
    });
    return data.explanation || '';
  },

  validateExtended: async (code: string): Promise<ValidateExtendedResult> => {
    const data = await codeAssistClient.validateStrategyExtended({ code });

    // Read structured proto fields directly (Python backend — zero-trust).
    // quality_hints, sweep_dimensions, and strategy_directives are now
    // first-class proto message fields — no JSON-in-warnings parsing needed.
    return {
      valid: data.valid,
      errors: data.errors || [],
      warnings: data.warnings || [],
      qualityHints: (data.qualityHints || []).map((h) => ({
        category: h.category,
        severity: h.severity,
        message: h.message,
        line: h.line,
        snippet: h.snippet,
      })),
      sweepDimensions: (data.sweepDimensions || []).map((d) => ({
        key: d.key,
        type: (d.type || 'float') as 'int' | 'float',
        default: d.default,
        min: d.min,
        max: d.max,
        step: d.step,
        hasRange: d.hasRange,
      })),
      strategyDirectives: (data.strategyDirectives || []).map((d) => ({
        key: d.key,
        value: d.value,
      })),
      strategyType: (data as Record<string, unknown>).strategyType as string || 'run_context',
      parametersJson: (data as Record<string, unknown>).parametersJson as string || (data as Record<string, unknown>).parameters_json as string || '[]',
      parameters: (data.parameters || []).map((p) => ({
        key: p.key,
        required: p.required,
        type: (p.type || undefined) as RequiredParamSpec['type'],
        default: parseParamValue(p.defaultValue, p.type as RequiredParamSpec['type']),
        suggested: parseParamValue(p.suggestedValue, p.type as RequiredParamSpec['type']),
      })),
    };
  },
};
