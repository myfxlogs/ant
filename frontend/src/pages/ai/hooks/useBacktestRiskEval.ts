import { useEffect, useRef, useState } from 'react';
import { strategyApi } from '@/client/strategy';
import i18n from '@/i18n';

export type BacktestRiskEval = {
  loading: boolean;
  score?: number;
  level?: string;
  isReliable?: boolean;
  reasons?: string[];
  warnings?: string[];
};

export function useBacktestRiskEval(params: {
  enabled: boolean;
  templateId: string;
  accountId: string;
  symbol: string;
  timeframe: string;
  datasetId?: string;
  backtestSucceeded?: boolean;
}) {
  const { enabled, templateId, accountId, symbol, timeframe, datasetId, backtestSucceeded } = params;

  const [risk, setRisk] = useState<BacktestRiskEval>({ loading: false });
  const inflightRef = useRef(false);
  const lastKeyRef = useRef('');

  useEffect(() => {
    if (!enabled) return;
    let mounted = true;
    if (!backtestSucceeded) return () => {
      mounted = false;
    };
    if (!templateId || !accountId || !symbol || !timeframe) return () => {
      mounted = false;
    };

    const key = [templateId, accountId, symbol, timeframe, datasetId || ''].join('|');
    if (key === lastKeyRef.current) {
      return () => {
        mounted = false;
      };
    }
    if (inflightRef.current) {
      return () => {
        mounted = false;
      };
    }
    lastKeyRef.current = key;
    inflightRef.current = true;

    void (async () => {
      try {
        if (!mounted) return;
        setRisk((prev) => ({ ...prev, loading: true }));
        const resp = await strategyApi.runBacktest({
          templateId,
          accountId,
          symbol,
          timeframe,
          parameters: {},
          initialCapital: 10000,
          ...(datasetId ? { datasetId } : {}),
        } as any);

        if (!mounted) return;
        setRisk({
          loading: false,
          score: resp.riskScore,
          level: resp.riskLevel,
          isReliable: resp.isReliable,
          reasons: resp.riskReasons || [],
          warnings: resp.riskWarnings || [],
        });
      } catch (e: any) {
        if (!mounted) return;
        setRisk({
          loading: false,
          score: undefined,
          level: 'unknown',
          isReliable: false,
          reasons: [String(e?.message || e || i18n.t('ai.riskEval.failed'))],
          warnings: [],
        });
      } finally {
        inflightRef.current = false;
      }
    })();
    return () => {
      mounted = false;
    };
  }, [enabled, templateId, accountId, symbol, timeframe, datasetId, backtestSucceeded]);

  const reset = () => setRisk({ loading: false });

  return { risk, reset };
}
