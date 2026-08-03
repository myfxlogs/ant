import { useState, useCallback } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import type { BacktestMetrics } from './useBacktestParams';

function isMQLCode(code: string): boolean {
  if (code.includes('package ') && code.includes('import (')) return false;
  return code.includes('OnTick') || code.includes('OnBar') ||
    code.includes('OnInit') || code.includes('extern ') ||
    code.includes('input ') || code.includes('#property');
}

/** Snapshot of issues at a point in time (used for before/after comparison). */
interface IssueSnapshot {
  errors: string[];
  warnings: string[];
  hints: { category: string; message: string; line: number }[];
}

/** Debug summary shown after auto-fix completes. */
export interface AutoFixDebug {
  iterations: number;
  passed: boolean;
  /** Issues that existed before but were resolved. */
  fixed: IssueSnapshot;
  /** Issues that still exist after all iterations. */
  remaining: IssueSnapshot;
  /** Issues that were NOT present before but appeared after fix (regressions). */
  introduced: IssueSnapshot;
}

function snapshotFromResult(r: {
  errors?: string[]; warnings?: string[];
  qualityHints?: { category: string; message: string; line: number }[];
}): IssueSnapshot {
  return {
    errors: [...(r.errors || [])],
    warnings: [...(r.warnings || [])],
    hints: (r.qualityHints || []).map(h => ({
      category: h.category, message: h.message, line: h.line,
    })),
  };
}

/** Compute diff: (before minus after) = fixed, (after minus before) = introduced, (intersection) = remaining. */
function diffIssues(before: IssueSnapshot, after: IssueSnapshot): { fixed: IssueSnapshot; remaining: IssueSnapshot; introduced: IssueSnapshot } {
  const setFrom = (errors: string[]) => new Set(errors);
  const hintSet = (hints: { category: string; message: string; line: number }[]) =>
    new Set(hints.map(h => `${h.category}::${h.message}::${h.line}`));

  const bErr = setFrom(before.errors);
  const aErr = setFrom(after.errors);
  const bWarn = setFrom(before.warnings);
  const aWarn = setFrom(after.warnings);
  const bHint = hintSet(before.hints);
  const aHint = hintSet(after.hints);

  return {
    fixed: {
      errors: before.errors.filter(e => !aErr.has(e)),
      warnings: before.warnings.filter(w => !aWarn.has(w)),
      hints: before.hints.filter(h => !aHint.has(`${h.category}::${h.message}::${h.line}`)),
    },
    remaining: {
      errors: after.errors.filter(e => bErr.has(e)),
      warnings: after.warnings.filter(w => bWarn.has(w)),
      hints: after.hints.filter(h => bHint.has(`${h.category}::${h.message}::${h.line}`)),
    },
    introduced: {
      errors: after.errors.filter(e => !bErr.has(e)),
      warnings: after.warnings.filter(w => !bWarn.has(w)),
      hints: after.hints.filter(h => !bHint.has(`${h.category}::${h.message}::${h.line}`)),
    },
  };
}

interface AICodeContext {
  code: string;
  setCode: (code: string) => void;
  validationResult: { valid: boolean; errors?: string[]; warnings?: string[]; parameters?: { key: string; type?: string; required?: boolean; default?: unknown; suggested?: unknown }[]; qualityHints?: { category: string; message: string; line: number }[] } | null;
  setValidationResult: (r: unknown) => void;
  setLastValidatedCode: (code: string) => void;
  loadTemplates: () => Promise<void>;
}

