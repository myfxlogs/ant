import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { BacktestMetrics } from './useBacktestParams';

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
  setValidationResult: (r: any) => void;
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
    if (!vr || vr.valid) return;
    const errors = (vr.errors || []).map(e => `- ${e}`).join('\n');
    const warnings = (vr.warnings || []).map(w => `- ${w}`).join('\n');
    const prompt = [
      'I need help understanding and fixing validation issues in my Python trading strategy.',
      'Please analyze these issues and ask me clarifying questions about my trading logic,',
      'so I can explain what I intended. Help me fix them step by step.',
      '', '**Validation errors:**', errors || '(none)', '',
      warnings ? '**Warnings:**' : '', warnings || '',
      '', 'Please ask me one question at a time. Do not rewrite the code yet — help me understand the problems first.',
    ].filter(Boolean).join('\n');
    setChatAutoApply(false);
    setAiOptimizePrompt(prompt);
    setCodePanelVisible(true);
  }, [codeCtx.validationResult, setCodePanelVisible]);

  const handleAutoFix = useCallback(async () => {
    const vr = codeCtx.validationResult;
    const currentCode = codeCtx.code;
    if (!vr || vr.valid || !currentCode) return;

    // Snapshot pre-fix state for debug card.
    const preSnapshot = snapshotFromResult(vr);

    setAutoFixing(true);
    setAutoFixDebug(null);
    const maxIters = 3;
    let code = currentCode;
    let lastErrors: string[] = vr.errors || [];
    let lastWarnings: string[] = vr.warnings || [];
    let lastParams = vr.parameters || [];
    let lastQualityHints: { category: string; message: string; line: number }[] =
      (vr as any).qualityHints || [];
    try {
      for (let iter = 1; iter <= maxIters; iter++) {
        const paramHints = lastParams
          .filter(p => p.required || p.suggested !== undefined)
          .map(p => {
            const parts = [`@param ${p.key}`];
            if (p.type) parts.push(`type=${p.type}`);
            if (p.default !== undefined) parts.push(`default=${p.default}`);
            if (p.suggested !== undefined) parts.push(`suggested=${p.suggested}`);
            return parts.join(' ');
          });
        const errorsText = lastErrors.map(e => `- ${e}`).join('\n');
        const warningsText = lastWarnings.map(w => `- ${w}`).join('\n');
        const instruction = [
          'Fix ALL of the following validation errors in this Python trading strategy.',
          'Return the COMPLETE corrected code — do not omit any part.',
          '', '**Validation errors to fix:**', errorsText || '(none)', '',
          warningsText ? '**Warnings:**' : '', warningsText || '',
          '', paramHints.length ? '**Required parameters (add @param annotations at top of code):**' : '',
          ...paramHints.map(p => `  ${p}`),
          '', 'Rules:', '1. Keep all existing logic unchanged unless it causes an error.',
          '2. Add missing @param annotations with reasonable defaults.',
          '3. Fix calculation errors (EMA, data length checks, etc).',
          '4. Return ONLY valid Python code — no explanations, no markdown.',
        ].filter(Boolean).join('\n');
        try {
          const { codeAssistApi } = await import('@/client/codeAssist');
          const result = await codeAssistApi.revise({ code, instruction });
          if (!result.python) throw new Error('AI returned no code');
          code = result.python;
          const recheck = await codeAssistApi.validateExtended(code);
          const postSnapshot = snapshotFromResult(recheck);

          if (recheck.valid) {
            codeCtx.setCode(code);
            codeCtx.setLastValidatedCode(code);
            codeCtx.setValidationResult(recheck);
            const diff = diffIssues(preSnapshot, postSnapshot);
            setAutoFixDebug({ iterations: iter, passed: true, ...diff });
            message.success(`Auto-fix passed after ${iter} iteration${iter > 1 ? 's' : ''}`);
            setAutoFixing(false);
            return;
          }
          lastErrors = recheck.errors || [];
          lastWarnings = recheck.warnings || [];
          lastParams = recheck.parameters || [];
          lastQualityHints = recheck.qualityHints || [];
          codeCtx.setCode(code);

          // On last iteration, compute debug diff for the final state.
          if (iter === maxIters) {
            codeCtx.setValidationResult(recheck);
            const diff = diffIssues(preSnapshot, postSnapshot);
            setAutoFixDebug({ iterations: maxIters, passed: false, ...diff });
            message.warning(`Auto-fix: ${lastErrors.length} issue(s) remain after ${maxIters} iterations`);
          }
        } catch (e: unknown) {
          if (iter < maxIters) continue;
          throw e;
        }
      }
    } catch (e: unknown) {
      message.error((e as Error)?.message || 'Auto-fix failed');
    } finally {
      setAutoFixing(false);
    }
  }, [codeCtx.code, codeCtx.validationResult, codeCtx.setCode, codeCtx.setLastValidatedCode, codeCtx.setValidationResult]);

  return {
    aiOptimizePrompt, chatAutoApply, autoFixing, autoFixDebug, dismissDebug,
    handleAIOptimize, handleAskAIForValidation, handleAutoFix,
    setAiOptimizePrompt,
  };
}
