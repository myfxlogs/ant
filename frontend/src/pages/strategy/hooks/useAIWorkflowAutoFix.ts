import { message } from 'antd';
import { snapshotFromResult, diffIssues, isMQLCode } from './useAIWorkflow';

import type { IssueSnapshot } from './useAIWorkflow';

export type ParamHint = { key: string; required?: boolean; type?: string; default?: unknown; suggested?: unknown };

export type ValidationSnapshot = {
  errors?: string[];
  warnings?: string[];
  parameters?: ParamHint[];
  qualityHints?: { category: string; message: string; line: number }[];
  valid?: boolean;
};

export function buildFixInstruction(code: string, errors: string[], warnings: string[], params: ParamHint[]): string {
  const paramHints = params
    .filter(p => p.required || p.suggested !== undefined)
    .map(p => {
      const parts = [`@param ${p.key}`];
      if (p.type) parts.push(`type=${p.type}`);
      if (p.default !== undefined) parts.push(`default=${p.default}`);
      if (p.suggested !== undefined) parts.push(`suggested=${p.suggested}`);
      return parts.join(' ');
    });
  const errorsText = errors.map(e => `- ${e}`).join('\n');
  const warningsText = warnings.map(w => `- ${w}`).join('\n');
  const strategyLang = isMQLCode(code) ? 'MQL4' : 'Go';
  return [
    `Fix ALL of the following validation errors in this ${strategyLang} trading strategy.`,
    'Return the COMPLETE corrected code — do not omit any part.',
    '', '**Validation errors to fix:**', errorsText || '(none)', '',
    warningsText ? '**Warnings:**' : '', warningsText || '',
    '', paramHints.length ? '**Required parameters (add @param annotations at top of code):**' : '',
    ...paramHints.map(p => `  ${p}`),
    '', 'Rules:', '1. Keep all existing logic unchanged unless it causes an error.',
    '2. Add missing @param annotations with reasonable defaults.',
    '3. Fix calculation errors (EMA, data length checks, etc).',
    `4. Return ONLY valid ${strategyLang} code — no explanations, no markdown.`,
  ].filter(Boolean).join('\n');
}

export interface AutoFixIterationResult {
  done: boolean;
  passed: boolean;
  debug: { iterations: number; passed: boolean } & Record<string, unknown>;
  code?: string;
  errors?: string[];
  warnings?: string[];
  params?: ParamHint[];
  qualityHints?: { category: string; message: string; line: number }[];
}

export async function runAutoFixIteration(
  code: string,
  instruction: string,
  iter: number,
  maxIters: number,
  preSnapshot: IssueSnapshot,
  codeCtx: { setCode: (s: string) => void; setLastValidatedCode: (s: string) => void; setValidationResult: (r: ValidationSnapshot) => void },
  t: (k: string, o?: Record<string, unknown>) => string,
): Promise<AutoFixIterationResult> {
  const { codeAssistApi } = await import('@/client/codeAssist');
  const result = await codeAssistApi.revise({ code, instruction });
  if (!result.text) throw new Error('AI returned no code');
  code = result.text;
  const recheck = await codeAssistApi.validateExtended(code);
  const postSnapshot = snapshotFromResult(recheck);

  if (recheck.valid) {
    codeCtx.setCode(code);
    codeCtx.setLastValidatedCode(code);
    codeCtx.setValidationResult(recheck);
    const diff = diffIssues(preSnapshot, postSnapshot);
    message.success(t('strategy.validate.autoFixPassed', { defaultValue: 'Auto-fix passed after {{iter}} iteration(s)', iter, count: iter }));
    return { done: true, passed: true, debug: { iterations: iter, passed: true, ...diff } };
  }

  codeCtx.setCode(code);

  if (iter === maxIters) {
    codeCtx.setValidationResult(recheck);
    const diff = diffIssues(preSnapshot, postSnapshot);
    message.warning(t('strategy.validate.autoFixRemaining', { defaultValue: 'Auto-fix: {{count}} issue(s) remain after {{maxIters}} iterations', count: recheck.errors?.length ?? 0, maxIters }));
    return { done: true, passed: false, debug: { iterations: maxIters, passed: false, ...diff }, code, errors: recheck.errors, warnings: recheck.warnings, params: recheck.parameters, qualityHints: recheck.qualityHints };
  }

  return { done: false, passed: false, debug: { iterations: 0, passed: false }, code, errors: recheck.errors, warnings: recheck.warnings, params: recheck.parameters, qualityHints: recheck.qualityHints };
}

export interface AutoFixState {
  code: string;
  errors: string[];
  warnings: string[];
  params: ParamHint[];
  qualityHints: { category: string; message: string; line: number }[];
}

export function initAutoFixState(vr: ValidationSnapshot, currentCode: string): AutoFixState {
  return {
    code: currentCode,
    errors: vr.errors ?? [],
    warnings: vr.warnings ?? [],
    params: vr.parameters ?? [],
    qualityHints: vr.qualityHints ?? [],
  };
}

export function updateAutoFixState(state: AutoFixState, result: AutoFixIterationResult): void {
  state.code = result.code ?? state.code;
  state.errors = result.errors ?? [];
  state.warnings = result.warnings ?? [];
  state.params = result.params ?? [];
  state.qualityHints = result.qualityHints ?? [];
}