export function useAIWorkflow(
  codeCtx: AICodeContext,
  btMetrics: BacktestMetrics | null,
  setCodePanelVisible: (v: boolean) => void,
) {
  const [aiOptimizePrompt, setAiOptimizePrompt] = useState<string | null>(null);
  const [chatAutoApply, setChatAutoApply] = useState(true);
  const [autoFixing, setAutoFixing] = useState(false);
  const [autoFixDebug, setAutoFixDebug] = useState<AutoFixDebug | null>(null);
  const { t } = useTranslation();
  const dismissDebug = useCallback(() => setAutoFixDebug(null), []);

  const handleAIOptimize = useCallback(() => {
    const m = btMetrics;
    if (!m) return;
    const prompt = [
      'Optimize the strategy based on these backtest results:',
      `Total Return: ${((m.totalReturn ?? 0) * 100).toFixed(2)}%`,
      `Max Drawdown: ${((m.maxDrawdown ?? 0) * 100).toFixed(2)}%`,
      `Sharpe Ratio: ${(m.sharpeRatio ?? 0).toFixed(3)}`,
      `Win Rate: ${((m.winRate ?? 0) * 100).toFixed(1)}%`,
      `Total Trades: ${m.totalTrades ?? 0}`,
      'Please suggest parameter improvements and generate the updated strategy code.',
    ].join('\n');
    setChatAutoApply(true);
    setAiOptimizePrompt(prompt);
    setCodePanelVisible(true);
  }, [btMetrics, setCodePanelVisible]);

  const handleAskAIForValidation = useCallback(() => {
    const vr = codeCtx.validationResult;
    const strategyLang = isMQLCode(codeCtx.code) ? 'MQL' : 'Go';
    let prompt: string;
    if (vr && !vr.valid) {
      const errors = (vr.errors || []).map(e => `- ${e}`).join('\n');
      const warnings = (vr.warnings || []).map(w => `- ${w}`).join('\n');
      prompt = [
        `I need help fixing validation issues in my ${strategyLang} strategy.`,
        '', '**Errors:**', errors || '(none)', '',
        warnings ? '**Warnings:**\n' + warnings : '',
        '', 'Ask me one question at a time.',
      ].filter(Boolean).join('\n');
    } else {
      prompt = [
        `Please review my ${strategyLang} trading strategy below.`,
        'Analyze the logic, check for common pitfalls (look-ahead bias, overfitting, position sizing),',
        'and suggest improvements. Be specific and actionable.',
      ].join('\n');
    }
    setChatAutoApply(false);
    setAiOptimizePrompt(prompt);
    setCodePanelVisible(true);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- codeCtx.code accessed via closure, not reactive  | REF: rd.md#part-0.2-hooks-deps
  }, [codeCtx.validationResult, setCodePanelVisible]);

  const handleAutoFix = useCallback(async () => {
    const vr = codeCtx.validationResult;
    const currentCode = codeCtx.code;
    if (!vr || vr.valid || !currentCode) return;

    const preSnapshot = snapshotFromResult(vr);
    setAutoFixing(true);
    setAutoFixDebug(null);
    const maxIters = 3;
    const state = initAutoFixState(vr, currentCode);
    try {
      for (let iter = 1; iter <= maxIters; iter++) {
        const instruction = buildFixInstruction(state.code, state.errors, state.warnings, state.params);
        const result = await runAutoFixIteration(state.code, instruction, iter, maxIters, preSnapshot, codeCtx, t);
        if (result.done) {
          setAutoFixDebug(result.debug as unknown as AutoFixDebug);
          if (result.passed) setAutoFixing(false);
          return;
        }
        updateAutoFixState(state, result);
      }
    } catch (e: unknown) {
      message.error((e as Error)?.message || t('strategy.validate.autoFixFailed', { defaultValue: 'Auto-fix failed' }));
    } finally {
      setAutoFixing(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional subset of codeCtx to prevent infinite loop  | REF: rd.md#part-0.2-hooks-deps
  }, [codeCtx.code, codeCtx.validationResult, codeCtx.setCode, codeCtx.setLastValidatedCode, codeCtx.setValidationResult]);

  return {
    aiOptimizePrompt, chatAutoApply, autoFixing, autoFixDebug, dismissDebug,
    handleAIOptimize, handleAskAIForValidation, handleAutoFix,
    setAiOptimizePrompt,
  };
}

type ParamHint = { key: string; required?: boolean; type?: string; default?: unknown; suggested?: unknown };

type ValidationSnapshot = {
  errors?: string[];
  warnings?: string[];
  parameters?: ParamHint[];
  qualityHints?: { category: string; message: string; line: number }[];
  valid?: boolean;
};

function buildFixInstruction(code: string, errors: string[], warnings: string[], params: ParamHint[]): string {
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

interface AutoFixIterationResult {
  done: boolean;
  passed: boolean;
  debug: { iterations: number; passed: boolean } & Record<string, unknown>;
  code?: string;
  errors?: string[];
  warnings?: string[];
  params?: ParamHint[];
  qualityHints?: { category: string; message: string; line: number }[];
}

async function runAutoFixIteration(
  code: string,
  instruction: string,
  iter: number,
  maxIters: number,
  preSnapshot: ReturnType<typeof snapshotFromResult>,
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

interface AutoFixState {
  code: string;
  errors: string[];
  warnings: string[];
  params: ParamHint[];
  qualityHints: { category: string; message: string; line: number }[];
}

function initAutoFixState(vr: ValidationSnapshot, currentCode: string): AutoFixState {
  return {
    code: currentCode,
    errors: vr.errors ?? [],
    warnings: vr.warnings ?? [],
    params: vr.parameters ?? [],
    qualityHints: vr.qualityHints ?? [],
  };
}

function updateAutoFixState(state: AutoFixState, result: AutoFixIterationResult): void {
  state.code = result.code ?? state.code;
  state.errors = result.errors ?? [];
  state.warnings = result.warnings ?? [];
  state.params = result.params ?? [];
  state.qualityHints = result.qualityHints ?? [];
}
