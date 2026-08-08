import { useState, useCallback } from 'react';
import { message, Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import type { BacktestBlindSpotItem } from '@/components/backtest/backtestRunnerWatch';

function buildBlindSpotInstruction(blindSpots: BacktestBlindSpotItem[]): string {
  const fatalSpots = blindSpots.filter(b => b.severity === '致命' || b.severity === 'fatal');
  const otherSpots = blindSpots.filter(b => b.severity !== '致命' && b.severity !== 'fatal');

  const lines: string[] = [
    'Fix the following diagnostic issues found in this MQL trading strategy.',
    'Return the COMPLETE corrected code — do not omit any part.',
    '',
  ];

  if (fatalSpots.length > 0) {
    lines.push('**Critical issues (must fix):**');
    for (const bs of fatalSpots) {
      lines.push(`- [${bs.category || 'invariant'}] ${bs.description}${bs.location ? ` @ ${bs.location}` : ''}`);
    }
    lines.push('');
  }

  if (otherSpots.length > 0) {
    lines.push('**Quality issues (improve if possible):**');
    for (const bs of otherSpots) {
      lines.push(`- [${bs.category || 'quality'}] ${bs.description}${bs.location ? ` @ ${bs.location}` : ''}`);
    }
    lines.push('');
  }

  lines.push('Rules:');
  lines.push('1. Keep all existing logic unchanged unless it causes a diagnostic issue.');
  lines.push('2. Fix invariant violations (e.g. volume=0, negative prices) by ensuring correct calculations.');
  lines.push('3. Fix lookahead bias by using only past/current bar data (shift index >= 1 for future references).');
  lines.push('4. Return ONLY valid MQL code — no explanations, no markdown.');

  return lines.join('\n');
}

interface UseAIFixOptions {
  strategyId?: string;
  currentCode: string;
  onApplyCode: (code: string) => void;
  onRerunBacktest?: () => void;
}

export function useAIFix({ strategyId, currentCode, onApplyCode, onRerunBacktest }: UseAIFixOptions) {
  const { t } = useTranslation();
  const [aiFixing, setAIFixing] = useState(false);
  const [diffCode, setDiffCode] = useState<string | null>(null);
  const [diffOpen, setDiffOpen] = useState(false);

  const handleAIFix = useCallback(async (blindSpots: BacktestBlindSpotItem[]) => {
    if (!currentCode) {
      message.warning(t('strategy.backtest.diagnostic.noCode', 'No strategy code to fix'));
      return;
    }
    setAIFixing(true);
    try {
      const { codeAssistApi } = await import('@/client/codeAssist');
      const instruction = buildBlindSpotInstruction(blindSpots);
      const result = await codeAssistApi.revise({ code: currentCode, instruction });
      if (!result.text) {
        message.error(t('strategy.backtest.diagnostic.aiNoResult', 'AI returned no code'));
        return;
      }
      setDiffCode(result.text);
      setDiffOpen(true);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.backtest.diagnostic.aiFailed', 'AI fix failed'));
    } finally {
      setAIFixing(false);
    }
  }, [currentCode, t]);

  const handleApplyDiff = useCallback(async () => {
    if (!diffCode || !strategyId) {
      setDiffOpen(false);
      return;
    }
    try {
      const { strategyVersionApi } = await import('@/client/strategy_version');
      const result = await strategyVersionApi.updateCode(strategyId, diffCode, 'AI fix for blind spots', true);
      onApplyCode(diffCode);
      setDiffOpen(false);
      setDiffCode(null);
      if (result.compileSuccess) {
        message.success(t('strategy.backtest.diagnostic.fixApplied', 'Fix applied — re-running backtest'));
        onRerunBacktest?.();
      } else {
        message.warning(t('strategy.backtest.diagnostic.fixAppliedCompileWarn', 'Fix applied but compile has warnings'));
      }
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.backtest.diagnostic.applyFailed', 'Failed to apply fix'));
    }
  }, [diffCode, strategyId, onApplyCode, onRerunBacktest, t]);

  const handleCancelDiff = useCallback(() => {
    setDiffOpen(false);
    setDiffCode(null);
  }, []);

  const diffModal = (
    <Modal
      title={t('strategy.backtest.diagnostic.diffPreview', 'AI Fix Preview')}
      open={diffOpen}
      onOk={handleApplyDiff}
      onCancel={handleCancelDiff}
      okText={t('strategy.backtest.diagnostic.apply', 'Apply & Re-run')}
      cancelText={t('common.cancel', { defaultValue: 'Cancel' })}
      width={800}
    >
      {diffCode && (
        <div style={{ marginBottom: 8 }}>
          <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>
            {t('strategy.backtest.diagnostic.diffHint', 'Review the AI-generated code below. Apply to create a new version and re-run backtest.')}
          </div>
          <pre style={{
            maxHeight: 400, overflow: 'auto', fontSize: 12, padding: 12,
            background: '#f5f5f5', borderRadius: 8, whiteSpace: 'pre-wrap',
          }}>
            {diffCode}
          </pre>
        </div>
      )}
    </Modal>
  );

  return {
    aiFixing,
    handleAIFix,
    diffModal,
  };
}
