import { codeAssistClient } from './connect';

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

export interface ValidateExtendedResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  parameters: RequiredParamSpec[];
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

  explain: async (input: ExplainCodeInput): Promise<string> => {
    const data = await codeAssistClient.explainCode({
      code: input.code,
      locale: input.locale || '',
    });
    return data.explanation || '';
  },

  validateExtended: async (code: string): Promise<ValidateExtendedResult> => {
    const data = await codeAssistClient.validateStrategyExtended({ code });
    return {
      valid: data.valid,
      errors: data.errors || [],
      warnings: data.warnings || [],
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
