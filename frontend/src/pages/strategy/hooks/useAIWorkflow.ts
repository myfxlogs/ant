import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { BacktestMetrics } from './useBacktestParams';

interface AICodeContext {
  code: string;
  setCode: (code: string) => void;
  validationResult: { valid: boolean; errors?: string[]; warnings?: string[]; parameters?: { key: string; type?: string; required?: boolean; default?: unknown; suggested?: unknown }[] } | null;
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
    setAutoFixing(true);
    const maxIters = 3;
    let code = currentCode;
    let lastErrors: string[] = vr.errors || [];
    let lastWarnings: string[] = vr.warnings || [];
    let lastParams = vr.parameters || [];
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
          if (recheck.valid) {
            codeCtx.setCode(code);
            codeCtx.setLastValidatedCode(code);
            codeCtx.setValidationResult(recheck);
            message.success(`Auto-fix passed after ${iter} iteration${iter > 1 ? 's' : ''}`);
            setAutoFixing(false);
            return;
          }
          lastErrors = recheck.errors || [];
          lastWarnings = recheck.warnings || [];
          lastParams = recheck.parameters || [];
          codeCtx.setCode(code);
        } catch (e: unknown) {
          if (iter < maxIters) continue;
          throw e;
        }
      }
      codeCtx.setCode(code);
      codeCtx.setValidationResult({ valid: false, errors: lastErrors, warnings: lastWarnings, parameters: lastParams });
      message.warning(`Auto-fix: ${lastErrors.length} issue(s) remain after ${maxIters} iterations`);
    } catch (e: unknown) {
      message.error((e as Error)?.message || 'Auto-fix failed');
    } finally {
      setAutoFixing(false);
    }
  }, [codeCtx.code, codeCtx.validationResult, codeCtx.setCode, codeCtx.setLastValidatedCode, codeCtx.setValidationResult]);

  return {
    aiOptimizePrompt, chatAutoApply, autoFixing,
    handleAIOptimize, handleAskAIForValidation, handleAutoFix,
    setAiOptimizePrompt,
  };
}
