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
  const inflightSeqRef = useRef(0);
  const lastKeyRef = useRef('');

  useEffect(() => {
    let mounted = true;
    if (!enabled) {
      return () => { mounted = false; };
    }
    if (!backtestSucceeded) {
      return () => { mounted = false; };
    }
    if (!templateId || !accountId || !symbol || !timeframe) {
      return () => { mounted = false; };
    }

    const key = [templateId, accountId, symbol, timeframe, datasetId || ''].join('|');
    if (key === lastKeyRef.current) {
      return () => { mounted = false; };
    }
    if (inflightSeqRef.current > 0) {
      return () => { mounted = false; };
    }
    lastKeyRef.current = key;
    const seq = ++inflightSeqRef.current;

    void (async () => {
      try {
        if (!mounted) return;
        if (seq !== inflightSeqRef.current) return;
        setRisk((prev) => ({ ...prev, loading: true }));
        const backtestParams: Parameters<typeof strategyApi.runBacktest>[0] & { datasetId?: string } = {
          templateId,
          accountId,
          symbol,
          timeframe,
          parameters: {},
          initialCapital: 10000,
          ...(datasetId ? { datasetId } : {}),
        };
        const resp = await strategyApi.runBacktest(backtestParams);

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
          reasons: [e instanceof Error ? e.message : (typeof e === 'string' ? e : i18n.t('ai.riskEval.failed'))],
          warnings: [],
        });
      } finally {
        if (seq === inflightSeqRef.current) inflightSeqRef.current = 0;
      }
    })();
    return () => {
      mounted = false;
    };
  }, [enabled, templateId, accountId, symbol, timeframe, datasetId, backtestSucceeded]);

  const reset = () => { inflightSeqRef.current = 0; lastKeyRef.current = ''; setRisk({ loading: false }); };

  return { risk, reset };
}
